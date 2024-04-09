package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"jaytaylor.com/html2text"

	"github.com/mudler/local-agent-framework/llm"
	sitemap "github.com/oxffaa/gopher-parse-sitemap"
)

type InMemoryDatabase struct {
	sync.Mutex
	Database []string
	path     string
}

func loadDB(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	poolData := []string{}
	err = json.Unmarshal(data, &poolData)
	return poolData, err
}

func NewInMemoryDB(knowledgebase string) (*InMemoryDatabase, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(knowledgebase, "knowledgebase.json")

	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, return a new pool
		return &InMemoryDatabase{
			Database: []string{},
			path:     poolfile,
		}, nil
	}

	poolData, err := loadDB(poolfile)
	if err != nil {
		return nil, err
	}
	return &InMemoryDatabase{
		Database: poolData,
		path:     poolfile,
	}, nil
}

func (db *InMemoryDatabase) SaveToStore(apiKey string, apiURL string) error {
	for _, d := range db.Database {
		lai := llm.NewClient(apiKey, apiURL+"/v1")
		laiStore := llm.NewStoreClient(apiURL, apiKey)

		err := llm.StoreStringEmbeddingInVectorDB(laiStore, lai, d)
		if err != nil {
			return fmt.Errorf("Error storing in the KB: %w", err)
		}
	}

	return nil
}
func (db *InMemoryDatabase) AddEntry(entry string) error {
	db.Lock()
	defer db.Unlock()
	db.Database = append(db.Database, entry)
	return nil
}

func (db *InMemoryDatabase) SaveDB() error {
	db.Lock()
	defer db.Unlock()
	data, err := json.Marshal(db.Database)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, data, 0644)
	return err
}

func getWebPage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return html2text.FromString(string(body), html2text.Options{PrettyTables: true})
}

func Sitemap(url string) (res []string, err error) {
	err = sitemap.ParseFromSite(url, func(e sitemap.Entry) error {
		fmt.Println("Sitemap page: " + e.GetLocation())
		content, err := getWebPage(e.GetLocation())
		if err == nil {
			res = append(res, content)
		}
		return nil
	})
	return
}

// splitParagraphIntoChunks takes a paragraph and a maxChunkSize as input,
// and returns a slice of strings where each string is a chunk of the paragraph
// that is at most maxChunkSize long, ensuring that words are not split.
func splitParagraphIntoChunks(paragraph string, maxChunkSize int) []string {
	// Check if the paragraph length is less than or equal to maxChunkSize.
	// If so, return the paragraph as the only chunk.
	if len(paragraph) <= maxChunkSize {
		return []string{paragraph}
	}

	var chunks []string
	var currentChunk strings.Builder

	words := strings.Fields(paragraph) // Splits the paragraph into words.

	for _, word := range words {
		// Check if adding the next word would exceed the maxChunkSize.
		// If so, add the currentChunk to the chunks slice and start a new chunk.
		if currentChunk.Len()+len(word) > maxChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		// Add a space before the word if it's not the beginning of a new chunk.
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}

		// Add the word to the current chunk.
		currentChunk.WriteString(word)
	}

	// Add the last chunk if it's not empty.
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
