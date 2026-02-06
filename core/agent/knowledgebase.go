package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func (a *Agent) knowledgeBaseLookup(job *types.Job, conv Messages) Messages {
	// Only run KB recall/lookup when KB is explicitly enabled; long-term/summary memory
	// only affect saving in saveConversation, not this lookup.
	if !a.options.enableKB || len(conv) <= 0 {
		xlog.Debug("[Knowledge Base Lookup] Disabled, skipping", "agent", a.Character.Name)
		return conv
	}
	if !a.options.kbAutoSearch {
		xlog.Debug("[Knowledge Base Lookup] Auto search disabled, skipping", "agent", a.Character.Name)
		return conv
	}
	if a.options.ragdb == nil {
		xlog.Debug("[Knowledge Base Lookup] No RAG DB configured, skipping", "agent", a.Character.Name)
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

	xlog.Info("[Knowledge Base Lookup] Last user message", "agent", a.Character.Name, "message", userMessage, "lastMessage", conv.GetLatestUserMessage())

	if userMessage == "" {
		xlog.Info("[Knowledge Base Lookup] No user message found in conversation", "agent", a.Character.Name)
		if obs != nil {
			obs.Completion = &types.Completion{
				Error: "No user message found in conversation",
			}
			a.observer.Update(*obs)
		}
		return conv
	}

	results, err := a.options.ragdb.Search(userMessage, a.options.kbResults)
	if err != nil {
		xlog.Info("Error finding similar strings inside KB:", "error", err)
		if obs != nil {
			obs.AddProgress(types.Progress{
				Error: fmt.Sprintf("Error searching knowledge base: %v", err),
			})
			a.observer.Update(*obs)
		}
	}

	if len(results) == 0 {
		xlog.Info("[Knowledge Base Lookup] No similar strings found in KB", "agent", a.Character.Name)
		if obs != nil {
			obs.Completion = &types.Completion{
				ActionResult: "No similar strings found in knowledge base",
			}
			a.observer.Update(*obs)
		}
		return conv
	}

	formatResults := ""
	for _, r := range results {
		formatResults += fmt.Sprintf("- %s \n", r)
	}
	xlog.Info("[Knowledge Base Lookup] Found similar strings in KB", "agent", a.Character.Name, "results", formatResults)

	if obs != nil {
		obs.AddProgress(types.Progress{
			ActionResult: fmt.Sprintf("Found %d results in knowledge base", len(results)),
		})
		a.observer.Update(*obs)
	}

	// Create the message to add to conversation
	systemMessage := openai.ChatCompletionMessage{
		Role:    "system",
		Content: fmt.Sprintf("Given the user input you have the following in memory:\n%s", formatResults),
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

	xlog.Info("Saving conversation", "agent", a.Character.Name, "conversation size", len(conv))

	if a.options.enableSummaryMemory && len(conv) > 0 {
		fragment := cogito.NewEmptyFragment().AddStartMessage("user", "Summarize the conversation below, keep the highlights as a bullet list:\n"+Messages(conv).String())
		fragment, err := a.llm.Ask(a.context.Context, fragment)
		if err != nil {
			xlog.Error("Error summarizing conversation", "error", err)
		}
		msg := fragment.LastMessage()

		if err := a.options.ragdb.Store(msg.Content); err != nil {
			xlog.Error("Error storing into memory", "error", err)
		}
	} else {
		for _, message := range conv {
			if message.Role == "user" {
				if err := a.options.ragdb.Store(message.Content); err != nil {
					xlog.Error("Error storing into memory", "error", err)
				}
			}
		}
	}
}

// KBWrapperActions wraps RAGDB functionality as actions
type KBWrapperActions struct {
	ragdb     RAGDB
	kbResults int
}

type SearchKnowledgeBaseAction struct {
	*KBWrapperActions
}

type AddToKnowledgeBaseAction struct {
	*KBWrapperActions
}

// NewKBWrapperActions creates factory functions for KB wrapper actions
func NewKBWrapperActions(ragdb RAGDB, kbResults int) (*SearchKnowledgeBaseAction, *AddToKnowledgeBaseAction) {
	wrapper := &KBWrapperActions{
		ragdb:     ragdb,
		kbResults: kbResults,
	}
	return &SearchKnowledgeBaseAction{wrapper}, &AddToKnowledgeBaseAction{wrapper}
}

func (a *SearchKnowledgeBaseAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if a.ragdb == nil {
		return types.ActionResult{}, fmt.Errorf("knowledge base is not configured")
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if req.Query == "" {
		return types.ActionResult{}, fmt.Errorf("query cannot be empty")
	}

	results, err := a.ragdb.Search(req.Query, a.kbResults)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	if len(results) == 0 {
		return types.ActionResult{
			Result: fmt.Sprintf("No results found for query: %q", req.Query),
		}, nil
	}

	formatResults := ""
	for i, r := range results {
		formatResults += fmt.Sprintf("%d. %s\n", i+1, r)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Found %d result(s) for query %q:\n%s", len(results), req.Query, formatResults),
		Metadata: map[string]interface{}{
			"query":   req.Query,
			"results": results,
			"count":   len(results),
		},
	}, nil
}

func (a *SearchKnowledgeBaseAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName("search_knowledge_base"),
		Description: "Search the knowledge base for relevant information using a query string",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The search query to find relevant information in the knowledge base",
			},
		},
		Required: []string{"query"},
	}
}

func (a *AddToKnowledgeBaseAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if a.ragdb == nil {
		return types.ActionResult{}, fmt.Errorf("knowledge base is not configured")
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if req.Content == "" {
		return types.ActionResult{}, fmt.Errorf("content cannot be empty")
	}

	if err := a.ragdb.Store(req.Content); err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to store content in knowledge base: %w", err)
	}

	return types.ActionResult{
		Result: "Successfully added content to knowledge base",
		Metadata: map[string]interface{}{
			"content": req.Content,
		},
	}, nil
}

func (a *AddToKnowledgeBaseAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName("add_to_knowledge_base"),
		Description: "Add new content to the knowledge base for future retrieval",
		Properties: map[string]jsonschema.Definition{
			"content": {
				Type:        jsonschema.String,
				Description: "The content to store in the knowledge base",
			},
		},
		Required: []string{"content"},
	}
}
