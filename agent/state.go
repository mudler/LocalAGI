package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mudler/local-agent-framework/llm"
)

// PromptHUD contains
// all information that should be displayed to the LLM
// in the prompts
type PromptHUD struct {
	Character    Character `json:"character"`
	CurrentState State     `json:"current_state"`
}

// State is the structure
// that is used to keep track of the current state
// and the Agent's short memory that it can update
// Besides a long term memory that is accessible by the agent (With vector database),
// And a context memory (that is always powered by a vector database),
// this memory is the shorter one that the LLM keeps across conversation and across its
// reasoning process's and life time.
// TODO: A special action is then used to let the LLM itself update its memory
// periodically during self-processing, and the same action is ALSO exposed
// during the conversation to let the user put for example, a new goal to the agent.
type State struct {
	NowDoing    string   `json:"doing_now"`
	DoingNext   string   `json:"doing_next"`
	DoneHistory []string `json:"done_history"`
	Memories    []string `json:"memories"`
}

type Character struct {
	Name       string   `json:"name"`
	Age        int      `json:"age"`
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

func (a *Agent) Save(path string) error {
	data, err := json.Marshal(a.options.character)
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
		return fmt.Errorf("generated character is not valid ( guidance: %s ): %v", guidance, a.String())
	}
	return nil
}

func (a *Agent) validCharacter() bool {
	return a.Character.Name != "" &&
		a.Character.Age != 0 &&
		a.Character.Occupation != "" &&
		len(a.Character.Hobbies) != 0 &&
		len(a.Character.MusicTaste) != 0
}

const fmtT = `=====================
Name: %s
Age: %d
Occupation: %s
Hobbies: %v
Music taste: %v
=====================`

func (a *Agent) String() string {
	return fmt.Sprintf(
		fmtT,
		a.Character.Name,
		a.Character.Age,
		a.Character.Occupation,
		a.Character.Hobbies,
		a.Character.MusicTaste,
	)
}
