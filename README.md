```
# Go-MCP: 简易MCP服务器

这是一个用Go语言实现的简易MCP(Multi-Content Protocol)服务器，支持WebSocket连接和REST API。

## 功能特性

- WebSocket长连接支持
- REST API工具接口
- 工具管理机制
- 健康检查接口

## 项目结构

```
.
├── cmd
│   └── server        # 服务器实现
├── internal
│   ├── document      # 文档工具实现
│   ├── search        # 搜索工具实现
│   ├── server        # 服务器核心实现
│   └── tools         # 工具管理器
├── go.mod
├── go.sum
└── README.md
```

## 安装

```bash
git clone https://github.com/droid/go-mcp.git
cd go-mcp
go mod tidy
```

## 运行服务器

```bash
go run cmd/server/main.go
```

服务器将在本地的8080端口启动。也可以通过参数指定端口：

```bash
go run cmd/server/main.go -port=9000
```

## API接口

### WebSocket

- 路径: `/ws`
- 通过WebSocket连接与MCP服务器进行交互

### 工具API

- 路径: `/tool`
- 方法: `POST`
- 用于发送工具请求到服务器

### 获取可用工具

- 路径: `/tools`
- 方法: `GET`
- 获取服务器支持的所有工具列表

### 健康检查

- 路径: `/health`
- 方法: `GET`
- 响应:
  ```json
  {"status":"ok"}
  ```

## 支持的工具

服务器内置支持以下工具：

1. 搜索工具 - 提供文本搜索功能
2. 文档工具 - 提供文档读取和操作功能

## 许可证

MIT
```