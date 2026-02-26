package models

import "time"

type ChatMember struct {
	ChatID     string    `gorm:"type:varchar(36);primaryKey" json:"chat_id"`
	UserID     string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Role       string    `gorm:"type:varchar(10);not null;default:member" json:"role"` // "admin" | "member"
	JoinedAt   time.Time `json:"joined_at"`
	LastReadAt time.Time `json:"last_read_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Chat Chat `gorm:"foreignKey:ChatID" json:"chat,omitempty"`
}
