// Package browser 是内置「阅读模式浏览器」的与界面无关的部分:页面模型、历史前进 / 后退、地址规整。
// 真实抓取与正文提取以后实现 Source 接口即可;演示数据源在 internal/demo。
package browser

import (
	"errors"
	"strings"
)

// Section 是正文的一节:标题(Level 0 = 文章标题)+ Markdown 正文。
type Section struct {
	Heading string
	Level   int
	Body    string
}

// Link 是页面里的外链。URL 为空表示只展示、不能在内置浏览器里打开。
type Link struct{ Title, Host, URL string }

// Page 是提取好的阅读模式页面。
type Page struct {
	URL      string
	Site     string
	ReadMin  int // 阅读约几分钟
	Sections []Section
	Summary  []string
	Links    []Link
}

// Title 返回文章标题。
func (p Page) Title() string {
	if len(p.Sections) > 0 {
		return p.Sections[0].Heading
	}
	return p.URL
}

// OutlineItem 是大纲项。
type OutlineItem struct {
	Index int // 在 Sections 里的下标
	Text  string
	Level int
}

// Outline 由各节标题得出。
func (p Page) Outline() []OutlineItem {
	var o []OutlineItem
	for i, s := range p.Sections {
		if s.Heading != "" {
			o = append(o, OutlineItem{i, s.Heading, s.Level})
		}
	}
	return o
}

// ErrNoFetch 表示当前数据源不能抓取这个地址。
var ErrNoFetch = errors.New("尚未接入网页抓取")

// Source 取一个页面。
type Source interface {
	Fetch(url string) (Page, error)
}

// Browser 带历史的浏览器状态。
type Browser struct {
	src  Source
	hist []string
	cur  int
	page Page
	Err  error // 最近一次导航的错误(成功则为 nil)
	Fav  map[string]bool

	failed string          // 最近一次失败导航的地址(规整后);「重新载入」优先重试它
	plain  map[string]bool // 用户明确写了 http:// 的地址(规整后),重新载入 / 前进后退沿用
}

// New 建立浏览器,起始页为 home(空则不导航)。
func New(src Source, home string) *Browser {
	b := &Browser{src: src, cur: -1, Fav: map[string]bool{}}
	if home != "" {
		b.Go(home)
	}
	return b
}

// Normalize 去掉协议头与末尾斜杠,作为历史与收藏的键;空白返回空串。
func Normalize(u string) string {
	u = strings.TrimSpace(u)
	for _, p := range []string{"https://", "http://"} {
		u = strings.TrimPrefix(u, p)
	}
	return strings.TrimRight(u, "/")
}

// Go 导航到 url 并截断前进历史;失败时保留旧页面,Err 记录原因,地址记入 failed 供「重新载入」重试。
func (b *Browser) Go(url string) error {
	u := Normalize(url)
	if u == "" {
		return nil
	}
	if b.plain == nil {
		b.plain = map[string]bool{}
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "http://") {
		b.plain[u] = true
	}
	if err := b.load(u); err != nil {
		b.failed = u
		return err
	}
	b.failed = ""
	b.hist = append(b.hist[:b.cur+1], u)
	b.cur = len(b.hist) - 1
	return nil
}

// load 取页面;用户明确写过 http:// 的地址带上协议交给数据源(Normalize 会丢掉协议)。
func (b *Browser) load(u string) error {
	arg := u
	if b.plain[u] {
		arg = "http://" + u
	}
	p, err := b.src.Fetch(arg)
	b.Err = err
	if err != nil {
		return err
	}
	b.page = p
	return nil
}

// CanBack / CanForward 判断能否后退 / 前进。
func (b *Browser) CanBack() bool    { return b.cur > 0 }
func (b *Browser) CanForward() bool { return b.cur >= 0 && b.cur < len(b.hist)-1 }

// Back 后退一步。
func (b *Browser) Back() {
	if b.CanBack() {
		b.failed = ""
		b.cur--
		_ = b.load(b.hist[b.cur])
	}
}

// Forward 前进一步。
func (b *Browser) Forward() {
	if b.CanForward() {
		b.failed = ""
		b.cur++
		_ = b.load(b.hist[b.cur])
	}
}

// Reload 重新载入当前页;若上一次导航失败,则重试失败的地址。
func (b *Browser) Reload() {
	if b.failed != "" { // 上一次导航失败:重试它(失败时还没进历史,所以单靠历史重载什么也不做)
		_ = b.Go(b.failed)
		return
	}
	if b.cur >= 0 {
		_ = b.load(b.hist[b.cur])
	}
}

// Page 返回当前页(尚无页面时为零值)。
func (b *Browser) Page() Page { return b.page }

// URL 返回当前地址(无页面为空)。
func (b *Browser) URL() string {
	if b.cur < 0 {
		return ""
	}
	return b.hist[b.cur]
}

// ToggleFav 切换当前页的收藏,返回切换后的状态。
func (b *Browser) ToggleFav() bool {
	u := b.URL()
	if u == "" {
		return false
	}
	b.Fav[u] = !b.Fav[u]
	return b.Fav[u]
}
