package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var outputTopic = "output/topic"
var broker = "tcp://localhost:1883"
var inputTopic = "input/topic"
var clientID = "go-mqtt-graceful"

func processMessage(msg string) string {
	time.Sleep(500 * time.Millisecond) // heavy work
	return strings.ToUpper(msg)
}

func onMessage(client mqtt.Client, msg mqtt.Message) {
	topic, payload := msg.Topic(), msg.Payload()

	fmt.Printf("Received on %s: %s\n", topic, payload)
	result := processMessage(string(payload))

	token := client.Publish(outputTopic, 0, false, result)

	if token.Wait() && token.Error() != nil {
		fmt.Printf("Publish error: %v\n", token.Error())
	} else {
		fmt.Println("Published processed message")
	}
}

func main() {

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle <Ctrl+C>
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetOrderMatters(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(250)

	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		wg.Add(1)
		go func(client mqtt.Client, msg mqtt.Message) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				fmt.Println("Context canceled, skipping message")
				return
			default:
			}

			onMessage(client, msg)
		}(client, msg)
	}

	if token := client.Subscribe(inputTopic, 1, messageHandler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Println("Subscribed. Waiting for messages...")

	// Wait for sigint
	<-c
	fmt.Println("\nShutdown signal received")
	cancel()
	fmt.Println("Waiting for active tasks to complete...")
	wg.Wait()
	fmt.Println("All tasks complete. Exiting.")
}
