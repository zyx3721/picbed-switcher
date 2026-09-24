package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Mail     MailConfig
	App      AppConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Host string
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type MailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
	Security string
}

type AppConfig struct {
	BaseURL                          string
	PasswordResetTokenTTLMinutes     int
	EmailVerificationTokenTTLMinutes int
}

type RedisConfig struct {
	Enabled           bool
	Addr              string
	Password          string
	DB                int
	ConvertQueue      string
	WorkerConcurrency int
}

// Load 加载配置，等价于 LoadWithOverrides(nil)
func Load() *Config {
	return LoadWithOverrides(nil)
}

// LoadWithOverrides 加载配置，overrides 中的显式值优先于环境变量（含 .env 文件），
// 取值优先级为：命令行参数覆盖 > 环境变量 > 默认值
func LoadWithOverrides(overrides map[string]string) *Config {
	expireHours, err := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	if err != nil || expireHours <= 0 {
		expireHours = 24
	}
	resetTTL, err := strconv.Atoi(getEnv("PASSWORD_RESET_TOKEN_TTL_MINUTES", "5"))
	if err != nil || resetTTL <= 0 {
		resetTTL = 5
	}
	verificationTTL, err := strconv.Atoi(getEnv("EMAIL_VERIFICATION_TOKEN_TTL_MINUTES", "5"))
	if err != nil || verificationTTL <= 0 {
		verificationTTL = 5
	}
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil || redisDB < 0 {
		redisDB = 0
	}
	workerConcurrency, err := strconv.Atoi(getEnv("CONVERT_WORKER_CONCURRENCY", "1"))
	if err != nil || workerConcurrency <= 0 {
		workerConcurrency = 1
	}

	return &Config{
		Server: ServerConfig{
			Host: lookupValue(overrides, "host", "SERVER_HOST", "localhost"),
			Port: lookupValue(overrides, "port", "SERVER_PORT", "8080"),
			Mode: lookupValue(overrides, "mode", "GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     lookupValue(overrides, "db_host", "DB_HOST", "localhost"),
			Port:     lookupValue(overrides, "db_port", "DB_PORT", "5432"),
			Name:     lookupValue(overrides, "db_name", "DB_NAME", "picbed_switcher"),
			User:     lookupValue(overrides, "db_user", "DB_USER", "postgres"),
			Password: lookupValue(overrides, "db_password", "DB_PASSWORD", "postgres"),
			SSLMode:  lookupValue(overrides, "db_sslmode", "DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:      lookupValue(overrides, "jwt_secret", "JWT_SECRET", "dev-change-me"),
			ExpireHours: expireHours,
		},
		Mail: MailConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", ""),
			FromName: getEnv("SMTP_FROM_NAME", ""),
			Security: mailSecurity(),
		},
		App: AppConfig{
			BaseURL:                          getEnv("APP_BASE_URL", "http://localhost:5173"),
			PasswordResetTokenTTLMinutes:     resetTTL,
			EmailVerificationTokenTTLMinutes: verificationTTL,
		},
		Redis: RedisConfig{
			Enabled:           boolEnv("REDIS_ENABLED", false),
			Addr:              getEnv("REDIS_ADDR", "localhost:6379"),
			Password:          getEnv("REDIS_PASSWORD", ""),
			DB:                redisDB,
			ConvertQueue:      getEnv("REDIS_CONVERT_QUEUE", "picbed:convert_tasks"),
			WorkerConcurrency: workerConcurrency,
		},
	}
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host,
		d.User,
		d.Password,
		d.Name,
		d.Port,
		d.SSLMode,
	)
}

func mailSecurity() string {
	security := getEnv("SMTP_SECURITY", "")
	if security != "" {
		return security
	}
	ssl := getEnv("SMTP_SSL", "")
	switch ssl {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return "ssl"
	case "0", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off":
		return "none"
	default:
		return "auto"
	}
}

func boolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off":
		return false
	default:
		return defaultValue
	}
}

// lookupValue 按 overrides 显式值、环境变量、默认值的顺序取值，
// overrides 的 key 为命令行参数对应的语义键，与具体环境变量名解耦
func lookupValue(overrides map[string]string, key, envKey, defaultValue string) string {
	if overrides != nil {
		if value, ok := overrides[key]; ok && value != "" {
			return value
		}
	}
	return getEnv(envKey, defaultValue)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
