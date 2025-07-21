package models

import (
	"time"

	"github.com/google/uuid"
)

type ActionExecution struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	ActionName string    `gorm:"type:varchar(255);not null;index" json:"actionName"`
	Status     string    `gorm:"type:varchar(50);not null;index" json:"status"` // "success" or "error"
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
