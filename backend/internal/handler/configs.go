package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/picbed"
	"github.com/jerion/picbed-switcher/internal/utils"
	"gorm.io/gorm"
)

// picbedTypes godoc
// @Summary 获取支持的图床类型与配置字段
// @Tags picbed
// @Produce json
// @Security BearerAuth
// @Success 200 {object} picbedTypesResponse
// @Failure 401 {object} errorResponse
// @Router /api/picbed/types [get]
func (a *API) picbedTypes(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"types": picbedTypeDefs}) }

// listConfigs godoc
// @Summary 获取所有图床配置
// @Tags picbed
// @Produce json
// @Security BearerAuth
// @Success 200 {object} configsResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs [get]
func (a *API) listConfigs(c *gin.Context) {
	userID := middleware.UserID(c)
	if err := a.normalizeDefaultConfig(userID); err != nil {
		respondError(c, http.StatusInternalServerError, "默认配置状态整理失败")
		return
	}
	var configs []model.PicBedConfig
	if err := a.db.Where("user_id = ?", userID).Order("is_default desc, updated_at desc").Find(&configs).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "读取图床配置失败")
		return
	}
	items := make([]gin.H, 0, len(configs))
	for _, item := range configs {
		configMap, _ := a.decryptConfigMap(item.EncryptedConfig)
		items = append(items, configResponse(item, true, editableConfig(item.PicBedType, configMap)))
	}
	c.JSON(http.StatusOK, gin.H{"configs": items})
}

// createConfig godoc
// @Summary 添加图床配置
// @Tags picbed
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body picbedConfigRequest true "图床配置"
// @Success 201 {object} configResponseDoc
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs [post]
func (a *API) createConfig(c *gin.Context) {
	var req picbedConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if err := validatePicbedConfig(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	encrypted, err := a.encryptConfig(normalizeConfig(req.PicBedType, req.Config))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "配置加密失败")
		return
	}
	userID := middleware.UserID(c)
	configName := strings.TrimSpace(req.ConfigName)
	exists, err := a.configNameExists(userID, configName, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "检查配置名称失败")
		return
	}
	if exists {
		respondError(c, http.StatusConflict, "配置名称不能重复")
		return
	}
	item := model.PicBedConfig{UserID: userID, PicBedType: req.PicBedType, ConfigName: configName, EncryptedConfig: encrypted, IsDefault: req.IsDefault}
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if req.IsDefault {
			if err := a.clearDefault(tx, userID, 0); err != nil {
				return err
			}
		}
		return tx.Create(&item).Error
	}); err != nil {
		respondError(c, http.StatusConflict, "配置名称不能重复")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"config": configResponse(item, true, nil)})
}

// updateConfig godoc
// @Summary 更新图床配置
// @Tags picbed
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "配置 ID"
// @Param request body picbedConfigRequest true "图床配置"
// @Success 200 {object} configResponseDoc
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs/{id} [put]
func (a *API) updateConfig(c *gin.Context) {
	item, ok := a.findConfig(c)
	if !ok {
		return
	}
	var req picbedConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if err := validatePicbedConfig(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	encrypted, err := a.encryptConfig(normalizeConfig(req.PicBedType, req.Config))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "配置加密失败")
		return
	}
	userID := middleware.UserID(c)
	configName := strings.TrimSpace(req.ConfigName)
	exists, err := a.configNameExists(userID, configName, item.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "检查配置名称失败")
		return
	}
	if exists {
		respondError(c, http.StatusConflict, "配置名称不能重复")
		return
	}
	item.PicBedType = req.PicBedType
	item.ConfigName = configName
	item.EncryptedConfig = encrypted
	item.IsDefault = req.IsDefault
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if req.IsDefault {
			if err := a.clearDefault(tx, userID, item.ID); err != nil {
				return err
			}
		}
		return tx.Save(&item).Error
	}); err != nil {
		respondError(c, http.StatusConflict, "保存图床配置失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": configResponse(item, true, nil)})
}

// testConfigDraft godoc
// @Summary 测试未保存的图床配置
// @Tags picbed
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body picbedConfigRequest true "图床配置"
// @Success 200 {object} messageResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs/test [post]
func (a *API) testConfigDraft(c *gin.Context) {
	var req picbedConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if err := validatePicbedConfig(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := picbed.TestConfig(c.Request.Context(), req.PicBedType, normalizeConfig(req.PicBedType, req.Config)); err != nil {
		respondError(c, http.StatusBadRequest, "配置测试失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "配置测试通过"})
}

// testConfigSaved godoc
// @Summary 测试已保存的图床配置
// @Tags picbed
// @Produce json
// @Security BearerAuth
// @Param id path int true "配置 ID"
// @Success 200 {object} messageResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs/{id}/test [post]
func (a *API) testConfigSaved(c *gin.Context) {
	item, ok := a.findConfig(c)
	if !ok {
		return
	}
	configMap, err := a.decryptConfigMap(item.EncryptedConfig)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "读取图床配置失败")
		return
	}
	if err := picbed.TestConfig(c.Request.Context(), item.PicBedType, configMap); err != nil {
		respondError(c, http.StatusBadRequest, "配置测试失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "配置测试通过"})
}

// deleteConfig godoc
// @Summary 删除图床配置
// @Tags picbed
// @Produce json
// @Security BearerAuth
// @Param id path int true "配置 ID"
// @Success 200 {object} messageResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs/{id} [delete]
func (a *API) deleteConfig(c *gin.Context) {
	item, ok := a.findConfig(c)
	if !ok {
		return
	}
	if err := a.db.Delete(&item).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "删除图床配置失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "图床配置已删除"})
}

// setDefaultConfig godoc
// @Summary 设置默认图床配置
// @Tags picbed
// @Produce json
// @Security BearerAuth
// @Param id path int true "配置 ID"
// @Success 200 {object} configResponseDoc
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/picbed/configs/{id}/default [put]
func (a *API) setDefaultConfig(c *gin.Context) {
	item, ok := a.findConfig(c)
	if !ok {
		return
	}
	userID := middleware.UserID(c)
	item.IsDefault = true
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := a.clearDefault(tx, userID, item.ID); err != nil {
			return err
		}
		return tx.Save(&item).Error
	}); err != nil {
		respondError(c, http.StatusInternalServerError, "设置默认配置失败")
		return
	}
	configMap, _ := a.decryptConfigMap(item.EncryptedConfig)
	c.JSON(http.StatusOK, gin.H{"config": configResponse(item, true, editableConfig(item.PicBedType, configMap))})
}

func (a *API) findConfig(c *gin.Context) (model.PicBedConfig, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "配置 ID 不正确")
		return model.PicBedConfig{}, false
	}
	item, ok := a.findConfigByID(c, uint(id))
	if !ok {
		respondError(c, http.StatusNotFound, "图床配置不存在")
	}
	return item, ok
}
func (a *API) findConfigByID(c *gin.Context, id uint) (model.PicBedConfig, bool) {
	return a.findConfigByIDForUser(middleware.UserID(c), id)
}

func (a *API) findConfigByIDForUser(userID uint, id uint) (model.PicBedConfig, bool) {
	var item model.PicBedConfig
	if err := a.db.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		return model.PicBedConfig{}, false
	}
	return item, true
}
func (a *API) configNameExists(userID uint, configName string, excludeID uint) (bool, error) {
	query := a.db.Model(&model.PicBedConfig{}).Where("user_id = ? AND config_name = ?", userID, configName)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
func (a *API) clearDefault(tx *gorm.DB, userID uint, excludeID uint) error {
	query := tx.Model(&model.PicBedConfig{}).Where("user_id = ? AND is_default = ?", userID, true)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	return query.Update("is_default", false).Error
}
func (a *API) normalizeDefaultConfig(userID uint) error {
	var defaults []model.PicBedConfig
	if err := a.db.Where("user_id = ? AND is_default = ?", userID, true).Order("updated_at desc, id desc").Find(&defaults).Error; err != nil {
		return err
	}
	if len(defaults) <= 1 {
		return nil
	}
	ids := make([]uint, 0, len(defaults)-1)
	for _, item := range defaults[1:] {
		ids = append(ids, item.ID)
	}
	return a.db.Model(&model.PicBedConfig{}).Where("id IN ? AND user_id = ?", ids, userID).Update("is_default", false).Error
}
func (a *API) encryptConfig(config map[string]string) (string, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return utils.EncryptString(string(raw))
}
func (a *API) decryptConfigMap(encrypted string) (map[string]string, error) {
	raw, err := utils.DecryptString(encrypted)
	if err != nil {
		return nil, err
	}
	var config map[string]string
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return nil, err
	}
	return config, nil
}

func validatePicbedConfig(req picbedConfigRequest) error {
	def, ok := findPicbedType(req.PicBedType)
	if !ok {
		return errors.New("不支持的图床类型")
	}
	if strings.TrimSpace(req.ConfigName) == "" {
		return errors.New("请填写配置名称")
	}
	if req.Config == nil {
		return errors.New("请填写图床配置")
	}
	for _, field := range def.Fields {
		if field.Required && strings.TrimSpace(req.Config[field.Key]) == "" {
			return fmt.Errorf("请填写%s", displayFieldLabel(field))
		}
	}
	return nil
}
func findPicbedType(value string) (picbedTypeDef, bool) {
	for _, item := range picbedTypeDefs {
		if item.Value == value {
			return item, true
		}
	}
	return picbedTypeDef{}, false
}
func displayFieldLabel(field configField) string {
	labels := map[string]string{
		"repository":        "仓库名",
		"branch":            "分支名",
		"token":             "Token",
		"storage_path":      "存储路径",
		"custom_domain":     "自定义域名",
		"secret_id":         "SecretId",
		"secret_key":        "SecretKey",
		"bucket":            "存储桶",
		"region":            "地域",
		"access_key_id":     "AccessKeyId",
		"access_key_secret": "AccessKeySecret",
		"secret_access_key": "SecretAccessKey",
		"endpoint":          "Endpoint",
		"access_key":        "AccessKey",
		"operator":          "操作员",
		"password":          "密码",
		"use_ssl":           "是否使用 SSL",
		"api_url":           "API 地址",
		"base_url":          "公开访问根地址",
		"auth_token":        "认证 Token",
	}
	if label, ok := labels[field.Key]; ok {
		return label
	}
	return field.Label
}
func normalizeConfig(picbedType string, input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = strings.TrimSpace(value)
	}
	if picbedType == "aliyun" {
		if output["region"] == "" {
			output["region"] = output["endpoint"]
		}
		delete(output, "endpoint")
	}
	return output
}
func buildPublicBaseURL(picbedType string, cfg map[string]string) string {
	customDomain := strings.TrimRight(cfg["custom_domain"], "/")
	storagePath := cleanURLPath(cfg["storage_path"])
	if customDomain != "" {
		return joinURLPath(customDomain, storagePath)
	}
	switch picbedType {
	case "github":
		return joinURLPath("https://raw.githubusercontent.com", cfg["repository"], cfg["branch"], storagePath)
	case "gitee":
		return joinURLPath("https://gitee.com", cfg["repository"], "raw", cfg["branch"], storagePath)
	case "easyimage":
		return joinURLPath(strings.TrimRight(cfg["api_url"], "/"), storagePath)
	case "other":
		return joinURLPath(cfg["base_url"], storagePath)
	default:
		return joinURLPath(customDomain, storagePath)
	}
}

func cleanURLPath(value string) string {
	return strings.Trim(path.Clean("/"+strings.TrimSpace(value)), "/")
}
func joinURLPath(base string, parts ...string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = cleanURLPath(part)
		if part != "" && part != "." {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return base
	}
	return base + "/" + strings.Join(cleanParts, "/")
}
func editableConfig(picbedType string, config map[string]string) map[string]string {
	if config == nil {
		return map[string]string{"masked": "true"}
	}
	def, ok := findPicbedType(picbedType)
	if !ok {
		return map[string]string{"masked": "true"}
	}
	output := map[string]string{"masked": "true"}
	if picbedType == "aliyun" && strings.TrimSpace(config["region"]) == "" {
		config["region"] = strings.TrimSpace(config["endpoint"])
	}
	for _, field := range def.Fields {
		if value := strings.TrimSpace(config[field.Key]); value != "" {
			output[field.Key] = value
		}
	}
	return output
}

func configResponse(item model.PicBedConfig, masked bool, config map[string]string) gin.H {
	response := gin.H{"id": item.ID, "picbed_type": item.PicBedType, "config_name": item.ConfigName, "is_default": item.IsDefault, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
	if masked {
		if config != nil {
			response["config"] = config
		} else {
			response["config"] = gin.H{"masked": true}
		}
	} else {
		response["config"] = config
	}
	return response
}
