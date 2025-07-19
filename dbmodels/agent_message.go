package models

import (
	"time"

	"github.com/google/uuid"
)

type AgentMessage struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	AgentID   uuid.UUID `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"agentId"`
	Sender    string    `gorm:"type:varchar(255);not null" json:"sender"` // "user" or "agent"
	Content   string    `gorm:"type:text;not null" json:"content"`
	Type      string    `gorm:"type:varchar(50);not null;default:'message'" json:"type"` // "message" or "error"
	CreatedAt time.Time `json:"createdAt"`

	Agent Agent `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
