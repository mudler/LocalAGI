package agent

import (
	"context"
	"strings"
)

type Option func(*options) error
type llmOptions struct {
	APIURL string
	APIKey string
	Model  string
}

type options struct {
	LLMAPI                 llmOptions
	character              Character
	randomIdentityGuidance string
	randomIdentity         bool
	actions                Actions
	enableHUD              bool
	context                context.Context
}

func defaultOptions() *options {
	return &options{
		LLMAPI: llmOptions{
			APIURL: "http://localhost:8080",
			Model:  "echidna",
		},
		character: Character{
			Name:        "John Doe",
			Age:         0,
			Occupation:  "Unemployed",
			NowDoing:    "Nothing",
			DoingNext:   "Nothing",
			DoneHistory: []string{},
			Memories:    []string{},
			Hobbies:     []string{},
			MusicTaste:  []string{},
		},
	}
}

func newOptions(opts ...Option) (*options, error) {
	options := defaultOptions()
	for _, o := range opts {
		if err := o(options); err != nil {
			return nil, err
		}
	}
	return options, nil
}

var EnableHUD = func(o *options) error {
	o.enableHUD = true
	return nil
}

func WithLLMAPIURL(url string) Option {
	return func(o *options) error {
		o.LLMAPI.APIURL = url
		return nil
	}
}

func WithLLMAPIKey(key string) Option {
	return func(o *options) error {
		o.LLMAPI.APIKey = key
		return nil
	}
}

func WithContext(ctx context.Context) Option {
	return func(o *options) error {
		o.context = ctx
		return nil
	}
}

func WithModel(model string) Option {
	return func(o *options) error {
		o.LLMAPI.Model = model
		return nil
	}
}

func WithCharacter(c Character) Option {
	return func(o *options) error {
		o.character = c
		return nil
	}
}

func FromFile(path string) Option {
	return func(o *options) error {
		c, err := Load(path)
		if err != nil {
			return err
		}
		o.character = *c
		return nil
	}
}

func WithRandomIdentity(guidance ...string) Option {
	return func(o *options) error {
		o.randomIdentityGuidance = strings.Join(guidance, "")
		o.randomIdentity = true
		return nil
	}
}

func WithActions(actions ...Action) Option {
	return func(o *options) error {
		o.actions = actions
		return nil
	}
}
