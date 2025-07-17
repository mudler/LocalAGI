package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Observable struct {
	ID       uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	ParentID *uuid.UUID `gorm:"type:char(36);index" json:"parent_id,omitempty"`
	Agent    string     `gorm:"type:varchar(255);not null;index" json:"agent"`
	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	Icon     string     `gorm:"type:varchar(100)" json:"icon"`
	UserID   uuid.UUID  `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	AgentID  uuid.UUID  `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"agentId"`

	// JSON fields for complex data
	Creation   datatypes.JSON `gorm:"type:json" json:"creation,omitempty"`
	Progress   datatypes.JSON `gorm:"type:json" json:"progress,omitempty"`
	Completion datatypes.JSON `gorm:"type:json" json:"completion,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Foreign key relationships
	User       User         `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	AgentModel Agent        `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Parent     *Observable  `gorm:"foreignKey:ParentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Children   []Observable `gorm:"foreignKey:ParentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (o *Observable) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return
}
