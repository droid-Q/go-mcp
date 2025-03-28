package tools

import (
	"encoding/json"
	"fmt"
	"log"
)

// ToolRequest 代表一个工具请求
type ToolRequest struct {
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters"`
}

// ToolResponse 代表一个工具响应
type ToolResponse struct {
	Status  string      `json:"status"`
	Content interface{} `json:"content,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Tool 定义MCP工具接口
type Tool interface {
	// Name 返回工具名称
	Name() string

	// Description 返回工具描述
	Description() string

	// ParameterSchema 返回参数JSON Schema
	ParameterSchema() string

	// Execute 执行工具并返回结果
	Execute(params json.RawMessage) (interface{}, error)
}

// ToolManager 管理MCP工具
type ToolManager struct {
	tools map[string]Tool
}

// NewToolManager 创建新的工具管理器
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册一个工具
func (tm *ToolManager) RegisterTool(tool Tool) {
	tm.tools[tool.Name()] = tool
	log.Printf("Tool registered: %s", tool.Name())
}

// ExecuteTool 执行工具请求
func (tm *ToolManager) ExecuteTool(request ToolRequest) ToolResponse {
	tool, exists := tm.tools[request.Name]
	if !exists {
		return ToolResponse{
			Status: "error",
			Error:  fmt.Sprintf("Tool '%s' not found", request.Name),
		}
	}

	result, err := tool.Execute(request.Parameters)
	if err != nil {
		return ToolResponse{
			Status: "error",
			Error:  err.Error(),
		}
	}

	return ToolResponse{
		Status:  "success",
		Content: result,
	}
}

// GetToolsSchema 获取所有工具的JSON Schema
func (tm *ToolManager) GetToolsSchema() []map[string]interface{} {
	schemas := make([]map[string]interface{}, 0, len(tm.tools))

	for _, tool := range tm.tools {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(tool.ParameterSchema()), &schema); err != nil {
			log.Printf("Error parsing schema for tool %s: %v", tool.Name(), err)
			continue
		}

		toolSchema := map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  schema,
		}

		schemas = append(schemas, toolSchema)
	}

	return schemas
}
