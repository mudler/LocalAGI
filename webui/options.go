package webui

import "github.com/mudler/LocalAgent/core/state"

type Config struct {
	DefaultChunkSize int
	Pool             *state.AgentPool
	ApiKeys          []string
	LLMAPIURL        string
	LLMAPIKey        string
	LLMModel         string
	StateDir         string
}

type Option func(*Config)

func WithDefaultChunkSize(size int) Option {
	return func(c *Config) {
		c.DefaultChunkSize = size
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
