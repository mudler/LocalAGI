// TODO: this is a duplicate of LocalRAG/pkg/client
package localrag

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/pkg/xlog"
)

var _ agent.RAGDB = &WrappedClient{}

type WrappedClient struct {
	*Client
	collection string
}

func NewWrappedClient(baseURL, collection string) *WrappedClient {
	wc := &WrappedClient{
		Client:     NewClient(baseURL),
		collection: collection,
	}

	wc.CreateCollection(collection)

	return wc
}

func (c *WrappedClient) Count() int {
	entries, err := c.ListEntries(c.collection)
	if err != nil {
		return 0
	}
	return len(entries)
}

func (c *WrappedClient) Reset() error {
	return c.Client.Reset(c.collection)
}

func (c *WrappedClient) Search(s string, similarity int) ([]string, error) {
	results, err := c.Client.Search(c.collection, s, similarity)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, r := range results {
		res = append(res, fmt.Sprintf("%s (%+v)", r.Content, r.Metadata))
	}
	return res, nil
}

func (c *WrappedClient) Store(s string) error {
	// the Client API of LocalRAG takes only files at the moment.
	// So we take the string that we want to store, write it to a file, and then store the file.

	hash := md5.Sum([]byte(s))
	fileName := hex.EncodeToString(hash[:]) + ".txt"

	xlog.Debug("Storing string in LocalRAG", "collection", c.collection, "fileName", fileName)

	tempdir, err := os.MkdirTemp("", "localrag")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempdir)

	f := filepath.Join(tempdir, fileName)
	err = os.WriteFile(f, []byte(s), 0644)
	if err != nil {
		return err
	}

	defer os.Remove(f)
	return c.Client.Store(c.collection, f)
}

// Result represents a single result from a query.
type Result struct {
	ID        string
	Metadata  map[string]string
	Embedding []float32
	Content   string

	// The cosine similarity between the query and the document.
	// The higher the value, the more similar the document is to the query.
	// The value is in the range [-1, 1].
	Similarity float32
}

// Client is a client for the RAG API
type Client struct {
	BaseURL string
}

// NewClient creates a new RAG API client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
	}
}

// CreateCollection creates a new collection
func (c *Client) CreateCollection(name string) error {
	url := fmt.Sprintf("%s/api/collections", c.BaseURL)

	type request struct {
		Name string `json:"name"`
	}

	payload, err := json.Marshal(request{Name: name})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.New("failed to create collection")
	}

	return nil
}

// ListCollections lists all collections
func (c *Client) ListCollections() ([]string, error) {
	url := fmt.Sprintf("%s/api/collections", c.BaseURL)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to list collections")
	}

	var collections []string
	err = json.NewDecoder(resp.Body).Decode(&collections)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

// ListCollections lists all collections
func (c *Client) ListEntries(collection string) ([]string, error) {
	url := fmt.Sprintf("%s/api/collections/%s/entries", c.BaseURL, collection)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to list collections")
	}

	var entries []string
	err = json.NewDecoder(resp.Body).Decode(&entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// DeleteEntry deletes an Entry in a collection and return the entries left
func (c *Client) DeleteEntry(collection, entry string) ([]string, error) {
	url := fmt.Sprintf("%s/api/collections/%s/entry/delete", c.BaseURL, collection)

	type request struct {
		Entry string `json:"entry"`
	}
	client := &http.Client{}
	payload, err := json.Marshal(request{Entry: entry})
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyResult := new(bytes.Buffer)
		bodyResult.ReadFrom(resp.Body)
		return nil, errors.New("failed to delete collection: " + bodyResult.String())
	}

	var results []string
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Search searches a collection
func (c *Client) Search(collection, query string, maxResults int) ([]Result, error) {
	url := fmt.Sprintf("%s/api/collections/%s/search", c.BaseURL, collection)

	type request struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}

	payload, err := json.Marshal(request{Query: query, MaxResults: maxResults})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to search collection")
	}

	var results []Result
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (c *Client) Reset(collection string) error {
	url := fmt.Sprintf("%s/api/collections/%s/reset", c.BaseURL, collection)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b := new(bytes.Buffer)
		b.ReadFrom(resp.Body)

		return errors.New("failed to reset collection: " + b.String())
	}

	return nil
}

// Store uploads a file to a collection
func (c *Client) Store(collection, filePath string) error {
	url := fmt.Sprintf("%s/api/collections/%s/upload", c.BaseURL, collection)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b := new(bytes.Buffer)
		b.ReadFrom(resp.Body)

		type response struct {
			Error string `json:"error"`
		}

		var r response
		err = json.Unmarshal(b.Bytes(), &r)
		if err == nil {
			return errors.New("failed to upload file: " + r.Error)
		}

		return errors.New("failed to upload file")
	}

	return nil
}
