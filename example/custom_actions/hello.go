package custom_actions

import (
	"encoding/json"
)

type Params struct {
	Message string `json:"message"` // field name
}

func Run(config map[string]interface{}) (string, map[string]interface{}, error) {
	p := Params{}
	b, err := json.Marshal(config)
	if err != nil {
		return "", map[string]interface{}{}, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return "", map[string]interface{}{}, err
	}

	return "Hello, " + p.Message + "!", map[string]interface{}{}, nil
}

func Description() string {
	return "Send a message to the user"
}

func Definition() map[string][]string {
	return map[string][]string{
		"message": { // field name
			"string",              // type
			"The message to send", // description
		},
	}
}

func RequiredFields() []string {
	return []string{"message"} // field name
}

var config string

func Init(configuration string) error {
	// Do something with the configuration that was passed-by
	config = configuration
	return nil
}

// DynamicPrompt
func Render() (string, string, error) {
	return "Hello, " + config + "!", "", nil
}

func Role() string {
	return "system" // Role for the dynamic prompt
}
