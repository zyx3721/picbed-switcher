package model

import "time"

type ConversionTask struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	TaskType  string     `gorm:"size:20;not null" json:"task_type"`
	Status    string     `gorm:"size:20;not null;index" json:"status"`
	Total     int        `gorm:"default:0" json:"total"`
	Success   int        `gorm:"default:0" json:"success"`
	Failed    int        `gorm:"default:0" json:"failed"`
	Message   string     `gorm:"size:255" json:"message"`
	Payload   string     `gorm:"type:text" json:"-"`
	Error     string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}

func (ConversionTask) TableName() string {
	return "conversion_tasks"
}
