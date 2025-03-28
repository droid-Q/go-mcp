package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message 定义MCP消息结构（与服务器相同）
type Message struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

func main() {
	// 中断信号处理
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 设置WebSocket URL
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	clientID := uuid.New().String()
	query := url.Values{}
	query.Add("id", clientID)
	u.RawQuery = query.Encode()

	fmt.Printf("连接到 %s\n", u.String())
	fmt.Printf("客户端ID: %s\n", clientID)

	// 连接WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	// 消息通道
	done := make(chan struct{})

	// 读取消息的goroutine
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("读取错误:", err)
				return
			}
			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("解码错误: %v", err)
				continue
			}

			fmt.Printf("收到消息: %s\n类型: %s\n", msg.ID, msg.Type)
			contentJSON, _ := json.MarshalIndent(msg.Content, "", "  ")
			fmt.Printf("内容:\n%s\n", string(contentJSON))
		}
	}()

	// 输入处理
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("输入消息（格式: <type>:<content>）或 'quit' 退出:")

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			fmt.Println("接收到中断，关闭连接...")

			// 发送关闭消息
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("写入关闭消息错误:", err)
				return
			}

			// 等待服务器关闭连接
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		default:
			if scanner.Scan() {
				input := scanner.Text()

				if input == "quit" {
					return
				}

				parts := strings.SplitN(input, ":", 2)
				if len(parts) != 2 {
					fmt.Println("无效格式。请使用 <type>:<content>")
					continue
				}

				msgType := strings.TrimSpace(parts[0])
				content := strings.TrimSpace(parts[1])

				msg := Message{
					ID:      uuid.New().String(),
					Type:    msgType,
					Content: content,
				}

				msgJSON, err := json.Marshal(msg)
				if err != nil {
					log.Println("编码错误:", err)
					continue
				}

				if err := conn.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
					log.Println("写入错误:", err)
					return
				}

				fmt.Printf("已发送消息: %s\n", msg.ID)
			}
		}
	}
}
