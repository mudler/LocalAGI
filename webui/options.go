package webui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/services/skills"
)

type Config struct {
	DefaultChunkSize          int
	Pool                      *state.AgentPool
	SkillsService             *skills.Service
	ApiKeys                   []string
	LLMAPIURL                 string
	LLMAPIKey                 string
	LLMModel                  string
	StateDir                  string
	CustomActionsDir          string
	ConversationStoreDuration time.Duration

	// Collections / knowledge base (LocalRecall)
	CollectionDBPath string
	FileAssets       string
	VectorEngine     string
	EmbeddingModel   string
	MaxChunkingSize  int
	ChunkOverlap     int
	CollectionAPIKeys []string
	DatabaseURL      string
}

func collectionsDefaults(c *Config) {
	if c.CollectionDBPath == "" {
		c.CollectionDBPath = os.Getenv("COLLECTION_DB_PATH")
		if c.CollectionDBPath == "" {
			c.CollectionDBPath = filepath.Join(c.StateDir, "collections")
		}
	}
	if c.FileAssets == "" {
		c.FileAssets = os.Getenv("FILE_ASSETS")
		if c.FileAssets == "" {
			c.FileAssets = filepath.Join(c.StateDir, "assets")
		}
	}
	if c.VectorEngine == "" {
		c.VectorEngine = os.Getenv("VECTOR_ENGINE")
		if c.VectorEngine == "" {
			c.VectorEngine = "chromem"
		}
	}
	if c.EmbeddingModel == "" {
		c.EmbeddingModel = os.Getenv("EMBEDDING_MODEL")
		if c.EmbeddingModel == "" {
			c.EmbeddingModel = "granite-embedding-107m-multilingual"
		}
	}
	if c.MaxChunkingSize == 0 {
		if s := os.Getenv("MAX_CHUNKING_SIZE"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				c.MaxChunkingSize = n
			}
		}
		if c.MaxChunkingSize == 0 {
			c.MaxChunkingSize = 400
		}
	}
	if c.ChunkOverlap == 0 {
		if s := os.Getenv("CHUNK_OVERLAP"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				c.ChunkOverlap = n
			}
		}
	}
	if c.DatabaseURL == "" {
		c.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if len(c.CollectionAPIKeys) == 0 {
		if s := os.Getenv("API_KEYS"); s != "" {
			c.CollectionAPIKeys = strings.Split(s, ",")
		}
	}
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

func WithSkillsService(svc *skills.Service) Option {
	return func(c *Config) {
		c.SkillsService = svc
	}
}

func WithApiKeys(keys ...string) Option {
	return func(c *Config) {
		c.ApiKeys = keys
	}
}

func WithCollectionDBPath(path string) Option {
	return func(c *Config) {
		c.CollectionDBPath = path
	}
}

func WithFileAssets(path string) Option {
	return func(c *Config) {
		c.FileAssets = path
	}
}

func WithVectorEngine(engine string) Option {
	return func(c *Config) {
		c.VectorEngine = engine
	}
}

func WithEmbeddingModel(model string) Option {
	return func(c *Config) {
		c.EmbeddingModel = model
	}
}

func WithMaxChunkingSize(size int) Option {
	return func(c *Config) {
		c.MaxChunkingSize = size
	}
}

func WithChunkOverlap(overlap int) Option {
	return func(c *Config) {
		c.ChunkOverlap = overlap
	}
}

func WithCollectionAPIKeys(keys ...string) Option {
	return func(c *Config) {
		c.CollectionAPIKeys = keys
	}
}

func WithDatabaseURL(url string) Option {
	return func(c *Config) {
		c.DatabaseURL = url
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
	collectionsDefaults(c)
	return c
}
