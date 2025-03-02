package agent

type PromptBlock interface {
	Render(a *Agent) (string, error)
	Role() string
}
