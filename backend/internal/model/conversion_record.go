/*
项目名称：图床转站助手
文件名称：conversion_record.go
创建时间：2026-05-13 01:52:26

系统用户：jerion
作　　者：Jerion
联系邮箱：416685476@qq.com
功能描述：转换记录数据模型
*/

package model

import (
	"time"
)

// ConversionRecord 转换记录模型
type ConversionRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"not null;index" json:"user_id"`
	OriginalFilename string    `gorm:"size:255;not null" json:"original_filename"`
	SourcePicBed     string    `gorm:"size:20;not null" json:"source_picbed"`
	TargetPicBed     string    `gorm:"size:20;not null" json:"target_picbed"`
	Status           string    `gorm:"size:20;not null" json:"status"` // success/failed/processing
	ErrorMessage     string    `gorm:"type:text" json:"error_message,omitempty"`
	ImageCount       int       `gorm:"default:0" json:"image_count"`
	CreatedAt        time.Time `gorm:"index:idx_created_at" json:"created_at"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName 指定表名
func (ConversionRecord) TableName() string {
	return "conversion_records"
}
