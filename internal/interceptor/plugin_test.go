package interceptor

import (
	"net/http"
	"strings"
	"testing"

	"wx_channel/internal/interceptor/proxy"
)

type channelPluginContext struct {
	req  *proxy.ContextReq
	res  *proxy.ContextRes
	body string
}

func (ctx *channelPluginContext) Req() *proxy.ContextReq {
	return ctx.req
}

func (ctx *channelPluginContext) Res() *proxy.ContextRes {
	return ctx.res
}

func (ctx *channelPluginContext) Mock(int, map[string]string, string) {}

func (ctx *channelPluginContext) GetResponseHeader(key string) string {
	return ctx.res.Header.Get(key)
}

func (ctx *channelPluginContext) SetResponseHeader(key string, val string) {
	ctx.res.Header.Set(key, val)
}

func (ctx *channelPluginContext) SetResponseBody(body string) {
	ctx.body = body
}

func (ctx *channelPluginContext) GetResponseBody() ([]byte, error) {
	return []byte(ctx.body), nil
}

func (ctx *channelPluginContext) SetStatusCode(code int) {
	ctx.res.StatusCode = code
}

func TestChannelInjectsWeuiCSSAndOtherAssetsFromSameOrigin(t *testing.T) {
	cfg := &InterceptorConfig{
		APIServerProtocol: "http",
		APIServerHostname: "127.0.0.1",
		APIServerPort:     2022,
		APIServerAddr:     "127.0.0.1:2022",
	}
	files := newTestChannelInjectedFiles(t, nil)
	plugins := CreateChannelInterceptorPlugins(&Interceptor{
		Version:           "test-version",
		Settings:          cfg,
		FrontendVariables: map[string]any{},
	}, files)
	ctx := &channelPluginContext{
		req: &proxy.ContextReq{
			URL: &proxy.ContextURL{
				Path:     "/web/pages/feed",
				Hostname: func() string { return "channels.weixin.qq.com" },
			},
			Header: make(http.Header),
		},
		res: &proxy.ContextRes{
			Header: http.Header{},
		},
		body: "<!doctype html><html><head></head><body><p>feed</p></body></html>",
	}
	ctx.res.Header.Set("Content-Type", "text/html; charset=utf-8")

	plugins[0].OnResponse(ctx)

	weuiCSS := `href="/__wx_channels_assets/lib/timeless/0.28.0/timeless.weui.css?v=test-version"`
	if !strings.Contains(ctx.body, `<link rel="stylesheet" `+weuiCSS+`>`) {
		t.Fatalf("channel HTML does not contain same-origin weui CSS link:\n%s", ctx.body)
	}
	componentsCSS := `href="/__wx_channels_assets/src/components.css"`
	if !strings.Contains(ctx.body, `<link rel="stylesheet" `+componentsCSS+`>`) {
		t.Fatalf("channel HTML does not contain components CSS stylesheet link:\n%s", ctx.body)
	}
	if strings.Contains(ctx.body, "127.0.0.1:2022/__wx_channels_assets") {
		t.Fatalf("channel HTML should use same-origin asset URLs:\n%s", ctx.body)
	}
	if !strings.Contains(ctx.body, `assetsBaseURL: "/__wx_channels_assets"`) {
		t.Fatalf("channel HTML does not override runtime asset base URL:\n%s", ctx.body)
	}
	cssIdx := strings.Index(ctx.body, "timeless.weui.css?v=test-version")
	jsIdx := strings.Index(ctx.body, "timeless.weui.umd.min.js?v=test-version")
	envOverrideIdx := strings.Index(ctx.body, `assetsBaseURL: "/__wx_channels_assets"`)
	envScriptIdx := strings.Index(ctx.body, "/__wx_channels_assets/src/env.js")
	utilsScriptIdx := strings.Index(ctx.body, "/__wx_channels_assets/src/utils.js")
	channelsScriptIdx := strings.Index(ctx.body, "/__wx_channels_assets/src/channels.js")
	feedScriptIdx := strings.Index(ctx.body, "/__wx_channels_assets/src/feed.js")
	if cssIdx < 0 || jsIdx < 0 {
		t.Fatalf("expected both weui CSS and JS assets in injected HTML:\n%s", ctx.body)
	}
	if cssIdx > jsIdx {
		t.Fatal("weui CSS should be injected before weui JS")
	}
	if envOverrideIdx < 0 || envScriptIdx < 0 || envOverrideIdx > envScriptIdx {
		t.Fatalf("runtime asset base URL override should be injected before env.js:\n%s", ctx.body)
	}
	if channelsScriptIdx < 0 {
		t.Fatalf("channel HTML does not include channels.js:\n%s", ctx.body)
	}
	if utilsScriptIdx < 0 || channelsScriptIdx < utilsScriptIdx {
		t.Fatalf("channels.js should be injected after utils.js:\n%s", ctx.body)
	}
	if feedScriptIdx < 0 || channelsScriptIdx > feedScriptIdx {
		t.Fatalf("channels.js should be injected before page script:\n%s", ctx.body)
	}
}

func TestChannelNavigationHooksDoNotDependOnStaticAssetFilename(t *testing.T) {
	plugins := CreateChannelInterceptorPlugins(&Interceptor{
		Version:  "test-version",
		Settings: &InterceptorConfig{},
	}, nil)
	ctx := &channelPluginContext{
		req: &proxy.ContextReq{
			URL: &proxy.ContextURL{
				Path:     "/t/wx_fed/web_res/js/renamed-flow.publish.js",
				Hostname: func() string { return "res.wx.qq.com" },
			},
			Header: make(http.Header),
		},
		res: &proxy.ContextRes{
			Header: http.Header{},
		},
		body: "const feedStore={value:{feeds:[],currentFeedIndex:0}},localStore={value:{feeds:[],currentFeedIndex:0}};const hooks={flowTab:feedStore,localFlowTab:localStore,goToNextFlowFeed:nextHandler,goToPrevFlowFeed:prevHandler,loadLocalPlaylist:localHandler};export{hooks}",
	}
	ctx.res.Header.Set("Content-Type", "application/javascript")

	plugins[1].OnResponse(ctx)

	for _, eventName := range []string{"GotoNextFeed", "GotoPrevFeed", "HomeFeedChanged"} {
		if !strings.Contains(ctx.body, "WXU.emit(WXU.Events."+eventName+", feed);") {
			t.Fatalf("renamed static asset did not receive %s hook:\n%s", eventName, ctx.body)
		}
	}
}

func TestChannelNavigationHooksRequireMatchingFeedState(t *testing.T) {
	plugins := CreateChannelInterceptorPlugins(&Interceptor{
		Version:  "test-version",
		Settings: &InterceptorConfig{},
	}, nil)
	originalBody := "const hooks={goToNextFlowFeed:nextHandler,goToPrevFlowFeed:prevHandler,loadLocalPlaylist:localHandler};export{hooks}"
	ctx := &channelPluginContext{
		req: &proxy.ContextReq{
			URL: &proxy.ContextURL{
				Path:     "/t/wx_fed/web_res/js/partial-flow.publish.js",
				Hostname: func() string { return "res.wx.qq.com" },
			},
			Header: make(http.Header),
		},
		res: &proxy.ContextRes{
			Header: http.Header{},
		},
		body: originalBody,
	}
	ctx.res.Header.Set("Content-Type", "application/javascript")

	plugins[1].OnResponse(ctx)

	if ctx.body != originalBody {
		t.Fatalf("navigation functions without matching feed state must remain unchanged:\n%s", ctx.body)
	}
}

func TestChannelHTMLPreservesBundleVersionWhileUsingFreshInjectionRevision(t *testing.T) {
	plugins := CreateChannelInterceptorPlugins(&Interceptor{
		Version:           "test-version",
		Settings:          &InterceptorConfig{},
		FrontendVariables: map[string]any{},
	}, newTestChannelInjectedFiles(t, nil))
	ctx := &channelPluginContext{
		req: &proxy.ContextReq{
			URL: &proxy.ContextURL{
				Path:     "/web/pages/feed",
				Hostname: func() string { return "channels.weixin.qq.com" },
			},
			Header: make(http.Header),
		},
		res: &proxy.ContextRes{
			Header: http.Header{
				"Cache-Control": []string{"public, max-age=86400"},
				"ETag":          []string{`"upstream"`},
				"Last-Modified": []string{"Fri, 15 Aug 2026 00:00:00 GMT"},
				"Content-Type":  []string{"text/html; charset=utf-8"},
			},
		},
		body: `<!doctype html><html><head><script src="/t/wx_fed/web_res/js/app.js"></script></head><body></body></html>`,
	}

	plugins[0].OnResponse(ctx)

	if !strings.Contains(ctx.body, `src="/t/wx_fed/web_res/js/app.js?t=test-version&wx_channels_inject=inject-r2"`) {
		t.Fatalf("injected HTML did not preserve the bundle version with a separate revision key:\n%s", ctx.body)
	}
	if got := ctx.res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	for _, header := range []string{"ETag", "Last-Modified", "Expires"} {
		if got := ctx.res.Header.Get(header); got != "" {
			t.Fatalf("%s = %q, want cleared after HTML injection", header, got)
		}
	}
}

func TestChannelModifiedJavaScriptDoesNotCacheUpstreamResponse(t *testing.T) {
	plugins := CreateChannelInterceptorPlugins(&Interceptor{
		Version:  "test-version",
		Settings: &InterceptorConfig{},
	}, nil)
	ctx := &channelPluginContext{
		req: &proxy.ContextReq{
			URL: &proxy.ContextURL{
				Path:     "/t/wx_fed/web_res/js/runtime.publish.js",
				Hostname: func() string { return "res.wx.qq.com" },
			},
			Header: make(http.Header),
		},
		res: &proxy.ContextRes{
			Header: http.Header{
				"Cache-Control": []string{"public, max-age=86400"},
				"ETag":          []string{`"upstream"`},
				"Last-Modified": []string{"Fri, 15 Aug 2026 00:00:00 GMT"},
				"Content-Type":  []string{"application/javascript"},
			},
		},
		body: "const runtime = true;",
	}

	plugins[1].OnResponse(ctx)

	if got := ctx.res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	for _, header := range []string{"ETag", "Last-Modified", "Expires"} {
		if got := ctx.res.Header.Get(header); got != "" {
			t.Fatalf("%s = %q, want cleared after JavaScript injection", header, got)
		}
	}
}
