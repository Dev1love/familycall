package models

import "time"

type GroupCallParticipant struct {
	CallID   string     `gorm:"type:varchar(36);primaryKey" json:"call_id"`
	UserID   string     `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`

	User User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Call GroupCall  `gorm:"foreignKey:CallID" json:"call,omitempty"`
}
