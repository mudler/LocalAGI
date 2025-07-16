package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reminder struct {
	ID          uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"userId"`
	AgentID     uuid.UUID  `gorm:"type:char(36);index;not null;constraint:OnDelete:CASCADE" json:"agentId"`
	Message     string     `gorm:"type:text;not null" json:"message"`
	CronExpr    string     `gorm:"type:varchar(255);not null" json:"cronExpr"`
	LastRun     *time.Time `gorm:"type:datetime" json:"lastRun"`
	NextRun     time.Time  `gorm:"type:datetime;not null" json:"nextRun"`
	IsRecurring bool       `gorm:"type:boolean;default:false;not null" json:"isRecurring"`
	Active      bool       `gorm:"type:boolean;default:true;not null" json:"active"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	User  User  `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Agent Agent `gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (r *Reminder) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()
	return
}
