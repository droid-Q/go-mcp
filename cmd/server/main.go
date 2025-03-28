package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/droid/go-mcp/internal/document"
	"github.com/droid/go-mcp/internal/search"
	"github.com/droid/go-mcp/internal/server"
	"github.com/droid/go-mcp/internal/tools"
)

func main() {
	// 命令行参数
	port := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

	// 创建工具管理器
	toolMgr := tools.NewToolManager()

	// 注册MCP工具
	toolMgr.RegisterTool(search.NewSearchTool())
	toolMgr.RegisterTool(document.NewDocumentTool())

	// 创建并运行MCP服务器
	mcpServer := server.NewMCPServer(toolMgr)
	go mcpServer.Run()

	// 设置HTTP路由
	http.HandleFunc("/ws", mcpServer.HandleWebSocket)
	http.HandleFunc("/tool", mcpServer.HandleToolRequest)
	http.HandleFunc("/tools", mcpServer.GetAvailableTools)

	// 健康检查
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 首页信息
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "Go MCP Server",
			"description": "一个Model Context Protocol服务器实现",
			"version": "1.0.0",
			"endpoints": {
				"/ws": "WebSocket连接",
				"/tool": "REST API工具请求",
				"/tools": "获取可用工具列表",
				"/health": "健康检查"
			}
		}`))
	})

	// 启动HTTP服务器
	serverAddr := ":" + *port
	log.Printf("MCP Server starting on http://localhost%s", serverAddr)
	log.Printf("Available endpoints:")
	log.Printf("  - ws:     http://localhost%s/ws", serverAddr)
	log.Printf("  - tool:   http://localhost%s/tool", serverAddr)
	log.Printf("  - tools:  http://localhost%s/tools", serverAddr)
	log.Printf("  - health: http://localhost%s/health", serverAddr)
	log.Printf("Available tools: %d", len(toolMgr.GetToolsSchema()))

	for _, tool := range toolMgr.GetToolsSchema() {
		log.Printf("  - %s: %s", tool["name"], tool["description"])
	}

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
