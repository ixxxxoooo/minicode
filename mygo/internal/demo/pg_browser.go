// 演示数据索引 · 浏览器
//
// 套 0:docs.rs 两篇(默认首页 routing),见 browser.go。
// 套 1:长文 80 节、大表格、中文站、60 链接、重定向说明、极简一句。
// 套 2:专用 URL 返回各种抓取错误(超时 / 404 / 证书 / 被阻止 / 非 HTML)。
// 套 3:无摘要页、单句页。
//
// 在地址栏输入 demo.minicode/... 路径(见 demoURL 常量);切换套会预置书签与历史链。
// 设计稿缺口:独立「加载中」态未接(无中间态页面)。
package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/browser"
)

const (
	demoHost       = "demo.minicode"
	demoLong       = demoHost + "/long-guide"
	demoCN         = demoHost + "/zh-deep-dive"
	demoMini       = demoHost + "/one-liner"
	demoNoSum      = demoHost + "/no-summary"
	demoLinks60    = demoHost + "/link-hub"
	demoRedirect   = demoHost + "/redirect-chain"
	demoRich       = demoHost + "/rich-article"
	demoErrTimeout = demoHost + "/err/timeout"
	demoErr404     = demoHost + "/err/not-found"
	demoErrTLS     = demoHost + "/err/bad-cert"
	demoErrBlock   = demoHost + "/err/blocked"
	demoErrBinary  = demoHost + "/err/not-html"
)

// BrowserSource 演示网页数据源;Set 选择预置库(见 BrowserSetNames)。
type BrowserSource struct{ Set int }

func (bs BrowserSource) Fetch(u string) (browser.Page, error) {
	u = browser.Normalize(u)
	if pg, err := legacyBrowserFetch(u); err == nil {
		return pg, nil
	}
	if pg, ok, err := demoBrowserPage(bs.Set, u); ok {
		return pg, err
	}
	return browser.Page{}, browser.ErrNoFetch
}

func legacyBrowserFetch(u string) (browser.Page, error) {
	switch u {
	case routing.URL:
		return routing, nil
	case service.URL:
		return service, nil
	}
	return browser.Page{}, browser.ErrNoFetch
}

func demoBrowserPage(set int, u string) (browser.Page, bool, error) {
	switch u {
	case demoErrTimeout:
		return browser.Page{}, true, fmt.Errorf("连接超时(演示)")
	case demoErr404:
		return browser.Page{}, true, fmt.Errorf("服务器返回 404;稍后重试或换用系统浏览器")
	case demoErrTLS:
		return browser.Page{}, true, fmt.Errorf("证书无效:无法建立 TLS(演示)")
	case demoErrBlock:
		return browser.Page{}, true, fmt.Errorf("访问被阻止:疑似机器人验证(演示)")
	case demoErrBinary:
		return browser.Page{}, true, fmt.Errorf("非 HTML 内容:无法提取阅读模式(演示)")
	}
	if set == 2 && strings.HasPrefix(u, demoHost+"/err/") {
		return browser.Page{}, true, fmt.Errorf("未知演示错误地址")
	}
	switch u {
	case demoLong:
		return pageLong80(), true, nil
	case demoCN:
		return pageChinese(), true, nil
	case demoMini:
		return pageOneLiner(), true, nil
	case demoNoSum:
		return pageNoSummary(), true, nil
	case demoLinks60:
		return pageLinks60(), true, nil
	case demoRedirect:
		return pageRedirect(), true, nil
	case demoRich:
		return pageRich(), true, nil
	}
	if set >= 1 {
		if pg, ok := extraSet1(u); ok {
			return pg, true, nil
		}
	}
	return browser.Page{}, false, nil
}

func extraSet1(u string) (browser.Page, bool) {
	switch u {
	case demoLong, demoCN, demoRich, demoLinks60, demoRedirect, demoMini, demoNoSum:
		pg, _, _ := demoBrowserPage(1, u)
		return pg, true
	}
	return browser.Page{}, false
}

func pageLong80() browser.Page {
	var sec []browser.Section
	sec = append(sec, browser.Section{Heading: "超长指南(80 节)", Body: "用于测试大纲滚动与阅读模式性能。"})
	for i := 1; i <= 80; i++ {
		sec = append(sec, browser.Section{Heading: fmt.Sprintf("第 %d 节", i), Level: 1, Body: fmt.Sprintf("本节正文 #%d。\n\n- 要点 A\n- 要点 B", i)})
	}
	return browser.Page{URL: demoLong, Site: demoHost, ReadMin: 45, Sections: sec,
		Summary: []string{"共 80 个小节", "适合测大纲与滚动"}, Links: demoSampleLinks(8)}
}

func pageChinese() browser.Page {
	return browser.Page{
		URL: demoCN, Site: "示例中文站", ReadMin: 12,
		Sections: []browser.Section{
			{Heading: "内置浏览器中文阅读模式", Body: "验证中文字体、行高与标点换行。"},
			{Heading: "表格与代码混排", Level: 1, Body: "| 列 | 值 |\n| --- | --- |\n| 延迟 | 32 ms |\n\n```python\nprint('你好')\n```"},
			{Heading: "图片占位", Level: 1, Body: "![示意图](placeholder.png)\n\n图为演示占位,不加载真实图片。"},
		},
		Summary: []string{"中文正文渲染", "含表格与代码围栏"},
		Links:   []browser.Link{{Title: "返回长文", Host: demoHost, URL: demoLong}},
	}
}

func pageOneLiner() browser.Page {
	return browser.Page{URL: demoMini, Site: demoHost, ReadMin: 1,
		Sections: []browser.Section{{Heading: "极简", Body: "只有一句话的页面。"}},
		Summary:  []string{"一句话"},
	}
}

func pageNoSummary() browser.Page {
	return browser.Page{URL: demoNoSum, Site: demoHost, ReadMin: 3,
		Sections: []browser.Section{{Heading: "无摘要页", Body: "抽屉「摘要」应为空列表。"}},
		Summary:  nil,
		Links:    []browser.Link{{Title: "docs.rs", Host: "docs.rs", URL: BrowserHome}},
	}
}

func pageLinks60() browser.Page {
	links := demoSampleLinks(60)
	return browser.Page{URL: demoLinks60, Site: demoHost, ReadMin: 8,
		Sections: []browser.Section{{Heading: "链接聚合", Body: "右侧「链接」应有 60 条,可滚动。"}},
		Summary:  []string{"60 个外链"},
		Links:    links,
	}
}

func demoSampleLinks(n int) []browser.Link {
	var out []browser.Link
	for i := 0; i < n; i++ {
		out = append(out, browser.Link{Title: fmt.Sprintf("相关文章 %d", i+1), Host: "example.org", URL: fmt.Sprintf("example.org/article-%d", i+1)})
	}
	return out
}

func pageRedirect() browser.Page {
	return browser.Page{URL: demoRedirect, Site: demoHost, ReadMin: 2,
		Sections: []browser.Section{{Heading: "重定向链", Body: "演示:历史里先打开 /old-url 再 302 到此页(用历史后退验证)。"}},
		Summary:  []string{"模拟经过 302 的最终页"},
		Links:    []browser.Link{{Title: "旧地址(演示)", Host: demoHost, URL: demoHost + "/old-url"}},
	}
}

func pageRich() browser.Page {
	table := strings.Builder{}
	table.WriteString("| 服务 | RPS | P99 |\n| --- | ---: | ---: |\n")
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&table, "| svc-%d | %d | %dms |\n", i, 1000+i*50, 20+i%15)
	}
	code := "```rust\nasync fn handler() -> Result<(), Error> {\n    Ok(())\n}\n```"
	return browser.Page{
		URL: demoRich, Site: demoHost, ReadMin: 18,
		Sections: []browser.Section{
			{Heading: "富内容文章", Body: "大表格 + 代码 + 图片占位。"},
			{Heading: "宽表", Level: 1, Body: table.String()},
			{Heading: "代码块", Level: 1, Body: code},
			{Heading: "图片", Level: 1, Body: "[图:监控大盘截图占位]"},
		},
		Summary: []string{"24 行指标表", "含 Rust 代码"},
		Links:   demoSampleLinks(6),
	}
}

// BrowserDemoHome 返回某套的建议起始 URL(供切换命令导航)。
func BrowserDemoHome(set int) string {
	switch set {
	case 1:
		return demoLong
	case 2:
		return demoErrTimeout
	case 3:
		return demoNoSum
	default:
		return BrowserHome
	}
}

// BrowserDemoSeedHistory 返回切换套时要依次打开的 URL(模拟历史与书签)。
func BrowserDemoSeedHistory(set int) []string {
	switch set {
	case 1:
		return []string{demoLong, demoCN, demoRich, demoLinks60}
	case 2:
		return []string{demoErr404, demoLong}
	case 3:
		return []string{demoNoSum, demoMini}
	default:
		return nil
	}
}
