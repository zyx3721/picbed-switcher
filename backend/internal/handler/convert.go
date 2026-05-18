package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/picbed"
	"github.com/jerion/picbed-switcher/internal/utils"
)

const maxLocalBatchUploadSize = 256 << 20

// analyzeMarkdown godoc
// @Summary 分析 Markdown 文档中的图片地址
// @Tags convert
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body markdownRequest true "Markdown 内容"
// @Success 200 {object} analyzeMarkdownResponse
// @Failure 400 {object} errorResponse
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
// @Failure 404 {object} errorResponse
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
// @Success 200 {object} batchConvertResponse
// @Failure 400 {object} errorResponse
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
	if req.TargetConfigID == 0 {
		respondError(c, http.StatusBadRequest, "请先选择目标图床配置")
		return
	}
	if len(req.Documents) == 0 {
		respondError(c, http.StatusBadRequest, "请至少上传一个 Markdown 文档")
		return
	}
	if len(req.Documents) > 20 {
		respondError(c, http.StatusBadRequest, "单次最多处理 20 个 Markdown 文档")
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
			return uploadedURL, nil
		}
		imageFile, err := readMultipartImage(files, fileKey)
		if err != nil {
			return "", err
		}
		result, err := picbed.Upload(c.Request.Context(), targetPicBed, targetConfig, imageFile)
		if err != nil {
			return "", fmt.Errorf("上传图片 %s 失败：%w", currentURL, err)
		}
		uploadCache[fileKey] = result.URL
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
	record := model.ConversionRecord{UserID: middleware.UserID(c), OriginalFilename: filename, SourcePicBed: "local", TargetPicBed: targetPicBed, Status: status, ErrorMessage: message, ImageCount: changed}
	if err := a.db.Create(&record).Error; err != nil {
		return nil, errors.New("保存转换记录失败")
	}
	if status == "failed" {
		return nil, errors.New(message)
	}
	return gin.H{"filename": filename, "content": output, "changed": changed, "status": status, "record": record}, nil
}

func (a *API) convertOne(c *gin.Context, req markdownRequest) (gin.H, int, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, http.StatusBadRequest, errors.New("Markdown 内容不能为空")
	}
	target, ok := a.findConfigByID(c, req.TargetConfigID)
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
	output, changed, err := utils.ReplaceImageURLs(req.Content, func(currentURL string) (string, error) {
		if uploadedURL, ok := uploadCache[currentURL]; ok {
			return uploadedURL, nil
		}
		image, err := picbed.DownloadImage(c.Request.Context(), currentURL)
		if err != nil {
			return "", fmt.Errorf("下载图片 %s 失败：%w", currentURL, err)
		}
		result, err := picbed.Upload(c.Request.Context(), target.PicBedType, targetConfig, image)
		if err != nil {
			return "", fmt.Errorf("上传图片 %s 失败：%w", currentURL, err)
		}
		uploadCache[currentURL] = result.URL
		return result.URL, nil
	})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	status := "success"
	message := ""
	if changed == 0 {
		status = "failed"
		message = "没有图片地址被转换"
	}
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = "untitled.md"
	}
	record := model.ConversionRecord{UserID: middleware.UserID(c), OriginalFilename: filename, SourcePicBed: sourcePicBed, TargetPicBed: target.PicBedType, Status: status, ErrorMessage: message, ImageCount: changed}
	if err := a.db.Create(&record).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("保存转换记录失败")
	}
	if status == "failed" {
		return nil, http.StatusBadRequest, errors.New(message)
	}
	return gin.H{"filename": filename, "content": output, "changed": changed, "status": status, "record": record}, http.StatusOK, nil
}

// listRecords godoc
// @Summary 获取转换历史
// @Tags convert
// @Produce json
// @Security BearerAuth
// @Success 200 {object} recordsResponse
// @Failure 401 {object} errorResponse
// @Router /api/convert/records [get]
func (a *API) listRecords(c *gin.Context) {
	var records []model.ConversionRecord
	if err := a.db.Where("user_id = ?", middleware.UserID(c)).Order("created_at desc").Limit(50).Find(&records).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "读取转换记录失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
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
