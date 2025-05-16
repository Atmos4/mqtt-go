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
	broker := "tcp://localhost:1883"
	clientID := "go-mqtt-tester"
	inputTopic := "input/topic"
	outputTopic := "output/topic"

	// MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID)

	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("Connected to broker")

		// Subscribe to output topic
		if token := c.Subscribe(outputTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
			fmt.Printf("Received on %s: %s\n", msg.Topic(), msg.Payload())
		}); token.Wait() && token.Error() != nil {
			fmt.Printf("Subscribe error: %v\n", token.Error())
		}
	}

	// Handle <Ctrl+C>
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(250)

	// ticker
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	count := 1
	for {
		select {
		case <-ticker.C:
			payload := fmt.Sprintf("Count = %d", count)
			fmt.Printf("Publishing: %s\n", payload)
			token := client.Publish(inputTopic, 0, false, payload)
			token.Wait()
			count++
		case <-c:
			fmt.Println("\nShutting down...")
			return
		}
	}

}
