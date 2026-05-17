/*
项目名称：图床转站助手
文件名称：picbed_config.go
创建时间：2026-05-13 01:52:26

系统用户：jerion
作　　者：Jerion
联系邮箱：416685476@qq.com
功能描述：图床配置数据模型
*/

package model

import (
	"time"
)

// PicBedConfig 图床配置模型
type PicBedConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	PicBedType      string    `gorm:"size:20;not null" json:"picbed_type"` // github/gitee/tencent/aliyun/qiniu
	ConfigName      string    `gorm:"size:100;not null" json:"config_name"`
	EncryptedConfig string    `gorm:"type:text;not null" json:"-"`
	IsDefault       bool      `gorm:"default:false" json:"is_default"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName 指定表名
func (PicBedConfig) TableName() string {
	return "picbed_configs"
}
