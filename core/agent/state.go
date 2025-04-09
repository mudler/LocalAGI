package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// PromptHUD contains
// all information that should be displayed to the LLM
// in the prompts
type PromptHUD struct {
	Character     Character                 `json:"character"`
	CurrentState  action.AgentInternalState `json:"current_state"`
	PermanentGoal string                    `json:"permanent_goal"`
	ShowCharacter bool                      `json:"show_character"`
}

type Character struct {
	Name       string   `json:"name"`
	Age        string   `json:"age"`
	Occupation string   `json:"job_occupation"`
	Hobbies    []string `json:"hobbies"`
	MusicTaste []string `json:"favorites_music_genres"`
	Sex        string   `json:"sex"`
}

func (c *Character) ToJSONSchema() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type:        jsonschema.String,
				Description: "The name of the character",
			},
			"age": {
				Type:        jsonschema.String,
				Description: "The age of the character",
			},
			"job_occupation": {
				Type:        jsonschema.String,
				Description: "The occupation of the character",
			},
			"hobbies": {
				Type:        jsonschema.Array,
				Description: "The hobbies of the character",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"favorites_music_genres": {
				Type:        jsonschema.Array,
				Description: "The favorite music genres of the character",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"sex": {
				Type:        jsonschema.String,
				Description: "The character sex (male, female)",
			},
		},
	}
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

func (a *Agent) State() action.AgentInternalState {
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

func (a *Agent) validCharacter() bool {
	return a.Character.Name != ""
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
