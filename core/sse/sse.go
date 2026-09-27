package sse

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type (
	// Listener defines the interface for the receiving end.
	Listener interface {
		ID() string
		Chan() chan Envelope
	}

	// Envelope defines the interface for content that can be broadcast to clients.
	Envelope interface {
		String() string // Represent the envelope contents as a string for transmission.
	}

	// Manager defines the interface for managing clients and broadcasting messages.
	Manager interface {
		Send(message Envelope)
		Handle(ctx *fiber.Ctx, cl Listener)
		Register(cl Listener)
		Unregister(id string)
		Clients() []string
	}

	History interface {
		Add(message Envelope) // Add adds a message to the history.
		Send(c Listener)      // Send sends the history to a client.
	}
)

type Client struct {
	id string
	ch chan Envelope
}

func NewClient(id string) Listener {
	return &Client{
		id: id,
		ch: make(chan Envelope, 50),
	}
}

func (c *Client) ID() string          { return c.id }
func (c *Client) Chan() chan Envelope { return c.ch }

// Message represents a simple message implementation.
type Message struct {
	Event string
	Time  time.Time
	Data  string
}

// NewMessage returns a new message instance.
func NewMessage(data string) *Message {
	return &Message{
		Data: data,
		Time: time.Now(),
	}
}

// String returns the message as a string.
func (m *Message) String() string {
	sb := strings.Builder{}

	if m.Event != "" {
		sb.WriteString(fmt.Sprintf("event: %s\n", m.Event))
	}
	sb.WriteString(fmt.Sprintf("data: %v\n\n", m.Data))

	return sb.String()
}

// WithEvent sets the event name for the message.
func (m *Message) WithEvent(event string) Envelope {
	m.Event = event
	return m
}

// broadcastManager manages the clients and broadcasts messages to them.
type broadcastManager struct {
	clients        sync.Map
	broadcast      chan Envelope
	workerPoolSize int
	messageHistory *history
}

// NewManager initializes and returns a new Manager instance.
// bufferSize sets the broadcast channel capacity so Send() rarely blocks;
// delivery itself is handled by a single goroutine to preserve message order
// (SSE consumers rely on it, e.g. for token streams).
func NewManager(bufferSize int) Manager {
	if bufferSize < 1 {
		bufferSize = 1
	}
	manager := &broadcastManager{
		broadcast:      make(chan Envelope, bufferSize*16),
		workerPoolSize: bufferSize,
		messageHistory: newHistory(10),
	}

	manager.startWorkers()

	return manager
}

// Send broadcasts a message to all connected clients.
func (manager *broadcastManager) Send(message Envelope) {
	manager.broadcast <- message
}

// Register adds a client to the broadcast list and sends message history.
func (manager *broadcastManager) Register(cl Listener) {
	manager.register(cl)
	manager.messageHistory.Send(cl)
}

// Unregister removes a client from the broadcast list and closes its channel.
func (manager *broadcastManager) Unregister(id string) {
	manager.unregister(id)
}

// Handle sets up a new client and handles the connection.
func (manager *broadcastManager) Handle(c *fiber.Ctx, cl Listener) {

	manager.register(cl)
	ctx := c.Context()

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Cache-Control")
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")

	// Send history to the newly connected client
	manager.messageHistory.Send(cl)
	ctx.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case msg, ok := <-cl.Chan():
				if !ok {
					// If the channel is closed, return from the function
					return
				}
				_, err := fmt.Fprint(w, msg.String())
				if err != nil {
					// If an error occurs (e.g., client has disconnected), return from the function
					return
				}

				w.Flush()

			case <-ctx.Done():
				manager.unregister(cl.ID())
				close(cl.Chan())
				return
			}
		}
	}))
}

// Clients method to list connected client IDs
func (manager *broadcastManager) Clients() []string {
	var clients []string
	manager.clients.Range(func(key, value any) bool {
		id, ok := key.(string)
		if ok {
			clients = append(clients, id)
		}
		return true
	})
	return clients
}

// startWorkers starts the delivery goroutine for message broadcasting.
//
// Delivery is intentionally single-threaded: a pool of competing workers on
// the same channel delivers messages to clients in nondeterministic order,
// which scrambles token-level SSE streams (observed as interleaved/corrupted
// chat output in the UI). One goroutine consuming a buffered channel keeps
// FIFO order end-to-end; per-client backpressure is still handled by the
// non-blocking send below (slow clients drop messages instead of stalling
// everyone else).
func (manager *broadcastManager) startWorkers() {
	go func() {
		for message := range manager.broadcast {
			// Record once per message — not once per delivered client —
			// so history stays duplicate-free and is kept even when no
			// client is currently connected.
			manager.messageHistory.Add(message)
			manager.clients.Range(func(key, value any) bool {
				client, ok := value.(Listener)
				if !ok {
					return true // Continue iteration
				}
				select {
				case client.Chan() <- message:
				default:
					// If the client's channel is full, drop the message
				}
				return true // Continue iteration
			})
		}
	}()
}

// register adds a client to the manager.
func (manager *broadcastManager) register(client Listener) {
	manager.clients.Store(client.ID(), client)
}

// unregister removes a client from the manager.
func (manager *broadcastManager) unregister(clientID string) {
	manager.clients.Delete(clientID)
}

type history struct {
	mu       sync.Mutex
	messages []Envelope
	maxSize  int // Maximum number of messages to retain
}

func newHistory(maxSize int) *history {
	return &history{
		messages: []Envelope{},
		maxSize:  maxSize,
	}
}

func (h *history) Add(message Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, message)
	// Ensure history does not exceed maxSize
	if len(h.messages) > h.maxSize {
		// Remove the oldest messages to fit the maxSize
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}
}

// Send replays the retained history to a client. It snapshots the messages
// under the lock but delivers outside of it: the client channel is written
// from the connection handler's goroutine while the delivery goroutine may
// concurrently Add, so holding the lock across channel sends could deadlock
// a full client against the broadcaster.
func (h *history) Send(c Listener) {
	h.mu.Lock()
	snapshot := make([]Envelope, len(h.messages))
	copy(snapshot, h.messages)
	h.mu.Unlock()
	for _, msg := range snapshot {
		select {
		case c.Chan() <- msg:
		default:
			// Drop replayed history for clients whose buffer is already full.
		}
	}
}
