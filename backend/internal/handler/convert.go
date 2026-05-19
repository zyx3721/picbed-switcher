package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/picbed"
	"github.com/jerion/picbed-switcher/internal/utils"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const maxLocalBatchUploadSize = 256 << 20
const localConvertTaskType = "local_upload"

// analyzeMarkdown godoc
// @Summary 分析 Markdown 文档中的图片地址
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body markdownRequest true "Markdown 内容"
// @Success 200 {object} analyzeMarkdownResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /api/convert/analyze [post]
func (a *API) analyzeMarkdown(c *gin.Context) {
	var req markdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	images := utils.ExtractMarkdownImages(req.Content)
	counts := map[string]int{}
	for _, image := range images {
		counts[image.PicBed]++
	}
	c.JSON(http.StatusOK, gin.H{"images": images, "counts": counts, "total": len(images)})
}

// convertMarkdown godoc
// @Summary 执行单个 Markdown 文档转换
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body markdownRequest true "转换请求"
// @Success 200 {object} convertMarkdownResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/process [post]
func (a *API) convertMarkdown(c *gin.Context) {
	var req markdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	result, status, err := a.convertOne(c, req)
	if err != nil {
		respondError(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// convertMarkdownBatch godoc
// @Summary 批量执行 Markdown 文档转换
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body batchMarkdownRequest true "批量转换请求"
// @Success 200 {object} batchConvertResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /api/convert/batch [post]
func (a *API) convertMarkdownBatch(c *gin.Context) {
	var req batchMarkdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if len(req.Files) == 0 {
		respondError(c, http.StatusBadRequest, "请至少上传一个 Markdown 文件")
		return
	}
	if len(req.Files) > 20 {
		respondError(c, http.StatusBadRequest, "单次最多转换 20 个 Markdown 文件")
		return
	}
	results := make([]gin.H, 0, len(req.Files))
	for _, file := range req.Files {
		if file.TargetConfigID == 0 {
			file.TargetConfigID = req.TargetConfigID
		}
		result, _, err := a.convertOne(c, file)
		if err != nil {
			results = append(results, gin.H{"filename": file.Filename, "status": "failed", "error": err.Error()})
			continue
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// convertLocalMarkdownBatch godoc
// @Summary 批量上传 Markdown 中引用的本地图片并替换地址
// @Tags convert
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param manifest formData string true "本地批量上传清单 JSON，结构为 localBatchManifest"
// @Param images formData file false "本地图片文件，可按 manifest.images[].file_key 指定字段名上传；同一字段可提交多个文件"
// @Success 200 {object} batchConvertResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/local-batch [post]
func (a *API) convertLocalMarkdownBatch(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLocalBatchUploadSize)
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		respondError(c, http.StatusBadRequest, "请上传 Markdown、本地图片和路径映射")
		return
	}
	var req localBatchManifest
	if err := json.Unmarshal([]byte(c.PostForm("manifest")), &req); err != nil {
		respondError(c, http.StatusBadRequest, "本地上传清单格式不正确")
		return
	}
	if err := validateLocalBatchManifest(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	target, ok := a.findConfigByID(c, req.TargetConfigID)
	if !ok {
		respondError(c, http.StatusNotFound, "目标图床配置不存在")
		return
	}
	targetConfig, err := a.decryptConfigMap(target.EncryptedConfig)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "读取目标图床配置失败")
		return
	}
	files := map[string][]*multipart.FileHeader{}
	if c.Request.MultipartForm != nil {
		files = c.Request.MultipartForm.File
	}
	uploadCache := map[string]string{}
	results := make([]gin.H, 0, len(req.Documents))
	for _, document := range req.Documents {
		result, err := a.convertLocalOne(c, document, target.PicBedType, targetConfig, files, uploadCache)
		if err != nil {
			results = append(results, gin.H{"filename": documentFilename(document.Filename), "status": "failed", "error": err.Error()})
			continue
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (a *API) convertLocalOne(c *gin.Context, document localMarkdownDocument, targetPicBed string, targetConfig map[string]string, files map[string][]*multipart.FileHeader, uploadCache map[string]string) (gin.H, error) {
	return a.convertLocalOneForUser(c.Request.Context(), middleware.UserID(c), document, targetPicBed, targetConfig, func(fileKey string) (picbed.ImageFile, error) {
		return readMultipartImage(files, fileKey)
	}, uploadCache)
}

func (a *API) convertLocalOneForUser(ctx context.Context, userID uint, document localMarkdownDocument, targetPicBed string, targetConfig map[string]string, loadImage func(string) (picbed.ImageFile, error), uploadCache map[string]string, taskID ...uint) (gin.H, error) {
	if strings.TrimSpace(document.Content) == "" {
		return nil, errors.New("Markdown 内容不能为空")
	}
	images := utils.ExtractMarkdownImages(document.Content)
	if len(images) == 0 {
		return nil, errors.New("未识别到图片地址")
	}
	mappingBySource := map[string]string{}
	mappingByNormalizedSource := map[string]string{}
	for _, image := range document.Images {
		source := strings.TrimSpace(image.Source)
		fileKey := strings.TrimSpace(image.FileKey)
		if source == "" || fileKey == "" || isHTTPImageURL(source) {
			continue
		}
		mappingBySource[source] = fileKey
		mappingByNormalizedSource[normalizeLocalImageSource(source)] = fileKey
	}
	details := make([]model.ConversionRecordDetail, 0, len(images))
	output, changed, err := utils.ReplaceImageURLs(document.Content, func(currentURL string) (string, error) {
		if isHTTPImageURL(currentURL) {
			return currentURL, nil
		}
		fileKey, ok := mappingBySource[currentURL]
		if !ok {
			fileKey, ok = mappingByNormalizedSource[normalizeLocalImageSource(currentURL)]
		}
		if !ok {
			return "", fmt.Errorf("本地图片 %s 未匹配到上传文件", currentURL)
		}
		if uploadedURL, ok := uploadCache[fileKey]; ok {
			details = append(details, model.ConversionRecordDetail{OriginalURL: currentURL, TargetURL: uploadedURL, Status: "success"})
			return uploadedURL, nil
		}
		imageFile, err := loadImage(fileKey)
		if err != nil {
			return "", err
		}
		result, err := picbed.Upload(ctx, targetPicBed, targetConfig, imageFile)
		if err != nil {
			return "", fmt.Errorf("上传图片 %s 失败：%w", currentURL, err)
		}
		uploadCache[fileKey] = result.URL
		details = append(details, model.ConversionRecordDetail{OriginalURL: currentURL, TargetURL: result.URL, Status: "success"})
		return result.URL, nil
	})
	if err != nil {
		return nil, err
	}
	status := "success"
	message := ""
	if changed == 0 {
		status = "failed"
		message = "没有本地图片地址被转换"
	}
	filename := documentFilename(document.Filename)
	record := model.ConversionRecord{UserID: userID, OriginalFilename: filename, SourcePicBed: "local", TargetPicBed: targetPicBed, Status: status, ErrorMessage: message, ImageCount: changed, ConvertedContent: output}
	if len(taskID) > 0 && taskID[0] > 0 {
		record.TaskID = &taskID[0]
	}
	if err := a.db.Create(&record).Error; err != nil {
		return nil, errors.New("保存转换记录失败")
	}
	a.saveRecordDetails(record.ID, details)
	if status == "failed" {
		return nil, errors.New(message)
	}
	return gin.H{"filename": filename, "content": output, "changed": changed, "status": status, "record": record}, nil
}

func (a *API) convertOneForUser(ctx context.Context, userID uint, req markdownRequest, taskID ...uint) (gin.H, int, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, http.StatusBadRequest, errors.New("Markdown 内容不能为空")
	}
	target, ok := a.findConfigByIDForUser(userID, req.TargetConfigID)
	if !ok {
		return nil, http.StatusNotFound, errors.New("目标图床配置不存在")
	}
	targetConfig, err := a.decryptConfigMap(target.EncryptedConfig)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("读取目标图床配置失败")
	}
	images := utils.ExtractMarkdownImages(req.Content)
	if len(images) == 0 {
		return nil, http.StatusBadRequest, errors.New("未识别到图片地址")
	}
	sourcePicBed := summarizeSourcePicBeds(images)
	uploadCache := map[string]string{}
	details := make([]model.ConversionRecordDetail, 0, len(images))
	skippedUnsupportedImageCount := 0
	output, changed, err := utils.ReplaceImageURLs(req.Content, func(currentURL string) (string, error) {
		imageURL := strings.TrimSpace(currentURL)
		if !isHTTPImageURL(imageURL) {
			skippedUnsupportedImageCount++
			details = append(details, model.ConversionRecordDetail{OriginalURL: currentURL, Status: "failed", Error: "本地路径无法通过批量转换上传"})
			return currentURL, nil
		}
		if uploadedURL, ok := uploadCache[imageURL]; ok {
			details = append(details, model.ConversionRecordDetail{OriginalURL: currentURL, TargetURL: uploadedURL, Status: "success"})
			return uploadedURL, nil
		}
		image, err := picbed.DownloadImage(ctx, imageURL)
		if err != nil {
			return "", fmt.Errorf("下载图片 %s 失败：%w", imageURL, err)
		}
		result, err := picbed.Upload(ctx, target.PicBedType, targetConfig, image)
		if err != nil {
			return "", fmt.Errorf("上传图片 %s 失败：%w", imageURL, err)
		}
		uploadCache[imageURL] = result.URL
		details = append(details, model.ConversionRecordDetail{OriginalURL: currentURL, TargetURL: result.URL, Status: "success"})
		return result.URL, nil
	})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	status := "success"
	message := ""
	if skippedUnsupportedImageCount > 0 {
		message = fmt.Sprintf("存在 %d 个图片无法转换", skippedUnsupportedImageCount)
	}
	if changed == 0 {
		status = "failed"
		if message == "" {
			message = "没有图片地址被转换"
		}
	}
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = "untitled.md"
	}
	record := model.ConversionRecord{UserID: userID, OriginalFilename: filename, SourcePicBed: sourcePicBed, TargetPicBed: target.PicBedType, Status: status, ErrorMessage: message, ImageCount: changed, ConvertedContent: output}
	if len(taskID) > 0 && taskID[0] > 0 {
		record.TaskID = &taskID[0]
	}
	if err := a.db.Create(&record).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("保存转换记录失败")
	}
	a.saveRecordDetails(record.ID, details)
	if status == "failed" {
		return nil, http.StatusBadRequest, errors.New(message)
	}
	return gin.H{"filename": filename, "content": output, "changed": changed, "status": status, "record": record}, http.StatusOK, nil
}
func (a *API) convertOne(c *gin.Context, req markdownRequest) (gin.H, int, error) {
	return a.convertOneForUser(c.Request.Context(), middleware.UserID(c), req)
}

// listRecords godoc
// @Summary 获取转换历史
// @Tags convert
// @Produce json
// @Security BearerAuth
// @Success 200 {object} recordsResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/records [get]
func (a *API) listRecords(c *gin.Context) {
	var records []model.ConversionRecord
	if err := a.db.Where("user_id = ?", middleware.UserID(c)).Order("created_at desc").Limit(50).Find(&records).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "读取转换记录失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

// getRecord godoc
// @Summary 获取转换历史详情
// @Tags convert
// @Produce json
// @Security BearerAuth
// @Param id path int true "记录 ID"
// @Success 200 {object} recordResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/convert/records/{id} [get]
func (a *API) getRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "记录 ID 不正确")
		return
	}
	var record model.ConversionRecord
	if err := a.db.Preload("Details").Where("id = ? AND user_id = ?", id, middleware.UserID(c)).First(&record).Error; err != nil {
		respondError(c, http.StatusNotFound, "转换记录不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": record})
}

// deleteRecords godoc
// @Summary 批量删除转换历史
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body deleteRecordsRequest true "记录 ID 列表"
// @Success 200 {object} messageResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/records [delete]
func (a *API) deleteRecords(c *gin.Context) {
	var req deleteRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	ids := uniquePositiveIDs(req.IDs)
	if len(ids) == 0 {
		respondError(c, http.StatusBadRequest, "请选择要删除的历史记录")
		return
	}
	if len(ids) > 50 {
		respondError(c, http.StatusBadRequest, "单次最多删除 50 条历史记录")
		return
	}

	userID := middleware.UserID(c)
	var ownedIDs []uint
	if err := a.db.Model(&model.ConversionRecord{}).Where("user_id = ? AND id IN ?", userID, ids).Pluck("id", &ownedIDs).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "查询转换记录失败")
		return
	}
	if len(ownedIDs) == 0 {
		respondError(c, http.StatusBadRequest, "请选择要删除的历史记录")
		return
	}
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("record_id IN ?", ownedIDs).Delete(&model.ConversionRecordDetail{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND id IN ?", userID, ownedIDs).Delete(&model.ConversionRecord{}).Error
	}); err != nil {
		respondError(c, http.StatusInternalServerError, "删除转换记录失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("已删除 %d 条转换记录", len(ownedIDs))})
}

func uniquePositiveIDs(input []uint) []uint {
	seen := map[uint]bool{}
	ids := make([]uint, 0, len(input))
	for _, id := range input {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// createConvertTask godoc
// @Summary 创建转换任务
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createConvertTaskRequest true "转换任务请求"
// @Success 202 {object} taskCreateResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/tasks [post]
func (a *API) createConvertTask(c *gin.Context) {
	var req createConvertTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if len(req.Files) == 0 {
		respondError(c, http.StatusBadRequest, "请至少上传一个 Markdown 文件")
		return
	}
	if len(req.Files) > 20 {
		respondError(c, http.StatusBadRequest, "单次最多转换 20 个 Markdown 文件")
		return
	}
	for index := range req.Files {
		if req.Files[index].TargetConfigID == 0 {
			req.Files[index].TargetConfigID = req.TargetConfigID
		}
		if req.Files[index].TargetConfigID == 0 {
			respondError(c, http.StatusBadRequest, "请先选择目标图床配置")
			return
		}
	}
	payload, err := json.Marshal(req.Files)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "转换任务载荷生成失败")
		return
	}
	task := model.ConversionTask{
		UserID:   middleware.UserID(c),
		TaskType: "convert",
		Status:   "queued",
		Total:    len(req.Files),
		Message:  "转换任务已加入队列",
		Payload:  string(payload),
	}
	if err := a.db.Create(&task).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "创建转换任务失败")
		return
	}
	if err := a.enqueueConvertTask(c.Request.Context(), task.ID); err != nil {
		ended := time.Now()
		_ = a.db.Model(&task).Updates(map[string]any{"status": "failed", "message": "转换任务入队失败", "error": err.Error(), "ended_at": ended}).Error
		respondError(c, http.StatusInternalServerError, "转换任务入队失败")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": task, "results": []gin.H{}})
}

// createLocalConvertTask godoc
// @Summary 创建本地图片上传替换任务
// @Tags convert
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param manifest formData string true "本地批量上传清单 JSON，结构为 localBatchManifest"
// @Param images formData file false "本地图片文件，可按 manifest.images[].file_key 指定字段名上传"
// @Success 202 {object} taskCreateResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/local-tasks [post]
func (a *API) createLocalConvertTask(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLocalBatchUploadSize)
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		respondError(c, http.StatusBadRequest, "请上传 Markdown、本地图片和路径映射")
		return
	}
	var req localBatchManifest
	if err := json.Unmarshal([]byte(c.PostForm("manifest")), &req); err != nil {
		respondError(c, http.StatusBadRequest, "本地上传清单格式不正确")
		return
	}
	if err := validateLocalBatchManifest(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := a.findConfigByID(c, req.TargetConfigID); !ok {
		respondError(c, http.StatusNotFound, "目标图床配置不存在")
		return
	}

	task := model.ConversionTask{
		UserID:   middleware.UserID(c),
		TaskType: localConvertTaskType,
		Status:   "queued",
		Total:    len(req.Documents),
		Message:  "本地上传任务已加入队列",
	}
	if err := a.db.Create(&task).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "创建本地上传任务失败")
		return
	}

	files := map[string][]*multipart.FileHeader{}
	if c.Request.MultipartForm != nil {
		files = c.Request.MultipartForm.File
	}
	storedFiles, err := storeLocalTaskFiles(task.ID, files)
	if err != nil {
		cleanupLocalTaskFiles(task.ID)
		a.failConvertTask(task.ID, "保存本地上传文件失败", err)
		respondError(c, http.StatusInternalServerError, "保存本地上传文件失败")
		return
	}
	payload, err := json.Marshal(localConvertTaskPayload{Manifest: req, Files: storedFiles})
	if err != nil {
		cleanupLocalTaskFiles(task.ID)
		a.failConvertTask(task.ID, "本地上传任务载荷生成失败", err)
		respondError(c, http.StatusInternalServerError, "本地上传任务载荷生成失败")
		return
	}
	if err := a.db.Model(&task).Update("payload", string(payload)).Error; err != nil {
		cleanupLocalTaskFiles(task.ID)
		a.failConvertTask(task.ID, "保存本地上传任务载荷失败", err)
		respondError(c, http.StatusInternalServerError, "保存本地上传任务载荷失败")
		return
	}
	if err := a.enqueueConvertTask(c.Request.Context(), task.ID); err != nil {
		cleanupLocalTaskFiles(task.ID)
		ended := time.Now()
		_ = a.db.Model(&task).Updates(map[string]any{"status": "failed", "message": "本地上传任务入队失败", "error": err.Error(), "ended_at": ended}).Error
		respondError(c, http.StatusInternalServerError, "本地上传任务入队失败")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": task, "results": []gin.H{}})
}

func (a *API) enqueueConvertTask(ctx context.Context, taskID uint) error {
	if a.redis != nil && a.cfg.Redis.Enabled {
		return a.redis.RPush(ctx, a.cfg.Redis.ConvertQueue, taskID).Err()
	}
	select {
	case a.convertQueue <- taskID:
		return nil
	default:
		return fmt.Errorf("本地转换队列已满，请稍后重试")
	}
}

func validateLocalBatchManifest(req localBatchManifest) error {
	if req.TargetConfigID == 0 {
		return errors.New("请先选择目标图床配置")
	}
	if len(req.Documents) == 0 {
		return errors.New("请至少上传一个 Markdown 文档")
	}
	if len(req.Documents) > 20 {
		return errors.New("单次最多处理 20 个 Markdown 文档")
	}
	return nil
}

func localTaskDir(taskID uint) string {
	return filepath.Join(os.TempDir(), "picbed-switcher", "local-tasks", strconv.FormatUint(uint64(taskID), 10))
}

func storeLocalTaskFiles(taskID uint, files map[string][]*multipart.FileHeader) (map[string][]localTaskStoredFile, error) {
	baseDir := localTaskDir(taskID)
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, err
	}
	stored := map[string][]localTaskStoredFile{}
	for fileKey, headers := range files {
		cleanKey := strings.TrimSpace(fileKey)
		if cleanKey == "" {
			continue
		}
		for index, header := range headers {
			source, err := header.Open()
			if err != nil {
				return nil, err
			}
			filename := fmt.Sprintf("%s-%d-%s", safeLocalTaskName(cleanKey), index, safeLocalTaskName(header.Filename))
			path := filepath.Join(baseDir, filename)
			target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				_ = source.Close()
				return nil, err
			}
			_, copyErr := io.Copy(target, source)
			closeErr := target.Close()
			_ = source.Close()
			if copyErr != nil {
				return nil, copyErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			stored[cleanKey] = append(stored[cleanKey], localTaskStoredFile{Filename: header.Filename, Path: path})
		}
	}
	return stored, nil
}

func safeLocalTaskName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	value = filepath.Base(value)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, ".-")
	if value == "" {
		return "file"
	}
	return value
}

func cleanupLocalTaskFiles(taskID uint) {
	_ = os.RemoveAll(localTaskDir(taskID))
}

func (a *API) startConvertWorkers() {
	if a.workerStarted {
		return
	}
	a.workerStarted = true
	ctx, cancel := context.WithCancel(context.Background())
	a.workerCancel = cancel
	concurrency := a.cfg.Redis.WorkerConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if a.redis == nil || !a.cfg.Redis.Enabled {
		concurrency = 1
	}
	for index := 0; index < concurrency; index++ {
		go a.runConvertWorker(ctx, index+1)
	}
	go a.requeueQueuedConvertTasks(ctx)
}

func (a *API) requeueQueuedConvertTasks(ctx context.Context) {
	var tasks []model.ConversionTask
	if err := a.db.Where("status = ?", "queued").Order("created_at asc").Find(&tasks).Error; err != nil {
		log.Printf("failed to requeue conversion tasks: %v", err)
		return
	}
	for _, task := range tasks {
		if err := a.enqueueConvertTask(ctx, task.ID); err != nil {
			log.Printf("failed to requeue conversion task %d: %v", task.ID, err)
		}
	}
}

func (a *API) runConvertWorker(ctx context.Context, workerID int) {
	for {
		taskID, err := a.nextConvertTask(ctx)
		if err != nil {
			if err == context.Canceled || err == redis.Nil {
				return
			}
			log.Printf("conversion worker %d failed to read task: %v", workerID, err)
			continue
		}
		a.runConvertTask(ctx, taskID)
	}
}

func (a *API) nextConvertTask(ctx context.Context) (uint, error) {
	if a.redis != nil && a.cfg.Redis.Enabled {
		result, err := a.redis.BLPop(ctx, 0, a.cfg.Redis.ConvertQueue).Result()
		if err != nil {
			return 0, err
		}
		if len(result) < 2 {
			return 0, fmt.Errorf("Redis 队列返回值不正确")
		}
		id, err := strconv.ParseUint(result[1], 10, 64)
		if err != nil || id == 0 {
			return 0, fmt.Errorf("Redis 队列任务 ID 不正确：%s", result[1])
		}
		return uint(id), nil
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case taskID := <-a.convertQueue:
		return taskID, nil
	}
}

func (a *API) runConvertTask(ctx context.Context, taskID uint) {
	var task model.ConversionTask
	if err := a.db.First(&task, taskID).Error; err != nil {
		log.Printf("conversion task %d not found: %v", taskID, err)
		return
	}
	if task.Status != "queued" {
		return
	}
	started := time.Now()
	claimed := a.db.Model(&model.ConversionTask{}).Where("id = ? AND status = ?", task.ID, "queued").Updates(map[string]any{
		"status":     "running",
		"message":    "转换任务执行中",
		"started_at": started,
	})
	if claimed.Error != nil {
		log.Printf("failed to claim conversion task %d: %v", task.ID, claimed.Error)
		return
	}
	if claimed.RowsAffected == 0 {
		return
	}
	task.Status = "running"
	if task.TaskType == localConvertTaskType {
		a.runLocalConvertTask(ctx, task)
		return
	}
	if task.TaskType != "convert" {
		a.failConvertTask(task.ID, "不支持的转换任务类型", fmt.Errorf("task_type: %s", task.TaskType))
		return
	}
	var files []markdownRequest
	if err := json.Unmarshal([]byte(task.Payload), &files); err != nil {
		a.failConvertTask(task.ID, "转换任务载荷解析失败", err)
		return
	}

	success := 0
	failed := 0
	for index, file := range files {
		_ = a.db.Model(&task).Updates(map[string]any{
			"message": fmt.Sprintf("正在转换第 %d / %d 个文档：%s", index+1, len(files), documentFilename(file.Filename)),
			"success": success,
			"failed":  failed,
		}).Error

		_, _, err := a.convertOneForUser(ctx, task.UserID, file, task.ID)
		if err != nil {
			failed++
		} else {
			success++
		}
		_ = a.db.Model(&task).Updates(map[string]any{"success": success, "failed": failed}).Error
	}

	ended := time.Now()
	status := "success"
	if failed > 0 {
		status = "failed"
	}
	_ = a.db.Model(&task).Updates(map[string]any{
		"status":   status,
		"success":  success,
		"failed":   failed,
		"message":  fmt.Sprintf("转换完成，成功 %d 个，失败 %d 个", success, failed),
		"ended_at": ended,
	}).Error
}

func (a *API) runLocalConvertTask(ctx context.Context, task model.ConversionTask) {
	defer cleanupLocalTaskFiles(task.ID)
	var payload localConvertTaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		a.failConvertTask(task.ID, "本地上传任务载荷解析失败", err)
		return
	}
	if err := validateLocalBatchManifest(payload.Manifest); err != nil {
		a.failConvertTask(task.ID, "本地上传任务载荷不正确", err)
		return
	}
	target, ok := a.findConfigByIDForUser(task.UserID, payload.Manifest.TargetConfigID)
	if !ok {
		a.failConvertTask(task.ID, "目标图床配置不存在", nil)
		return
	}
	targetConfig, err := a.decryptConfigMap(target.EncryptedConfig)
	if err != nil {
		a.failConvertTask(task.ID, "读取目标图床配置失败", err)
		return
	}

	success := 0
	failed := 0
	uploadCache := map[string]string{}
	loadImage := func(fileKey string) (picbed.ImageFile, error) {
		storedFiles := payload.Files[fileKey]
		if len(storedFiles) == 0 {
			return picbed.ImageFile{}, fmt.Errorf("本地图片文件 %s 不存在", fileKey)
		}
		return readStoredLocalTaskImage(storedFiles[0])
	}

	for index, document := range payload.Manifest.Documents {
		_ = a.db.Model(&task).Updates(map[string]any{
			"message": fmt.Sprintf("正在上传替换第 %d / %d 个文档：%s", index+1, len(payload.Manifest.Documents), documentFilename(document.Filename)),
			"success": success,
			"failed":  failed,
		}).Error

		_, err := a.convertLocalOneForUser(ctx, task.UserID, document, target.PicBedType, targetConfig, loadImage, uploadCache, task.ID)
		if err != nil {
			failed++
		} else {
			success++
		}
		_ = a.db.Model(&task).Updates(map[string]any{"success": success, "failed": failed}).Error
	}

	ended := time.Now()
	status := "success"
	if failed > 0 {
		status = "failed"
	}
	_ = a.db.Model(&task).Updates(map[string]any{
		"status":   status,
		"success":  success,
		"failed":   failed,
		"message":  fmt.Sprintf("本地图片上传完成，成功 %d 个，失败 %d 个", success, failed),
		"ended_at": ended,
	}).Error
}

func readStoredLocalTaskImage(file localTaskStoredFile) (picbed.ImageFile, error) {
	path := strings.TrimSpace(file.Path)
	if path == "" {
		return picbed.ImageFile{}, errors.New("本地图片临时文件路径为空")
	}
	opened, err := os.Open(path)
	if err != nil {
		return picbed.ImageFile{}, err
	}
	defer opened.Close()
	return picbed.NewImageFile(file.Filename, opened)
}

func (a *API) failConvertTask(taskID uint, message string, err error) {
	ended := time.Now()
	updates := map[string]any{"status": "failed", "message": message, "ended_at": ended}
	if err != nil {
		updates["error"] = err.Error()
	}
	_ = a.db.Model(&model.ConversionTask{}).Where("id = ?", taskID).Updates(updates).Error
}

// listConvertTasks godoc
// @Summary 获取转换任务列表
// @Tags convert
// @Produce json
// @Security BearerAuth
// @Success 200 {object} tasksResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/convert/tasks [get]
func (a *API) listConvertTasks(c *gin.Context) {
	var tasks []model.ConversionTask
	if err := a.db.Where("user_id = ?", middleware.UserID(c)).Order("created_at desc").Limit(50).Find(&tasks).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "读取转换任务失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// getConvertTask godoc
// @Summary 获取转换任务详情
// @Tags convert
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务 ID"
// @Success 200 {object} taskDetailResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/convert/tasks/{id} [get]
func (a *API) getConvertTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "任务 ID 不正确")
		return
	}
	var task model.ConversionTask
	if err := a.db.Where("id = ? AND user_id = ?", id, middleware.UserID(c)).First(&task).Error; err != nil {
		respondError(c, http.StatusNotFound, "转换任务不存在")
		return
	}
	var records []model.ConversionRecord
	_ = a.db.Preload("Details").Where("task_id = ? AND user_id = ?", task.ID, middleware.UserID(c)).Order("id asc").Find(&records).Error
	c.JSON(http.StatusOK, gin.H{"task": task, "records": records})
}

func (a *API) saveRecordDetails(recordID uint, details []model.ConversionRecordDetail) {
	if len(details) == 0 {
		return
	}
	for index := range details {
		details[index].RecordID = recordID
	}
	_ = a.db.Create(&details).Error
}

func summarizeSourcePicBeds(images []utils.MarkdownImage) string {
	if len(images) == 0 {
		return "unknown"
	}
	seen := map[string]bool{}
	labels := make([]string, 0)
	for _, image := range images {
		picbedType := strings.TrimSpace(image.PicBed)
		if picbedType == "" {
			picbedType = "unknown"
		}
		if seen[picbedType] {
			continue
		}
		seen[picbedType] = true
		labels = append(labels, picbedType)
	}
	if len(labels) == 1 {
		return labels[0]
	}
	return "mixed"
}

func readMultipartImage(files map[string][]*multipart.FileHeader, fileKey string) (picbed.ImageFile, error) {
	fileHeaders := files[fileKey]
	if len(fileHeaders) == 0 {
		return picbed.ImageFile{}, fmt.Errorf("本地图片文件 %s 不存在", fileKey)
	}
	opened, err := fileHeaders[0].Open()
	if err != nil {
		return picbed.ImageFile{}, err
	}
	defer opened.Close()
	return picbed.NewImageFile(fileHeaders[0].Filename, opened)
}

func documentFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "untitled.md"
	}
	return filename
}

func isHTTPImageURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func normalizeLocalImageSource(raw string) string {
	value := strings.TrimSpace(raw)
	if decoded, err := url.QueryUnescape(value); err == nil {
		value = decoded
	}
	value = strings.TrimPrefix(value, "file:///")
	value = strings.TrimPrefix(value, "file://")
	value = strings.ReplaceAll(value, `\`, "/")
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	return strings.ToLower(strings.Trim(value, "/"))
}
