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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/xlog"
)

var _ agent.RAGDB = &WrappedClient{}

type WrappedClient struct {
	*Client
	collection string
}

func NewWrappedClient(baseURL, apiKey, c string) *WrappedClient {
	collection := strings.TrimSpace(strings.ToLower(c))
	wc := &WrappedClient{
		Client:     NewClient(baseURL, apiKey),
		collection: collection,
	}

	wc.CreateCollection(collection)

	return wc
}

// Collection returns the collection name for this client.
func (c *WrappedClient) Collection() string {
	return c.collection
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
	t := time.Now()
	dateTime := t.Format("2006-01-02-15-04-05")
	hash := md5.Sum([]byte(s))
	fileName := fmt.Sprintf("%s-%s.%s", dateTime, hex.EncodeToString(hash[:]), "txt")

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

// GetEntryContent returns the full file content (no chunk overlap) and the number of chunks for the entry.
func (c *WrappedClient) GetEntryContent(entry string) (content string, chunkCount int, err error) {
	return c.Client.GetEntryContent(c.collection, entry)
}

// apiResponse is the standardized LocalRecall API response wrapper (since 3f73ff3a).
type apiResponse struct {
	Success bool            `json:"success"`
	Message string         `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *apiError       `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// parseAPIError reads the response body and returns an error from the API response or a generic message.
func parseAPIError(resp *http.Response, body []byte, fallback string) error {
	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err == nil && wrap.Error != nil {
		if wrap.Error.Details != "" {
			return fmt.Errorf("%s: %s", wrap.Error.Message, wrap.Error.Details)
		}
		return errors.New(wrap.Error.Message)
	}
	return fmt.Errorf("%s: %s", fallback, string(body))
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

// EntryChunk represents a single chunk (legacy; GetEntryContent now returns full file content).
type EntryChunk struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

// Client is a client for the RAG API
type Client struct {
	BaseURL string
	APIKey  string
}

// NewClient creates a new RAG API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

// Add a helper method to set the Authorization header
func (c *Client) addAuthHeader(req *http.Request) {
	if c.APIKey == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp, body, "failed to create collection")
	}

	return nil
}

// ListCollections lists all collections
func (c *Client) ListCollections() ([]string, error) {
	url := fmt.Sprintf("%s/api/collections", c.BaseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp, body, "failed to list collections")
	}

	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return nil, errors.New(wrap.Error.Message)
		}
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var data struct {
		Collections []string `json:"collections"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, err
	}
	return data.Collections, nil
}

// ListEntries lists all entries in a collection
func (c *Client) ListEntries(collection string) ([]string, error) {
	url := fmt.Sprintf("%s/api/collections/%s/entries", c.BaseURL, collection)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp, body, "failed to list entries")
	}

	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return nil, errors.New(wrap.Error.Message)
		}
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var data struct {
		Entries []string `json:"entries"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, err
	}
	return data.Entries, nil
}

// GetEntryContent returns the full file content (no chunk overlap) and the number of chunks for the entry.
func (c *Client) GetEntryContent(collection, entry string) (content string, chunkCount int, err error) {
	entryEscaped := url.PathEscape(entry)
	reqURL := fmt.Sprintf("%s/api/collections/%s/entries/%s", c.BaseURL, collection, entryEscaped)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", 0, err
	}
	c.addAuthHeader(req)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, parseAPIError(resp, body, "failed to get entry content")
	}

	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return "", 0, errors.New(wrap.Error.Message)
		}
		return "", 0, fmt.Errorf("invalid response: %w", err)
	}

	var data struct {
		Content    string `json:"content"`
		ChunkCount int    `json:"chunk_count"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return "", 0, err
	}
	return data.Content, data.ChunkCount, nil
}

// DeleteEntry deletes an entry in a collection
func (c *Client) DeleteEntry(collection, entry string) ([]string, error) {
	url := fmt.Sprintf("%s/api/collections/%s/entry/delete", c.BaseURL, collection)

	type request struct {
		Entry string `json:"entry"`
	}

	payload, err := json.Marshal(request{Entry: entry})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp, body, "failed to delete entry")
	}

	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return nil, errors.New(wrap.Error.Message)
		}
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var data struct {
		RemainingEntries []string `json:"remaining_entries"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, err
	}
	return data.RemainingEntries, nil
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp, body, "failed to search collection")
	}

	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return nil, errors.New(wrap.Error.Message)
		}
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var data struct {
		Results []Result `json:"results"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, err
	}
	return data.Results, nil
}

// Reset resets a collection
func (c *Client) Reset(collection string) error {
	url := fmt.Sprintf("%s/api/collections/%s/reset", c.BaseURL, collection)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp, body, "failed to reset collection")
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
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp, body, "failed to upload file")
	}

	return nil
}

// SourceInfo represents an external source for a collection (LocalRecall API contract).
type SourceInfo struct {
	URL            string `json:"url"`
	UpdateInterval int    `json:"update_interval"` // minutes
	LastUpdate     string `json:"last_update"`      // RFC3339
}

// AddSource registers an external source for a collection.
func (c *Client) AddSource(collection, url string, updateIntervalMinutes int) error {
	reqURL := fmt.Sprintf("%s/api/collections/%s/sources", c.BaseURL, collection)
	var body struct {
		URL            string `json:"url"`
		UpdateInterval int    `json:"update_interval"`
	}
	body.URL = url
	body.UpdateInterval = updateIntervalMinutes
	if body.UpdateInterval < 1 {
		body.UpdateInterval = 60
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp, b, "failed to add source")
	}
	return nil
}

// RemoveSource removes an external source from a collection.
func (c *Client) RemoveSource(collection, url string) error {
	reqURL := fmt.Sprintf("%s/api/collections/%s/sources", c.BaseURL, collection)
	payload, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, reqURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp, b, "failed to remove source")
	}
	return nil
}

// ListSources returns external sources for a collection.
func (c *Client) ListSources(collection string) ([]SourceInfo, error) {
	reqURL := fmt.Sprintf("%s/api/collections/%s/sources", c.BaseURL, collection)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.addAuthHeader(req)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp, body, "failed to list sources")
	}
	var wrap apiResponse
	if err := json.Unmarshal(body, &wrap); err != nil || !wrap.Success {
		if wrap.Error != nil {
			return nil, errors.New(wrap.Error.Message)
		}
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	var data struct {
		Sources []SourceInfo `json:"sources"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, err
	}
	return data.Sources, nil
}
