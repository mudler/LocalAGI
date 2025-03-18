package localagent

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	Name          string                 `json:"name"`
	Actions       []string               `json:"actions,omitempty"`
	Connectors    []string               `json:"connectors,omitempty"`
	PromptBlocks  []string               `json:"prompt_blocks,omitempty"`
	InitialPrompt string                 `json:"initial_prompt,omitempty"`
	Parallel      bool                   `json:"parallel,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// AgentStatus represents the status of an agent
type AgentStatus struct {
	Status string `json:"status"`
}

// ListAgents returns a list of all agents
func (c *Client) ListAgents() ([]string, error) {
	resp, err := c.doRequest(http.MethodGet, "/agents", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// The response is HTML, so we'll need to parse it properly
	// For now, we'll just return a placeholder implementation
	return []string{}, fmt.Errorf("ListAgents not implemented")
}

// GetAgentConfig retrieves the configuration for a specific agent
func (c *Client) GetAgentConfig(name string) (*AgentConfig, error) {
	path := fmt.Sprintf("/api/agent/%s/config", name)
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config AgentConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &config, nil
}

// CreateAgent creates a new agent with the given configuration
func (c *Client) CreateAgent(config *AgentConfig) error {
	resp, err := c.doRequest(http.MethodPost, "/create", config)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if status, ok := response["status"]; ok && status == "ok" {
		return nil
	}
	return fmt.Errorf("failed to create agent: %v", response)
}

// UpdateAgentConfig updates the configuration for an existing agent
func (c *Client) UpdateAgentConfig(name string, config *AgentConfig) error {
	// Ensure the name in the URL matches the name in the config
	config.Name = name
	path := fmt.Sprintf("/api/agent/%s/config", name)

	resp, err := c.doRequest(http.MethodPut, path, config)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if status, ok := response["status"]; ok && status == "ok" {
		return nil
	}
	return fmt.Errorf("failed to update agent: %v", response)
}

// DeleteAgent removes an agent
func (c *Client) DeleteAgent(name string) error {
	path := fmt.Sprintf("/delete/%s", name)
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if status, ok := response["status"]; ok && status == "ok" {
		return nil
	}
	return fmt.Errorf("failed to delete agent: %v", response)
}

// PauseAgent pauses an agent
func (c *Client) PauseAgent(name string) error {
	path := fmt.Sprintf("/pause/%s", name)
	resp, err := c.doRequest(http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if status, ok := response["status"]; ok && status == "ok" {
		return nil
	}
	return fmt.Errorf("failed to pause agent: %v", response)
}

// StartAgent starts a paused agent
func (c *Client) StartAgent(name string) error {
	path := fmt.Sprintf("/start/%s", name)
	resp, err := c.doRequest(http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if status, ok := response["status"]; ok && status == "ok" {
		return nil
	}
	return fmt.Errorf("failed to start agent: %v", response)
}

// ExportAgent exports an agent configuration
func (c *Client) ExportAgent(name string) (*AgentConfig, error) {
	path := fmt.Sprintf("/settings/export/%s", name)
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config AgentConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &config, nil
}
