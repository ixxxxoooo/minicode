package app

import (
	"os/exec"

	"github.com/ixxxxoooo/minicode/internal/browser"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

const (
	prefBrowserFetch = "browser.fetch"
	prefBrowserTO    = "browser.timeout"
	prefBrowserUA    = "browser.ua"
)

type comboSource struct {
	demo browser.Source
	http browser.HTTPSource
}

func (c comboSource) Fetch(u string) (browser.Page, error) {
	if pg, err := c.demo.Fetch(u); err == nil {
		return pg, nil
	}
	return c.http.Fetch(u)
}

func (a *App) browserOpts() browser.FetchOpts {
	if a.prefs == nil {
		return browser.FetchOpts{Enabled: true, TimeoutSec: 20}
	}
	return browser.FetchOpts{
		Enabled:    a.prefs.Bool(prefBrowserFetch, true),
		TimeoutSec: a.prefs.Int(prefBrowserTO, 20),
		UserAgent:  a.prefs.Str(prefBrowserUA, ""),
	}
}

func (a *App) initBrowser() {
	if a.br != nil {
		return
	}
	src := comboSource{demo: demo.BrowserSource{Set: a.demoPgBrowser}, http: browser.HTTPSource{Opts: a.browserOpts()}}
	home := ""
	if a.demoExtra {
		home = demo.BrowserHome
	}
	a.br = browser.New(src, home)
}

func (a *App) openExtBrowser(u string) {
	u = browser.Normalize(u)
	if u == "" {
		return
	}
	url := u
	if len(url) < 4 || url[:4] != "http" {
		url = "https://" + url
	}
	a.bg("open-browser", func() func() {
		err := exec.Command("/usr/bin/open", url).Run()
		return func() {
			if err != nil {
				a.toast = "无法用系统浏览器打开"
				log.Warn("系统浏览器打开失败", "host", u, "err", err)
			}
		}
	})
}

// pageRefForChat 把当前阅读页正文压成对话引用(不含 URL 查询串中的敏感参数)。
func (a *App) pageRefForChat(pg browser.Page) string {
	var body string
	for i, s := range pg.Sections {
		if i > 8 {
			body += "\n…(正文已截断)"
			break
		}
		if s.Heading != "" {
			body += "\n## " + s.Heading + "\n"
		}
		if len(s.Body) > 1200 {
			body += s.Body[:1200] + "…"
		} else {
			body += s.Body
		}
	}
	host := pg.URL
	if i := len(host); i > 120 {
		host = host[:120] + "…"
	}
	return "@网页:" + host + "\n" + body
}

func (a *App) fillDraftWithPage(pg browser.Page, question string) {
	a.draft = question + "\n\n" + a.pageRefForChat(pg)
	a.screen = ScreenWork
	if a.mode != ModeAgent {
		a.SetMode(ModeAgent)
	}
}
