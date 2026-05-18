package utils

import (
	"net/url"
	"regexp"
	"strings"
)

type MarkdownImage struct {
	Raw    string `json:"raw"`
	URL    string `json:"url"`
	Alt    string `json:"alt"`
	PicBed string `json:"picbed"`
}

var imagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^\s)]+)(?:\s+"[^"]*")?\)|<img[^>]+src=["']([^"']+)["'][^>]*>`)

func ExtractMarkdownImages(content string) []MarkdownImage {
	matches := imagePattern.FindAllStringSubmatch(content, -1)
	images := make([]MarkdownImage, 0, len(matches))

	for _, match := range matches {
		imageURL := match[2]
		alt := match[1]
		if imageURL == "" {
			imageURL = match[3]
		}
		images = append(images, MarkdownImage{
			Raw:    match[0],
			URL:    imageURL,
			Alt:    alt,
			PicBed: DetectPicBed(imageURL),
		})
	}

	return images
}

func DetectPicBed(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "unknown"
	}
	if detected := detectPicBedByHostAndPath(parsed.Host, parsed.Path); detected != "other" {
		return detected
	}
	if embedded := extractEmbeddedURL(parsed); embedded != "" {
		return DetectPicBed(embedded)
	}
	return "other"
}

func detectPicBedByHostAndPath(hostValue string, pathValue string) string {
	host := strings.ToLower(hostValue)
	path := strings.ToLower(pathValue)

	switch {
	case strings.Contains(host, "githubusercontent.com") || strings.Contains(host, "github.com"):
		return "github"
	case strings.Contains(host, "gitee.com"):
		return "gitee"
	case strings.Contains(host, "myqcloud.com") || strings.Contains(host, "tencent"):
		return "tencent"
	case strings.Contains(host, "aliyuncs.com") || strings.Contains(host, "aliyun"):
		return "aliyun"
	case strings.Contains(host, "qiniucdn.com") || strings.Contains(host, "qiniucs.com") || strings.Contains(host, "clouddn.com") || strings.Contains(host, "qiniu"):
		return "qiniu"
	case strings.Contains(host, "easyimage") || strings.Contains(path, "easyimage") || strings.Contains(path, "/i/"):
		return "easyimage"
	default:
		return "other"
	}
}

func extractEmbeddedURL(parsed *url.URL) string {
	for _, value := range []string{parsed.Path, parsed.EscapedPath(), parsed.RawQuery} {
		if embedded := firstEmbeddedURL(value); embedded != "" {
			return embedded
		}
	}
	return ""
}

func firstEmbeddedURL(value string) string {
	current := value
	for range 3 {
		if embedded := firstHTTPURL(current); embedded != "" {
			return embedded
		}
		unescaped, err := url.QueryUnescape(current)
		if err != nil || unescaped == current {
			break
		}
		current = unescaped
	}
	return ""
}

func firstHTTPURL(value string) string {
	lower := strings.ToLower(value)
	start := -1
	for _, marker := range []string{"https://", "http://"} {
		if index := strings.Index(lower, marker); index >= 0 && (start == -1 || index < start) {
			start = index
		}
	}
	if start == -1 {
		return ""
	}
	candidate := value[start:]
	if end := strings.IndexAny(candidate, " \t\r\n\"'<>)&"); end >= 0 {
		candidate = candidate[:end]
	}
	return strings.TrimSpace(candidate)
}

func ReplaceImageHost(content string, source string, target string, targetBaseURL string) (string, int) {
	if targetBaseURL == "" {
		targetBaseURL = "https://example.com/" + target
	}
	targetBaseURL = strings.TrimRight(targetBaseURL, "/")

	changed := 0
	result := imagePattern.ReplaceAllStringFunc(content, func(raw string) string {
		matches := imagePattern.FindStringSubmatch(raw)
		if len(matches) == 0 {
			return raw
		}

		currentURL := ""
		if matches[2] != "" {
			currentURL = matches[2]
		} else {
			currentURL = matches[3]
		}
		if source != "" && source != "unknown" && DetectPicBed(currentURL) != source {
			return raw
		}

		newURL := BuildTargetURL(currentURL, targetBaseURL)
		changed++
		return strings.Replace(raw, currentURL, newURL, 1)
	})

	return result, changed
}

func ReplaceImageURLs(content string, replace func(currentURL string) (string, error)) (string, int, error) {
	changed := 0
	var replaceErr error
	result := imagePattern.ReplaceAllStringFunc(content, func(raw string) string {
		if replaceErr != nil {
			return raw
		}
		matches := imagePattern.FindStringSubmatch(raw)
		if len(matches) == 0 {
			return raw
		}

		currentURL := ""
		if matches[2] != "" {
			currentURL = matches[2]
		} else {
			currentURL = matches[3]
		}
		newURL, err := replace(currentURL)
		if err != nil {
			replaceErr = err
			return raw
		}
		if strings.TrimSpace(newURL) == "" || newURL == currentURL {
			return raw
		}
		changed++
		return strings.Replace(raw, currentURL, newURL, 1)
	})
	if replaceErr != nil {
		return content, changed, replaceErr
	}
	return result, changed, nil
}

func BuildTargetURL(currentURL string, targetBaseURL string) string {
	parsed, err := url.Parse(currentURL)
	filename := "image"
	if err == nil {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			filename = parts[len(parts)-1]
		}
	}
	return targetBaseURL + "/" + filename
}
