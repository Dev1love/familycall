package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ChatID    string     `gorm:"type:varchar(36);not null;index" json:"chat_id"`
	SenderID  string     `gorm:"type:varchar(36);not null;index" json:"sender_id"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`

	Sender User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Chat   Chat `gorm:"foreignKey:ChatID" json:"chat,omitempty"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
