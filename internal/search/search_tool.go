package search

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// SearchResult 表示一个搜索结果
type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url,omitempty"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Source  string  `json:"source"`
}

// SearchParams 搜索参数
type SearchParams struct {
	Query        string   `json:"query"`
	MaxResults   int      `json:"max_results,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	FilterTerms  []string `json:"filter_terms,omitempty"`
	ExcludeTerms []string `json:"exclude_terms,omitempty"`
}

// SearchTool 实现搜索功能
type SearchTool struct {
	// 模拟数据库
	knowledgeBase []SearchResult
}

// NewSearchTool 创建新的搜索工具
func NewSearchTool() *SearchTool {
	// 初始化一些模拟数据
	mockData := []SearchResult{
		{
			Title:   "使用Go实现WebSocket服务器",
			URL:     "https://example.com/go-websocket",
			Content: "本文介绍如何使用Go语言的gorilla/websocket库实现高性能的WebSocket服务器。WebSocket是一种在单个TCP连接上进行全双工通信的协议...",
			Source:  "tutorials",
		},
		{
			Title:   "Go语言并发编程模式",
			URL:     "https://example.com/go-concurrency",
			Content: "Go语言的并发特性是其最强大的功能之一。Goroutines和channels使得编写并发程序变得简单而高效...",
			Source:  "articles",
		},
		{
			Title:   "Model Context Protocol介绍",
			URL:     "https://example.com/mcp-intro",
			Content: "MCP（Model Context Protocol）是一种开放协议，用于标准化应用程序如何向大型语言模型（LLMs）提供上下文。这允许模型以安全、受控的方式访问工具和数据源...",
			Source:  "documentation",
		},
		{
			Title:   "构建MCP服务器最佳实践",
			URL:     "https://example.com/mcp-best-practices",
			Content: "本文档提供了构建高效、安全、可扩展的Model Context Protocol服务器的最佳实践。包括身份验证、工具定义、错误处理和性能优化等方面...",
			Source:  "documentation",
		},
		{
			Title:   "大型语言模型安全访问策略",
			URL:     "https://example.com/llm-security",
			Content: "为LLMs提供外部工具访问权限带来了许多安全挑战。本文讨论如何实现适当的安全措施，包括输入验证、沙箱执行、访问控制和审计日志...",
			Source:  "security",
		},
	}

	return &SearchTool{
		knowledgeBase: mockData,
	}
}

// Name 实现Tool接口
func (t *SearchTool) Name() string {
	return "search"
}

// Description 实现Tool接口
func (t *SearchTool) Description() string {
	return "搜索知识库和文档以获取相关信息"
}

// ParameterSchema 实现Tool接口
func (t *SearchTool) ParameterSchema() string {
	return `{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "搜索查询"
			},
			"max_results": {
				"type": "integer",
				"description": "最大结果数",
				"default": 3
			},
			"sources": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"description": "限制搜索的来源"
			},
			"filter_terms": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"description": "必须包含的术语"
			},
			"exclude_terms": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"description": "必须排除的术语"
			}
		},
		"required": ["query"]
	}`
}

// Execute 实现Tool接口
func (t *SearchTool) Execute(paramsJSON json.RawMessage) (interface{}, error) {
	var params SearchParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return nil, errors.New("无效的搜索参数: " + err.Error())
	}

	if params.Query == "" {
		return nil, errors.New("搜索查询不能为空")
	}

	maxResults := 3
	if params.MaxResults > 0 {
		maxResults = params.MaxResults
	}

	// 执行模拟搜索
	results := t.search(params)

	// 限制结果数量
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// search 执行搜索并对结果评分
func (t *SearchTool) search(params SearchParams) []SearchResult {
	query := strings.ToLower(params.Query)
	results := make([]SearchResult, 0)

	// 遍历知识库并评分
	for _, item := range t.knowledgeBase {
		// 如果指定了来源过滤，检查当前项目是否匹配
		if len(params.Sources) > 0 {
			found := false
			for _, source := range params.Sources {
				if source == item.Source {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 检查排除项
		if containsAny(item.Title, params.ExcludeTerms) || containsAny(item.Content, params.ExcludeTerms) {
			continue
		}

		// 检查必需项
		if len(params.FilterTerms) > 0 {
			if !containsAll(item.Title, params.FilterTerms) && !containsAll(item.Content, params.FilterTerms) {
				continue
			}
		}

		// 计算相关性分数
		score := calculateScore(query, item)

		// 分数大于0表示有相关性
		if score > 0 {
			item.Score = score
			results = append(results, item)
		}
	}

	// 按相关性分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// calculateScore 计算文档相关性分数
func calculateScore(query string, item SearchResult) float64 {
	title := strings.ToLower(item.Title)
	content := strings.ToLower(item.Content)

	// 标题完全匹配得分最高
	if title == query {
		return 10.0
	}

	score := 0.0

	// 标题包含查询 (部分匹配)
	if strings.Contains(title, query) {
		score += 5.0
	}

	// 内容包含查询
	if strings.Contains(content, query) {
		score += 3.0
	}

	// 检查查询中的每个单词
	queryWords := strings.Fields(query)
	for _, word := range queryWords {
		if len(word) < 3 {
			continue // 忽略太短的单词
		}

		// 标题中单词匹配
		if strings.Contains(title, word) {
			score += 1.0
		}

		// 内容中单词匹配
		if strings.Contains(content, word) {
			score += 0.5
		}
	}

	return score
}

// containsAny 检查字符串是否包含任意给定的项
func containsAny(s string, terms []string) bool {
	if len(terms) == 0 {
		return false
	}

	s = strings.ToLower(s)
	for _, term := range terms {
		if strings.Contains(s, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// containsAll 检查字符串是否包含所有给定的项
func containsAll(s string, terms []string) bool {
	if len(terms) == 0 {
		return true
	}

	s = strings.ToLower(s)
	for _, term := range terms {
		if !strings.Contains(s, strings.ToLower(term)) {
			return false
		}
	}
	return true
}
