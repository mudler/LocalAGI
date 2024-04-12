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

	. "github.com/mudler/local-agent-framework/agent"
	"jaytaylor.com/html2text"

	sitemap "github.com/oxffaa/gopher-parse-sitemap"
)

type InMemoryDatabase struct {
	sync.Mutex
	Database []string
	path     string
	rag      RAGDB
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

func NewInMemoryDB(knowledgebase string, store RAGDB) (*InMemoryDatabase, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(knowledgebase, "knowledgebase.json")

	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, return a new pool
		return &InMemoryDatabase{
			Database: []string{},
			path:     poolfile,
			rag:      store,
		}, nil
	}

	poolData, err := loadDB(poolfile)
	if err != nil {
		return nil, err
	}
	return &InMemoryDatabase{
		Database: poolData,
		path:     poolfile,
		rag:      store,
	}, nil
}

func (db *InMemoryDatabase) SaveToStore() error {
	for _, d := range db.Database {
		if d == "" {
			// skip empty chunks
			continue
		}
		err := db.rag.Store(d)
		if err != nil {
			return fmt.Errorf("Error storing in the KB: %w", err)
		}
	}

	return nil
}

func (db *InMemoryDatabase) Reset() error {
	db.Lock()
	db.Database = []string{}
	db.Unlock()
	if err := db.rag.Reset(); err != nil {
		return err
	}
	return db.SaveDB()
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

func WebsiteToKB(website string, chunkSize int, db *InMemoryDatabase) {
	content, err := Sitemap(website)
	if err != nil {
		fmt.Println("Error walking sitemap for website", err)
	}
	fmt.Println("Found pages: ", len(content))
	fmt.Println("ChunkSize: ", chunkSize)

	StringsToKB(db, chunkSize, content...)
}

func StringsToKB(db *InMemoryDatabase, chunkSize int, content ...string) {
	for _, c := range content {
		chunks := splitParagraphIntoChunks(c, chunkSize)
		fmt.Println("chunks: ", len(chunks))
		for _, chunk := range chunks {
			fmt.Println("Chunk size: ", len(chunk))
			db.AddEntry(chunk)
		}

		db.SaveDB()
	}

	if err := db.SaveToStore(); err != nil {
		fmt.Println("Error storing in the KB", err)
	}
}

// splitParagraphIntoChunks takes a paragraph and a maxChunkSize as input,
// and returns a slice of strings where each string is a chunk of the paragraph
// that is at most maxChunkSize long, ensuring that words are not split.
func splitParagraphIntoChunks(paragraph string, maxChunkSize int) []string {
	if len(paragraph) <= maxChunkSize {
		return []string{paragraph}
	}

	var chunks []string
	var currentChunk strings.Builder

	words := strings.Fields(paragraph) // Splits the paragraph into words.

	for _, word := range words {
		// If adding the next word would exceed maxChunkSize (considering a space if not the first word in a chunk),
		// add the currentChunk to chunks, and reset currentChunk.
		if currentChunk.Len() > 0 && currentChunk.Len()+len(word)+1 > maxChunkSize { // +1 for the space if not the first word
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		} else if currentChunk.Len() == 0 && len(word) > maxChunkSize { // Word itself exceeds maxChunkSize, split the word
			chunks = append(chunks, word)
			continue
		}

		// Add a space before the word if it's not the beginning of a new chunk.
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}

		// Add the word to the current chunk.
		currentChunk.WriteString(word)
	}

	// After the loop, add any remaining content in currentChunk to chunks.
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
