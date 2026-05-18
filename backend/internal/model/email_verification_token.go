package model

import "time"

type EmailVerificationToken struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	TokenHash string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	RequestIP string     `gorm:"size:64" json:"request_ip"`
	CreatedAt time.Time  `json:"created_at"`
	User      User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}
