package database

import (
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.User{}, &model.PicBedConfig{}, &model.ConversionRecord{}); err != nil {
		return nil, err
	}

	return db, nil
}
