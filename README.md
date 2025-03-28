# Go-MCP: 简易MCP服务器

这是一个用Go语言实现的简易MCP(Multi-Content Protocol)服务器，支持WebSocket连接和REST API。

## 功能特性

- WebSocket长连接支持
- REST API消息接口
- 内部消息广播机制
- 客户端连接管理
- 健康检查接口

## 项目结构

```
.
├── cmd
│   ├── client        # 客户端示例
│   └── server        # 服务器实现
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

服务器将在本地的8080端口启动。

## 运行客户端

在另一个终端窗口中运行：

```bash
go run cmd/client/main.go
```

## API接口

### WebSocket

- 路径: `/ws`
- 可选参数: `id` - 客户端ID，如不提供则自动生成

### REST消息API

- 路径: `/message`
- 方法: `POST`
- 请求体:
  ```json
  {
    "id": "可选，自动生成",
    "type": "消息类型",
    "content": "消息内容，可以是任何JSON对象"
  }
  ```

### 健康检查

- 路径: `/health`
- 方法: `GET`
- 响应:
  ```json
  {"status":"ok"}
  ```

## 消息格式

所有消息遵循以下格式：

```json
{
  "id": "唯一消息ID",
  "type": "消息类型",
  "content": "消息内容，可以是任何JSON对象"
}
```

## 客户端使用示例

连接WebSocket后，输入消息格式为：`<type>:<content>`

例如:
```
chat:你好
```

将发送以下消息：
```json
{
  "id": "自动生成的UUID",
  "type": "chat",
  "content": "你好"
}
```

## 许可证

MIT 