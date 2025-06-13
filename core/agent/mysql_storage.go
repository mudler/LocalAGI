package agent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
)

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

func (m *MySQLStorage) Search(query string, similarEntries int) ([]string, error) {
	var summaries []models.AgentMessage

	fmt.Printf("DEBUG: Searching for query: '%s'\n", query)

	// Extract keywords from the query
	keywords := extractKeywords(query)
	fmt.Printf("DEBUG: Extracted keywords: %v\n", keywords)

	if len(keywords) == 0 {
		// Fallback to original LIKE search if no keywords
		searchTerm := "%" + strings.ToLower(query) + "%"
		err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ?",
			m.agentID, searchTerm).
			Order("CreatedAt desc").
			Limit(similarEntries).
			Find(&summaries).Error

		if err != nil {
			return nil, err
		}
	} else {
		// Multi-strategy search
		results := make(map[string]*models.AgentMessage)

		// Strategy 1: Try full-text search if available (MySQL 5.6+)
		if len(keywords) > 0 {
			searchPhrase := strings.Join(keywords, " ")
			var ftResults []models.AgentMessage

			// Try MATCH AGAINST for full-text search
			err := db.DB.Where("AgentID = ? AND MATCH(Content) AGAINST(? IN NATURAL LANGUAGE MODE)",
				m.agentID, searchPhrase).
				Order("CreatedAt desc").
				Find(&ftResults).Error

			if err == nil && len(ftResults) > 0 {
				fmt.Printf("DEBUG: Full-text search found %d results\n", len(ftResults))
				for _, result := range ftResults {
					results[result.ID.String()] = &result
				}
			} else {
				fmt.Printf("DEBUG: Full-text search failed or no results: %v\n", err)
				// Continue with other strategies - fulltext search is optional
			}
		}

		// Strategy 2: Word-based search for each keyword
		for _, keyword := range keywords {
			var wordResults []models.AgentMessage
			searchTerm := "%" + keyword + "%"
			err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ?",
				m.agentID, searchTerm).
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
				err := db.DB.Where("AgentID = ? AND LOWER(Content) LIKE ?",
					m.agentID, searchTerm).
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
		// Note: This is a simple sort. For better performance with large datasets,
		// consider sorting in the database query
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

	fmt.Printf("DEBUG: Final search results count: %d\n", len(summaries))
	for i, summary := range summaries {
		fmt.Printf("DEBUG: Result %d: %s: %s\n", i+1, summary.Sender, summary.Content)
	}

	resultStrings := make([]string, len(summaries))
	for i, summary := range summaries {
		resultStrings[i] = fmt.Sprintf("%s: %s", summary.Sender, summary.Content)
	}

	return resultStrings, nil
}

func (m *MySQLStorage) Count() int {
	var count int64
	db.DB.Model(&models.AgentMessage{}).Where("AgentID = ?", m.agentID).Count(&count)
	return int(count)
}
