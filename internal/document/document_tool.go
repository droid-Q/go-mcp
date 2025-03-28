package document

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// DocumentType 定义文档类型
type DocumentType string

const (
	// 支持的文档类型
	TypeText     DocumentType = "text"
	TypeHTML     DocumentType = "html"
	TypeJSON     DocumentType = "json"
	TypeMarkdown DocumentType = "markdown"
)

// Document 表示一个文档
type Document struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Content  string       `json:"content"`
	Type     DocumentType `json:"type"`
	Metadata interface{}  `json:"metadata,omitempty"`
}

// SummarizeParams 摘要参数
type SummarizeParams struct {
	DocumentID string `json:"document_id"`
	Content    string `json:"content,omitempty"`
	MaxLength  int    `json:"max_length,omitempty"`
	Format     string `json:"format,omitempty"`
}

// ConvertParams 转换参数
type ConvertParams struct {
	Content  string       `json:"content"`
	FromType DocumentType `json:"from_type"`
	ToType   DocumentType `json:"to_type"`
}

// ExtractParams 提取参数
type ExtractParams struct {
	Content string   `json:"content"`
	Type    string   `json:"type"`
	Fields  []string `json:"fields,omitempty"`
}

// DocumentTool 实现文档处理功能
type DocumentTool struct {
	documents map[string]Document
	nextID    int
}

// NewDocumentTool 创建新的文档工具
func NewDocumentTool() *DocumentTool {
	// 初始化一些模拟文档
	docs := map[string]Document{
		"doc-1": {
			ID:      "doc-1",
			Title:   "MCP服务器实现指南",
			Content: "# MCP服务器实现指南\n\n## 简介\n\nMCP(Model Context Protocol)是一种开放协议，用于标准化应用程序如何向大型语言模型(LLMs)提供上下文。本指南将帮助您实现自己的MCP服务器。\n\n## 核心组件\n\n1. 工具定义\n2. 工具执行\n3. 上下文管理\n4. 安全控制\n\n## 实现步骤\n\n...",
			Type:    TypeMarkdown,
		},
		"doc-2": {
			ID:      "doc-2",
			Title:   "API文档",
			Content: "<html><body><h1>MCP API文档</h1><p>这是API文档的HTML格式</p><ul><li>GET /tools - 获取可用工具列表</li><li>POST /execute - 执行工具</li></ul></body></html>",
			Type:    TypeHTML,
		},
	}

	return &DocumentTool{
		documents: docs,
		nextID:    3,
	}
}

// Name 实现Tool接口
func (t *DocumentTool) Name() string {
	return "document"
}

// Description 实现Tool接口
func (t *DocumentTool) Description() string {
	return "处理、转换和提取文档内容"
}

// ParameterSchema 实现Tool接口
func (t *DocumentTool) ParameterSchema() string {
	return `{
		"oneOf": [
			{
				"type": "object",
				"properties": {
					"action": {
						"type": "string",
						"enum": ["summarize"],
						"description": "摘要操作"
					},
					"document_id": {
						"type": "string",
						"description": "要摘要的文档ID"
					},
					"content": {
						"type": "string",
						"description": "要摘要的内容（如果没有提供document_id）"
					},
					"max_length": {
						"type": "integer",
						"description": "摘要的最大长度"
					},
					"format": {
						"type": "string",
						"enum": ["text", "bullet_points", "json"],
						"description": "摘要的格式"
					}
				},
				"required": ["action"]
			},
			{
				"type": "object",
				"properties": {
					"action": {
						"type": "string",
						"enum": ["convert"],
						"description": "转换操作"
					},
					"content": {
						"type": "string",
						"description": "要转换的内容"
					},
					"from_type": {
						"type": "string",
						"enum": ["text", "html", "json", "markdown"],
						"description": "源格式"
					},
					"to_type": {
						"type": "string",
						"enum": ["text", "html", "json", "markdown"],
						"description": "目标格式"
					}
				},
				"required": ["action", "content", "from_type", "to_type"]
			},
			{
				"type": "object",
				"properties": {
					"action": {
						"type": "string",
						"enum": ["extract"],
						"description": "提取操作"
					},
					"content": {
						"type": "string",
						"description": "要提取的内容"
					},
					"type": {
						"type": "string",
						"enum": ["entities", "keywords", "structured_data"],
						"description": "提取类型"
					},
					"fields": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "要提取的字段（用于structured_data类型）"
					}
				},
				"required": ["action", "content", "type"]
			}
		]
	}`
}

// Execute 实现Tool接口
func (t *DocumentTool) Execute(paramsJSON json.RawMessage) (interface{}, error) {
	// 首先解析动作类型
	var baseParams struct {
		Action string `json:"action"`
	}

	if err := json.Unmarshal(paramsJSON, &baseParams); err != nil {
		return nil, errors.New("无效的参数: " + err.Error())
	}

	switch baseParams.Action {
	case "summarize":
		var params SummarizeParams
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, errors.New("无效的摘要参数: " + err.Error())
		}
		return t.summarize(params)

	case "convert":
		var params ConvertParams
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, errors.New("无效的转换参数: " + err.Error())
		}
		return t.convert(params)

	case "extract":
		var params ExtractParams
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, errors.New("无效的提取参数: " + err.Error())
		}
		return t.extract(params)

	default:
		return nil, fmt.Errorf("不支持的操作: %s", baseParams.Action)
	}
}

// summarize 模拟文档摘要功能
func (t *DocumentTool) summarize(params SummarizeParams) (interface{}, error) {
	// 获取内容
	content := params.Content
	if content == "" && params.DocumentID != "" {
		doc, exists := t.documents[params.DocumentID]
		if !exists {
			return nil, fmt.Errorf("文档不存在: %s", params.DocumentID)
		}
		content = doc.Content
	}

	if content == "" {
		return nil, errors.New("没有提供内容")
	}

	// 根据内容类型提取纯文本
	// 简化处理，实际应用中应根据文档类型采用不同的解析方法
	plainText := content

	// 模拟摘要（简单截取）
	maxLength := 200
	if params.MaxLength > 0 {
		maxLength = params.MaxLength
	}

	var summary string
	if len(plainText) > maxLength {
		summary = plainText[:maxLength] + "..."
	} else {
		summary = plainText
	}

	// 格式化输出
	format := params.Format
	if format == "" {
		format = "text"
	}

	switch format {
	case "text":
		return map[string]string{"summary": summary}, nil

	case "bullet_points":
		// 简单分割成句子，然后构建项目符号列表
		sentences := strings.Split(summary, ".")
		bulletPoints := make([]string, 0)

		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence != "" {
				bulletPoints = append(bulletPoints, "• "+sentence)
			}
		}

		return map[string][]string{"bullet_points": bulletPoints}, nil

	case "json":
		// 已经是JSON格式的映射，无需额外处理
		return map[string]string{"summary": summary}, nil

	default:
		return nil, fmt.Errorf("不支持的格式: %s", format)
	}
}

// convert 模拟文档格式转换
func (t *DocumentTool) convert(params ConvertParams) (interface{}, error) {
	if params.Content == "" {
		return nil, errors.New("内容不能为空")
	}

	// 验证支持的类型
	if !isValidDocType(params.FromType) || !isValidDocType(params.ToType) {
		return nil, errors.New("不支持的文档类型")
	}

	// 如果源类型和目标类型相同，直接返回
	if params.FromType == params.ToType {
		return map[string]string{"content": params.Content}, nil
	}

	var convertedContent string

	// 实际应用中，应该使用专门的库进行转换
	// 这里只是简单模拟一些转换操作
	switch {
	case params.FromType == TypeMarkdown && params.ToType == TypeHTML:
		// Markdown转HTML（简单模拟）
		content := params.Content
		// 处理标题
		content = strings.ReplaceAll(content, "# ", "<h1>") + "</h1>"
		content = strings.ReplaceAll(content, "## ", "<h2>") + "</h2>"
		// 处理段落
		paragraphs := strings.Split(content, "\n\n")
		for i, p := range paragraphs {
			if !strings.HasPrefix(p, "<h") {
				paragraphs[i] = "<p>" + p + "</p>"
			}
		}
		convertedContent = "<html><body>" + strings.Join(paragraphs, "") + "</body></html>"

	case params.FromType == TypeHTML && params.ToType == TypeText:
		// HTML转纯文本（简单模拟）
		content := params.Content
		// 移除HTML标签
		content = strings.ReplaceAll(content, "<html>", "")
		content = strings.ReplaceAll(content, "</html>", "")
		content = strings.ReplaceAll(content, "<body>", "")
		content = strings.ReplaceAll(content, "</body>", "")
		content = strings.ReplaceAll(content, "<h1>", "")
		content = strings.ReplaceAll(content, "</h1>", "\n\n")
		content = strings.ReplaceAll(content, "<h2>", "")
		content = strings.ReplaceAll(content, "</h2>", "\n\n")
		content = strings.ReplaceAll(content, "<p>", "")
		content = strings.ReplaceAll(content, "</p>", "\n\n")
		convertedContent = content

	case params.FromType == TypeJSON && params.ToType == TypeText:
		// 假设JSON是格式化的，直接返回
		convertedContent = params.Content

	default:
		// 对于其他转换，返回带有注释的原始内容
		convertedContent = fmt.Sprintf("/* 从 %s 转换到 %s */\n%s",
			params.FromType, params.ToType, params.Content)
	}

	return map[string]string{
		"from_type": string(params.FromType),
		"to_type":   string(params.ToType),
		"content":   convertedContent,
	}, nil
}

// extract 模拟从文档中提取信息
func (t *DocumentTool) extract(params ExtractParams) (interface{}, error) {
	if params.Content == "" {
		return nil, errors.New("内容不能为空")
	}

	// 根据提取类型进行不同处理
	switch params.Type {
	case "entities":
		// 模拟实体提取（简单的关键词匹配）
		entities := map[string][]string{
			"person":       {"John", "Alice", "Bob"},
			"location":     {"New York", "London", "Beijing"},
			"date":         {"2023", "January", "Monday"},
			"organization": {"Google", "Microsoft", "Amazon"},
		}

		result := make(map[string][]string)

		content := strings.ToLower(params.Content)
		for category, words := range entities {
			matches := make([]string, 0)
			for _, word := range words {
				if strings.Contains(content, strings.ToLower(word)) {
					matches = append(matches, word)
				}
			}

			if len(matches) > 0 {
				result[category] = matches
			}
		}

		return result, nil

	case "keywords":
		// 模拟关键词提取（简单的频率统计）
		words := strings.Fields(strings.ToLower(params.Content))
		freqMap := make(map[string]int)

		// 统计词频
		for _, word := range words {
			// 清理单词
			word = strings.Trim(word, ".,!?;:()")
			if len(word) > 3 { // 忽略太短的单词
				freqMap[word]++
			}
		}

		// 构建关键词列表
		type keyword struct {
			Word  string `json:"word"`
			Count int    `json:"count"`
		}

		keywords := make([]keyword, 0, len(freqMap))
		for word, count := range freqMap {
			keywords = append(keywords, keyword{Word: word, Count: count})
		}

		// 按频率排序
		sort.Slice(keywords, func(i, j int) bool {
			return keywords[i].Count > keywords[j].Count
		})

		// 只返回前10个
		if len(keywords) > 10 {
			keywords = keywords[:10]
		}

		return map[string]interface{}{
			"keywords": keywords,
		}, nil

	case "structured_data":
		// 模拟结构化数据提取
		// 实际应用可能使用正则表达式或专门的解析器

		// 如果未指定字段，返回所有可能的结构化数据
		if len(params.Fields) == 0 {
			// 模拟从内容中提取的一些结构化数据
			data := map[string]interface{}{
				"title":      "从内容中提取的标题",
				"sections":   []string{"简介", "方法", "结果", "讨论"},
				"references": 12,
				"metadata": map[string]string{
					"author": "从内容中提取的作者",
					"date":   "从内容中提取的日期",
				},
			}
			return data, nil
		}

		// 如果指定了字段，只返回这些字段
		result := make(map[string]interface{})
		for _, field := range params.Fields {
			switch field {
			case "title":
				result["title"] = "从内容中提取的标题"
			case "sections":
				result["sections"] = []string{"简介", "方法", "结果", "讨论"}
			case "references":
				result["references"] = 12
			case "metadata":
				result["metadata"] = map[string]string{
					"author": "从内容中提取的作者",
					"date":   "从内容中提取的日期",
				}
			}
		}

		return result, nil

	default:
		return nil, fmt.Errorf("不支持的提取类型: %s", params.Type)
	}
}

// 辅助函数：验证文档类型是否合法
func isValidDocType(docType DocumentType) bool {
	validTypes := []DocumentType{TypeText, TypeHTML, TypeJSON, TypeMarkdown}
	for _, t := range validTypes {
		if docType == t {
			return true
		}
	}
	return false
}
