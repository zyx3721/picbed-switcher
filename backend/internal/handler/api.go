package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

type API struct {
	db            *gorm.DB
	cfg           *config.Config
	convertQueue  chan uint
	redis         *redis.Client
	workerCancel  context.CancelFunc
	workerStarted bool
}

func NewAPI(db *gorm.DB, cfg *config.Config) *API {
	return &API{db: db, cfg: cfg, convertQueue: make(chan uint, 100)}
}

func (a *API) UseRedis(client *redis.Client) {
	a.redis = client
}

func (a *API) Close() {
	if a.workerCancel != nil {
		a.workerCancel()
	}
}

// health godoc
// @Summary 服务健康检查
// @Tags base
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (a *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "PicBed Switcher API 服务运行正常"})
}
func (a *API) Register(router *gin.Engine) {
	a.startConvertWorkers()
	router.GET("/health", a.health)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	api := router.Group("/api")
	auth := api.Group("/auth")
	auth.POST("/register", a.register)
	auth.POST("/login", a.login)
	auth.POST("/email/verify", a.verifyEmail)
	auth.POST("/password/forgot", a.forgotPassword)
	auth.POST("/password/reset", a.resetPassword)
	protected := api.Group("")
	protected.Use(middleware.Auth(a.cfg.JWT.Secret))
	protected.GET("/auth/profile", a.profile)
	protected.POST("/auth/email/verification", a.resendEmailVerification)
	protected.PUT("/auth/password", a.changePassword)
	protected.PUT("/auth/email", a.changeEmail)
	protected.GET("/picbed/types", a.picbedTypes)
	protected.GET("/picbed/configs", a.listConfigs)
	protected.POST("/picbed/configs", a.createConfig)
	protected.POST("/picbed/configs/test", a.testConfigDraft)
	protected.PUT("/picbed/configs/:id", a.updateConfig)
	protected.DELETE("/picbed/configs/:id", a.deleteConfig)
	protected.PUT("/picbed/configs/:id/default", a.setDefaultConfig)
	protected.POST("/picbed/configs/:id/test", a.testConfigSaved)
	protected.POST("/convert/analyze", a.analyzeMarkdown)
	protected.POST("/convert/process", a.convertMarkdown)
	protected.POST("/convert/batch", a.convertMarkdownBatch)
	protected.POST("/convert/local-batch", a.convertLocalMarkdownBatch)
	protected.POST("/convert/local-tasks", a.createLocalConvertTask)
	protected.POST("/convert/tasks", a.createConvertTask)
	protected.GET("/convert/tasks", a.listConvertTasks)
	protected.GET("/convert/tasks/:id", a.getConvertTask)
	protected.GET("/convert/records", a.listRecords)
	protected.GET("/convert/records/:id", a.getRecord)
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
