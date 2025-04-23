package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client implements the transport.Interface for stdio processes
type Client struct {
	baseURL   string
	processes map[string]*Process
	groups    map[string][]string
	mu        sync.RWMutex
}

// NewClient creates a new stdio transport client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:   baseURL,
		processes: make(map[string]*Process),
		groups:    make(map[string][]string),
	}
}

// CreateProcess starts a new process in a group
func (c *Client) CreateProcess(ctx context.Context, command string, args []string, env []string, groupID string) (*Process, error) {
	log.Printf("Creating process: command=%s, args=%v, groupID=%s", command, args, groupID)

	req := struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Env     []string `json:"env"`
		GroupID string   `json:"group_id"`
	}{
		Command: command,
		Args:    args,
		Env:     env,
		GroupID: groupID,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/processes", c.baseURL)
	log.Printf("Sending POST request to %s", url)

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response with status: %d", resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to decode response: %w. body: %s", err, string(body))
	}

	log.Printf("Successfully created process with ID: %s", result.ID)

	process := &Process{
		ID:        result.ID,
		GroupID:   groupID,
		CreatedAt: time.Now(),
	}

	c.mu.Lock()
	c.processes[process.ID] = process
	if groupID != "" {
		c.groups[groupID] = append(c.groups[groupID], process.ID)
	}
	c.mu.Unlock()

	return process, nil
}

// GetProcess returns a process by ID
func (c *Client) GetProcess(id string) (*Process, error) {
	c.mu.RLock()
	process, exists := c.processes[id]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return process, nil
}

// GetGroupProcesses returns all processes in a group
func (c *Client) GetGroupProcesses(groupID string) ([]*Process, error) {
	c.mu.RLock()
	processIDs, exists := c.groups[groupID]
	if !exists {
		c.mu.RUnlock()
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	processes := make([]*Process, 0, len(processIDs))
	for _, pid := range processIDs {
		if process, exists := c.processes[pid]; exists {
			processes = append(processes, process)
		}
	}
	c.mu.RUnlock()

	return processes, nil
}

// StopProcess stops a single process
func (c *Client) StopProcess(id string) error {
	c.mu.Lock()
	process, exists := c.processes[id]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("process not found: %s", id)
	}

	// Remove from group if it exists
	if process.GroupID != "" {
		groupProcesses := c.groups[process.GroupID]
		for i, pid := range groupProcesses {
			if pid == id {
				c.groups[process.GroupID] = append(groupProcesses[:i], groupProcesses[i+1:]...)
				break
			}
		}
		if len(c.groups[process.GroupID]) == 0 {
			delete(c.groups, process.GroupID)
		}
	}

	delete(c.processes, id)
	c.mu.Unlock()

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/processes/%s", c.baseURL, id),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}
	resp.Body.Close()

	return nil
}

// StopGroup stops all processes in a group
func (c *Client) StopGroup(groupID string) error {
	c.mu.Lock()
	processIDs, exists := c.groups[groupID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("group not found: %s", groupID)
	}
	c.mu.Unlock()

	for _, pid := range processIDs {
		if err := c.StopProcess(pid); err != nil {
			return fmt.Errorf("failed to stop process %s in group %s: %w", pid, groupID, err)
		}
	}

	return nil
}

// ListGroups returns all group IDs
func (c *Client) ListGroups() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	groups := make([]string, 0, len(c.groups))
	for groupID := range c.groups {
		groups = append(groups, groupID)
	}
	return groups
}

// GetProcessIO returns io.Reader and io.Writer for a process
func (c *Client) GetProcessIO(id string) (io.Reader, io.Writer, error) {
	log.Printf("Getting IO for process: %s", id)

	process, err := c.GetProcess(id)
	if err != nil {
		return nil, nil, err
	}

	// Parse the base URL to get the host
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Connect to WebSocket
	u := url.URL{
		Scheme: "ws",
		Host:   baseURL.Host,
		Path:   fmt.Sprintf("/ws/%s", process.ID),
	}

	log.Printf("Connecting to WebSocket at: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	log.Printf("Successfully connected to WebSocket for process: %s", id)

	// Create reader and writer
	reader := &websocketReader{conn: conn}
	writer := &websocketWriter{conn: conn}

	return reader, writer, nil
}

// websocketReader implements io.Reader for WebSocket
type websocketReader struct {
	conn *websocket.Conn
}

func (r *websocketReader) Read(p []byte) (n int, err error) {
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n = copy(p, message)
	return n, nil
}

// websocketWriter implements io.Writer for WebSocket
type websocketWriter struct {
	conn *websocket.Conn
}

func (w *websocketWriter) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes all connections and stops all processes
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop all processes
	for id := range c.processes {
		if err := c.StopProcess(id); err != nil {
			return fmt.Errorf("failed to stop process %s: %w", id, err)
		}
	}

	return nil
}

// RunProcess executes a command and returns its output
func (c *Client) RunProcess(ctx context.Context, command string, args []string, env []string) (string, error) {
	log.Printf("Running one-time process: command=%s, args=%v", command, args)

	req := struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Env     []string `json:"env"`
	}{
		Command: command,
		Args:    args,
		Env:     env,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/run", c.baseURL)
	log.Printf("Sending POST request to %s", url)

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to execute process: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response with status: %d", resp.StatusCode)

	var result struct {
		Output string `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to decode response: %w. body: %s", err, string(body))
	}

	log.Printf("Successfully executed process with output length: %d", len(result.Output))
	return result.Output, nil
}
