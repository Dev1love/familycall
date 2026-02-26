package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chat struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Type      string    `gorm:"type:varchar(10);not null;index" json:"type"` // "direct" | "group"
	Name      *string   `gorm:"type:varchar(200)" json:"name,omitempty"`
	CreatedBy string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Members  []ChatMember `gorm:"foreignKey:ChatID" json:"members,omitempty"`
	Messages []Message    `gorm:"foreignKey:ChatID" json:"messages,omitempty"`
	Creator  User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (c *Chat) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
