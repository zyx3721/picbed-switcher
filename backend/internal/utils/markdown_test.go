package utils

import "testing"

func TestDetectPicBed(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "github raw", raw: "https://raw.githubusercontent.com/owner/repo/main/image.png", want: "github"},
		{name: "github proxy path", raw: "https://gh-proxy.com/https://raw.githubusercontent.com/owner/repo/main/image.png", want: "github"},
		{name: "github proxy encoded path", raw: "https://gh-proxy.com/https%3A%2F%2Fraw.githubusercontent.com%2Fowner%2Frepo%2Fmain%2Fimage.png", want: "github"},
		{name: "github proxy query", raw: "https://proxy.example.com/fetch?url=https%3A%2F%2Fraw.githubusercontent.com%2Fowner%2Frepo%2Fmain%2Fimage.png", want: "github"},
		{name: "gitee", raw: "https://gitee.com/owner/repo/raw/master/image.png", want: "gitee"},
		{name: "other", raw: "https://img.example.com/image.png", want: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectPicBed(tt.raw); got != tt.want {
				t.Fatalf("DetectPicBed(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
