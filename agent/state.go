package agent

import (
	"encoding/json"
	"os"

	"github.com/mudler/local-agent-framework/llm"
)

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
	data, err := json.Marshal(a.options.Character)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *Agent) GenerateIdentity(guidance string) error {
	err := llm.GenerateJSONFromStruct(a.client, guidance, a.options.LLMAPI.Model, &a.options.Character)

	a.Character = a.options.Character
	return err
}
