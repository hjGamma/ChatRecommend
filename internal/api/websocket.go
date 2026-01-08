package api

import (
	"encoding/json"
	"net/http"
	"time"

	"ChatRecommend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// 写超时时间
	writeWait = 10 * time.Second
	// 读超时时间
	pongWait = 60 * time.Second
	// ping周期
	pingPeriod = (pongWait * 9) / 10
	// 最大消息大小
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该检查
	},
}

// Client WebSocket客户端
type Client struct {
	conn       *websocket.Conn
	handler    *Handler
	send       chan []byte
	conversationID string
	senderID   string
}

// WSMessage WebSocket消息
type WSMessage struct {
	Type           string                      `json:"type"`
	AutocompleteRequest *models.AutocompleteRequest `json:"autocomplete_request,omitempty"`
	Data           interface{}                 `json:"data,omitempty"`
	Error          string                      `json:"error,omitempty"`
}

// HandleWebSocket 处理WebSocket连接
func (h *Handler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("WebSocket升级失败")
		return
	}

	client := &Client{
		conn:    conn,
		handler: h,
		send:    make(chan []byte, 256),
	}

	// 启动读写goroutine
	go client.writePump()
	go client.readPump()
}

// readPump 读取消息
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket读取错误")
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			logrus.WithError(err).Error("解析WebSocket消息失败")
			continue
		}

		c.handleMessage(&wsMsg)
	}
}

// writePump 写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			logrus.WithField("message_size", len(message)).Debug("writePump: 从通道接收消息")

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logrus.WithError(err).Error("创建写入器失败")
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				logrus.WithError(err).Error("关闭写入器失败")
				return
			}

			logrus.Debug("writePump: 消息已发送")
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.WithError(err).Error("发送 Ping 失败")
				return
			}
		}
	}
}

// handleMessage 处理消息
func (c *Client) handleMessage(msg *WSMessage) {
	switch msg.Type {
	case "autocomplete":
		if msg.AutocompleteRequest == nil {
			c.sendError("autocomplete_request不能为空")
			return
		}

		logrus.WithFields(logrus.Fields{
			"conversation_id": msg.AutocompleteRequest.ConversationID,
			"input":           msg.AutocompleteRequest.Input,
		}).Debug("WebSocket 收到补全请求")

		// 保存conversation_id和sender_id
		c.conversationID = msg.AutocompleteRequest.ConversationID
		c.senderID = msg.AutocompleteRequest.SenderID

		// 获取补全建议
		resp, err := c.handler.autocomplete.GetSuggestionsWithDebounce(msg.AutocompleteRequest)
		if err != nil {
			logrus.WithError(err).Error("获取补全建议失败")
			c.sendError(err.Error())
			return
		}

		logrus.WithFields(logrus.Fields{
			"suggestions_count": len(resp.Suggestions),
			"suggestions":       resp.Suggestions,
		}).Debug("准备发送补全响应")

		// 发送响应
		response := WSMessage{
			Type: "autocomplete_response",
			Data: resp,
		}
		c.sendMessage(&response)

	default:
		c.sendError("未知的消息类型: " + msg.Type)
	}
}

// sendMessage 发送消息
func (c *Client) sendMessage(msg *WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("序列化消息失败")
		return
	}

	logrus.WithField("message", string(data)).Debug("发送 WebSocket 消息")

	select {
	case c.send <- data:
		logrus.Debug("消息已放入发送通道")
	default:
		logrus.Warn("发送通道已满，丢弃消息")
	}
}

// sendError 发送错误消息
func (c *Client) sendError(errMsg string) {
	msg := WSMessage{
		Type:  "error",
		Error: errMsg,
	}
	c.sendMessage(&msg)
}

