package browser

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var blog = diag.L("browser")

// FetchOpts 控制抓取上限与请求头。
type FetchOpts struct {
	TimeoutSec int
	MaxBody    int64
	UserAgent  string
	Enabled    bool
}

// HTTPSource 用 net/http 抓取并抽取阅读模式正文。
type HTTPSource struct {
	Opts FetchOpts
}

// Fetch 实现 Source。
func (h HTTPSource) Fetch(raw string) (Page, error) {
	if !h.Opts.Enabled {
		return Page{}, fmt.Errorf("内置浏览器抓取已在设置中关闭;可在 设置 → 浏览器 中开启")
	}
	u := Normalize(raw)
	if u == "" {
		return Page{}, fmt.Errorf("请输入网址")
	}
	explicitHTTP := strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "http://")
	full, ok := fullURL(u)
	if !ok {
		return Page{}, fmt.Errorf("网址格式不正确")
	}
	sec := h.Opts.TimeoutSec
	if sec <= 0 {
		sec = 20
	}
	max := h.Opts.MaxBody
	if max <= 0 {
		max = 2 << 20
	}
	start := time.Now()
	client := &http.Client{Timeout: time.Duration(sec) * time.Second}
	req, err := http.NewRequest(http.MethodGet, full, nil)
	if err != nil {
		return Page{}, err
	}
	ua := h.Opts.UserAgent
	if ua == "" {
		ua = "MiniCode/1.0 (reading-mode; +https://minicode.local)"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	if explicitHTTP && strings.HasPrefix(full, "https://") {
		full = "http://" + strings.TrimPrefix(full, "https://") // 用户明确写了 http://
		req.URL, _ = url.Parse(full)
	}
	resp, err := client.Do(req)
	if err != nil && strings.HasPrefix(full, "https://") && httpsUnsupported(err) {
		// 站点只有 http:自动补的 https 握手失败时退回 http 重试一次
		alt := "http://" + strings.TrimPrefix(full, "https://")
		if req2, e2 := http.NewRequest(http.MethodGet, alt, nil); e2 == nil {
			req2.Header = req.Header.Clone()
			if resp2, e3 := client.Do(req2); e3 == nil {
				blog.Info("https 不可用,已退回 http", "host", RedactURL(alt))
				full, resp, err = alt, resp2, nil
			}
		}
	}
	if err != nil {
		blog.Warn("抓取失败", "host", RedactURL(full), "err", err, "ms", time.Since(start).Milliseconds())
		return Page{}, fmt.Errorf("%s", friendlyNetErr(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		blog.Warn("HTTP 错误", "host", RedactURL(full), "code", resp.StatusCode)
		return Page{}, fmt.Errorf("服务器返回 %d;稍后重试或换用系统浏览器", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, max))
	if err != nil {
		return Page{}, err
	}
	if !utf8.Valid(body) {
		body, _ = io.ReadAll(transform.NewReader(bytes.NewReader(body), charmap.Windows1252.NewDecoder()))
	}
	pg, err := extractHTML(body, u)
	if err != nil {
		return Page{}, err
	}
	pg.Insecure = strings.HasPrefix(full, "http://")
	blog.Info("抓取完成", "host", RedactURL(full), "sections", len(pg.Sections), "ms", time.Since(start).Milliseconds())
	return pg, nil
}

// fullURL 把规整后的地址(已去掉协议)补成可请求的完整网址;主机名不合法返回 false。
// 本机地址(localhost / 127.x / ::1 / *.local / 带端口的内网 IP)默认走 http,方便预览本地开发服务器,其余走 https。
func fullURL(u string) (string, bool) {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		u = Normalize(u)
	}
	p, err := url.Parse("//" + u)
	if err != nil || p.Hostname() == "" {
		return "", false
	}
	host := strings.ToLower(p.Hostname())
	local := host == "localhost" || host == "::1" || strings.HasSuffix(host, ".local") ||
		strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.")
	// 既不是本机、也不含「.」(如 foo)基本是敲错了,不去请求
	if !local && !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return "", false
	}
	if local {
		return "http://" + u, true
	}
	return "https://" + u, true
}

// httpsUnsupported 判断错误是否意味着「对方不说 HTTPS」(而不是域名 / 超时 / 证书被拒这类别的问题)。
func httpsUnsupported(err error) bool {
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "server gave http response") || strings.Contains(m, "first record does not look like a tls handshake") ||
		strings.Contains(m, "connection reset") || strings.Contains(m, "connection refused") || strings.Contains(m, "protocol version not supported") || strings.Contains(m, "eof")
}

// friendlyNetErr 把网络错误翻成一句中文加下一步,原文只进日志。
func friendlyNetErr(err error) string {
	m := strings.ToLower(err.Error())
	switch {
	case strings.Contains(m, "connection refused"):
		return "连接被拒绝:这个地址上没有服务在监听(本地开发服务器是否已启动?)"
	case strings.Contains(m, "no such host"):
		return "找不到这个域名,请检查拼写或网络"
	case strings.Contains(m, "timeout") || strings.Contains(m, "deadline exceeded"):
		return "连接超时,可检查网络,或在 设置 → 浏览器 调大超时"
	case strings.Contains(m, "certificate") || strings.Contains(m, "tls"):
		return "证书校验失败,这个站点的 HTTPS 不可信;可在系统浏览器中打开"
	}
	return "无法连接,可检查网络或在系统浏览器中打开"
}

func hostOnly(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return u
	}
	return p.Host
}

func extractHTML(body []byte, normURL string) (Page, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Page{}, fmt.Errorf("无法解析 HTML")
	}
	title := findTitle(root)
	main := findMain(root)
	if main == nil {
		main = root
	}
	var secs []Section
	walkContent(main, &secs, 0)
	if len(secs) == 0 && title != "" {
		secs = []Section{{Heading: title, Level: 0}}
	}
	if len(secs) > 0 && secs[0].Heading == "" && secs[0].Level == 0 && title != "" {
		secs[0].Heading = title // 开头是没有标题的正文:标题取 <title>,正文留在这一节里
	} else if len(secs) > 0 && secs[0].Level != 0 {
		secs = append([]Section{{Heading: title, Level: 0}}, secs...)
	}
	site := normURL
	if i := strings.Index(site, "/"); i >= 0 {
		site = site[:i]
	}
	words := 0
	for _, s := range secs {
		words += len([]rune(s.Body + s.Heading))
	}
	mins := max(words/400, 1)
	sum, links := collectMeta(root, normURL)
	if len(sum) == 0 {
		for _, s := range secs {
			t := strings.TrimSpace(s.Body)
			if t != "" {
				sum = append(sum, t)
			}
			if len(sum) >= 4 {
				break
			}
		}
	}
	return Page{URL: normURL, Site: site, ReadMin: mins, Sections: secs, Summary: sum, Links: links}, nil
}

func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
		return strings.TrimSpace(n.FirstChild.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

func findMain(n *html.Node) *html.Node {
	var best *html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.ElementNode {
			switch x.Data {
			case "article", "main":
				best = x
				return
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			if best != nil {
				return
			}
		}
	}
	walk(n)
	if best == nil {
		best = findTag(n, "body") // 文档根下面是 <html>,<body> 在它里面,要递归找
	}
	return best
}

// findTag 深度优先找第一个指定标签。
func findTag(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := findTag(c, tag); r != nil {
			return r
		}
	}
	return nil
}

// skipTags 是不含正文的容器:导航、页眉页脚、脚本样式、表单等。
var skipTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true, "header": true,
	"footer": true, "aside": true, "form": true, "svg": true, "head": true, "template": true,
}

func walkContent(n *html.Node, out *[]Section, lvl int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "h1", "h2", "h3", "h4":
			lv := 1
			if c.Data == "h1" {
				lv = 1
			} else if c.Data == "h2" {
				lv = 1
			} else {
				lv = 2
			}
			*out = append(*out, Section{Heading: textOf(c), Level: lv})
		case "p":
			if t := strings.TrimSpace(textOf(c)); t != "" {
				if len(*out) == 0 {
					*out = append(*out, Section{Body: t})
				} else {
					last := &(*out)[len(*out)-1]
					if last.Body != "" {
						last.Body += "\n\n" + t
					} else {
						last.Body = t
					}
				}
			}
		case "pre", "code":
			if t := strings.TrimSpace(textOf(c)); t != "" {
				block := "\n```\n" + t + "\n```\n"
				if len(*out) == 0 {
					*out = append(*out, Section{Body: block})
				} else {
					(*out)[len(*out)-1].Body += block
				}
			}
		case "img":
			alt := attr(c, "alt")
			if alt != "" {
				if len(*out) == 0 {
					*out = append(*out, Section{Body: "*(" + alt + ")*"})
				} else {
					(*out)[len(*out)-1].Body += "\n*(" + alt + ")*\n"
				}
			}
		default:
			if !skipTags[c.Data] { // div / section / html / body 等容器一律往里找正文
				walkContent(c, out, lvl+1)
			}
		}
	}
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var w func(*html.Node)
	w = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			w(c)
		}
	}
	w(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func collectMeta(root *html.Node, base string) ([]string, []Link) {
	var sum []string
	var links []Link
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attr(n, "href")
			t := textOf(n)
			if href == "" || t == "" {
				goto kids
			}
			u := resolveURL(base, href)
			if u == "" {
				goto kids
			}
			h := hostOnly(u)
			if !seen[u] && len(links) < 40 {
				seen[u] = true
				links = append(links, Link{Title: t, Host: h, URL: Normalize(u)})
			}
		}
		if n.Type == html.ElementNode && n.Data == "p" {
			t := textOf(n)
			if len(sum) < 4 && len(strings.TrimSpace(t)) > 12 {
				sum = append(sum, strings.TrimSpace(t))
			}
		}
	kids:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return sum, links
}

// resolveURL 按标准规则把 href 解析成绝对地址。base 是不带协议的页面地址(如 "localhost:8765/docs/a.html"),
// 协议在后面的 Normalize 里会被丢掉,这里统一用 https 占位。
// 相对路径按页面所在目录解析(b.html、../x、?q 都对);mailto: / javascript: / 纯锚点返回空串,调用方跳过。
func resolveURL(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	low := strings.ToLower(href)
	for _, p := range []string{"javascript:", "mailto:", "tel:", "data:", "about:"} {
		if strings.HasPrefix(low, p) {
			return ""
		}
	}
	b, err := url.Parse("https://" + base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(href)
	if err != nil {
		return ""
	}
	u := b.ResolveReference(r)
	u.Fragment = ""
	return u.String()
}
