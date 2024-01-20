package agent

import (
	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

type llmOptions struct {
	APIURL string
	Model  string
}

type options struct {
	LLMAPI    llmOptions
	Character Character
}

type Agent struct {
	options   *options
	Character Character
	client    *openai.Client
}

type Option func(*options) error

func defaultOptions() *options {
	return &options{
		LLMAPI: llmOptions{
			APIURL: "http://localhost:8080",
			Model:  "echidna",
		},
		Character: Character{
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

func New(opts ...Option) (*Agent, error) {
	options, err := newOptions(opts...)
	if err != nil {
		return nil, err
	}

	client := llm.NewClient("", options.LLMAPI.APIURL)
	a := &Agent{
		options:   options,
		client:    client,
		Character: options.Character,
	}
	return a, nil
}

func WithLLMAPIURL(url string) Option {
	return func(o *options) error {
		o.LLMAPI.APIURL = url
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
		o.Character = c
		return nil
	}
}

func FromFile(path string) Option {
	return func(o *options) error {
		c, err := Load(path)
		if err != nil {
			return err
		}
		o.Character = *c
		return nil
	}
}
