package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yuhaohwang/bililive-go/src/instance"
)

// WebSocketManager 管理与客户端的WebSocket连接。
type WebSocketManager struct {
	clients  map[*websocket.Conn]bool // 当前已连接的客户端列表。
	upgrader websocket.Upgrader       // 用于升级HTTP连接到WebSocket连接的工具。
	lock     sync.Mutex               // 用于同步对clients的访问。
}

// NewWebSocketManager 初始化一个新的WebSocketManager并返回其指针。
func NewWebSocketManager(ctx context.Context) *WebSocketManager {
	wsm := &WebSocketManager{
		clients:  make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
	instance.GetInstance(ctx).WebsocketManager = wsm
	return wsm
}

// HandleConnection 处理新的WebSocket连接请求。
func (wsm *WebSocketManager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		inst.Logger.Error("Failed to upgrade ws: ", err)
		return
	}

	wsm.lock.Lock()
	wsm.clients[conn] = true
	wsm.lock.Unlock()

	// 仅仅为了检测连接断开
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			wsm.RemoveClient(conn)
			conn.Close()
			break
		}
	}
}

// RemoveClient 从管理器中移除一个WebSocket客户端连接。
func (wsm *WebSocketManager) RemoveClient(conn *websocket.Conn) {
	wsm.lock.Lock()
	delete(wsm.clients, conn)
	wsm.lock.Unlock()
}

// EventMessage 是发送给客户端的事件消息的结构。
type EventMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// SendEvent 将事件消息发送到给定的WebSocket连接。
func (wsm *WebSocketManager) SendEvent(conn *websocket.Conn, event string, data interface{}) error {
	msg := EventMessage{
		Event: event,
		Data:  data,
	}

	jsonData, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		fmt.Println("Error formatting:", err)
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, jsonData)
}

// SendMessageToClient 将消息发送到特定的WebSocket客户端。
func (wsm *WebSocketManager) SendMessageToClient(conn *websocket.Conn, event string, data interface{}) error {
	return wsm.SendEvent(conn, event, data)
}

// BroadcastMessage 将消息广播到所有连接的WebSocket客户端。
// 如果发送失败，返回发送失败的连接列表和最后一个错误。
func (wsm *WebSocketManager) BroadcastMessage(event string, data interface{}) ([]*websocket.Conn, error) {
	wsm.lock.Lock()
	defer wsm.lock.Unlock()

	var failedConns []*websocket.Conn
	var lastError error

	for client := range wsm.clients {
		if err := wsm.SendEvent(client, event, data); err != nil {
			client.Close()
			delete(wsm.clients, client)
			failedConns = append(failedConns, client)
			lastError = err
		}
	}

	if len(failedConns) > 0 {
		return failedConns, lastError
	}
	return nil, nil
}

// Close 关闭所有的WebSocket连接并清除clients。
func (wsm *WebSocketManager) Close(ctx context.Context) {
	wsm.lock.Lock()
	for client := range wsm.clients {
		client.Close()
	}
	wsm.clients = make(map[*websocket.Conn]bool)
	wsm.lock.Unlock()
}
