package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jerion/picbed-switcher/docs"
	"github.com/jerion/picbed-switcher/internal/buildinfo"
	"github.com/jerion/picbed-switcher/internal/config"
	"github.com/jerion/picbed-switcher/internal/database"
	"github.com/jerion/picbed-switcher/internal/handler"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// flagValues 汇总命令行参数的显式取值，零值表示未设置
type flagValues struct {
	envPath    string
	host       string
	port       string
	mode       string
	dbHost     string
	dbPort     string
	dbName     string
	dbUser     string
	dbPassword string
	dbSSLMode  string
	jwtSecret  string
}

// versionText 组装 -v/--version 输出的版本信息文本
func versionText() string {
	return fmt.Sprintf("picbed-switcher %s\ncommit: %s\nbuild: %s\ngo: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate, runtime.Version())
}

// setUsage 自定义 -h/--help 输出：-v 与 -version 合并一行，各参数描述统一换行缩进对齐
func setUsage() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			if f.Name == "version" {
				return
			}
			if f.Name == "v" {
				fmt.Fprintf(w, "  -v, -version\n    \t%s\n", f.Usage)
				return
			}
			name, usage := flag.UnquoteUsage(f)
			fmt.Fprintf(w, "  -%s %s\n    \t%s (default %q)\n", f.Name, name, usage, f.DefValue)
		})
	}
}

// overridesFromFlags 将命令行参数显式值映射为配置加载的覆盖键值
func overridesFromFlags(v flagValues) map[string]string {
	overrides := make(map[string]string)
	if v.envPath != "" {
		overrides["env"] = v.envPath
	}
	if v.host != "" {
		overrides["host"] = v.host
	}
	if v.port != "" {
		overrides["port"] = v.port
	}
	if v.mode != "" {
		overrides["mode"] = v.mode
	}
	if v.dbHost != "" {
		overrides["db_host"] = v.dbHost
	}
	if v.dbPort != "" {
		overrides["db_port"] = v.dbPort
	}
	if v.dbName != "" {
		overrides["db_name"] = v.dbName
	}
	if v.dbUser != "" {
		overrides["db_user"] = v.dbUser
	}
	if v.dbPassword != "" {
		overrides["db_password"] = v.dbPassword
	}
	if v.dbSSLMode != "" {
		overrides["db_sslmode"] = v.dbSSLMode
	}
	if v.jwtSecret != "" {
		overrides["jwt_secret"] = v.jwtSecret
	}
	return overrides
}

// loadDotEnv 加载 .env 配置文件；显式指定路径且文件不存在时输出警告
func loadDotEnv(customPath string) {
	envFile := ".env"
	if customPath != "" {
		envFile = customPath
	}
	if err := godotenv.Load(envFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if customPath != "" {
				log.Printf("Warning: .env file not found: %s", envFile)
			}
			return
		}
		log.Printf("Warning: failed to load .env: %v", err)
	}
}

// @title PicBed Switcher API
// @version 1.0
// @description Markdown 图床批量转换平台 API
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	showVersion := flag.Bool("v", false, "显示版本信息并退出")
	flag.BoolVar(showVersion, "version", false, "显示版本信息并退出")
	envPath := flag.String("env", "", ".env 配置文件路径，默认加载工作目录下的 .env")
	host := flag.String("host", "", "后端监听主机，等价环境变量 SERVER_HOST")
	port := flag.String("port", "", "后端监听端口，等价环境变量 SERVER_PORT")
	mode := flag.String("mode", "", "Gin 运行模式（debug/release/test），等价环境变量 GIN_MODE")
	dbHost := flag.String("db-host", "", "数据库主机，等价环境变量 DB_HOST")
	dbPort := flag.String("db-port", "", "数据库端口，等价环境变量 DB_PORT")
	dbName := flag.String("db-name", "", "数据库名称，等价环境变量 DB_NAME")
	dbUser := flag.String("db-user", "", "数据库用户，等价环境变量 DB_USER")
	dbPassword := flag.String("db-password", "", "数据库密码，等价环境变量 DB_PASSWORD")
	dbSSLMode := flag.String("db-sslmode", "", "数据库 SSL 模式，等价环境变量 DB_SSLMODE")
	jwtSecret := flag.String("jwt-secret", "", "JWT 签名密钥，等价环境变量 JWT_SECRET")
	setUsage()
	flag.Parse()

	if *showVersion {
		fmt.Print(versionText())
		return
	}

	overrides := overridesFromFlags(flagValues{
		envPath:    *envPath,
		host:       *host,
		port:       *port,
		mode:       *mode,
		dbHost:     *dbHost,
		dbPort:     *dbPort,
		dbName:     *dbName,
		dbUser:     *dbUser,
		dbPassword: *dbPassword,
		dbSSLMode:  *dbSSLMode,
		jwtSecret:  *jwtSecret,
	})

	loadDotEnv(*envPath)

	cfg := config.LoadWithOverrides(overrides)
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
	if cfg.Redis.Enabled {
		redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("Failed to connect Redis: %v", err)
		}
		defer redisClient.Close()
		api.UseRedis(redisClient)
		log.Printf("Redis conversion queue enabled: %s", cfg.Redis.ConvertQueue)
	}
	defer api.Close()
	api.Register(router)

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("PicBed Switcher %s starting on %s...", buildinfo.Version, addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
