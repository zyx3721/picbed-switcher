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
	BaseURL                      string
	PasswordResetTokenTTLMinutes int
}

func Load() *Config {
	expireHours, err := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	if err != nil || expireHours <= 0 {
		expireHours = 24
	}
	resetTTL, err := strconv.Atoi(getEnv("PASSWORD_RESET_TOKEN_TTL_MINUTES", "30"))
	if err != nil || resetTTL <= 0 {
		resetTTL = 30
	}

	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "picbed_switcher"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "dev-change-me"),
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
			BaseURL:                      getEnv("APP_BASE_URL", "http://localhost:5173"),
			PasswordResetTokenTTLMinutes: resetTTL,
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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
