package model

import "time"

type ConversionRecordDetail struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	RecordID    uint             `gorm:"not null;index" json:"record_id"`
	OriginalURL string           `gorm:"type:text;not null" json:"original_url"`
	TargetURL   string           `gorm:"type:text" json:"target_url"`
	Status      string           `gorm:"size:20;not null" json:"status"`
	Error       string           `gorm:"type:text" json:"error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	Record      ConversionRecord `gorm:"foreignKey:RecordID;constraint:OnDelete:CASCADE" json:"-"`
}

func (ConversionRecordDetail) TableName() string {
	return "conversion_record_details"
}
