package database

import (
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminEmail    = "admin@example.com"
	defaultAdminPassword = "123456"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.EmailVerificationToken{},
		&model.PasswordResetToken{},
		&model.PicBedConfig{},
		&model.ConversionTask{},
		&model.ConversionRecord{},
		&model.ConversionRecordDetail{},
	); err != nil {
		return nil, err
	}
	if err := seedDefaultAdmin(db); err != nil {
		return nil, err
	}

	return db, nil
}

func seedDefaultAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := model.User{Username: defaultAdminUsername, Email: defaultAdminEmail, PasswordHash: string(hash)}
	return db.Create(&admin).Error
}
