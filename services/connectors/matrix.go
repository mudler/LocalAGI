package connectors

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/xlog"
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
}

const matrixThinkingMessage = "🤔 thinking..."

func NewMatrix(config map[string]string) *Matrix {

	return &Matrix{
		homeserverURL: config["homeserverURL"],
		userID:        config["userID"],
		accessToken:   config["accessToken"],
		roomID:        config["roomID"],
		roomMode:      config["roomMode"] == "true",
		placeholders:  make(map[string]string),
		activeJobs:    make(map[string][]*types.Job),
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

func (m *Matrix) handleRoomMessage(a *agent.Agent, evt *event.Event) {
	if m.roomID != evt.RoomID.String() && m.roomMode { // If we have a roomID and it's not the same as the event room
		// Skip messages from other rooms
		xlog.Info("Skipping reply to room", "event room", evt.RoomID, "config room", m.roomID)
		return
	}

	if evt.Sender == id.UserID(m.userID) {
		// Skip messages from ourselves
		return
	}

	// Skip if message does not mention the bot
	mentioned := false
	msg := evt.Content.AsMessage()
	if msg.Mentions != nil {
		mentioned = slices.Contains(evt.Content.AsMessage().Mentions.UserIDs, m.client.UserID)
	}

	if !mentioned && !m.roomMode {
		xlog.Info("Skipping reply because it does not mention the bot", "mentions", evt.Content.AsMessage().Mentions.UserIDs)
		return
	}

	currentConv := a.SharedState().ConversationTracker.GetConversation(fmt.Sprintf("matrix:%s", evt.RoomID.String()))

	message := evt.Content.AsMessage().Body

	go func() {
		agentOptions := []types.JobOption{
			types.WithUUID(evt.ID.String()),
		}

		currentConv = append(currentConv, openai.ChatCompletionMessage{
			Role:    "user",
			Content: message,
		})

		a.SharedState().ConversationTracker.AddMessage(
			fmt.Sprintf("matrix:%s", evt.RoomID.String()), currentConv[len(currentConv)-1],
		)

		agentOptions = append(agentOptions, types.WithConversationHistory(currentConv))

		// Add room and conversation_id for tracking and cancel-previous-on-new-message
		metadata := map[string]any{
			"room":                          evt.RoomID.String(),
			types.MetadataKeyConversationID: "matrix:" + evt.RoomID.String(),
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
					m.activeJobs[evt.RoomID.String()] = slices.Delete(m.activeJobs[evt.RoomID.String()], i, i+1)
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

		a.SharedState().ConversationTracker.AddMessage(
			fmt.Sprintf("matrix:%s", evt.RoomID.String()), openai.ChatCompletionMessage{
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
	client, err := mautrix.NewClient(m.homeserverURL, id.UserID(m.userID), m.accessToken)
	if err != nil {
		xlog.Error(fmt.Sprintf("Error creating Matrix client: %v", err))
		return
	}
	xlog.Info("Matrix client created")
	m.client = client

	if m.roomID != "" {
		// handle new conversations
		a.AddSubscriber(func(ccm *types.ConversationMessage) {
			xlog.Debug("Subscriber(matrix)", "message", ccm.Message.Content)
			_, err := m.client.SendText(context.Background(), id.RoomID(m.roomID), ccm.Message.Content)
			if err != nil {
				xlog.Error(fmt.Sprintf("Error posting message: %v", err))
			}
			a.SharedState().ConversationTracker.AddMessage(
				fmt.Sprintf("matrix:%s", m.roomID),
				openai.ChatCompletionMessage{
					Content: ccm.Message.Content,
					Role:    "assistant",
				},
			)
		})
	}

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

	// This prevents the agent from picking up a backlog of messages and swamping the chat with responses.
	syncer.FilterJSON = &mautrix.Filter{
		Room: mautrix.RoomFilter{
			Timeline: mautrix.FilterPart{
				Limit: 1,
			},
		},
	}

	go func() {
		for {
			select {
			case <-a.Context().Done():
				xlog.Info("Context cancelled, stopping sync loop")
				return
			default:
				err := client.SyncWithContext(a.Context())

				xlog.Info("Syncing")
				if err != nil {
					xlog.Error(fmt.Sprintf("Error syncing: %v", err))
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

// MatrixConfigMeta returns the metadata for Matrix connector configuration fields
func MatrixConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "homeserverURL",
			Label:    "Homeserver URL",
			HelpText: "e.g. http://host.docker.internal:8008",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "userID",
			Label:    "User ID",
			HelpText: "e.g. @bot:host",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "accessToken",
			Label:    "Access Token",
			HelpText: "Token obtained from _matrix/client/v3/login",
			Type:     config.FieldTypeText,
			Required: true,
		},
		{
			Name:     "roomID",
			Label:    "Internal Room ID",
			HelpText: "The autogenerated unique identifier for a room",
			Type:     config.FieldTypeText,
		},
		{
			Name:     "roomMode",
			Label:    "Room Mode",
			HelpText: "Respond to all messages in the specified room",
			Type:     config.FieldTypeCheckbox,
		},
	}
}
