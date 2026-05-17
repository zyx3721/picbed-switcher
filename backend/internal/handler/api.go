package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"gorm.io/gorm"
)

type API struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAPI(db *gorm.DB, cfg *config.Config) *API { return &API{db: db, cfg: cfg} }

func (a *API) Register(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "PicBed Switcher API 服务运行正常"})
	})
	api := router.Group("/api")
	auth := api.Group("/auth")
	auth.POST("/register", a.register)
	auth.POST("/login", a.login)
	protected := api.Group("")
	protected.Use(middleware.Auth(a.cfg.JWT.Secret))
	protected.GET("/auth/profile", a.profile)
	protected.PUT("/auth/password", a.changePassword)
	protected.GET("/picbed/types", a.picbedTypes)
	protected.GET("/picbed/configs", a.listConfigs)
	protected.POST("/picbed/configs", a.createConfig)
	protected.PUT("/picbed/configs/:id", a.updateConfig)
	protected.DELETE("/picbed/configs/:id", a.deleteConfig)
	protected.PUT("/picbed/configs/:id/default", a.setDefaultConfig)
	protected.POST("/convert/analyze", a.analyzeMarkdown)
	protected.POST("/convert/process", a.convertMarkdown)
	protected.POST("/convert/batch", a.convertMarkdownBatch)
	protected.GET("/convert/records", a.listRecords)
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
