package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AgentState struct {
	AgentID     uuid.UUID      `gorm:"type:char(36);primaryKey;constraint:OnDelete:CASCADE" json:"agentId"`
	UserID      uuid.UUID      `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	NowDoing    string         `gorm:"type:text" json:"nowDoing"`
	DoingNext   string         `gorm:"type:text" json:"doingNext"`
	DoneHistory datatypes.JSON `gorm:"type:json" json:"doneHistory"`
	Memories    datatypes.JSON `gorm:"type:json" json:"memories"`
	Goal        string         `gorm:"type:text" json:"goal"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`

	Agent Agent `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	User  User  `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
