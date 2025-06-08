package models

import (
	"time"

	"github.com/google/uuid"
)

type LLMUsage struct {
	ID               uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID           uuid.UUID `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	AgentID          uuid.UUID `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"agentId"`
	Model            string    `gorm:"type:varchar(100);not null" json:"model"`
	PromptTokens     int       `gorm:"not null" json:"promptTokens"`
	CompletionTokens int       `gorm:"not null" json:"completionTokens"`
	TotalTokens      int       `gorm:"not null" json:"totalTokens"`
	Cost             float64   `gorm:"type:decimal(10,6);not null" json:"cost"`
	RequestType      string    `gorm:"type:varchar(50);not null" json:"requestType"` // e.g., "completion", "chat", "embedding"
	GenID            string    `gorm:"type:varchar(100)" json:"genId"`               // OpenRouter generation ID
	CreatedAt        time.Time `json:"createdAt"`

	User  User  `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Agent Agent `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
