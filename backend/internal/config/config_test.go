package config

import "testing"

func TestLoadWithOverrides(t *testing.T) {
	t.Run("默认值", func(t *testing.T) {
		t.Setenv("SERVER_HOST", "")
		t.Setenv("SERVER_PORT", "")
		cfg := LoadWithOverrides(nil)
		if cfg.Server.Host != "localhost" || cfg.Server.Port != "8080" {
			t.Fatalf("unexpected server config: %+v", cfg.Server)
		}
	})

	t.Run("环境变量生效", func(t *testing.T) {
		t.Setenv("SERVER_HOST", "0.0.0.0")
		t.Setenv("SERVER_PORT", "9090")
		cfg := LoadWithOverrides(nil)
		if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != "9090" {
			t.Fatalf("env values not applied: %+v", cfg.Server)
		}
	})

	t.Run("覆盖值优先于环境变量", func(t *testing.T) {
		t.Setenv("SERVER_HOST", "0.0.0.0")
		t.Setenv("SERVER_PORT", "9090")
		cfg := LoadWithOverrides(map[string]string{"host": "127.0.0.1"})
		if cfg.Server.Host != "127.0.0.1" {
			t.Fatalf("override not applied: %+v", cfg.Server)
		}
		if cfg.Server.Port != "9090" {
			t.Fatalf("env value should be kept when override is empty: %+v", cfg.Server)
		}
	})

	t.Run("空覆盖值回退环境变量", func(t *testing.T) {
		t.Setenv("DB_HOST", "db-from-env")
		cfg := LoadWithOverrides(map[string]string{"db_host": ""})
		if cfg.Database.Host != "db-from-env" {
			t.Fatalf("empty override should fall back to env: %+v", cfg.Database)
		}
	})

	t.Run("数据库与 JWT 覆盖值生效", func(t *testing.T) {
		cfg := LoadWithOverrides(map[string]string{
			"db_host":    "db.example.com",
			"db_port":    "6543",
			"db_name":    "picbed_test",
			"jwt_secret": "cli-secret",
		})
		if cfg.Database.Host != "db.example.com" || cfg.Database.Port != "6543" || cfg.Database.Name != "picbed_test" {
			t.Fatalf("database overrides not applied: %+v", cfg.Database)
		}
		if cfg.JWT.Secret != "cli-secret" {
			t.Fatalf("jwt secret override not applied: %+v", cfg.JWT)
		}
	})

	t.Run("Load 保持兼容行为", func(t *testing.T) {
		t.Setenv("SERVER_PORT", "7070")
		cfg := Load()
		if cfg.Server.Port != "7070" {
			t.Fatalf("Load should read env vars: %+v", cfg.Server)
		}
	})
}
