package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	broker       = "tcp://localhost:1883"
	clientID     = "go-mqtt-heartbeat"
	heartbeatTTL = 3 * time.Second
)

type JSONTime struct {
	time.Time
}

type Device struct {
	Type     string
	ID       string
	LastSeen JSONTime
}

var (
	mu         sync.Mutex
	devices    = make(map[string]Device)
	mqttClient mqtt.Client
)

func main() {
	// Handle <Ctrl+C>
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true)

	opts.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe("+/+/heartbeat", 0, onHeartbeat); token.Wait() && token.Error() != nil {
			fmt.Printf("Subscription error: %v\n", token.Error())
			os.Exit(1)
		}
		fmt.Println("Subscribed to all heartbeat topics")
	}

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer mqttClient.Disconnect(250)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			publishActiveDevices()
		case <-c:
			fmt.Println("Shutting down")
			return
		}
	}

}

func onHeartbeat(client mqtt.Client, msg mqtt.Message) {
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) != 3 {
		return
	}

	mu.Lock()
	devices[parts[1]] = Device{
		Type:     parts[0],
		ID:       parts[1],
		LastSeen: JSONTime{time.Now()},
	}
	mu.Unlock()

	fmt.Printf("Heartbeat received from %s\n", parts[1])
}

type DeviceJson struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	LastSeen string `json:"last_seen"`
}

func publishActiveDevices() {
	now := time.Now()
	inactiveDevices := make([]string, 0)

	mu.Lock()
	for k, device := range devices {
		if now.Sub(device.LastSeen.Time) > heartbeatTTL {
			inactiveDevices = append(inactiveDevices, k)
		}
	}
	for _, k := range inactiveDevices {
		delete(devices, k)
	}
	mu.Unlock()

	payload, err := json.Marshal(devices)
	if err != nil {
		fmt.Printf("JSON error: %v\n", err)
		return
	}

	token := mqttClient.Publish("global/heartbeat", 0, false, payload)
	token.Wait()
}
