package app

import (
	"time"

	"github.com/egoist/mygo"
)

const noticeCap = 100

// noticeItem 是一条 IDE 通知中心的历史记录。
type noticeItem struct {
	id      int64
	level   string // info / ok / warn / err
	title   string
	body    string
	at      time.Time
	read    bool
	actions []noticeAction
}

type noticeAction struct {
	label string
	run   func() // 主线程执行
}

type noticeStore struct {
	items   []noticeItem
	unread  int
	toasts  []noticeItem // 右下角叠放(每帧画最末几条)
	nextID  int64
	panelOn bool
}

func (a *App) noticeInit() {
	if a.nt == nil {
		a.nt = &noticeStore{}
	}
}

func (a *App) postNotice(level, title, body string, actions []noticeAction) {
	a.noticeInit()
	it := noticeItem{id: a.nt.nextID, level: level, title: title, body: body, at: a.now(), actions: actions}
	a.nt.nextID++
	a.nt.items = append([]noticeItem{it}, a.nt.items...)
	if len(a.nt.items) > noticeCap {
		a.nt.items = a.nt.items[:noticeCap]
	}
	a.nt.unread++
	a.nt.toasts = append(a.nt.toasts, it)
	if len(a.nt.toasts) > 4 {
		a.nt.toasts = a.nt.toasts[len(a.nt.toasts)-4:]
	}
	log.Info("通知", "level", level, "title", title)
}

func (a *App) noticeInfo(title, body string) { a.postNotice("info", title, body, nil) }
func (a *App) noticeOK(title, body string)   { a.postNotice("ok", title, body, nil) }
func (a *App) noticeWarn(title, body string, act []noticeAction) {
	a.postNotice("warn", title, body, act)
}
func (a *App) noticeErr(title, body string, act []noticeAction) {
	a.postNotice("err", title, body, act)
}

func (a *App) markNoticesRead() {
	if a.nt == nil {
		return
	}
	a.nt.unread = 0
	for i := range a.nt.items {
		a.nt.items[i].read = true
	}
}

func (a *App) clearNotices() {
	if a.nt == nil {
		return
	}
	a.nt.items = nil
	a.nt.unread = 0
	a.nt.toasts = nil
}

func (a *App) dismissNotice(id int64) {
	if a.nt == nil {
		return
	}
	var out []noticeItem
	for _, it := range a.nt.items {
		if it.id == id {
			if !it.read {
				a.nt.unread = max(0, a.nt.unread-1)
			}
			continue
		}
		out = append(out, it)
	}
	a.nt.items = out
}

func (a *App) appInForeground() bool {
	if a.win == nil {
		return true
	}
	return mygo.App.IsActive()
}
