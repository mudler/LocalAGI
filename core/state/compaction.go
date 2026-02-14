package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/mudler/LocalAGI/pkg/localrag"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
)

// datePrefixRegex matches YYYY-MM-DD at the start of a filename (e.g. 2006-01-02-15-04-05-hash.txt).
var datePrefixRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)

// summaryPrefix is the filename prefix for compaction summary entries; skip re-compacting these.
const summaryPrefix = "summary-"

// bucketKey returns the period bucket key for a date string (YYYY-MM-DD).
func bucketKey(dateStr, period string) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}
	switch period {
	case "daily":
		return dateStr, nil
	case "weekly":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week), nil
	case "monthly":
		return t.Format("2006-01"), nil
	default:
		return dateStr, nil
	}
}

// dateFromFilename extracts YYYY-MM-DD from the start of a filename if present.
func dateFromFilename(filename string) (string, bool) {
	base := filepath.Base(filename)
	matches := datePrefixRegex.FindStringSubmatch(base)
	if len(matches) < 2 {
		return "", false
	}
	return matches[1], true
}

// groupEntriesByPeriod groups entry names by period bucket (daily/weekly/monthly). Skips summary-* and entries without a parseable date.
func groupEntriesByPeriod(entries []string, period string) map[string][]string {
	groups := make(map[string][]string)
	for _, entry := range entries {
		if strings.HasPrefix(filepath.Base(entry), summaryPrefix) {
			continue
		}
		dateStr, ok := dateFromFilename(entry)
		if !ok {
			continue
		}
		key, err := bucketKey(dateStr, period)
		if err != nil {
			xlog.Debug("compaction: skip entry, invalid date", "entry", entry, "error", err)
			continue
		}
		groups[key] = append(groups[key], entry)
	}
	return groups
}

// summarizer summarizes text via the LLM.
type summarizer interface {
	Summarize(ctx context.Context, content string) (string, error)
}

type openAISummarizer struct {
	client *openai.Client
	model  string
}

func (s *openAISummarizer) Summarize(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", nil
	}
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "Summarize the following knowledge base entries into a concise summary. Preserve important facts and key points."},
			{Role: openai.ChatMessageRoleUser, Content: content},
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// RunCompaction runs one compaction pass: list entries, group by period, for each group fetch content, optionally summarize, store result, delete originals.
func RunCompaction(ctx context.Context, client *localrag.WrappedClient, period string, summarize bool, apiURL, apiKey, model string) error {
	collection := client.Collection()
	entries, err := client.Client.ListEntries(collection)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}
	groups := groupEntriesByPeriod(entries, period)
	if len(groups) == 0 {
		xlog.Debug("compaction: no groups to compact", "collection", collection, "period", period)
		return nil
	}

	var sum summarizer
	if summarize && apiURL != "" && model != "" {
		openAIClient := llm.NewClient(apiKey, apiURL, "120s")
		sum = &openAISummarizer{client: openAIClient, model: model}
	}

	for key, groupEntries := range groups {
		if len(groupEntries) == 0 {
			continue
		}
		var combined strings.Builder
		for _, entry := range groupEntries {
			entryContent, _, err := client.GetEntryContent(entry)
			if err != nil {
				xlog.Warn("compaction: get entry content failed", "entry", entry, "error", err)
				continue
			}
			if entryContent != "" {
				combined.WriteString(entryContent)
				combined.WriteString("\n\n")
			}
		}
		content := strings.TrimSpace(combined.String())
		if content == "" {
			xlog.Debug("compaction: empty content for group", "key", key)
			continue
		}

		if sum != nil {
			summary, err := sum.Summarize(ctx, content)
			if err != nil {
				xlog.Warn("compaction: summarize failed", "key", key, "error", err)
				continue
			}
			content = summary
		}

		// Store result as summary-<key>.txt
		resultFilename := fmt.Sprintf("%s%s.txt", summaryPrefix, key)
		tmpDir, err := os.MkdirTemp("", "localagi-compact")
		if err != nil {
			xlog.Warn("compaction: mkdir temp failed", "error", err)
			continue
		}
		tmpPath := filepath.Join(tmpDir, resultFilename)
		if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			xlog.Warn("compaction: write temp file failed", "error", err)
			continue
		}
		if err := client.Client.Store(collection, tmpPath); err != nil {
			os.RemoveAll(tmpDir)
			xlog.Warn("compaction: store failed", "key", key, "error", err)
			continue
		}
		os.RemoveAll(tmpDir)

		for _, entry := range groupEntries {
			if _, err := client.Client.DeleteEntry(collection, entry); err != nil {
				xlog.Warn("compaction: delete entry failed", "entry", entry, "error", err)
			}
		}
		xlog.Info("compaction: compacted group", "collection", collection, "period", period, "key", key, "entries", len(groupEntries))
	}
	return nil
}

// runCompactionTicker runs compaction on a schedule (daily/weekly/monthly). It stops when ctx is done.
func runCompactionTicker(ctx context.Context, client *localrag.WrappedClient, config *AgentConfig, apiURL, apiKey, model string) {
	// Run first compaction immediately on startup
	if err := RunCompaction(ctx, client, config.KBCompactionInterval, config.KBCompactionSummarize, apiURL, apiKey, model); err != nil {
		xlog.Warn("compaction ticker initial run failed", "collection", client.Collection(), "error", err)
	}

	interval := 24 * time.Hour
	switch config.KBCompactionInterval {
	case "weekly":
		interval = 7 * 24 * time.Hour
	case "monthly":
		interval = 30 * 24 * time.Hour
	default:
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			xlog.Debug("compaction ticker stopped", "collection", client.Collection())
			return
		case <-ticker.C:
			if err := RunCompaction(ctx, client, config.KBCompactionInterval, config.KBCompactionSummarize, apiURL, apiKey, model); err != nil {
				xlog.Warn("compaction ticker failed", "collection", client.Collection(), "error", err)
			}
		}
	}
}
