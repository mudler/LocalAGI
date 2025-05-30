package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Agent struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:char(36);index;not null" json:"userId"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Config    datatypes.JSON `gorm:"type:json;not null" json:"config"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}
