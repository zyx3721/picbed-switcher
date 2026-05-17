package picbed

import (
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
