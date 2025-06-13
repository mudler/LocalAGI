package agent

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/sashabaranov/go-openai/jsonschema"
	"gorm.io/gorm"
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

func (a *Agent) State() action.AgentInternalState {
	return *a.currentState
}

// LoadState loads agent state from database
func (a *Agent) LoadState() error {
	// Load from database only
	return a.LoadStateFromDB()
}

// SaveState saves agent state to database
func (a *Agent) SaveState() error {
	// Save to database only
	return a.SaveStateToDB()
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

// LoadCharacterFromDB loads character from database
func (a *Agent) LoadCharacterFromDB() error {
	// Validate that we have a valid agentID
	if a.options.agentID == uuid.Nil {
		return fmt.Errorf("invalid agent ID: cannot load character")
	}

	var dbCharacter models.Character
	err := db.DB.Where("AgentID = ?", a.options.agentID).First(&dbCharacter).Error
	if err != nil {
		// Check if it's a "record not found" error - this is expected behavior
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("character not found for agent")
		}
		// For other database errors, return them as-is
		return err
	}

	// Convert database character to agent character
	hobbies := []string{}
	if dbCharacter.Hobbies != nil {
		json.Unmarshal(dbCharacter.Hobbies, &hobbies)
	}

	musicTaste := []string{}
	if dbCharacter.MusicTaste != nil {
		json.Unmarshal(dbCharacter.MusicTaste, &musicTaste)
	}

	a.Character = Character{
		Name:       dbCharacter.Name,
		Age:        dbCharacter.Age,
		Occupation: dbCharacter.Occupation,
		Hobbies:    hobbies,
		MusicTaste: musicTaste,
		Sex:        dbCharacter.Sex,
	}

	return nil
}

// SaveCharacterToDB saves character to database
func (a *Agent) SaveCharacterToDB() error {
	// Validate that we have valid IDs
	if a.options.agentID == uuid.Nil {
		return fmt.Errorf("invalid agent ID: cannot save character")
	}
	if a.options.userID == uuid.Nil {
		return fmt.Errorf("invalid user ID: cannot save character")
	}

	hobbiesJSON, _ := json.Marshal(a.Character.Hobbies)
	musicTasteJSON, _ := json.Marshal(a.Character.MusicTaste)

	dbCharacter := models.Character{
		AgentID:    a.options.agentID,
		UserID:     a.options.userID,
		Name:       a.Character.Name,
		Age:        a.Character.Age,
		Occupation: a.Character.Occupation,
		Hobbies:    hobbiesJSON,
		MusicTaste: musicTasteJSON,
		Sex:        a.Character.Sex,
	}

	// Use upsert (create or update) - don't set ID manually, let GORM handle it
	return db.DB.Where("AgentID = ?", a.options.agentID).Assign(dbCharacter).FirstOrCreate(&dbCharacter).Error
}

// LoadStateFromDB loads agent state from database
func (a *Agent) LoadStateFromDB() error {
	// Validate that we have a valid agentID
	if a.options.agentID == uuid.Nil {
		return fmt.Errorf("invalid agent ID: cannot load state")
	}

	var dbState models.AgentState
	err := db.DB.Where("AgentID = ?", a.options.agentID).First(&dbState).Error
	if err != nil {
		// Check if it's a "record not found" error - this is expected behavior
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("state not found for agent")
		}
		// For other database errors, return them as-is
		return err
	}

	// Convert database state to agent state
	doneHistory := []string{}
	if dbState.DoneHistory != nil {
		json.Unmarshal(dbState.DoneHistory, &doneHistory)
	}

	memories := []string{}
	if dbState.Memories != nil {
		json.Unmarshal(dbState.Memories, &memories)
	}

	a.currentState = &action.AgentInternalState{
		NowDoing:    dbState.NowDoing,
		DoingNext:   dbState.DoingNext,
		DoneHistory: doneHistory,
		Memories:    memories,
		Goal:        dbState.Goal,
	}

	return nil
}

// SaveStateToDB saves agent state to database
func (a *Agent) SaveStateToDB() error {
	// Validate that we have valid IDs
	if a.options.agentID == uuid.Nil {
		return fmt.Errorf("invalid agent ID: cannot save state")
	}
	if a.options.userID == uuid.Nil {
		return fmt.Errorf("invalid user ID: cannot save state")
	}

	doneHistoryJSON, _ := json.Marshal(a.currentState.DoneHistory)
	memoriesJSON, _ := json.Marshal(a.currentState.Memories)

	dbState := models.AgentState{
		AgentID:     a.options.agentID,
		UserID:      a.options.userID,
		NowDoing:    a.currentState.NowDoing,
		DoingNext:   a.currentState.DoingNext,
		DoneHistory: doneHistoryJSON,
		Memories:    memoriesJSON,
		Goal:        a.currentState.Goal,
	}

	// Use upsert (create or update)
	return db.DB.Where("AgentID = ?", a.options.agentID).Assign(dbState).FirstOrCreate(&dbState).Error
}

// prepareState loads agent state from database during initialization
func (a *Agent) prepareState() error {
	// Try to load existing state from database
	if err := a.LoadStateFromDB(); err == nil {
		// State found in database, use it
		return nil
	}

	// No state found, initialize with empty state
	a.currentState = &action.AgentInternalState{
		NowDoing:    "",
		DoingNext:   "",
		DoneHistory: []string{},
		Memories:    []string{},
		Goal:        "",
	}

	// Save the initial empty state to database
	if err := a.SaveStateToDB(); err != nil {
		return fmt.Errorf("failed to save initial state to database: %v", err)
	}

	return nil
}

// LoadState loads agent state from database (replaces file-based loading)
