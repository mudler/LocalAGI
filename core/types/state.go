package types

import "fmt"

// State is the structure
// that is used to keep track of the current state
// and the Agent's short memory that it can update
// Besides a long term memory that is accessible by the agent (With vector database),
// And a context memory (that is always powered by a vector database),
// this memory is the shorter one that the LLM keeps across conversation and across its
// reasoning process's and life time.


type AgentInternalState struct {
	NowDoing    string   `json:"doing_now"`
	DoingNext   string   `json:"doing_next"`
	DoneHistory []string `json:"done_history"`
	Memories    []string `json:"memories"`
	Goal        string   `json:"goal"`
	Cronjobs    []string `json:"cronjobs"` // Added cronjobs field
}

//... (rest of the content with cronjob-related enhancements)