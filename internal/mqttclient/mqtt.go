package mqttclient

import (
	"log"

	"go-mqtt-test/internal/hub"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func ConnectAndSubscribe(h *hub.Hub, topic string, callback func([]byte)) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("go-htmx-mqtt-client")

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT broker")
		token := c.Subscribe(topic, 0, func(c mqtt.Client, m mqtt.Message) {
			callback(m.Payload())
		})
		token.Wait()
		if err := token.Error(); err != nil {
			log.Println("Subscribe error:", err)
		}
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	return client
}
