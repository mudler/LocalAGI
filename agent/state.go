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
	Character Character `json:"character"`
}

type Character struct {
	Name        string   `json:"name"`
	Age         int      `json:"age"`
	Occupation  string   `json:"job_occupation"`
	NowDoing    string   `json:"doing_now"`
	DoingNext   string   `json:"doing_next"`
	DoneHistory []string `json:"done_history"`
	Memories    []string `json:"memories"`
	Hobbies     []string `json:"hobbies"`
	MusicTaste  []string `json:"music_taste"`
	Sex         string   `json:"sex"`
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
		a.Character.NowDoing != "" &&
		a.Character.DoingNext != "" &&
		len(a.Character.DoneHistory) != 0 &&
		len(a.Character.Memories) != 0 &&
		len(a.Character.Hobbies) != 0 &&
		len(a.Character.MusicTaste) != 0
}

const fmtT = `=====================
Name: %s
Age: %d
Occupation: %s
Now doing: %s
Doing next: %s
Done history: %v
Memories: %v
Hobbies: %v
Music taste: %v
=====================`

func (a *Agent) String() string {
	return fmt.Sprintf(
		fmtT,
		a.Character.Name,
		a.Character.Age,
		a.Character.Occupation,
		a.Character.NowDoing,
		a.Character.DoingNext,
		a.Character.DoneHistory,
		a.Character.Memories,
		a.Character.Hobbies,
		a.Character.MusicTaste,
	)
}
