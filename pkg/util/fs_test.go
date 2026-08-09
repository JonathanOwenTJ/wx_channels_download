package util

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilenamePreservesExtensionWhenTruncated(t *testing.T) {
	processor := NewFilenameProcessor("", make(map[string]int))
	input := strings.Repeat("这是一个很长的视频标题", 20) + ".mp4"

	name, err := processor.SanitizeFilename(input)
	if err != nil {
		t.Fatalf("sanitize long filename: %v", err)
	}
	if !strings.HasSuffix(name, ".mp4") {
		t.Fatalf("long filename lost extension: %q", name)
	}
	if len(name) > processor.maxFilenameLength {
		t.Fatalf("filename length = %d, want <= %d", len(name), processor.maxFilenameLength)
	}
}

func TestSanitizeFilenameDoesNotChangeShortFilename(t *testing.T) {
	processor := NewFilenameProcessor("", make(map[string]int))
	const input = "普通标题.mp4"

	name, err := processor.SanitizeFilename(input)
	if err != nil {
		t.Fatalf("sanitize short filename: %v", err)
	}
	if name != input {
		t.Fatalf("short filename = %q, want %q", name, input)
	}
}

func TestProcessFilenamePreservesForwardSlashDirectories(t *testing.T) {
	processor := NewFilenameProcessor("", make(map[string]int))

	name, dir, err := processor.ProcessFilename("案痕推理薄/标题①②/③_xWT156.jpg")
	if err != nil {
		t.Fatalf("process nested filename: %v", err)
	}
	if name != "③_xWT156.jpg" {
		t.Fatalf("name = %q, want %q", name, "③_xWT156.jpg")
	}
	wantDir := filepath.Join("案痕推理薄", "标题①②")
	if dir != wantDir {
		t.Fatalf("dir = %q, want %q", dir, wantDir)
	}
}
