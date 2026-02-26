package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupCall struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ChatID    string     `gorm:"type:varchar(36);not null;index" json:"chat_id"`
	StartedBy string     `gorm:"type:varchar(36);not null" json:"started_by"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`

	Chat         Chat                  `gorm:"foreignKey:ChatID" json:"chat,omitempty"`
	Starter      User                  `gorm:"foreignKey:StartedBy" json:"starter,omitempty"`
	Participants []GroupCallParticipant `gorm:"foreignKey:CallID" json:"participants,omitempty"`
}

func (gc *GroupCall) BeforeCreate(tx *gorm.DB) error {
	if gc.ID == "" {
		gc.ID = uuid.New().String()
	}
	return nil
}
