package models

import (
	"time"

	"github.com/google/uuid"
)

type AgentMessage struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	AgentID   uuid.UUID `gorm:"type:char(36);index;not null" json:"agentId"`
	Sender    string    `gorm:"type:varchar(255);not null" json:"sender"` // "user" or "agent"
	Content   string    `gorm:"type:text;not null" json:"content"`
	Timestamp time.Time `gorm:"not null" json:"timestamp"`
}
