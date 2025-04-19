package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSONRPCNotification represents a JSON-RPC notification
type JSONRPCNotification struct {
	JSONRPC      string `json:"jsonrpc"`
	Notification struct {
		Method string      `json:"method"`
		Params interface{} `json:"params,omitempty"`
	} `json:"notification"`
}

// Client implements the transport.Interface for stdio processes
type Client struct {
	baseURL    string
	processID  string
	conn       *websocket.Conn
	mu         sync.Mutex
	notifyChan chan JSONRPCNotification
}

// NewClient creates a new stdio transport client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		notifyChan: make(chan JSONRPCNotification, 100),
	}
}

// Start initiates the connection to the server
func (c *Client) Start(ctx context.Context) error {
	// Start a new process
	req := struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{
		Command: "./mcp_server",
		Args:    []string{},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/processes", c.baseURL),
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	c.processID = result.ID

	// Connect to WebSocket
	u := url.URL{
		Scheme: "ws",
		Host:   c.baseURL,
		Path:   fmt.Sprintf("/ws/%s", c.processID),
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn

	// Start notification handler
	go c.handleNotifications()

	return nil
}

// Close shuts down the client and closes the transport
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	if c.processID != "" {
		req, err := http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/processes/%s", c.baseURL, c.processID),
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
	}

	return nil
}

// SendRequest sends a JSON-RPC request to the server
func (c *Client) SendRequest(
	ctx context.Context,
	request JSONRPCRequest,
) (*JSONRPCResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	if err := c.conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	var response JSONRPCResponse
	if err := c.conn.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &response, nil
}

// SendNotification sends a JSON-RPC notification to the server
func (c *Client) SendNotification(
	ctx context.Context,
	notification JSONRPCNotification,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteJSON(notification)
}

// SetNotificationHandler sets the handler for notifications
func (c *Client) SetNotificationHandler(
	handler func(notification JSONRPCNotification),
) {
	go func() {
		for notification := range c.notifyChan {
			handler(notification)
		}
	}()
}

func (c *Client) handleNotifications() {
	for {
		var notification JSONRPCNotification
		if err := c.conn.ReadJSON(&notification); err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		select {
		case c.notifyChan <- notification:
		default:
			// Drop notification if channel is full
		}
	}
}
