package actions

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
	"jaytaylor.com/html2text"
)

func NewBrowse(config map[string]string) *BrowseAction {

	return &BrowseAction{}
}

type BrowseAction struct{}

func (a *BrowseAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		URL string `json:"url"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	// Create HTTP client with proper configuration to prevent stream errors
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: false},
			DisableKeepAlives: false,
			// Force HTTP/1.1 to avoid HTTP/2 stream errors
			ForceAttemptHTTP2: false,
		},
	}

	req, err := http.NewRequest("GET", result.URL, nil)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Add proper browser-like headers to avoid bot detection
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Remove Accept-Encoding header to let Go handle compression automatically
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return types.ActionResult{}, err
	}
	defer resp.Body.Close()

	// Check if we got a valid response
	if resp.StatusCode >= 400 {
		return types.ActionResult{}, fmt.Errorf("website returned error %d: %s", resp.StatusCode, resp.Status)
	}

	pagebyte, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Check if content is too small (likely blocked/error page)
	if len(pagebyte) < 100 {
		return types.ActionResult{}, fmt.Errorf("website returned insufficient content (likely blocked or error page)")
	}

	rendered, err := html2text.FromString(string(pagebyte), html2text.Options{
		PrettyTables: true,
	})

	if err != nil {
		return types.ActionResult{}, err
	}

	// Filter out garbage content
	if len(rendered) < 50 {
		return types.ActionResult{}, fmt.Errorf("page content too short after conversion (likely JavaScript-only or blocked content)")
	}

	// Truncate very long content to prevent overwhelming the LLM
	if len(rendered) > 8000 {
		rendered = rendered[:8000] + "\n\n[Content truncated to prevent overwhelming response...]"
	}

	return types.ActionResult{Result: fmt.Sprintf("Successfully browsed '%s':\n\n%s", result.URL, rendered)}, nil
}

func (a *BrowseAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "browse",
		Description: "Use this tool to visit an URL. It browse a website page and return the text content.",
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "The website URL.",
			},
		},
		Required: []string{"url"},
	}
}

func (a *BrowseAction) Plannable() bool {
	return true
}
