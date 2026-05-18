/*
项目名称：图床转站助手
文件名称：001_init_schema.sql
创建时间：2026-05-13 01:52:26

系统用户：jerion
作　　者：Jerion
联系邮箱：416685476@qq.com
功能描述：数据库初始化脚本，创建所有表和索引
*/

-- ============================
-- 用户表
-- ============================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================
-- 密码重置令牌表
-- ============================
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    request_ip VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================
-- 图床配置表
-- ============================
CREATE TABLE IF NOT EXISTS picbed_configs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    picbed_type VARCHAR(20) NOT NULL,
    config_name VARCHAR(100) NOT NULL,
    encrypted_config TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, config_name)
);

-- ============================
-- 转换记录表
-- ============================
CREATE TABLE IF NOT EXISTS conversion_records (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_filename VARCHAR(255) NOT NULL,
    source_picbed VARCHAR(20) NOT NULL,
    target_picbed VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    image_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================
-- 索引
-- ============================
CREATE INDEX IF NOT EXISTS idx_picbed_configs_user_id ON picbed_configs(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_picbed_configs_user_default ON picbed_configs(user_id) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_conversion_records_user_id ON conversion_records(user_id);
CREATE INDEX IF NOT EXISTS idx_conversion_records_created_at ON conversion_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- ============================
-- 更新时间触发器函数
-- ============================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================
-- 为用户表添加更新时间触发器
-- ============================
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================
-- 为图床配置表添加更新时间触发器
-- ============================
DROP TRIGGER IF EXISTS update_picbed_configs_updated_at ON picbed_configs;
CREATE TRIGGER update_picbed_configs_updated_at
    BEFORE UPDATE ON picbed_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================
-- 插入测试数据（可选，仅用于开发环境）
-- ============================
-- INSERT INTO users (username, password_hash, email)
-- VALUES ('testuser', '$2a$10$...', 'test@example.com');
