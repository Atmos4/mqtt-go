package hub

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	sync.Mutex
	Clients map[*websocket.Conn]bool
}

func New() *Hub {
	return &Hub{
		Clients: make(map[*websocket.Conn]bool),
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

func (h *Hub) OnConnect() http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		h.Add(conn)
		defer func() {
			h.Remove(conn)
			conn.Close()
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket read error:", err)
				break
			}
			log.Println(data)
		}
	}
}
