package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/droid/go-mcp/internal/tools"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message 定义MCP消息结构
type Message struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Content  interface{} `json:"content"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// ToolRequest 工具请求
type ToolRequest struct {
	ID       string          `json:"id"`
	Tool     string          `json:"tool"`
	Params   json.RawMessage `json:"params"`
	Metadata interface{}     `json:"metadata,omitempty"`
}

// ToolResponse 工具响应
type ToolResponse struct {
	RequestID string      `json:"request_id"`
	Status    string      `json:"status"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// Client 客户端连接
type Client struct {
	ID         string
	Connection *websocket.Conn
	Send       chan Message
	Server     *MCPServer
}

// MCPServer MCP服务器实现
type MCPServer struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	toolMgr    *tools.ToolManager
	mutex      sync.RWMutex
}

// WebSocket配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求
	},
}

// NewMCPServer 创建新的MCP服务器
func NewMCPServer(toolMgr *tools.ToolManager) *MCPServer {
	return &MCPServer{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
		toolMgr:    toolMgr,
	}
}

// Run 启动MCP服务器的主循环
func (s *MCPServer) Run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client.ID] = client
			s.mutex.Unlock()
			log.Printf("Client registered: %s", client.ID)

		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.ID)
			}
			s.mutex.Unlock()

		case message := <-s.broadcast:
			s.mutex.RLock()
			for _, client := range s.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(s.clients, client.ID)
				}
			}
			s.mutex.RUnlock()
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (s *MCPServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to websocket:", err)
		return
	}

	// 生成客户端ID或使用请求参数
	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		clientID = uuid.New().String()
	}

	client := &Client{
		ID:         clientID,
		Connection: conn,
		Send:       make(chan Message, 256),
		Server:     s,
	}

	s.register <- client

	// 启动goroutine处理消息发送和接收
	go client.writePump()
	go client.readPump()

	// 发送连接成功消息
	client.Send <- Message{
		ID:   uuid.New().String(),
		Type: "connection",
		Content: map[string]string{
			"status":    "connected",
			"client_id": clientID,
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"version":   "1.0",
		},
	}

	// 发送可用工具清单
	client.Send <- Message{
		ID:   uuid.New().String(),
		Type: "tools",
		Content: map[string]interface{}{
			"tools": s.toolMgr.GetToolsSchema(),
		},
	}
}

// HandleToolRequest 处理工具请求
func (s *MCPServer) HandleToolRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// 确保请求ID存在
	if request.ID == "" {
		request.ID = uuid.New().String()
	}

	// 执行工具请求
	response := s.executeToolRequest(request)

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// executeToolRequest 执行工具请求并返回响应
func (s *MCPServer) executeToolRequest(request ToolRequest) ToolResponse {
	log.Printf("Executing tool request: %s - %s", request.ID, request.Tool)

	// 创建工具执行请求
	toolRequest := tools.ToolRequest{
		Name:       request.Tool,
		Parameters: request.Params,
	}

	// 执行工具
	result := s.toolMgr.ExecuteTool(toolRequest)

	// 构建响应
	response := ToolResponse{
		RequestID: request.ID,
		Status:    result.Status,
		Result:    result.Content,
		Error:     result.Error,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"tool":      request.Tool,
		},
	}

	// 广播工具执行结果（如果有需要）
	if result.Status == "success" {
		s.broadcast <- Message{
			ID:   uuid.New().String(),
			Type: "tool_result",
			Content: map[string]interface{}{
				"request_id": request.ID,
				"tool":       request.Tool,
				"result":     result.Content,
			},
		}
	}

	return response
}

// GetAvailableTools 获取可用工具列表
func (s *MCPServer) GetAvailableTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": s.toolMgr.GetToolsSchema(),
	})
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Connection.Close()
	}()

	c.Connection.SetReadLimit(512 * 1024) // 512KB
	c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Connection.SetPongHandler(func(string) error {
		c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// 尝试解析为工具请求
		var toolRequest ToolRequest
		if err := json.Unmarshal(message, &toolRequest); err == nil && toolRequest.Tool != "" {
			response := c.Server.executeToolRequest(toolRequest)

			// 将响应发送给客户端
			c.Send <- Message{
				ID:      uuid.New().String(),
				Type:    "tool_response",
				Content: response,
			}
			continue
		}

		// 否则尝试解析为一般消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error decoding message: %v", err)
			continue
		}

		// 如果消息没有ID，生成一个
		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}

		log.Printf("Received message from %s: %s", c.ID, string(message))

		// 根据消息类型处理
		switch msg.Type {
		case "ping":
			// 响应ping消息
			c.Send <- Message{
				ID:   uuid.New().String(),
				Type: "pong",
				Content: map[string]interface{}{
					"timestamp": time.Now().Unix(),
				},
			}
		case "get_tools":
			// 发送可用工具列表
			c.Send <- Message{
				ID:   uuid.New().String(),
				Type: "tools",
				Content: map[string]interface{}{
					"tools": c.Server.toolMgr.GetToolsSchema(),
				},
			}
		default:
			// 默认广播消息
			c.Server.broadcast <- msg
		}
	}
}

// writePump 向WebSocket连接发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭
				c.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Connection.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("error encoding message: %v", err)
				return
			}

			w.Write(messageBytes)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
