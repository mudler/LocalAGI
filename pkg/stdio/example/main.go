package main

import (
	"context"
	"fmt"
	"io"
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

	// Create a process group
	groupID := "test-group"

	// Start a process in the group
	process, err := client.CreateProcess(
		context.Background(),
		"echo",
		[]string{"Hello, World!"},
		[]string{"TEST=value"},
		groupID,
	)
	if err != nil {
		log.Fatalf("Failed to create process: %v", err)
	}

	// Get IO streams for the process
	reader, writer, err := client.GetProcessIO(process.ID)
	if err != nil {
		log.Fatalf("Failed to get process IO: %v", err)
	}

	// Write to the process
	_, err = writer.Write([]byte("Hello from client\n"))
	if err != nil {
		log.Fatalf("Failed to write to process: %v", err)
	}

	// Read from the process
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		log.Fatalf("Failed to read from process: %v", err)
	}
	fmt.Printf("Process output: %s", buf[:n])

	// Get all processes in the group
	processes, err := client.GetGroupProcesses(groupID)
	if err != nil {
		log.Printf("Failed to get group processes: %v", err)
	} else {
		fmt.Printf("Processes in group %s: %+v\n", groupID, processes)
	}

	// List all groups
	groups := client.ListGroups()
	fmt.Printf("All groups: %v\n", groups)

	// Stop the process
	if err := client.StopProcess(process.ID); err != nil {
		log.Fatalf("Failed to stop process: %v", err)
	}

	// Close the client
	if err := client.Close(); err != nil {
		log.Fatalf("Failed to close client: %v", err)
	}
}
