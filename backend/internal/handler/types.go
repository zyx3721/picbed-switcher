package handler

import (
	"github.com/jerion/picbed-switcher/internal/model"
	"github.com/jerion/picbed-switcher/internal/picbed"
	"github.com/jerion/picbed-switcher/internal/utils"
	"regexp"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type configField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
}
type picbedTypeDef struct {
	Value       string        `json:"value"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Fields      []configField `json:"fields"`
}

func filenameFormatField() configField {
	return configField{Key: "filename_format", Label: "Filename format", Placeholder: picbed.DefaultFilenameFormat}
}

var picbedTypeDefs = []picbedTypeDef{
	{Value: "github", Label: "GitHub", Description: "GitHub repository storage.", Fields: append([]configField{{Key: "repository", Label: "Repository", Placeholder: "owner/repo", Required: true}, {Key: "branch", Label: "Branch", Placeholder: "main", Required: true}, {Key: "token", Label: "Token", Placeholder: "GitHub Personal Access Token", Required: true, Secret: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "images/blog"}}, filenameFormatField())},
	{Value: "gitee", Label: "Gitee", Description: "Gitee repository storage.", Fields: append([]configField{{Key: "repository", Label: "Repository", Placeholder: "owner/repo", Required: true}, {Key: "branch", Label: "Branch", Placeholder: "master", Required: true}, {Key: "token", Label: "Token", Placeholder: "Gitee Access Token", Required: true, Secret: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "images/blog"}}, filenameFormatField())},
	{Value: "tencent", Label: "Tencent COS", Description: "Tencent Cloud COS storage.", Fields: append([]configField{{Key: "secret_id", Label: "SecretId", Placeholder: "AKID...", Required: true, Secret: true}, {Key: "secret_key", Label: "SecretKey", Placeholder: "SecretKey", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-1250000000", Required: true}, {Key: "region", Label: "Region", Placeholder: "ap-guangzhou", Required: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Public domain", Placeholder: "https://cdn.example.com"}}, filenameFormatField())},
	{Value: "aliyun", Label: "Aliyun OSS", Description: "Aliyun OSS storage.", Fields: append([]configField{{Key: "access_key_id", Label: "AccessKeyId", Placeholder: "LTAI...", Required: true, Secret: true}, {Key: "access_key_secret", Label: "AccessKeySecret", Placeholder: "AccessKeySecret", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-name", Required: true}, {Key: "region", Label: "Region", Placeholder: "cn-guangzhou", Required: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Public domain", Placeholder: "https://cdn.example.com"}}, filenameFormatField())},
	{Value: "qiniu", Label: "Qiniu", Description: "Qiniu Kodo storage.", Fields: append([]configField{{Key: "access_key", Label: "AccessKey", Placeholder: "AccessKey", Required: true, Secret: true}, {Key: "secret_key", Label: "SecretKey", Placeholder: "SecretKey", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-name", Required: true}, {Key: "region", Label: "Region", Placeholder: "cn-east-1", Required: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Custom domain / CDN test domain", Placeholder: "https://cdn.example.com", Required: true}}, filenameFormatField())},
	{Value: "baidu_bos", Label: "Baidu BOS", Description: "Baidu Cloud BOS storage.", Fields: append([]configField{{Key: "access_key_id", Label: "AccessKeyId", Placeholder: "AccessKeyId", Required: true, Secret: true}, {Key: "secret_access_key", Label: "SecretAccessKey", Placeholder: "SecretAccessKey", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-name", Required: true}, {Key: "region", Label: "Region", Placeholder: "bj", Required: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Public domain", Placeholder: "https://cdn.example.com"}}, filenameFormatField())},
	{Value: "huawei_obs", Label: "Huawei OBS", Description: "Huawei Cloud OBS storage.", Fields: append([]configField{{Key: "access_key_id", Label: "AccessKeyId", Placeholder: "AccessKeyId", Required: true, Secret: true}, {Key: "secret_access_key", Label: "SecretAccessKey", Placeholder: "SecretAccessKey", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-name", Required: true}, {Key: "region", Label: "Region", Placeholder: "cn-north-4", Required: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Public domain", Placeholder: "https://cdn.example.com"}}, filenameFormatField())},
	{Value: "upyun", Label: "UpYun USS", Description: "UpYun cloud storage.", Fields: append([]configField{{Key: "bucket", Label: "Service name", Placeholder: "service-name", Required: true}, {Key: "operator", Label: "Operator", Placeholder: "operator", Required: true}, {Key: "password", Label: "Password", Placeholder: "Operator password", Required: true, Secret: true}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Acceleration domain / test domain", Placeholder: "https://img.example.com", Required: true}}, filenameFormatField())},
	{Value: "minio", Label: "MinIO", Description: "Self-hosted S3-compatible storage.", Fields: append([]configField{{Key: "endpoint", Label: "Endpoint", Placeholder: "minio.example.com:9000", Required: true}, {Key: "access_key", Label: "AccessKey", Placeholder: "minioadmin", Required: true, Secret: true}, {Key: "secret_key", Label: "SecretKey", Placeholder: "minioadmin", Required: true, Secret: true}, {Key: "bucket", Label: "Bucket", Placeholder: "bucket-name", Required: true}, {Key: "region", Label: "Region", Placeholder: "us-east-1"}, {Key: "use_ssl", Label: "Use SSL", Placeholder: "true / false"}, {Key: "storage_path", Label: "Storage path", Placeholder: "markdown/images"}, {Key: "custom_domain", Label: "Public domain", Placeholder: "https://cdn.example.com"}}, filenameFormatField())},
	{Value: "easyimage", Label: "EasyImage", Description: "Self-hosted EasyImage service.", Fields: []configField{{Key: "api_url", Label: "API URL", Placeholder: "https://img.example.com/api/index.php", Required: true}, {Key: "token", Label: "Token", Placeholder: "EasyImage Token", Required: true, Secret: true}}},
	{Value: "other", Label: "Other", Description: "Generic image host service.", Fields: append([]configField{{Key: "api_url", Label: "API URL", Placeholder: "https://img.example.com/api/index.php", Required: true}, {Key: "token", Label: "Token", Placeholder: "Upload API token", Required: true, Secret: true}}, filenameFormatField())},
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type changePasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type resetPasswordRequest struct {
	Token           string `json:"token"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type changeEmailRequest struct {
	Email string `json:"email"`
}

type picbedConfigRequest struct {
	PicBedType string            `json:"picbed_type"`
	ConfigName string            `json:"config_name"`
	Config     map[string]string `json:"config"`
	IsDefault  bool              `json:"is_default"`
}
type markdownRequest struct {
	Filename       string `json:"filename"`
	Content        string `json:"content"`
	TargetConfigID uint   `json:"target_config_id"`
}
type batchMarkdownRequest struct {
	Files          []markdownRequest `json:"files"`
	TargetConfigID uint              `json:"target_config_id"`
}

type createConvertTaskRequest struct {
	TaskType       string            `json:"task_type"`
	Files          []markdownRequest `json:"files"`
	TargetConfigID uint              `json:"target_config_id"`
}

type localImageMapping struct {
	Source  string `json:"source"`
	FileKey string `json:"file_key"`
}

type localMarkdownDocument struct {
	Filename string              `json:"filename"`
	Content  string              `json:"content"`
	Images   []localImageMapping `json:"images"`
}

type localBatchManifest struct {
	TargetConfigID uint                    `json:"target_config_id"`
	Documents      []localMarkdownDocument `json:"documents"`
}

type localTaskStoredFile struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

type localConvertTaskPayload struct {
	Manifest localBatchManifest               `json:"manifest"`
	Files    map[string][]localTaskStoredFile `json:"files"`
}

type errorResponse struct {
	Error string `json:"error" example:"请求失败"`
}

type messageResponse struct {
	Message string `json:"message" example:"操作成功"`
}

type healthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message" example:"PicBed Switcher API 服务运行正常"`
}

type authResponse struct {
	Token string                 `json:"token"`
	User  map[string]interface{} `json:"user"`
}

type userResponse struct {
	User map[string]interface{} `json:"user"`
}

type picbedTypesResponse struct {
	Types []picbedTypeDef `json:"types"`
}

type configsResponse struct {
	Configs []map[string]interface{} `json:"configs"`
}

type configResponseDoc struct {
	Config map[string]interface{} `json:"config"`
}

type analyzeMarkdownResponse struct {
	Images []utils.MarkdownImage `json:"images"`
	Counts map[string]int        `json:"counts"`
	Total  int                   `json:"total"`
}

type convertMarkdownResponse struct {
	Filename string                 `json:"filename"`
	Content  string                 `json:"content"`
	Changed  int                    `json:"changed"`
	Status   string                 `json:"status"`
	Record   model.ConversionRecord `json:"record"`
}

type batchConvertResponse struct {
	Results []map[string]interface{} `json:"results"`
}

type recordsResponse struct {
	Records []model.ConversionRecord `json:"records"`
}
type recordResponse struct {
	Record model.ConversionRecord `json:"record"`
}

type taskCreateResponse struct {
	Task    model.ConversionTask     `json:"task"`
	Results []map[string]interface{} `json:"results"`
}

type tasksResponse struct {
	Tasks []model.ConversionTask `json:"tasks"`
}

type taskDetailResponse struct {
	Task    model.ConversionTask     `json:"task"`
	Records []model.ConversionRecord `json:"records"`
}
