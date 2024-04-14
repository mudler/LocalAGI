package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mudler/local-agent-framework/action"
	"github.com/mudler/local-agent-framework/llm"
)

// PromptHUD contains
// all information that should be displayed to the LLM
// in the prompts
type PromptHUD struct {
	Character     Character          `json:"character"`
	CurrentState  action.StateResult `json:"current_state"`
	PermanentGoal string             `json:"permanent_goal"`
	ShowCharacter bool               `json:"show_character"`
}

type Character struct {
	Name       string   `json:"name"`
	Age        string   `json:"age"`
	Occupation string   `json:"job_occupation"`
	Hobbies    []string `json:"hobbies"`
	MusicTaste []string `json:"music_taste"`
	Sex        string   `json:"sex"`
}

func Load(path string) (*Character, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Character
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (a *Agent) State() action.StateResult {
	return *a.currentState
}

func (a *Agent) LoadState(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, a.currentState)
}

func (a *Agent) LoadCharacter(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &a.Character)
}

func (a *Agent) SaveState(path string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.Marshal(a.currentState)
	if err != nil {
		return err
	}
	os.WriteFile(path, data, 0644)
	return nil
}

func (a *Agent) SaveCharacter(path string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.Marshal(a.Character)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *Agent) generateIdentity(guidance string) error {
	if guidance == "" {
		guidance = "Generate a random character for roleplaying."
	}
	err := llm.GenerateJSONFromStruct(a.context.Context, a.client, guidance, a.options.LLMAPI.Model, &a.options.character)
	a.Character = a.options.character
	if err != nil {
		return fmt.Errorf("failed to generate JSON from structure: %v", err)
	}

	if !a.validCharacter() {
		return fmt.Errorf("generated character is not valid ( guidance: %s ): %v", guidance, a.Character.String())
	}
	return nil
}

func (a *Agent) validCharacter() bool {
	return a.Character.Name != "" &&
		a.Character.Age != "" &&
		a.Character.Occupation != "" &&
		len(a.Character.Hobbies) != 0 &&
		len(a.Character.MusicTaste) != 0
}

const fmtT = `=====================
Name: %s
Age: %s
Occupation: %s
Hobbies: %v
Music taste: %v
=====================`

func (c *Character) String() string {
	return fmt.Sprintf(
		fmtT,
		c.Name,
		c.Age,
		c.Occupation,
		c.Hobbies,
		c.MusicTaste,
	)
}
