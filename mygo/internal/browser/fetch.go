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
	if !strings.Contains(u, ".") {
		return Page{}, fmt.Errorf("网址格式不正确")
	}
	full := u
	if !strings.HasPrefix(u, "http") {
		full = "https://" + u
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
	resp, err := client.Do(req)
	if err != nil {
		blog.Warn("抓取失败", "host", RedactURL(full), "err", err, "ms", time.Since(start).Milliseconds())
		return Page{}, fmt.Errorf("无法连接:%v;可检查网络或在系统浏览器中打开", err)
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
	blog.Info("抓取完成", "host", RedactURL(full), "sections", len(pg.Sections), "ms", time.Since(start).Milliseconds())
	return pg, nil
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

func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return "https://" + base + href
	}
	return "https://" + base + "/" + href
}
