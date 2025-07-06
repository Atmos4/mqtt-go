package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	deviceType := "Scanner"
	deviceId := "Scanner-A1"
	if len(os.Args) > 2 {
		deviceType = os.Args[1]
		deviceId = fmt.Sprintf("%s-%s", deviceType, os.Args[2])
	}

	// MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID(fmt.Sprintf("go-mqtt-%s", deviceId))

	// Handle <Ctrl+C>
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(250)

	// ticker
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	topic := fmt.Sprintf("%s/%s/heartbeat", deviceType, deviceId)

	for {
		select {
		case <-ticker.C:
			token := client.Publish(topic, 0, false, "online")
			token.Wait()
			fmt.Printf("Publishing to %s\n", topic)
		case <-c:
			fmt.Println("Shutting down")
			return
		}
	}

}
