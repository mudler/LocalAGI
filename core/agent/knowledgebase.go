package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

func (a *Agent) knowledgeBaseLookup(job *types.Job, conv Messages) Messages {
	fmt.Printf("DEBUG: Knowledge base lookup: %v %v %v \n", a.options.enableKB, a.options.enableLongTermMemory, a.options.enableSummaryMemory)
	if (!a.options.enableKB && !a.options.enableLongTermMemory && !a.options.enableSummaryMemory) ||
		len(conv) <= 0 {
		xlog.Debug("[Knowledge Base Lookup] Disabled, skipping", "agent", a.Character.Name)
		return conv
	}

	var obs *types.Observable
	if job != nil && job.Obs != nil && a.observer != nil {
		obs = a.observer.NewObservable()
		obs.Name = "Recall"
		obs.Icon = "database"
		obs.ParentID = job.Obs.ID
		a.observer.Update(*obs)
	}

	// Walk conversation from bottom to top, and find the first message of the user
	// to use it as a query to the KB
	userMessage := conv.GetLatestUserMessage().Content

	fmt.Println("userMessage", userMessage)

	xlog.Info("[Knowledge Base Lookup] Last user message", "agent", a.Character.Name, "message", userMessage, "lastMessage", conv.GetLatestUserMessage())

	if userMessage == "" {
		xlog.Info("[Knowledge Base Lookup] No user message found in conversation", "agent", a.Character.Name)
		return conv
	}

	var results []MemoryResult
	var err error

	if a.options.useMySQLForSummaries && a.options.enableKB {
		mysqlStorage := NewMySQLStorage(a.options.agentID, a.options.userID)
		if a.options.enableSummaryMemory {
			results, err = mysqlStorage.Search(userMessage, a.options.kbResults)
			fmt.Printf("DEBUG: MySQL search results: %d memories found\n", len(results))
		} else {
			excludeCount := 1
			fmt.Printf("DEBUG: Using count-based exclusion (excluding %d most recent messages)\n", excludeCount)
			results, err = mysqlStorage.GetLastMessagesExcludingCount(a.options.kbResults, excludeCount)
			fmt.Printf("DEBUG: MySQL get last messages results: %d memories found\n", len(results))
		}
	} else {
		return conv
	}

	if err != nil {
		xlog.Info("Error finding similar strings inside KB:", "error", err)
		if obs != nil {
			obs.AddProgress(types.Progress{
				Error: fmt.Sprintf("Error searching knowledge base: %v", err),
			})
			a.observer.Update(*obs)
		}
	}

	// Apply advanced filtering and processing
	processedResults := a.processMemoryResults(results)

	// Improved formatting with better context and structure
	formatResults := a.formatEnhancedMemoryResults(processedResults)
	xlog.Info("[Knowledge Base Lookup] Found similar strings in KB", "agent", a.Character.Name, "results", formatResults)

	if obs != nil {
		obs.AddProgress(types.Progress{
			ActionResult: fmt.Sprintf("Found %d results in knowledge base", len(processedResults)),
		})
		a.observer.Update(*obs)
	}

	// Create an improved system message with better context
	systemMessage := openai.ChatCompletionMessage{
		Role: "system",
		Content: fmt.Sprintf(`MEMORY CONTEXT: Based on the current user message, here are the relevant memories from your previous conversations:

%s

INSTRUCTIONS: Use this historical context to inform your response, but prioritize the current conversation. If the user is asking about previous interactions, reference these memories appropriately. If no relevant context is found above, respond based on the current conversation only.`,
			formatResults),
	}

	// Add the message to the conversation
	conv = append([]openai.ChatCompletionMessage{systemMessage}, conv...)

	if obs != nil {
		obs.Completion = &types.Completion{
			Conversation: []openai.ChatCompletionMessage{systemMessage},
		}
		a.observer.Update(*obs)
	}

	return conv
}

// processMemoryResults applies advanced filtering: deduplication, length limiting, etc.
func (a *Agent) processMemoryResults(results []MemoryResult) []MemoryResult {
	if len(results) == 0 {
		return results
	}

	// Step 1: Deduplication based on content similarity
	deduplicated := a.deduplicateMemories(results)

	// Step 2: Length limiting for individual memories
	lengthLimited := a.limitMemoryLength(deduplicated, 500) // Limit each memory to 500 chars

	// Step 3: Sort by time (most recent first)
	sorted := a.sortMemoriesByTime(lengthLimited)

	// Step 4: Limit total number of results
	maxResults := len(sorted)
	if maxResults > 10 {
		maxResults = 10
	}
	return sorted[:maxResults]
}

// deduplicateMemories removes very similar memories to avoid redundancy
func (a *Agent) deduplicateMemories(results []MemoryResult) []MemoryResult {
	if len(results) <= 1 {
		return results
	}

	var deduplicated []MemoryResult
	for i, result := range results {
		isDuplicate := false
		for j := 0; j < i; j++ {
			// Simple similarity check: if content is very similar, consider it duplicate
			similarity := a.calculateContentSimilarity(result.Content, results[j].Content)
			if similarity > 0.85 { // 85% similarity threshold
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			deduplicated = append(deduplicated, result)
		}
	}
	return deduplicated
}

// calculateContentSimilarity returns a simple similarity score between two strings
func (a *Agent) calculateContentSimilarity(content1, content2 string) float64 {
	if content1 == content2 {
		return 1.0
	}

	words1 := strings.Fields(strings.ToLower(content1))
	words2 := strings.Fields(strings.ToLower(content2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Simple Jaccard similarity
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	intersection := 0
	for _, word := range words2 {
		if set1[word] {
			intersection++
		}
	}

	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// limitMemoryLength truncates individual memories that are too long
func (a *Agent) limitMemoryLength(results []MemoryResult, maxLength int) []MemoryResult {
	for i := range results {
		if len(results[i].Content) > maxLength {
			results[i].Content = results[i].Content[:maxLength] + "..."
		}
	}
	return results
}

// sortMemoriesByTime sorts memories by creation time (most recent first)
func (a *Agent) sortMemoriesByTime(results []MemoryResult) []MemoryResult {
	// Sort by creation time (newest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].CreatedAt.After(results[i].CreatedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// formatEnhancedMemoryResults provides enhanced formatting with timestamps and relevance
func (a *Agent) formatEnhancedMemoryResults(results []MemoryResult) string {
	if len(results) == 0 {
		return "No relevant memories found."
	}

	var formattedResults strings.Builder

	// Add context about the search
	formattedResults.WriteString(fmt.Sprintf("Relevant memories (%d found, sorted by recency):\n\n", len(results)))

	now := time.Now()

	for i, result := range results {
		// Calculate relative time
		timeAgo := a.formatTimeAgo(now.Sub(result.CreatedAt))

		// Format sender with appropriate emoji/indicator
		senderIcon := "🤖"
		if result.Sender == "user" {
			senderIcon = "👤"
		}

		// Distinguish between user and assistant messages with clear formatting
		formattedResults.WriteString(fmt.Sprintf("%d. %s [%s] (%s ago)\n   %s\n\n",
			i+1,
			senderIcon,
			result.Sender,
			timeAgo,
			result.Content))
	}

	return formattedResults.String()
}

// formatTimeAgo formats a duration into a human-readable "time ago" string
func (a *Agent) formatTimeAgo(duration time.Duration) string {
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / (24 * 30))
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	} else {
		years := int(duration.Hours() / (24 * 365))
		if years == 1 {
			return "1 year"
		}
		return fmt.Sprintf("%d years", years)
	}
}

func (a *Agent) saveConversation(m Messages, prefix string) error {
	if a.options.conversationsPath == "" {
		return nil
	}
	dateTime := time.Now().Format("2006-01-02-15-04-05")
	fileName := a.Character.Name + "-" + dateTime + ".json"
	if prefix != "" {
		fileName = prefix + "-" + fileName
	}
	os.MkdirAll(a.options.conversationsPath, os.ModePerm)
	return m.Save(filepath.Join(a.options.conversationsPath, fileName))
}

func (a *Agent) saveCurrentConversation(conv Messages) {

	if err := a.saveConversation(conv, ""); err != nil {
		xlog.Error("Error saving conversation", "error", err)
	}

	if !a.options.enableLongTermMemory && !a.options.enableSummaryMemory {
		xlog.Debug("Long term memory is disabled", "agent", a.Character.Name)
		return
	}

	// xlog.Info("Saving conversation", "agent", a.Character.Name, "conversation size", len(conv))

	// if a.options.enableSummaryMemory && len(conv) > 0 {
	// 	msg, err := a.askLLM(a.context.Context, []openai.ChatCompletionMessage{{
	// 		Role:    "user",
	// 		Content: "Summarize the conversation below, keep the highlights as a bullet list:\n" + Messages(conv).String(),
	// 	}}, maxRetries)
	// 	if err != nil {
	// 		xlog.Error("Error summarizing conversation", "error", err)
	// 	}

	// 	// Use MySQL storage for summaries if configured
	// 	if a.options.useMySQLForSummaries {
	// 		fmt.Printf("DEBUG: Storing summary into MySQL: %s\n", msg.Content)
	// 		mysqlStorage := NewMySQLStorage(a.options.agentID, a.options.userID)
	// 		if err := mysqlStorage.Store(msg.Content); err != nil {
	// 			xlog.Error("Error storing summary into MySQL", "error", err)
	// 		}
	// 	} else {
	// 		// Fallback to RagDB
	// 		if err := a.options.ragdb.Store(msg.Content); err != nil {
	// 			xlog.Error("Error storing into memory", "error", err)
	// 		}
	// 	}
	// } else {
	// 	for _, message := range conv {
	// 		if message.Role == "user" {
	// 			if a.options.useMySQLForSummaries {
	// 				mysqlStorage := NewMySQLStorage(a.options.agentID, a.options.userID)
	// 				if err := mysqlStorage.Store(message.Content); err != nil {
	// 					xlog.Error("Error storing user message into MySQL", "error", err)
	// 				}
	// 			} else {
	// 				if err := a.options.ragdb.Store(message.Content); err != nil {
	// 					xlog.Error("Error storing into memory", "error", err)
	// 				}
	// 			}
	// 		}
	// 	}
	// }
}
