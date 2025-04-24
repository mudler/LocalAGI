package stdio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mudler/LocalAGI/pkg/xlog"
)

// Process represents a running process with its stdio streams
type Process struct {
	ID        string
	GroupID   string
	Cmd       *exec.Cmd
	Stdin     io.WriteCloser
	Stdout    io.ReadCloser
	Stderr    io.ReadCloser
	CreatedAt time.Time
}

// Server handles process management and stdio streaming
type Server struct {
	processes map[string]*Process
	groups    map[string][]string // maps group ID to process IDs
	mu        sync.RWMutex
	upgrader  websocket.Upgrader
}

// NewServer creates a new stdio server
func NewServer() *Server {
	return &Server{
		processes: make(map[string]*Process),
		groups:    make(map[string][]string),
		upgrader:  websocket.Upgrader{},
	}
}

// StartProcess starts a new process and returns its ID
func (s *Server) StartProcess(ctx context.Context, command string, args []string, env []string, groupID string) (string, error) {
	xlog.Debug("Starting process", "command", command, "args", args, "groupID", groupID)

	cmd := exec.CommandContext(ctx, command, args...)

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
		xlog.Debug("Process environment", "env", cmd.Env)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start process: %w", err)
	}

	process := &Process{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		GroupID:   groupID,
		Cmd:       cmd,
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.processes[process.ID] = process
	if groupID != "" {
		s.groups[groupID] = append(s.groups[groupID], process.ID)
	}
	s.mu.Unlock()

	xlog.Debug("Successfully started process", "id", process.ID, "pid", cmd.Process.Pid)
	return process.ID, nil
}

// StopProcess stops a running process
func (s *Server) StopProcess(id string) error {
	s.mu.Lock()
	process, exists := s.processes[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("process not found: %s", id)
	}

	xlog.Debug("Stopping process", "processID", id, "pid", process.Cmd.Process.Pid)

	// Remove from group if it exists
	if process.GroupID != "" {
		groupProcesses := s.groups[process.GroupID]
		for i, pid := range groupProcesses {
			if pid == id {
				s.groups[process.GroupID] = append(groupProcesses[:i], groupProcesses[i+1:]...)
				break
			}
		}
		if len(s.groups[process.GroupID]) == 0 {
			delete(s.groups, process.GroupID)
		}
	}

	delete(s.processes, id)
	s.mu.Unlock()

	if err := process.Cmd.Process.Kill(); err != nil {
		xlog.Debug("Failed to kill process", "processID", id, "pid", process.Cmd.Process.Pid, "error", err)
		return fmt.Errorf("failed to kill process: %w", err)
	}

	xlog.Debug("Successfully killed process", "processID", id, "pid", process.Cmd.Process.Pid)
	return nil
}

// StopGroup stops all processes in a group
func (s *Server) StopGroup(groupID string) error {
	s.mu.Lock()
	processIDs, exists := s.groups[groupID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("group not found: %s", groupID)
	}
	s.mu.Unlock()

	for _, pid := range processIDs {
		if err := s.StopProcess(pid); err != nil {
			return fmt.Errorf("failed to stop process %s in group %s: %w", pid, groupID, err)
		}
	}

	return nil
}

// GetGroupProcesses returns all processes in a group
func (s *Server) GetGroupProcesses(groupID string) ([]*Process, error) {
	s.mu.RLock()
	processIDs, exists := s.groups[groupID]
	if !exists {
		s.mu.RUnlock()
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	processes := make([]*Process, 0, len(processIDs))
	for _, pid := range processIDs {
		if process, exists := s.processes[pid]; exists {
			processes = append(processes, process)
		}
	}
	s.mu.RUnlock()

	return processes, nil
}

// ListGroups returns all group IDs
func (s *Server) ListGroups() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := make([]string, 0, len(s.groups))
	for groupID := range s.groups {
		groups = append(groups, groupID)
	}
	return groups
}

// GetProcess returns a process by ID
func (s *Server) GetProcess(id string) (*Process, error) {
	s.mu.RLock()
	process, exists := s.processes[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return process, nil
}

// ListProcesses returns all running processes
func (s *Server) ListProcesses() []*Process {
	s.mu.RLock()
	defer s.mu.RUnlock()

	processes := make([]*Process, 0, len(s.processes))
	for _, p := range s.processes {
		processes = append(processes, p)
	}

	return processes
}

// RunProcess executes a command and returns its output
func (s *Server) RunProcess(ctx context.Context, command string, args []string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("process failed: %w", err)
	}

	return string(output), nil
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	http.HandleFunc("/processes", s.handleProcesses)
	http.HandleFunc("/processes/", s.handleProcess)
	http.HandleFunc("/ws/", s.handleWebSocket)
	http.HandleFunc("/groups", s.handleGroups)
	http.HandleFunc("/groups/", s.handleGroup)
	http.HandleFunc("/run", s.handleRun)

	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling /processes request: method=%s", r.Method)

	switch r.Method {
	case http.MethodGet:
		processes := s.ListProcesses()
		json.NewEncoder(w).Encode(processes)
	case http.MethodPost:
		var req struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Env     []string `json:"env"`
			GroupID string   `json:"group_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := s.StartProcess(context.Background(), req.Command, req.Args, req.Env, req.GroupID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/processes/"):]

	switch r.Method {
	case http.MethodGet:
		process, err := s.GetProcess(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(process)
	case http.MethodDelete:
		if err := s.StopProcess(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/ws/"):]
	xlog.Debug("Handling WebSocket connection", "processID", id)

	process, err := s.GetProcess(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if process.Cmd.ProcessState != nil && process.Cmd.ProcessState.Exited() {
		xlog.Debug("Process already exited", "processID", id)
		http.Error(w, "Process already exited", http.StatusGone)
		return
	}

	xlog.Debug("Process is running", "processID", id, "pid", process.Cmd.Process.Pid)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	xlog.Debug("WebSocket connection established", "processID", id)

	// Create a done channel to signal process completion
	done := make(chan struct{})

	// Handle stdin
	go func() {
		defer func() {
			select {
			case <-done:
				xlog.Debug("Process stdin handler done", "processID", id)
			default:
				xlog.Debug("WebSocket stdin connection closed", "processID", id)
			}
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					xlog.Debug("WebSocket stdin unexpected error", "processID", id, "error", err)
				}
				return
			}
			xlog.Debug("Received message", "processID", id, "message", string(message))
			if _, err := process.Stdin.Write(message); err != nil {
				if err != io.EOF {
					xlog.Debug("WebSocket stdin write error", "processID", id, "error", err)
				}
				return
			}
			xlog.Debug("Message sent to process", "processID", id, "message", string(message))
		}
	}()

	// Handle stdout and stderr
	go func() {
		defer func() {
			select {
			case <-done:
				xlog.Debug("Process output handler done", "processID", id)
			default:
				xlog.Debug("WebSocket output connection closed", "processID", id)
			}
		}()

		// Create a buffer for reading
		buf := make([]byte, 4096)
		reader := io.MultiReader(process.Stdout, process.Stderr)

		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					xlog.Debug("Read error", "processID", id, "error", err)
				}
				return
			}

			if n > 0 {
				xlog.Debug("Sending message", "processID", id, "size", n)
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						xlog.Debug("WebSocket output write error", "processID", id, "error", err)
					}
					return
				}
				xlog.Debug("Message sent to client", "processID", id, "size", n)
			}
		}
	}()

	// Wait for process to exit
	xlog.Debug("Waiting for process to exit", "processID", id)
	err = process.Cmd.Wait()
	close(done) // Signal that the process is done

	if err != nil {
		xlog.Debug("Process exited with error",
			"processID", id,
			"pid", process.Cmd.Process.Pid,
			"error", err)
	} else {
		xlog.Debug("Process exited successfully",
			"processID", id,
			"pid", process.Cmd.Process.Pid)
	}
}

// Add new handlers for group management
func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		groups := s.ListGroups()
		json.NewEncoder(w).Encode(groups)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGroup(w http.ResponseWriter, r *http.Request) {
	groupID := r.URL.Path[len("/groups/"):]

	switch r.Method {
	case http.MethodGet:
		processes, err := s.GetGroupProcesses(groupID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(processes)
	case http.MethodDelete:
		if err := s.StopGroup(groupID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Handling /run request")

	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Env     []string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Executing one-time process: command=%s, args=%v", req.Command, req.Args)

	output, err := s.RunProcess(r.Context(), req.Command, req.Args, req.Env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("One-time process completed with output length: %d", len(output))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"output": output,
	})
}
