package picbed

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxImageSize = 20 << 20

const DefaultFilenameFormat = "{y}/{m}/{d}/{origin}{ext}"

var safeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var randTokenPattern = regexp.MustCompile(`\{rand:(\d+)\}`)

type ImageFile struct {
	Filename    string
	ContentType string
	Data        []byte
}

type UploadResult struct {
	URL string
}

func DownloadImage(ctx context.Context, rawURL string) (ImageFile, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ImageFile{}, errors.New("图片地址不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ImageFile{}, errors.New("图片地址仅支持 HTTP 或 HTTPS")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ImageFile{}, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ImageFile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ImageFile{}, fmt.Errorf("图片下载失败：%s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize+1))
	if err != nil {
		return ImageFile{}, err
	}
	if len(data) > maxImageSize {
		return ImageFile{}, errors.New("图片大小不能超过 20MB")
	}
	contentType := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return ImageFile{}, errors.New("该地址不是有效图片")
	}

	return ImageFile{Filename: buildFilename(parsed, contentType, data), ContentType: contentType, Data: data}, nil
}

func NewImageFile(filename string, reader io.Reader) (ImageFile, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxImageSize+1))
	if err != nil {
		return ImageFile{}, err
	}
	if len(data) > maxImageSize {
		return ImageFile{}, errors.New("图片大小不能超过 20MB")
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return ImageFile{}, errors.New("上传文件不是有效图片")
	}
	return ImageFile{Filename: buildLocalFilename(filename, contentType), ContentType: contentType, Data: data}, nil
}

func Upload(ctx context.Context, picbedType string, cfg map[string]string, image ImageFile) (UploadResult, error) {
	if picbedType != "easyimage" {
		image.Filename = formatFilename(cfg["filename_format"], image)
	}
	switch picbedType {
	case "github":
		return uploadGitHub(ctx, cfg, image)
	case "gitee":
		return uploadGitee(ctx, cfg, image)
	case "tencent":
		return uploadTencentCOS(ctx, cfg, image)
	case "aliyun":
		return uploadAliyunOSS(ctx, cfg, image)
	case "qiniu":
		return uploadQiniuKodo(ctx, cfg, image)
	case "baidu_bos":
		return uploadBaiduBOS(ctx, cfg, image)
	case "huawei_obs":
		return uploadHuaweiOBS(ctx, cfg, image)
	case "upyun":
		return uploadUpYun(ctx, cfg, image)
	case "minio":
		return uploadMinIO(ctx, cfg, image)
	case "easyimage", "other":
		return uploadEasyImage(ctx, cfg, image)
	default:
		return UploadResult{}, fmt.Errorf("暂不支持上传到%s", picbedType)
	}
}

func TestConfig(ctx context.Context, picbedType string, cfg map[string]string) error {
	image := ImageFile{
		Filename:    "picbed-switcher-test.png",
		ContentType: "image/png",
		Data:        []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82},
	}
	_, err := Upload(ctx, picbedType, cfg, image)
	return err
}

func uploadGitHub(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	repo := strings.Trim(strings.TrimSpace(cfg["repository"]), "/")
	branch := strings.TrimSpace(cfg["branch"])
	token := strings.TrimSpace(cfg["token"])
	if repo == "" || branch == "" || token == "" {
		return UploadResult{}, errors.New("GitHub 配置缺少仓库、分支或 Token")
	}
	objectPath := objectPath(cfg["storage_path"], image.Filename)
	body := map[string]string{
		"message": fmt.Sprintf("upload %s", image.Filename),
		"content": base64.StdEncoding.EncodeToString(image.Data),
		"branch":  branch,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return UploadResult{}, err
	}
	apiURL := "https://api.github.com/repos/" + repo + "/contents/" + url.PathEscape(objectPath)
	apiURL = strings.ReplaceAll(apiURL, "%2F", "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewReader(raw))
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	var response struct {
		Content struct {
			DownloadURL string `json:"download_url"`
		} `json:"content"`
	}
	if err := doJSON(req, &response); err != nil {
		return UploadResult{}, err
	}
	if response.Content.DownloadURL == "" {
		return UploadResult{}, errors.New("GitHub 上传响应缺少图片地址")
	}
	return UploadResult{URL: response.Content.DownloadURL}, nil
}

func uploadGitee(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	repo := strings.Trim(strings.TrimSpace(cfg["repository"]), "/")
	branch := strings.TrimSpace(cfg["branch"])
	token := strings.TrimSpace(cfg["token"])
	if repo == "" || branch == "" || token == "" {
		return UploadResult{}, errors.New("Gitee 配置缺少仓库、分支或 Token")
	}
	objectPath := objectPath(cfg["storage_path"], image.Filename)
	body := map[string]string{
		"access_token": token,
		"message":      fmt.Sprintf("upload %s", image.Filename),
		"content":      base64.StdEncoding.EncodeToString(image.Data),
		"branch":       branch,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return UploadResult{}, err
	}
	apiURL := "https://gitee.com/api/v5/repos/" + repo + "/contents/" + url.PathEscape(objectPath)
	apiURL = strings.ReplaceAll(apiURL, "%2F", "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(raw))
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var response struct {
		Content struct {
			DownloadURL string `json:"download_url"`
		} `json:"content"`
	}
	if err := doJSON(req, &response); err != nil {
		return UploadResult{}, err
	}
	if response.Content.DownloadURL != "" {
		return UploadResult{URL: response.Content.DownloadURL}, nil
	}
	return UploadResult{URL: joinURL("https://gitee.com", repo, "raw", branch, objectPath)}, nil
}

func uploadTencentCOS(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	secretID := strings.TrimSpace(cfg["secret_id"])
	secretKey := strings.TrimSpace(cfg["secret_key"])
	bucket := strings.TrimSpace(cfg["bucket"])
	region := strings.TrimSpace(cfg["region"])
	if secretID == "" || secretKey == "" || bucket == "" || region == "" {
		return UploadResult{}, errors.New("腾讯云 COS 配置缺少 SecretId、SecretKey、存储桶或地域")
	}
	endpoint := fmt.Sprintf("cos.%s.myqcloud.com", region)
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    secretID,
		SecretKey:    secretKey,
		Bucket:       bucket,
		Region:       region,
		Endpoint:     endpoint,
		Secure:       true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

func uploadAliyunOSS(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	accessKeyID := strings.TrimSpace(cfg["access_key_id"])
	accessKeySecret := strings.TrimSpace(cfg["access_key_secret"])
	bucket := strings.TrimSpace(cfg["bucket"])
	region := strings.TrimSpace(cfg["region"])
	if region == "" {
		region = strings.TrimSpace(cfg["endpoint"])
	}
	if accessKeyID == "" || accessKeySecret == "" || bucket == "" || region == "" {
		return UploadResult{}, errors.New("阿里云 OSS 配置缺少 AccessKeyId、AccessKeySecret、存储桶或地域")
	}
	endpoint := fmt.Sprintf("oss-%s.aliyuncs.com", region)
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    accessKeyID,
		SecretKey:    accessKeySecret,
		Bucket:       bucket,
		Region:       region,
		Endpoint:     endpoint,
		Secure:       true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

func uploadQiniuKodo(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	accessKey := strings.TrimSpace(cfg["access_key"])
	secretKey := strings.TrimSpace(cfg["secret_key"])
	bucket := strings.TrimSpace(cfg["bucket"])
	region := strings.TrimSpace(cfg["region"])
	if accessKey == "" || secretKey == "" || bucket == "" || region == "" {
		return UploadResult{}, errors.New("七牛云 Kodo 配置缺少 AccessKey、SecretKey、存储桶或地域")
	}
	endpoint := fmt.Sprintf("s3.%s.qiniucs.com", region)
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		Bucket:       bucket,
		Region:       region,
		Endpoint:     endpoint,
		Secure:       true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

func uploadBaiduBOS(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	accessKeyID := strings.TrimSpace(cfg["access_key_id"])
	secretAccessKey := strings.TrimSpace(cfg["secret_access_key"])
	bucket := strings.TrimSpace(cfg["bucket"])
	region := strings.TrimSpace(cfg["region"])
	if accessKeyID == "" || secretAccessKey == "" || bucket == "" || region == "" {
		return UploadResult{}, errors.New("百度云 BOS 配置缺少 AccessKeyId、SecretAccessKey、存储桶或地域")
	}
	endpoint := fmt.Sprintf("s3.%s.bcebos.com", region)
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    accessKeyID,
		SecretKey:    secretAccessKey,
		Bucket:       bucket,
		Region:       region,
		Endpoint:     endpoint,
		Secure:       true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

func uploadHuaweiOBS(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	accessKeyID := strings.TrimSpace(cfg["access_key_id"])
	secretAccessKey := strings.TrimSpace(cfg["secret_access_key"])
	bucket := strings.TrimSpace(cfg["bucket"])
	region := strings.TrimSpace(cfg["region"])
	if accessKeyID == "" || secretAccessKey == "" || bucket == "" || region == "" {
		return UploadResult{}, errors.New("华为云 OBS 配置缺少 AccessKeyId、SecretAccessKey、存储桶或地域")
	}
	endpoint := fmt.Sprintf("obs.%s.myhuaweicloud.com", region)
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    accessKeyID,
		SecretKey:    secretAccessKey,
		Bucket:       bucket,
		Region:       region,
		Endpoint:     endpoint,
		Secure:       true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

func uploadUpYun(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	bucket := strings.TrimSpace(cfg["bucket"])
	operator := strings.TrimSpace(cfg["operator"])
	password := strings.TrimSpace(cfg["password"])
	if bucket == "" || operator == "" || password == "" {
		return UploadResult{}, errors.New("又拍云 USS 配置缺少服务名、操作员或密码")
	}
	objectPath := objectPath(cfg["storage_path"], image.Filename)
	uploadURL := joinURL("https://v0.api.upyun.com", bucket, objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(image.Data))
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(operator+":"+password)))
	req.Header.Set("Content-Type", image.ContentType)
	req.Header.Set("Content-Length", strconv.Itoa(len(image.Data)))
	if err := doJSON(req, nil); err != nil {
		return UploadResult{}, err
	}
	if customURL := customPublicURL(cfg["custom_domain"], objectPath); customURL != "" {
		return UploadResult{URL: customURL}, nil
	}
	return UploadResult{}, errors.New("又拍云 USS 配置缺少加速域名，无法生成公开访问地址")
}

func uploadMinIO(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	accessKey := strings.TrimSpace(cfg["access_key"])
	secretKey := strings.TrimSpace(cfg["secret_key"])
	bucket := strings.TrimSpace(cfg["bucket"])
	endpoint := strings.TrimSpace(cfg["endpoint"])
	if accessKey == "" || secretKey == "" || bucket == "" || endpoint == "" {
		return UploadResult{}, errors.New("MinIO 配置缺少 Endpoint、AccessKey、SecretKey 或存储桶")
	}
	endpoint, secure := normalizeS3Endpoint(endpoint, cfg["use_ssl"])
	return uploadS3Compatible(ctx, s3CompatibleConfig{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		Bucket:       bucket,
		Region:       strings.TrimSpace(cfg["region"]),
		Endpoint:     endpoint,
		Secure:       secure,
		PathStyle:    true,
		StoragePath:  cfg["storage_path"],
		CustomDomain: cfg["custom_domain"],
	}, image)
}

type s3CompatibleConfig struct {
	AccessKey    string
	SecretKey    string
	Bucket       string
	Region       string
	Endpoint     string
	Secure       bool
	PathStyle    bool
	StoragePath  string
	CustomDomain string
}

func uploadS3Compatible(ctx context.Context, cfg s3CompatibleConfig, image ImageFile) (UploadResult, error) {
	bucketLookup := minio.BucketLookupDNS
	if cfg.PathStyle {
		bucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.Secure,
		Region:       cfg.Region,
		BucketLookup: bucketLookup,
	})
	if err != nil {
		return UploadResult{}, err
	}
	objectPath := objectPath(cfg.StoragePath, image.Filename)
	reader := bytes.NewReader(image.Data)
	_, err = client.PutObject(ctx, cfg.Bucket, objectPath, reader, int64(len(image.Data)), minio.PutObjectOptions{
		ContentType: image.ContentType,
	})
	if err != nil {
		return UploadResult{}, err
	}
	if customURL := customPublicURL(cfg.CustomDomain, objectPath); customURL != "" {
		return UploadResult{URL: customURL}, nil
	}
	scheme := "https"
	if !cfg.Secure {
		scheme = "http"
	}
	if cfg.PathStyle {
		return UploadResult{URL: fmt.Sprintf("%s://%s/%s/%s", scheme, cfg.Endpoint, cfg.Bucket, objectPath)}, nil
	}
	return UploadResult{URL: fmt.Sprintf("%s://%s.%s/%s", scheme, cfg.Bucket, cfg.Endpoint, objectPath)}, nil
}

func normalizeS3Endpoint(endpoint string, useSSL string) (string, bool) {
	secure := true
	if strings.EqualFold(strings.TrimSpace(useSSL), "false") || strings.TrimSpace(useSSL) == "0" {
		secure = false
	}
	if parsed, err := url.Parse(strings.TrimSpace(endpoint)); err == nil && parsed.Host != "" {
		secure = parsed.Scheme != "http"
		endpoint = parsed.Host
	}
	return strings.TrimRight(strings.TrimSpace(endpoint), "/"), secure
}

func uploadEasyImage(ctx context.Context, cfg map[string]string, image ImageFile) (UploadResult, error) {
	apiURL := strings.TrimSpace(cfg["api_url"])
	token := strings.TrimSpace(cfg["token"])
	if apiURL == "" || token == "" {
		return UploadResult{}, errors.New("EasyImage 配置缺少 API 地址或 Token")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("token", token)
	part, err := writer.CreateFormFile("image", image.Filename)
	if err != nil {
		return UploadResult{}, err
	}
	if _, err := part.Write(image.Data); err != nil {
		return UploadResult{}, err
	}
	if err := writer.Close(); err != nil {
		return UploadResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &body)
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	var response map[string]any
	if err := doJSON(req, &response); err != nil {
		return UploadResult{}, err
	}
	if uploadedURL := findURL(response); uploadedURL != "" {
		return UploadResult{URL: uploadedURL}, nil
	}
	return UploadResult{}, errors.New("EasyImage 上传响应缺少图片地址")
}

func doJSON(req *http.Request, output any) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("上传失败：%s", strings.TrimSpace(string(data)))
	}
	if output == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, output)
}

func buildFilename(parsed *url.URL, contentType string, data []byte) string {
	base := path.Base(parsed.Path)
	if base == "." || base == "/" || base == "" {
		base = "image"
	}
	base = safeFilenameChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, ".-")
	if base == "" {
		base = "image"
	}
	if path.Ext(base) == "" {
		if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
			base += exts[0]
		}
	}
	return base
}

func buildLocalFilename(filename string, contentType string) string {
	base := strings.ReplaceAll(strings.TrimSpace(filename), `\`, "/")
	base = path.Base(base)
	if base == "." || base == "/" || base == "" {
		base = "image"
	}
	base = safeFilenameChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, ".-")
	if base == "" {
		base = "image"
	}
	if path.Ext(base) == "" {
		if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
			base += exts[0]
		}
	}
	return base
}

func formatFilename(format string, image ImageFile) string {
	format = strings.TrimSpace(format)
	if format == "" {
		format = DefaultFilenameFormat
	}
	now := time.Now()
	ext := path.Ext(image.Filename)
	origin := strings.TrimSuffix(path.Base(image.Filename), ext)
	if origin == "" || origin == "." || origin == "/" {
		origin = "image"
	}
	hash := sha1.Sum(image.Data)
	hashValue := hex.EncodeToString(hash[:])
	replacements := map[string]string{
		"{timestamp}": fmt.Sprintf("%d", now.Unix()),
		"{y}":         now.Format("2006"),
		"{m}":         now.Format("01"),
		"{d}":         now.Format("02"),
		"{hash}":      hashValue,
		"{origin}":    origin,
		"{random}":    randomString(image.Data, 8),
		"{ext}":       ext,
		"{name}":      origin,
		"{filename}":  path.Base(image.Filename),
	}
	filename := format
	filename = randTokenPattern.ReplaceAllStringFunc(filename, func(token string) string {
		matches := randTokenPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		length, err := strconv.Atoi(matches[1])
		if err != nil || length <= 0 {
			return ""
		}
		return randomString(image.Data, length)
	})
	for token, value := range replacements {
		filename = strings.ReplaceAll(filename, token, value)
	}
	filename = sanitizeObjectName(filename)
	if filename == "" {
		return image.Filename
	}
	return filename
}

func randomString(fallback []byte, length int) string {
	if length <= 0 {
		return ""
	}
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)[:length]
	}
	hash := sha1.Sum(append(fallback, []byte(fmt.Sprintf("%d", time.Now().UnixNano()))...))
	value := hex.EncodeToString(hash[:])
	if length <= len(value) {
		return value[:length]
	}
	return value
}

func sanitizeObjectName(value string) string {
	segments := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '\\' })
	cleanSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = safeFilenameChars.ReplaceAllString(segment, "-")
		segment = strings.Trim(segment, ".-")
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		cleanSegments = append(cleanSegments, segment)
	}
	return strings.Join(cleanSegments, "/")
}

func objectPath(storagePath string, filename string) string {
	return strings.Trim(path.Join("/", strings.TrimSpace(storagePath), filename), "/")
}

func customPublicURL(customDomain string, objectPath string) string {
	customDomain = strings.TrimSpace(customDomain)
	if customDomain == "" {
		return ""
	}
	return joinURL(customDomain, objectPath)
}

func joinURL(base string, parts ...string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(path.Clean("/"+strings.TrimSpace(part)), "/")
		if part != "" && part != "." {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return base
	}
	return base + "/" + strings.Join(cleanParts, "/")
}

func findURL(value any) string {
	switch current := value.(type) {
	case map[string]any:
		for _, key := range []string{"url", "src", "path", "image", "links"} {
			if found := findURL(current[key]); found != "" {
				return found
			}
		}
		for _, nested := range current {
			if found := findURL(nested); found != "" {
				return found
			}
		}
	case []any:
		for _, nested := range current {
			if found := findURL(nested); found != "" {
				return found
			}
		}
	case string:
		if strings.HasPrefix(current, "http://") || strings.HasPrefix(current, "https://") {
			return current
		}
	}
	return ""
}
