package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-mqtt-test/internal/hub"
	"log"
	"net/http"

	"github.com/a-h/templ"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Device struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	LastSeen string `json:"last_seen"`
}

var (
	broker = "tcp://localhost:1883"
)

func main() {
	h := hub.New()
	// Serve static files (HTMX, etc.)
	http.Handle("/static/", http.FileServer(http.Dir("static")))
	http.Handle("/{$}", templ.Handler(Home()))
	http.HandleFunc("/ws", h.OnConnect())

	go subscribeToHeartbeatMQTT(h)

	fmt.Println("UI server running at http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}

func subscribeToHeartbeatMQTT(h *hub.Hub) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("ui-app").
		SetCleanSession(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	client.Subscribe("global/heartbeat", 0, func(c mqtt.Client, m mqtt.Message) {
		log.Printf("Received heartbeat: %s", m.Payload())
		var devices map[string]Device
		if err := json.Unmarshal(m.Payload(), &devices); err != nil {
			log.Println("Error unmarshalling: ", err)
			return
		}

		var buf bytes.Buffer
		heartbeatList(devices).Render(context.Background(), &buf)
		h.Broadcast(buf.Bytes())
	})
}
