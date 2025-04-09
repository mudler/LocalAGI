package twitter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// TwitterAPIBase is the base URL for Twitter API v2
const TwitterAPIBase = "https://api.twitter.com/2"

// TwitterClient represents a Twitter API client
type TwitterClient struct {
	BearerToken string
	Client      *http.Client
}

// NewTwitterClient initializes a new Twitter API client
func NewTwitterClient(bearerToken string) *TwitterClient {
	return &TwitterClient{
		BearerToken: bearerToken,
		Client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// makeRequest is a helper for making authenticated HTTP requests
func (t *TwitterClient) makeRequest(method, url string, body map[string]interface{}) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+t.BearerToken)
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Twitter API error: %s", string(body))
	}

	return ioutil.ReadAll(resp.Body)
}

// GetStreamRules fetches existing stream rules
func (t *TwitterClient) GetStreamRules() ([]byte, error) {
	url := TwitterAPIBase + "/tweets/search/stream/rules"
	return t.makeRequest("GET", url, nil)
}

// AddStreamRule adds a rule to listen for mentions
func (t *TwitterClient) AddStreamRule(username string) error {
	url := TwitterAPIBase + "/tweets/search/stream/rules"
	body := map[string]interface{}{
		"add": []map[string]string{
			{"value": "@" + username, "tag": "Listen for mentions"},
		},
	}

	_, err := t.makeRequest("POST", url, body)
	return err
}

// DeleteStreamRules removes specific stream rules
func (t *TwitterClient) DeleteStreamRules(ruleIDs []string) error {
	url := TwitterAPIBase + "/tweets/search/stream/rules"
	body := map[string]interface{}{
		"delete": map[string]interface{}{
			"ids": ruleIDs,
		},
	}

	_, err := t.makeRequest("POST", url, body)
	return err
}

// ListenForMentions listens to the stream for mentions
func (t *TwitterClient) ListenForMentions() (*Tweet, error) {
	url := TwitterAPIBase + "/tweets/search/stream"
	resp, err := t.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var tweetResponse struct {
		Data Tweet `json:"data"`
	}

	err = json.Unmarshal(resp, &tweetResponse)
	if err != nil {
		return nil, err
	}

	return &tweetResponse.Data, nil
}

// GetReplies fetches all replies to a tweet
func (t *TwitterClient) GetReplies(tweetID, botUsername string) ([]Tweet, error) {
	url := fmt.Sprintf("%s/tweets/search/recent?query=conversation_id:%s from:%s", TwitterAPIBase, tweetID, botUsername)
	resp, err := t.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []Tweet `json:"data"`
	}

	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

// HasReplied checks if the bot has already replied to a tweet
func (t *TwitterClient) HasReplied(tweetID, botUsername string) (bool, error) {
	replies, err := t.GetReplies(tweetID, botUsername)
	if err != nil {
		return false, err
	}

	return len(replies) > 0, nil
}

// ReplyToTweet replies to a given tweet
func (t *TwitterClient) ReplyToTweet(tweetID, message string) error {
	url := TwitterAPIBase + "/tweets"
	body := map[string]interface{}{
		"text": message,
		"reply": map[string]string{
			"in_reply_to_tweet_id": tweetID,
		},
	}

	_, err := t.makeRequest("POST", url, body)
	return err
}

func (t *TwitterClient) Post(message string) error {
	url := TwitterAPIBase + "/tweets"
	body := map[string]interface{}{
		"text": message,
	}

	_, err := t.makeRequest("POST", url, body)
	return err
}

// Tweet represents a tweet object
type Tweet struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
