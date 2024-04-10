package rag

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type LocalAIRAGDB struct {
	client       *StoreClient
	openaiClient *openai.Client
}

func NewLocalAIRAGDB(storeClient *StoreClient, openaiClient *openai.Client) *LocalAIRAGDB {
	return &LocalAIRAGDB{
		client:       storeClient,
		openaiClient: openaiClient,
	}
}

func (db *LocalAIRAGDB) Store(s string) error {
	resp, err := db.openaiClient.CreateEmbeddings(context.TODO(),
		openai.EmbeddingRequestStrings{
			Input: []string{s},
			Model: openai.AdaEmbeddingV2,
		},
	)
	if err != nil {
		return fmt.Errorf("error getting keys: %v", err)
	}

	if len(resp.Data) == 0 {
		return fmt.Errorf("no response from OpenAI API")
	}

	embedding := resp.Data[0].Embedding

	setReq := SetRequest{
		Keys:   [][]float32{embedding},
		Values: []string{s},
	}
	err = db.client.Set(setReq)
	if err != nil {
		return fmt.Errorf("error setting keys: %v", err)
	}

	return nil
}

func (db *LocalAIRAGDB) Search(s string, similarEntries int) ([]string, error) {
	resp, err := db.openaiClient.CreateEmbeddings(context.TODO(),
		openai.EmbeddingRequestStrings{
			Input: []string{s},
			Model: openai.AdaEmbeddingV2,
		},
	)
	if err != nil {
		return []string{}, fmt.Errorf("error getting keys: %v", err)
	}

	if len(resp.Data) == 0 {
		return []string{}, fmt.Errorf("no response from OpenAI API")
	}
	embedding := resp.Data[0].Embedding

	// Find example
	findReq := FindRequest{
		TopK: similarEntries, // Number of similar entries you want to find
		Key:  embedding,      // The key you're looking for similarities to
	}
	findResp, err := db.client.Find(findReq)
	if err != nil {
		return []string{}, fmt.Errorf("error finding keys: %v", err)
	}

	return findResp.Values, nil
}
