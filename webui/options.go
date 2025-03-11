package webui

import "github.com/mudler/LocalAgent/core/state"

type Config struct {
	DefaultChunkSize int
	Pool             *state.AgentPool
	ApiKeys          []string
}

type Option func(*Config)

func WithDefaultChunkSize(size int) Option {
	return func(c *Config) {
		c.DefaultChunkSize = size
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
