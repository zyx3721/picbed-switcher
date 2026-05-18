package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// register godoc
// @Summary 用户注册
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authRequest true "注册信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/auth/register [post]
func (a *API) register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(c, http.StatusBadRequest, "用户名、邮箱或密码不能为空")
		return
	}
	if len(req.Username) < 3 {
		respondError(c, http.StatusBadRequest, "用户名至少需要 3 个字符")
		return
	}
	if !emailPattern.MatchString(req.Email) {
		respondError(c, http.StatusBadRequest, "邮箱格式不正确，请填写有效邮箱地址")
		return
	}
	if len(req.Password) < 6 {
		respondError(c, http.StatusBadRequest, "密码至少需要 6 个字符")
		return
	}
	var count int64
	if err := a.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "检查用户名失败")
		return
	}
	if count > 0 {
		respondError(c, http.StatusConflict, "用户名已存在，请更换用户名")
		return
	}
	if err := a.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "检查邮箱失败")
		return
	}
	if count > 0 {
		respondError(c, http.StatusConflict, "邮箱已存在，请更换邮箱")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "密码加密失败")
		return
	}
	user := model.User{Username: req.Username, PasswordHash: string(hash), Email: req.Email}
	if err := a.db.Create(&user).Error; err != nil {
		respondError(c, http.StatusConflict, "注册失败，请检查用户名和邮箱后重试")
		return
	}
	token, err := utils.GenerateToken(a.cfg.JWT.Secret, a.cfg.JWT.ExpireHours, user.ID, user.Username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "登录令牌生成失败")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": publicUser(user)})
}

// login godoc
// @Summary 用户名或邮箱登录
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authRequest true "登录信息"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/auth/login [post]
func (a *API) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	identity := strings.TrimSpace(req.Username)
	if identity == "" || req.Password == "" {
		respondError(c, http.StatusBadRequest, "用户名或邮箱、密码不能为空")
		return
	}
	var user model.User
	if err := a.db.Where("username = ? OR lower(email) = ?", identity, strings.ToLower(identity)).First(&user).Error; err != nil {
		respondError(c, http.StatusUnauthorized, "用户名或密码不正确")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(c, http.StatusUnauthorized, "用户名或密码不正确")
		return
	}
	token, err := utils.GenerateToken(a.cfg.JWT.Secret, a.cfg.JWT.ExpireHours, user.ID, user.Username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "登录令牌生成失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": publicUser(user)})
}

// forgotPassword godoc
// @Summary 发送密码重置邮件
// @Tags auth
// @Accept json
// @Produce json
// @Param request body forgotPasswordRequest true "邮箱信息"
// @Success 200 {object} map[string]string
// @Router /api/auth/password/forgot [post]
func (a *API) forgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		respondError(c, http.StatusBadRequest, "邮箱不能为空")
		return
	}
	if !emailPattern.MatchString(email) {
		respondError(c, http.StatusBadRequest, "邮箱格式不正确")
		return
	}

	var user model.User
	if err := a.db.Where("lower(email) = ?", email).First(&user).Error; err != nil {
		respondError(c, http.StatusNotFound, "邮箱不存在")
		return
	}

	token, err := generateResetToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "重置令牌生成失败")
		return
	}
	now := time.Now()
	reset := model.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashResetToken(token),
		ExpiresAt: now.Add(time.Duration(a.cfg.App.PasswordResetTokenTTLMinutes) * time.Minute),
		RequestIP: c.ClientIP(),
	}
	if err := a.db.Model(&model.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", user.ID).
		Update("used_at", now).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "重置令牌更新失败")
		return
	}
	if err := a.db.Create(&reset).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "重置令牌保存失败")
		return
	}

	mailCfg := utils.MailConfig{
		Host:     a.cfg.Mail.Host,
		Port:     a.cfg.Mail.Port,
		Username: a.cfg.Mail.Username,
		Password: a.cfg.Mail.Password,
		From:     a.cfg.Mail.From,
		FromName: a.cfg.Mail.FromName,
		Security: a.cfg.Mail.Security,
	}
	if err := utils.SendPasswordResetMail(mailCfg, email, passwordResetURL(a.cfg.App.BaseURL, token)); err != nil {
		usedAt := time.Now()
		_ = a.db.Model(&reset).Update("used_at", usedAt).Error
		respondError(c, http.StatusInternalServerError, "密码重置邮件发送失败，请检查配置")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码重置邮件已发送，请检查邮箱"})
}

// resetPassword godoc
// @Summary 使用重置令牌设置新密码
// @Tags auth
// @Accept json
// @Produce json
// @Param request body resetPasswordRequest true "重置密码信息"
// @Success 200 {object} map[string]string
// @Router /api/auth/password/reset [post]
func (a *API) resetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		respondError(c, http.StatusBadRequest, "重置令牌不能为空")
		return
	}
	if len(req.NewPassword) < 6 || len(req.ConfirmPassword) < 6 {
		respondError(c, http.StatusBadRequest, "密码至少 6 个字符")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		respondError(c, http.StatusBadRequest, "新密码与确认密码不一致")
		return
	}

	var reset model.PasswordResetToken
	if err := a.db.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hashResetToken(token), time.Now()).First(&reset).Error; err != nil {
		respondError(c, http.StatusBadRequest, "重置链接无效或已过期")
		return
	}
	var user model.User
	if err := a.db.First(&user, reset.UserID).Error; err != nil {
		respondError(c, http.StatusNotFound, "用户不存在")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.NewPassword)); err == nil {
		respondError(c, http.StatusBadRequest, "新密码不能与旧密码相同")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "密码加密失败")
		return
	}
	usedAt := time.Now()
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
			return err
		}
		return tx.Model(&reset).Update("used_at", usedAt).Error
	}); err != nil {
		respondError(c, http.StatusInternalServerError, "密码重置失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置，请重新登录"})
}

// profile godoc
// @Summary 获取当前用户信息
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/auth/profile [get]
func (a *API) profile(c *gin.Context) {
	var user model.User
	if err := a.db.First(&user, middleware.UserID(c)).Error; err != nil {
		respondError(c, http.StatusNotFound, "用户不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": publicUser(user)})
}

// changePassword godoc
// @Summary 修改当前用户密码
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body changePasswordRequest true "密码信息"
// @Success 200 {object} map[string]string
// @Router /api/auth/password [put]
func (a *API) changePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if len(req.OldPassword) < 6 || len(req.NewPassword) < 6 || len(req.ConfirmPassword) < 6 {
		respondError(c, http.StatusBadRequest, "密码至少 6 个字符")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		respondError(c, http.StatusBadRequest, "新密码与确认密码不一致")
		return
	}
	if req.NewPassword == req.OldPassword {
		respondError(c, http.StatusBadRequest, "新密码不能与旧密码相同")
		return
	}

	var user model.User
	if err := a.db.First(&user, middleware.UserID(c)).Error; err != nil {
		respondError(c, http.StatusNotFound, "用户不存在")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		respondError(c, http.StatusUnauthorized, "旧密码不正确")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "密码加密失败")
		return
	}
	if err := a.db.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "密码修改失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已修改"})
}

// changeEmail godoc
// @Summary 修改当前用户邮箱
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body changeEmailRequest true "邮箱信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/auth/email [put]
func (a *API) changeEmail(c *gin.Context) {
	var req changeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if newEmail == "" {
		respondError(c, http.StatusBadRequest, "邮箱不能为空")
		return
	}
	if !emailPattern.MatchString(newEmail) {
		respondError(c, http.StatusBadRequest, "邮箱格式不正确")
		return
	}

	var user model.User
	if err := a.db.First(&user, middleware.UserID(c)).Error; err != nil {
		respondError(c, http.StatusNotFound, "用户不存在")
		return
	}
	if strings.EqualFold(user.Email, newEmail) {
		respondError(c, http.StatusBadRequest, "新邮箱不能与当前邮箱相同")
		return
	}
	var count int64
	if err := a.db.Model(&model.User{}).Where("lower(email) = ? AND id <> ?", newEmail, user.ID).Count(&count).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "检查邮箱失败")
		return
	}
	if count > 0 {
		respondError(c, http.StatusConflict, "邮箱已存在，请更换邮箱")
		return
	}
	if err := a.db.Model(&user).Update("email", newEmail).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "邮箱修改失败")
		return
	}
	user.Email = newEmail
	c.JSON(http.StatusOK, gin.H{"message": "邮箱已修改", "user": publicUser(user)})
}
func publicUser(user model.User) gin.H {
	return gin.H{"id": user.ID, "username": user.Username, "email": user.Email, "created_at": user.CreatedAt}
}

func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func passwordResetURL(baseURL, token string) string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = "http://localhost:5173"
	}
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Sprintf("%s?reset_token=%s", strings.TrimRight(base, "/"), url.QueryEscape(token))
	}
	query := u.Query()
	query.Set("reset_token", token)
	u.RawQuery = query.Encode()
	return u.String()
}
