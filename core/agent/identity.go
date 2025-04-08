package agent

import (
	"fmt"
	"os"

	"github.com/mudler/LocalAgent/pkg/llm"
)

func (a *Agent) generateIdentity(guidance string) error {
	if guidance == "" {
		guidance = "Generate a random character for roleplaying."
	}

	err := llm.GenerateTypedJSON(a.context.Context, a.client, "Generate a character as JSON data. "+guidance, a.options.LLMAPI.Model, a.options.character.ToJSONSchema(), &a.options.character)
	//err := llm.GenerateJSONFromStruct(a.context.Context, a.client, guidance, a.options.LLMAPI.Model, &a.options.character)
	a.Character = a.options.character
	if err != nil {
		return fmt.Errorf("failed to generate JSON from structure: %v", err)
	}

	if !a.validCharacter() {
		return fmt.Errorf("generated character is not valid ( guidance: %s ): %v", guidance, a.Character.String())
	}
	return nil
}

func (a *Agent) prepareIdentity() error {
	if !a.options.randomIdentity {
		// No identity to generate
		return nil
	}

	if a.options.characterfile == "" {
		return a.generateIdentity(a.options.randomIdentityGuidance)
	}

	if _, err := os.Stat(a.options.characterfile); err == nil {
		// if there is a file, load the character back
		return a.LoadCharacter(a.options.characterfile)
	}

	if err := a.generateIdentity(a.options.randomIdentityGuidance); err != nil {
		return fmt.Errorf("failed to generate identity: %v", err)
	}

	// otherwise save it for next time
	if err := a.SaveCharacter(a.options.characterfile); err != nil {
		return fmt.Errorf("failed to save character: %v", err)
	}

	return nil
}
