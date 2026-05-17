package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/database"
	"github.com/jerion/picbed-switcher/internal/handler"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: failed to load .env: %v", err)
	}

	cfg := config.Load()
	gin.SetMode(cfg.Server.Mode)

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit(120, time.Minute))

	api := handler.NewAPI(db, cfg)
	api.Register(router)

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s...", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
