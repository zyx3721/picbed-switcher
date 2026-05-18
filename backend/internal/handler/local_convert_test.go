package handler

import "testing"

func TestNormalizeLocalImageSource(t *testing.T) {
	tests := map[string]string{
		`./images/source image.png`:               "images/source image.png",
		`images%2Fsource%20image.png`:             "images/source image.png",
		`file:///C:/Users/demo/Pictures/a.PNG`:    "c:/users/demo/pictures/a.png",
		`C:\Users\demo\Pictures\source-image.jpg`: "c:/users/demo/pictures/source-image.jpg",
	}
	for input, want := range tests {
		if got := normalizeLocalImageSource(input); got != want {
			t.Fatalf("normalizeLocalImageSource(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsHTTPImageURL(t *testing.T) {
	if !isHTTPImageURL("https://example.com/image.png") {
		t.Fatal("expected https URL to be detected")
	}
	if isHTTPImageURL("./images/image.png") {
		t.Fatal("expected local path to be ignored")
	}
}
