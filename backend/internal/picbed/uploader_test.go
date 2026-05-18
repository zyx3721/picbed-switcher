package picbed

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestFormatFilenameUsesDefaultFormat(t *testing.T) {
	filename := formatFilename("", ImageFile{Filename: "source.png", Data: []byte("image-data")})

	matched, err := regexp.MatchString(`^\d{4}/\d{2}/\d{2}/source\.png$`, filename)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatalf("expected default date/origin/extension format, got %q", filename)
	}
}

func TestFormatFilenameReplacesSupportedTokens(t *testing.T) {
	filename := formatFilename("{name}_{rand:6}{ext}", ImageFile{Filename: "source-image.jpg", Data: []byte("image-data")})

	matched, err := regexp.MatchString(`^source-image_[a-f0-9]{6}\.jpg$`, filename)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatalf("expected custom name/random/ext format, got %q", filename)
	}
}

func TestFormatFilenameSanitizesUnsafeCharacters(t *testing.T) {
	filename := formatFilename("../{filename}", ImageFile{Filename: "source image.png", Data: []byte("image-data")})

	if strings.Contains(filename, `\\`) || strings.Contains(filename, " ") || strings.Contains(filename, "..") {
		t.Fatalf("expected unsafe path and space characters to be sanitized, got %q", filename)
	}
}

func TestNewImageFileAcceptsLocalImage(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52}
	image, err := NewImageFile(`C:\Users\demo\source image.png`, bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	if image.Filename != "source-image.png" {
		t.Fatalf("expected sanitized basename, got %q", image.Filename)
	}
	if image.ContentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", image.ContentType)
	}
}

func TestNewImageFileRejectsNonImage(t *testing.T) {
	if _, err := NewImageFile("notes.txt", strings.NewReader("not an image")); err == nil {
		t.Fatal("expected non-image input to fail")
	}
}
