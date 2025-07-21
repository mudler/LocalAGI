package agent

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/core/sse"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"gorm.io/datatypes"
)

type Observer interface {
	NewObservable() *types.Observable
	Update(types.Observable)
	History() []types.Observable
}

type SSEObserver struct {
	agent   string
	userID  uuid.UUID
	agentID uuid.UUID
	manager sse.Manager
	mutex   sync.Mutex
}

func NewSSEObserver(agent string, manager sse.Manager) *SSEObserver {
	return &SSEObserver{
		agent:   agent,
		manager: manager,
	}
}

func NewSSEObserverWithIDs(agent string, userID, agentID uuid.UUID, manager sse.Manager) *SSEObserver {
	return &SSEObserver{
		agent:   agent,
		userID:  userID,
		agentID: agentID,
		manager: manager,
	}
}

func (s *SSEObserver) NewObservable() *types.Observable {
	return &types.Observable{
		ID:    uuid.New().String(),
		Agent: s.agent,
	}
}

func (s *SSEObserver) Update(obs types.Observable) {
	// Send SSE message first (for real-time updates)
	data, err := json.Marshal(obs)
	if err != nil {
		xlog.Error("Error marshaling observable", "error", err)
		return
	}
	msg := sse.NewMessage(string(data)).WithEvent("observable_update")
	s.manager.Send(msg)

	if s.userID == uuid.Nil || s.agentID == uuid.Nil {
		xlog.Debug("Observer missing userID or agentID, skipping database storage", "userID", s.userID, "agentID", s.agentID)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var dbObs models.Observable

	obsID, err := uuid.Parse(obs.ID)
	if err != nil {
		xlog.Error("Error parsing observable ID as UUID", "error", err, "id", obs.ID)
		return
	}

	err = db.DB.Where("ID = ? AND UserID = ? AND AgentID = ?", obsID, s.userID, s.agentID).First(&dbObs).Error
	if err != nil {

		dbObs = models.Observable{
			ID:      obsID,
			Agent:   obs.Agent,
			Name:    obs.Name,
			Icon:    obs.Icon,
			UserID:  s.userID,
			AgentID: s.agentID,
		}

		if obs.ParentID != "" {
			parentID, err := uuid.Parse(obs.ParentID)
			if err != nil {
				xlog.Error("Error parsing parent ID as UUID", "error", err, "parentID", obs.ParentID)
			} else {
				dbObs.ParentID = &parentID
			}
		}
	}

	if obs.Creation != nil {
		creationJSON, err := json.Marshal(obs.Creation)
		if err != nil {
			xlog.Error("Error marshaling creation", "error", err)
		} else {
			dbObs.Creation = datatypes.JSON(creationJSON)
		}
	}

	if len(obs.Progress) > 0 {
		progressJSON, err := json.Marshal(obs.Progress)
		if err != nil {
			xlog.Error("Error marshaling progress", "error", err)
		} else {
			dbObs.Progress = datatypes.JSON(progressJSON)
		}
	}

	if obs.Completion != nil {
		completionJSON, err := json.Marshal(obs.Completion)
		if err != nil {
			xlog.Error("Error marshaling completion", "error", err)
		} else {
			dbObs.Completion = datatypes.JSON(completionJSON)
		}
	}

	if err := db.DB.Save(&dbObs).Error; err != nil {
		xlog.Error("Error saving observable to database", "error", err, "observable_id", obs.ID)
	}
}

func (s *SSEObserver) History() []types.Observable {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.userID == uuid.Nil || s.agentID == uuid.Nil {
		xlog.Debug("Observer missing userID or agentID, returning empty history", "userID", s.userID, "agentID", s.agentID)
		return []types.Observable{}
	}

	var dbObservables []models.Observable

	err := db.DB.Where("UserID = ? AND AgentID = ?", s.userID, s.agentID).
		Order("CreatedAt DESC").
		Limit(20).
		Find(&dbObservables).Error

	if err != nil {
		xlog.Error("Error fetching observable history from database", "error", err)
		return []types.Observable{}
	}

	history := make([]types.Observable, 0, len(dbObservables))
	for _, dbObs := range dbObservables {
		obs := types.Observable{
			ID:    dbObs.ID.String(),
			Agent: dbObs.Agent,
			Name:  dbObs.Name,
			Icon:  dbObs.Icon,
		}

		if dbObs.ParentID != nil {
			obs.ParentID = dbObs.ParentID.String()
		}

		if len(dbObs.Creation) > 0 {
			var creation types.Creation
			if err := json.Unmarshal(dbObs.Creation, &creation); err == nil {
				obs.Creation = &creation
			}
		}

		if len(dbObs.Progress) > 0 {
			var progress []types.Progress
			if err := json.Unmarshal(dbObs.Progress, &progress); err == nil {
				obs.Progress = progress
			}
		}

		if len(dbObs.Completion) > 0 {
			var completion types.Completion
			if err := json.Unmarshal(dbObs.Completion, &completion); err == nil {
				obs.Completion = &completion
			}
		}

		history = append(history, obs)
	}

	return history
}
