package hub

import (
	"html/template"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	User    string `json:"user" validate:"required"`
	Content string `json:"content" validate:"required"`
}

type Hub struct {
	sync.Mutex
	Clients map[*websocket.Conn]bool
	Tpl     *template.Template
}

func New(tpl *template.Template) *Hub {
	return &Hub{
		Clients: make(map[*websocket.Conn]bool),
		Tpl:     tpl,
	}
}

func (h *Hub) Add(conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()
	h.Clients[conn] = true
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()
	delete(h.Clients, conn)
}

func (h *Hub) Broadcast(b []byte) {
	h.Lock()
	defer h.Unlock()
	for conn := range h.Clients {
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Println("WebSocket write error:", err)
			conn.Close()
			delete(h.Clients, conn)
		}
	}
}
