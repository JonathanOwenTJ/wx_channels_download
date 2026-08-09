package api

import (
	"strings"
	"testing"

	"wx_channel/pkg/util"
)

func TestResolveAuthorFolder(t *testing.T) {
	processor := util.NewFilenameProcessor("", make(map[string]int))
	tests := []struct {
		name     string
		nickname string
		username string
		want     string
	}{
		{name: "nickname", nickname: " 阿俊说商业 ", username: "wxid_123", want: "阿俊说商业"},
		{name: "username fallback", nickname: "", username: " wxid_123 ", want: "wxid_123"},
		{name: "unknown fallback", nickname: "", username: "", want: "未知视频号"},
		{name: "remove path separators", nickname: `博主/栏目\\精选`, username: "wxid_123", want: "博主栏目精选"},
		{name: "invalid nickname falls back", nickname: `<>:\\|?*`, username: "wxid_123", want: "wxid_123"},
		{name: "dot nickname falls back", nickname: "..", username: "wxid_123", want: "wxid_123"},
		{name: "reserved device name", nickname: "CON", username: "wxid_123", want: "CON_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAuthorFolder(processor, tt.nickname, tt.username); got != tt.want {
				t.Fatalf("resolveAuthorFolder(%q, %q) = %q, want %q", tt.nickname, tt.username, got, tt.want)
			}
		})
	}
}

func TestResolveAuthorFolderTruncatesLongNickname(t *testing.T) {
	processor := util.NewFilenameProcessor("", make(map[string]int))
	got := resolveAuthorFolder(processor, strings.Repeat("视", 100), "wxid_123")
	if len(got) > 235 {
		t.Fatalf("author folder length = %d, want <= 235 bytes", len(got))
	}
	if !strings.HasPrefix(got, "视") {
		t.Fatalf("author folder = %q, want truncated nickname", got)
	}
}

func TestRenderFilenameTemplateIncludesAuthorFolder(t *testing.T) {
	got := renderFilenameTemplate(
		"{{author_folder}}/{{filename}}_{{spec}}",
		"默认标题",
		map[string]string{
			"author_folder": "阿俊说商业",
			"filename":      "视频标题",
			"spec":          "1080p",
		},
	)
	if got != "阿俊说商业/视频标题_1080p" {
		t.Fatalf("rendered filename = %q", got)
	}
}

func TestSanitizeFilenameTemplateComponentRemovesPathSeparators(t *testing.T) {
	processor := util.NewFilenameProcessor("", make(map[string]int))
	got := sanitizeFilenameTemplateComponent(processor, `标题①/②\③:*?`)
	if got != "标题①②③" {
		t.Fatalf("sanitized template component = %q, want %q", got, "标题①②③")
	}
}
