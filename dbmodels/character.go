package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Character struct {
	AgentID    uuid.UUID      `gorm:"type:char(36);primaryKey;constraint:OnDelete:CASCADE" json:"agentId"`
	UserID     uuid.UUID      `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	Age        string         `gorm:"type:varchar(50)" json:"age"`
	Occupation string         `gorm:"type:varchar(255)" json:"job_occupation"`
	Hobbies    datatypes.JSON `gorm:"type:json" json:"hobbies"`
	MusicTaste datatypes.JSON `gorm:"type:json" json:"favorites_music_genres"`
	Sex        string         `gorm:"type:varchar(50)" json:"sex"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`

	Agent Agent `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	User  User  `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
