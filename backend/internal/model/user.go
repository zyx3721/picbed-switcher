/*
项目名称：图床转站助手
文件名称：user.go
创建时间：2026-05-13 01:52:26

系统用户：jerion
作　　者：Jerion
联系邮箱：416685476@qq.com
功能描述：用户数据模型
*/

package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Email        string    `gorm:"uniqueIndex;size:100" json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
