package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jerion/picbed-switcher/internal/middleware"
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/picbed"
	"github.com/jerion/picbed-switcher/internal/utils"
)

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
