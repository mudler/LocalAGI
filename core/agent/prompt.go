package agent

type DynamicPrompt interface {
	Render(a *Agent) (string, error)
	Role() string
}
