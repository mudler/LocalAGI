package agent

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/pkg/llm"
)

func (a *Agent) generateIdentity(guidance string, userID, agentID uuid.UUID, model string) error {
	if guidance == "" {
		guidance = "Generate a random character for roleplaying."
	}

	err := llm.GenerateTypedJSONWithGuidance(a.context.Context, a.client, "Generate a character as JSON data. "+guidance, model, userID, agentID, a.options.character.ToJSONSchema(), &a.options.character)
	//err := llm.GenerateJSONFromStruct(a.context.Context, a.client, guidance, a.options.LLMAPI.Model, &a.options.character)
	a.Character = a.options.character

	fmt.Println("generateIdentity", a.options.character, a.options.LLMAPI.Model, a.options.LLMAPI.APIKey)

	if err != nil {
		return fmt.Errorf("failed to generate JSON from structure: %v", err)
	}

	if !a.validCharacter() {
		return fmt.Errorf("generated character is not valid ( guidance: %s ): %v", guidance, a.Character.String())
	}
	return nil
}

func (a *Agent) prepareIdentity() error {
	println(a.options.randomIdentityGuidance, a.options.randomIdentity)

	if !a.options.randomIdentity {
		// No identity to generate
		return nil
	}

	// Try to load existing character from database
	if err := a.LoadCharacterFromDB(); err == nil {
		// Character found in database, use it
		return nil
	}

	// No character found, generate a new one
	if err := a.generateIdentity(a.options.randomIdentityGuidance, a.options.userID, a.options.agentID, a.options.LLMAPI.Model); err != nil {
		return fmt.Errorf("failed to generate identity: %v", err)
	}

	// Save the generated character to database
	if err := a.SaveCharacterToDB(); err != nil {
		return fmt.Errorf("failed to save character to database: %v", err)
	}

	return nil
}
