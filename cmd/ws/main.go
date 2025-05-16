package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"go-mqtt-test/internal/hub"
	"go-mqtt-test/internal/mqttclient"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var validate *validator.Validate

func main() {
	tpl, _ := template.New("msg").Parse(``)
	h := hub.New(tpl)

	validate = validator.New(validator.WithRequiredStructEnabled())

	mqttclient.ConnectAndSubscribe(h, "demo/topic", func(m []byte) {
		var msg hub.Message
		if err := json.Unmarshal(m, &msg); err != nil {
			log.Println("MQTT message parse error:", err)
			return
		}

		var buf bytes.Buffer
		normalMessage(msg.User, msg.Content).Render(context.Background(), &buf)
		h.Broadcast(buf.Bytes())
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
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

			var msg hub.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Println("Invalid message:", err)
				continue
			}

			if err := validate.Struct(msg); err != nil {
				var buf bytes.Buffer
				errorMessage(fmt.Sprintf("Invalid message: %s", err)).Render(context.Background(), &buf)
				conn.WriteMessage(websocket.TextMessage, buf.Bytes())
				log.Println("Invalid message:", err)
				continue
			}

			var buf bytes.Buffer
			normalMessage(msg.User, msg.Content).Render(context.Background(), &buf)
			h.Broadcast(buf.Bytes())
		}
	})

	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
