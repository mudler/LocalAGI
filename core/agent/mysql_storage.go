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

func (m *MySQLStorage) Search(query string, similarEntries int, excludeCount int) ([]MemoryResult, error) {
	return m.SearchExcludingRecentCount(query, similarEntries, excludeCount) // Exclude most recent messages by count
}

// SearchExcludingRecentCount searches for messages but excludes the most recent ones by count to avoid circular references
func (m *MySQLStorage) SearchExcludingRecentCount(query string, similarEntries int, excludeCount int) ([]MemoryResult, error) {
	var summaries []models.AgentMessage

	fmt.Printf("DEBUG: Searching for query: '%s' (excluding %d most recent messages)\n", query, excludeCount)

	// Get IDs of the most recent messages to exclude
	var recentMessageIDs []uuid.UUID
	if excludeCount > 0 {
		var recentMessages []models.AgentMessage
		err := db.DB.Where("AgentID = ? AND Type = ?", m.agentID, "message").
			Order("CreatedAt desc").
			Limit(excludeCount).
			Find(&recentMessages).Error
		if err != nil {
			return nil, err
		}

		for _, msg := range recentMessages {
			recentMessageIDs = append(recentMessageIDs, msg.ID)
		}
	}

	// Extract keywords from the query
	keywords := extractKeywords(query)
	fmt.Printf("DEBUG: Extracted keywords: %v\n", keywords)

	if len(keywords) == 0 {
		// Fallback to original LIKE search if no keywords, but exclude recent messages
		searchTerm := "%" + strings.ToLower(query) + "%"
		query := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND Type = ?",
			m.agentID, searchTerm, "message")

		// Exclude recent messages by ID if any
		if len(recentMessageIDs) > 0 {
			query = query.Where("ID NOT IN ?", recentMessageIDs)
		}

		err := query.Order("CreatedAt desc").
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
			query := db.DB.Where("AgentID = ? AND MATCH(Content) AGAINST(? IN NATURAL LANGUAGE MODE) AND Type = ?",
				m.agentID, searchPhrase, "message")

			// Exclude recent messages by ID if any
			if len(recentMessageIDs) > 0 {
				query = query.Where("ID NOT IN ?", recentMessageIDs)
			}

			err := query.Order("CreatedAt desc").Find(&ftResults).Error

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
			query := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND Type = ?",
				m.agentID, searchTerm, "message")

			// Exclude recent messages by ID if any
			if len(recentMessageIDs) > 0 {
				query = query.Where("ID NOT IN ?", recentMessageIDs)
			}

			err := query.Order("CreatedAt desc").Find(&wordResults).Error

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
				query := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ? AND Type = ?",
					m.agentID, searchTerm, "message")

				// Exclude recent messages by ID if any
				if len(recentMessageIDs) > 0 {
					query = query.Where("ID NOT IN ?", recentMessageIDs)
				}

				err := query.Order("CreatedAt desc").Find(&phraseResults).Error

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

	fmt.Printf("DEBUG: Final search results count: %d (excluded %d most recent messages)\n", len(summaries), excludeCount)
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

func (m *MySQLStorage) GetLastMessagesExcludingCount(limit int, excludeCount int) ([]MemoryResult, error) {
	var messages []models.AgentMessage

	fmt.Printf("DEBUG: Getting last %d messages (excluding %d most recent messages)\n", limit, excludeCount)

	err := db.DB.Where("AgentID = ? AND Type = ?", m.agentID, "message").
		Order("CreatedAt desc").
		Offset(excludeCount).
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG: Retrieved %d messages (skipped %d most recent)\n", len(messages), excludeCount)
	for i, message := range messages {
		fmt.Printf("DEBUG: Message %d: %s: %s (created: %v)\n", i+1, message.Sender, message.Content, message.CreatedAt)
	}

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
