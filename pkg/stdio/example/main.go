package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mudler/LocalAGI/pkg/stdio"
)

func main() {
	// Start the server
	server := stdio.NewServer()
	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(time.Second)

	// Create a client
	client := stdio.NewClient("localhost:8080")

	// Start the client
	if err := client.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	defer client.Close()

	// Set up notification handler
	client.SetNotificationHandler(func(notification stdio.JSONRPCNotification) {
		fmt.Printf("Received notification: %+v\n", notification)
	})

	// Send a request
	request := stdio.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"hello": "world"},
	}

	response, err := client.SendRequest(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}

	fmt.Printf("Received response: %+v\n", response)

	// Send a notification
	notification := stdio.JSONRPCNotification{
		JSONRPC: "2.0",
		Notification: struct {
			Method string      `json:"method"`
			Params interface{} `json:"params,omitempty"`
		}{
			Method: "test",
			Params: map[string]string{"hello": "world"},
		},
	}

	if err := client.SendNotification(context.Background(), notification); err != nil {
		log.Fatalf("Failed to send notification: %v", err)
	}

	// Keep the program running
	select {}
}
