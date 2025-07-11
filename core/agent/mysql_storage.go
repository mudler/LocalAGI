package agent

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
)

// MemoryResult represents a structured memory result with metadata
type MemoryResult struct {
	ID        uuid.UUID `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type MySQLStorage struct {
	agentID uuid.UUID
	userID  uuid.UUID
}

func NewMySQLStorage(agentID, userID uuid.UUID) *MySQLStorage {
	return &MySQLStorage{
		agentID: agentID,
		userID:  userID,
	}
}

// extractKeywords extracts meaningful words from the query
func extractKeywords(query string) []string {
	// Convert to lowercase and remove punctuation
	query = strings.ToLower(query)
	re := regexp.MustCompile(`[^\w\s]`)
	query = re.ReplaceAllString(query, " ")

	// Split into words and filter out common stop words
	words := strings.Fields(query)
	stopWords := map[string]bool{
		"the": true, "is": true, "at": true, "which": true, "on": true,
		"a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "with": true, "to": true, "for": true, "of": true,
		"as": true, "by": true, "that": true, "this": true, "it": true,
		"what": true, "what's": true, "whats": true, "my": true,
	}

	var keywords []string
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func (m *MySQLStorage) Search(query string, similarEntries int) ([]MemoryResult, error) {
	return m.SearchExcludingRecent(query, similarEntries, 30*time.Second) // Exclude messages from last 30 seconds
}

// SearchExcludingRecent searches for messages but excludes very recent ones to avoid circular references
func (m *MySQLStorage) SearchExcludingRecent(query string, similarEntries int, excludeWithin time.Duration) ([]MemoryResult, error) {
	var summaries []models.AgentMessage

	fmt.Printf("DEBUG: Searching for query: '%s' (excluding messages within %v)\n", query, excludeWithin)

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-excludeWithin)

	// Extract keywords from the query
	keywords := extractKeywords(query)
	fmt.Printf("DEBUG: Extracted keywords: %v\n", keywords)

	if len(keywords) == 0 {
		// Fallback to original LIKE search if no keywords, but exclude recent messages
		searchTerm := "%" + strings.ToLower(query) + "%"
		err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND CreatedAt < ?",
			m.agentID, searchTerm, cutoffTime).
			Order("CreatedAt desc").
			Limit(similarEntries).
			Find(&summaries).Error

		if err != nil {
			return nil, err
		}
	} else {
		// Multi-strategy search - simplified to just find matches and sort by time
		results := make(map[string]*models.AgentMessage)

		// Strategy 1: Try full-text search if available (MySQL 5.6+)
		if len(keywords) > 0 {
			searchPhrase := strings.Join(keywords, " ")
			var ftResults []models.AgentMessage

			// Try MATCH AGAINST for full-text search
			err := db.DB.Where("AgentID = ? AND MATCH(Content) AGAINST(? IN NATURAL LANGUAGE MODE) AND CreatedAt < ?",
				m.agentID, searchPhrase, cutoffTime).
				Order("CreatedAt desc").
				Find(&ftResults).Error

			if err == nil && len(ftResults) > 0 {
				fmt.Printf("DEBUG: Full-text search found %d results\n", len(ftResults))
				for _, result := range ftResults {
					results[result.ID.String()] = &result
				}
			} else {
				fmt.Printf("DEBUG: Full-text search failed or no results: %v\n", err)
			}
		}

		// Strategy 2: Word-based search for each keyword
		for _, keyword := range keywords {
			var wordResults []models.AgentMessage
			searchTerm := "%" + keyword + "%"
			err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND CreatedAt < ?",
				m.agentID, searchTerm, cutoffTime).
				Order("CreatedAt desc").
				Find(&wordResults).Error

			if err == nil {
				fmt.Printf("DEBUG: Word search for '%s' found %d results\n", keyword, len(wordResults))
				for _, result := range wordResults {
					results[result.ID.String()] = &result
				}
			}
		}

		// Strategy 3: Partial phrase matching
		if len(keywords) > 1 {
			for i := 0; i < len(keywords)-1; i++ {
				phrase := keywords[i] + " " + keywords[i+1]
				var phraseResults []models.AgentMessage
				searchTerm := "%" + phrase + "%"
				err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND CreatedAt < ?",
					m.agentID, searchTerm, cutoffTime).
					Order("CreatedAt desc").
					Find(&phraseResults).Error

				if err == nil {
					fmt.Printf("DEBUG: Phrase search for '%s' found %d results\n", phrase, len(phraseResults))
					for _, result := range phraseResults {
						results[result.ID.String()] = &result
					}
				}
			}
		}

		// Convert map to slice
		for _, summary := range results {
			summaries = append(summaries, *summary)
		}

		// Sort by creation time (newest first)
		if len(summaries) > 1 {
			for i := 0; i < len(summaries)-1; i++ {
				for j := i + 1; j < len(summaries); j++ {
					if summaries[i].CreatedAt.Before(summaries[j].CreatedAt) {
						summaries[i], summaries[j] = summaries[j], summaries[i]
					}
				}
			}
		}

		// Limit results
		if len(summaries) > similarEntries {
			summaries = summaries[:similarEntries]
		}
	}

	fmt.Printf("DEBUG: Final search results count: %d (excluded messages after %v)\n", len(summaries), cutoffTime)
	for i, summary := range summaries {
		fmt.Printf("DEBUG: Result %d: %s: %s (created: %v)\n", i+1, summary.Sender, summary.Content, summary.CreatedAt)
	}

	// Convert to MemoryResult with structured data
	results := make([]MemoryResult, len(summaries))
	for i, summary := range summaries {
		results[i] = MemoryResult{
			ID:        summary.ID,
			Sender:    summary.Sender,
			Content:   summary.Content,
			CreatedAt: summary.CreatedAt,
		}
	}

	return results, nil
}

func (m *MySQLStorage) Count() int {
	var count int64
	db.DB.Model(&models.AgentMessage{}).Where("AgentID = ?", m.agentID).Count(&count)
	return int(count)
}

// GetLastMessages retrieves the most recent messages for the agent up to the specified limit
func (m *MySQLStorage) GetLastMessages(limit int) ([]MemoryResult, error) {
	return m.GetLastMessagesExcludingRecent(limit, 30*time.Second)
}

// GetLastMessagesExcludingRecent retrieves the most recent messages but excludes very recent ones
func (m *MySQLStorage) GetLastMessagesExcludingRecent(limit int, excludeWithin time.Duration) ([]MemoryResult, error) {
	var messages []models.AgentMessage

	fmt.Printf("DEBUG: Getting last %d messages (excluding messages within %v)\n", limit, excludeWithin)

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-excludeWithin)

	err := db.DB.Where("AgentID = ? AND CreatedAt < ?", m.agentID, cutoffTime).
		Order("CreatedAt desc").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG: Retrieved %d recent messages (excluded messages after %v)\n", len(messages), cutoffTime)
	for i, message := range messages {
		fmt.Printf("DEBUG: Message %d: %s: %s (created: %v)\n", i+1, message.Sender, message.Content, message.CreatedAt)
	}

	// Convert to MemoryResult with structured data
	results := make([]MemoryResult, len(messages))
	for i, message := range messages {
		results[i] = MemoryResult{
			ID:        message.ID,
			Sender:    message.Sender,
			Content:   message.Content,
			CreatedAt: message.CreatedAt,
		}
	}

	return results, nil
}
