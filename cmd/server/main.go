package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message 定义MCP消息结构
type Message struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// MCPServer 简易MCP服务器
type MCPServer struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	mutex      sync.RWMutex
}

// Client 客户端连接
type Client struct {
	ID     string
	conn   *websocket.Conn
	send   chan Message
	server *MCPServer
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
func NewMCPServer() *MCPServer {
	return &MCPServer{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
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
				close(client.send)
				log.Printf("Client unregistered: %s", client.ID)
			}
			s.mutex.Unlock()

		case message := <-s.broadcast:
			s.mutex.RLock()
			for _, client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
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
		ID:     clientID,
		conn:   conn,
		send:   make(chan Message, 256),
		server: s,
	}

	s.register <- client

	// 启动goroutine处理消息发送和接收
	go client.writePump()
	go client.readPump()

	// 发送连接成功消息
	client.send <- Message{
		ID:      uuid.New().String(),
		Type:    "connection",
		Content: map[string]string{"status": "connected", "client_id": clientID},
	}
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

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
		c.server.broadcast <- msg
	}
}

// writePump 向WebSocket连接发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
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
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// HandleRESTMessage 处理REST API消息
func (s *MCPServer) HandleRESTMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	// 如果消息没有ID，生成一个
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	s.broadcast <- message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"status":"message accepted","id":"%s"}`, message.ID)))
}

func main() {
	server := NewMCPServer()
	go server.Run()

	// API端点
	http.HandleFunc("/ws", server.HandleWebSocket)
	http.HandleFunc("/message", server.HandleRESTMessage)

	// 健康检查
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := ":8080"
	fmt.Printf("MCP Server starting on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
