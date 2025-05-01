package connectors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Matrix struct {
	homeserverURL string
	userID        string
	accessToken   string
	roomID        string
	roomMode      bool

	// To track placeholder messages
	placeholders     map[string]string // map[jobUUID]messageID
	placeholderMutex sync.RWMutex
	client           *mautrix.Client

	// Track active jobs for cancellation
	activeJobs      map[string][]*types.Job // map[roomID]bool to track if a room has active processing
	activeJobsMutex sync.RWMutex

	conversationTracker *ConversationTracker[string]
}

const matrixThinkingMessage = "🤔 thinking..."

func NewMatrix(config map[string]string) *Matrix {
	duration, err := time.ParseDuration(config["lastMessageDuration"])
	if err != nil {
		duration = 5 * time.Minute
	}

	return &Matrix{
		homeserverURL:       config["homeserverURL"],
		userID:              config["userID"],
		accessToken:         config["accessToken"],
		roomID:              config["roomID"],
		roomMode:            config["roomMode"] == "true",
		conversationTracker: NewConversationTracker[string](duration),
		placeholders:        make(map[string]string),
		activeJobs:          make(map[string][]*types.Job),
	}
}

func (m *Matrix) AgentResultCallback() func(state types.ActionState) {
	return func(state types.ActionState) {
		// Mark the job as completed when we get the final result
		if state.ActionCurrentState.Job != nil && state.ActionCurrentState.Job.Metadata != nil {
			if room, ok := state.ActionCurrentState.Job.Metadata["room"].(string); ok && room != "" {
				m.activeJobsMutex.Lock()
				delete(m.activeJobs, room)
				m.activeJobsMutex.Unlock()
			}
		}
	}
}

func (m *Matrix) AgentReasoningCallback() func(state types.ActionCurrentState) bool {
	return func(state types.ActionCurrentState) bool {
		// Check if we have a placeholder message for this job
		m.placeholderMutex.RLock()
		msgID, exists := m.placeholders[state.Job.UUID]
		room := ""
		if state.Job.Metadata != nil {
			if r, ok := state.Job.Metadata["room"].(string); ok {
				room = r
			}
		}
		m.placeholderMutex.RUnlock()

		if !exists || msgID == "" || room == "" || m.client == nil {
			return true // Skip if we don't have a message to update
		}

		thought := matrixThinkingMessage + "\n\n"
		if state.Reasoning != "" {
			thought += "Current thought process:\n" + state.Reasoning
		}

		// Update the placeholder message with the current reasoning
		_, err := m.client.SendText(context.Background(), id.RoomID(room), thought)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error updating reasoning message: %v", err))
		}
		return true
	}
}

// cancelActiveJobForRoom cancels any active job for the given room
func (m *Matrix) cancelActiveJobForRoom(roomID string) {
	m.activeJobsMutex.RLock()
	ctxs, exists := m.activeJobs[roomID]
	m.activeJobsMutex.RUnlock()

	if exists {
		xlog.Info(fmt.Sprintf("Cancelling active job for room: %s", roomID))

		// Mark the job as inactive
		m.activeJobsMutex.Lock()
		for _, c := range ctxs {
			c.Cancel()
		}
		delete(m.activeJobs, roomID)
		m.activeJobsMutex.Unlock()
	}
}

func (m *Matrix) handleRoomMessage(a *agent.Agent, evt *event.Event) {
	if m.roomID != evt.RoomID.String() && m.roomMode { // If we have a roomID and it's not the same as the event room
		// Skip messages from other rooms
		xlog.Info("Skipping reply to room", evt.RoomID, m.roomID)
		return
	}

	if evt.Sender == id.UserID(m.userID) {
		// Skip messages from ourselves
		return
	}

	// Skip if message does not mention the bot
	mentioned := false
	if evt.Content.AsMessage().Mentions != nil {
		for _, mention := range evt.Content.AsMessage().Mentions.UserIDs {
			if mention == m.client.UserID {
				mentioned = true
				break
			}
		}
	}

	if !mentioned && !m.roomMode {
		xlog.Info("Skipping reply because it does not mention the bot", evt.RoomID, m.roomID)
		return
	}

	// Cancel any active job for this room before starting a new one
	m.cancelActiveJobForRoom(evt.RoomID.String())

	currentConv := m.conversationTracker.GetConversation(evt.RoomID.String())

	message := evt.Content.AsMessage().Body

	go func() {
		agentOptions := []types.JobOption{
			types.WithUUID(evt.ID.String()),
		}

		currentConv = append(currentConv, openai.ChatCompletionMessage{
			Role:    "user",
			Content: message,
		})

		m.conversationTracker.AddMessage(
			evt.RoomID.String(), currentConv[len(currentConv)-1],
		)

		agentOptions = append(agentOptions, types.WithConversationHistory(currentConv))

		// Add room to metadata for tracking
		metadata := map[string]interface{}{
			"room": evt.RoomID.String(),
		}
		agentOptions = append(agentOptions, types.WithMetadata(metadata))

		job := types.NewJob(agentOptions...)

		// Mark this room as having an active job
		m.activeJobsMutex.Lock()
		m.activeJobs[evt.RoomID.String()] = append(m.activeJobs[evt.RoomID.String()], job)
		m.activeJobsMutex.Unlock()

		defer func() {
			// Mark job as complete
			m.activeJobsMutex.Lock()
			job.Cancel()
			for i, j := range m.activeJobs[evt.RoomID.String()] {
				if j.UUID == job.UUID {
					m.activeJobs[evt.RoomID.String()] = append(m.activeJobs[evt.RoomID.String()][:i], m.activeJobs[evt.RoomID.String()][i+1:]...)
					break
				}
			}
			m.activeJobsMutex.Unlock()
		}()

		res := a.Ask(
			agentOptions...,
		)

		if res.Response == "" {
			xlog.Debug(fmt.Sprintf("Empty response from agent"))
			return
		}

		if res.Error != nil {
			xlog.Error(fmt.Sprintf("Error from agent: %v", res.Error))
			return
		}

		m.conversationTracker.AddMessage(
			evt.RoomID.String(), openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: res.Response,
			},
		)

		// Send the response to the room
		_, err := m.client.SendText(context.Background(), evt.RoomID, res.Response)
		if err != nil {
			xlog.Error(fmt.Sprintf("Error sending message: %v", err))
		}
	}()
}

func (m *Matrix) Start(a *agent.Agent) {
	// Create Matrix client
	client, err := mautrix.NewClient(m.homeserverURL, id.UserID(m.userID), m.accessToken)
	if err != nil {
		xlog.Error(fmt.Sprintf("Error creating Matrix client: %v", err))
		return
	}
	xlog.Info("Matrix client created")
	m.client = client

	// Set up event handler
	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		xlog.Info("Received message", evt.Content.AsMessage().Body)
		m.handleRoomMessage(a, evt)
	})

	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := client.JoinRoomByID(ctx, evt.RoomID)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error joining room: %v", err))
			}
			xlog.Info(fmt.Sprintf("Joined room: %s (%s)", evt.RoomID.String(), evt.RoomID.URI()))
		}
	})

	syncer.OnEventType(event.EventEncrypted, func(ctx context.Context, evt *event.Event) {
		xlog.Info("Received encrypted message, this does not work yet", evt.RoomID.String())
		//m.handleRoomMessage(a, evt)
	})

	// Start syncing
	go func() {
		for {
			err := client.SyncWithContext(a.Context())

			xlog.Info("Syncing")
			if err != nil {
				xlog.Error(fmt.Sprintf("Error syncing: %v", err))
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// Handle shutdown
	go func() {
		<-a.Context().Done()
		client.StopSync()
	}()
}

// MatrixConfigMeta returns the metadata for Matrix connector configuration fields
func MatrixConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "homeserverURL",
			Label:    "Homeserver URL",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "userID",
			Label:    "User ID",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "accessToken",
			Label:    "Access Token",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:  "roomID",
			Label: "Room ID",
			Type:  config.FieldTypeText,
		},
		{
			Name:  "roomMode",
			Label: "Room Mode",
			Type:  config.FieldTypeCheckbox,
		},
		{
			Name:         "lastMessageDuration",
			Label:        "Last Message Duration",
			Type:         config.FieldTypeText,
			DefaultValue: "5m",
		},
	}
}
