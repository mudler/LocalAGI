package webui

import (
	"time"

	"github.com/mudler/LocalAGI/core/state"
)

type Config struct {
	DefaultChunkSize          int
	Pool                      *state.AgentPool
	ApiKeys                   []string
	LLMAPIURL                 string
	LLMAPIKey                 string
	LLMModel                  string
	StateDir                  string
	CustomActionsDir          string
	ConversationStoreDuration time.Duration
}

type Option func(*Config)

func WithDefaultChunkSize(size int) Option {
	return func(c *Config) {
		c.DefaultChunkSize = size
	}
}

func WithConversationStoreduration(duration string) Option {
	return func(c *Config) {
		d, err := time.ParseDuration(duration)
		if err != nil {
			d = 1 * time.Hour
		}
		c.ConversationStoreDuration = d
	}
}

func WithStateDir(dir string) Option {
	return func(c *Config) {
		c.StateDir = dir
	}
}

func WithLLMModel(model string) Option {
	return func(c *Config) {
		c.LLMModel = model
	}
}

func WithLLMAPIUrl(url string) Option {
	return func(c *Config) {
		c.LLMAPIURL = url
	}
}

func WithLLMAPIKey(key string) Option {
	return func(c *Config) {
		c.LLMAPIKey = key
	}
}

func WithCustomActionsDir(dir string) Option {
	return func(c *Config) {
		c.CustomActionsDir = dir
	}
}

func WithPool(pool *state.AgentPool) Option {
	return func(c *Config) {
		c.Pool = pool
	}
}

func WithApiKeys(keys ...string) Option {
	return func(c *Config) {
		c.ApiKeys = keys
	}
}

func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func NewConfig(opts ...Option) *Config {
	c := &Config{
		DefaultChunkSize: 2048,
	}
	c.Apply(opts...)
	return c
}
