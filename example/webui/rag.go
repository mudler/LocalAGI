package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mudler/local-agent-framework/xlog"

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
	RAGDB
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

func NewInMemoryDB(knowledgebase string, store RAGDB) (*InMemoryDatabase, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(knowledgebase, "knowledgebase.json")

	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, return a new pool
		return &InMemoryDatabase{
			Database: []string{},
			path:     poolfile,
			RAGDB:    store,
		}, nil
	}

	poolData, err := loadDB(poolfile)
	if err != nil {
		return nil, err
	}
	return &InMemoryDatabase{
		RAGDB:    store,
		Database: poolData,
		path:     poolfile,
	}, nil
}

func (db *InMemoryDatabase) PopulateRAGDB() error {
	for _, d := range db.Database {
		if d == "" {
			// skip empty chunks
			continue
		}
		err := db.RAGDB.Store(d)
		if err != nil {
			return fmt.Errorf("error storing in the KB: %w", err)
		}
	}

	return nil
}

func (db *InMemoryDatabase) Reset() error {
	db.Lock()
	db.Database = []string{}
	db.Unlock()
	if err := db.RAGDB.Reset(); err != nil {
		return err
	}
	return db.SaveDB()
}

func (db *InMemoryDatabase) save() error {
	data, err := json.Marshal(db.Database)
	if err != nil {
		return err
	}

	return os.WriteFile(db.path, data, 0644)
}

func (db *InMemoryDatabase) Store(entry string) error {
	db.Lock()
	defer db.Unlock()
	db.Database = append(db.Database, entry)
	if err := db.RAGDB.Store(entry); err != nil {
		return err
	}
	return db.save()
}

func (db *InMemoryDatabase) SaveDB() error {
	db.Lock()
	defer db.Unlock()
	return db.save()
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

func getWebSitemap(url string) (res []string, err error) {
	err = sitemap.ParseFromSite(url, func(e sitemap.Entry) error {
		xlog.Info("Sitemap page: " + e.GetLocation())
		content, err := getWebPage(e.GetLocation())
		if err == nil {
			res = append(res, content)
		}
		return nil
	})
	return
}

func WebsiteToKB(website string, chunkSize int, db *InMemoryDatabase) {
	content, err := getWebSitemap(website)
	if err != nil {
		xlog.Info("Error walking sitemap for website", err)
	}
	xlog.Info("Found pages: ", len(content))
	xlog.Info("ChunkSize: ", chunkSize)

	StringsToKB(db, chunkSize, content...)
}

func StringsToKB(db *InMemoryDatabase, chunkSize int, content ...string) {
	for _, c := range content {
		chunks := splitParagraphIntoChunks(c, chunkSize)
		xlog.Info("chunks: ", len(chunks))
		for _, chunk := range chunks {
			xlog.Info("Chunk size: ", len(chunk))
			db.Store(chunk)
		}
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
