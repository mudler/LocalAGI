package rag

import (
	"context"
	"fmt"
	"runtime"

	"github.com/philippgille/chromem-go"
	"github.com/sashabaranov/go-openai"
)

type ChromemDB struct {
	collectionName string
	collection     *chromem.Collection
	index          int
}

func NewChromemDB(collection, path string, openaiClient *openai.Client) (*ChromemDB, error) {
	// db, err := chromem.NewPersistentDB(path, true)
	// if err != nil {
	// 	return nil, err
	// }
	db := chromem.NewDB()

	embeddingFunc := chromem.EmbeddingFunc(
		func(ctx context.Context, text string) ([]float32, error) {
			fmt.Println("Creating embeddings")
			resp, err := openaiClient.CreateEmbeddings(ctx,
				openai.EmbeddingRequestStrings{
					Input: []string{text},
					Model: openai.AdaEmbeddingV2,
				},
			)
			if err != nil {
				return []float32{}, fmt.Errorf("error getting keys: %v", err)
			}

			if len(resp.Data) == 0 {
				return []float32{}, fmt.Errorf("no response from OpenAI API")
			}

			embedding := resp.Data[0].Embedding

			return embedding, nil
		},
	)

	c, err := db.GetOrCreateCollection(collection, nil, embeddingFunc)
	if err != nil {
		return nil, err
	}

	return &ChromemDB{
		collectionName: collection,
		collection:     c,
		index:          1,
	}, nil
}

func (c *ChromemDB) Store(s string) error {
	defer func() {
		c.index++
	}()
	if s == "" {
		return fmt.Errorf("empty string")
	}
	fmt.Println("Trying to store", s)
	return c.collection.AddDocuments(context.Background(), []chromem.Document{
		{
			Content: s,
			ID:      fmt.Sprint(c.index),
		},
	}, runtime.NumCPU())
}

func (c *ChromemDB) Search(s string, similarEntries int) ([]string, error) {
	res, err := c.collection.Query(context.Background(), s, similarEntries, nil, nil)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, r := range res {
		results = append(results, r.Content)
	}

	return results, nil
}
