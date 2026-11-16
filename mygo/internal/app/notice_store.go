package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/demo"
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
	toasts  []noticeItem // 右上角叠放(每帧画最末几条)
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
	a.pushToast(it)
	log.Info("通知", "level", level, "title", title)
}

// flashLevel 按文案推断轻提示的级别:失败类 = err,受限 / 提醒类 = warn,完成类 = ok,其余 = info。
func flashLevel(msg string) string {
	has := func(ks ...string) bool {
		for _, k := range ks {
			if strings.Contains(msg, k) {
				return true
			}
		}
		return false
	}
	switch {
	case has("失败", "出错", "错误", "无法", "不能", "被拒", "拒绝", "冲突"):
		return "err"
	case has("禁止", "请先", "没有", "暂无", "未能", "未找到", "不支持", "置灰", "超过", "已存在"):
		return "warn"
	case has("已", "成功", "完成"):
		return "ok"
	}
	return "info"
}

// flash 是一句话轻提示(原来的 toast),统一画成通知样式的卡片(右下角,几秒后淡出)。
// info / ok 只闪一下;warn / err 同时记进通知中心,漏看了还能回头查。
func (a *App) flash(msg string) {
	if msg == "" {
		return
	}
	lv := flashLevel(msg)
	if lv == "warn" || lv == "err" {
		a.postNotice(lv, msg, "", nil)
		return
	}
	a.noticeInit()
	it := noticeItem{id: a.nt.nextID, level: lv, title: msg, at: a.now(), read: true}
	a.nt.nextID++
	a.pushToast(it)
	log.Debug("轻提示", "level", lv, "msg", msg)
}

// pushToast 把一条记录排进右下角叠放区,并安排两次唤醒让它淡出、移除(无输入时界面不会自己重绘)。
func (a *App) pushToast(it noticeItem) {
	a.nt.toasts = append(a.nt.toasts, it)
	if len(a.nt.toasts) > 4 {
		a.nt.toasts = a.nt.toasts[len(a.nt.toasts)-4:]
	}
	if a.win == nil {
		return
	}
	for _, d := range []time.Duration{3900 * time.Millisecond, 6200 * time.Millisecond} {
		time.AfterFunc(d, func() { a.win.Update(func() { a.frameSoon() }) })
	}
}

// postNoticeQuiet 只记进通知中心(已读、不弹浮层):事件本身已经在眼前(如会话里的权限卡)时用,
// 避免浮层盖住要操作的卡片,同时历史里仍查得到。
func (a *App) postNoticeQuiet(level, title, body string, actions []noticeAction) {
	a.noticeInit()
	it := noticeItem{id: a.nt.nextID, level: level, title: title, body: body, at: a.now(), actions: actions, read: true}
	a.nt.nextID++
	a.nt.items = append([]noticeItem{it}, a.nt.items...)
	if len(a.nt.items) > noticeCap {
		a.nt.items = a.nt.items[:noticeCap]
	}
	log.Info("通知(静默)", "level", level, "title", title)
}

func (a *App) noticeInfo(title, body string) { a.postNotice("info", title, body, nil) }
func (a *App) noticeOK(title, body string)   { a.postNotice("ok", title, body, nil) }
func (a *App) noticeWarn(title, body string, act []noticeAction) {
	a.postNotice("warn", title, body, act)
}
func (a *App) noticeErr(title, body string, act []noticeAction) {
	a.postNotice("err", title, body, act)
}

// dismissToast 关掉右下角的一张卡片(不动通知中心里的历史)。
func (a *App) dismissToast(id int64) {
	if a.nt == nil {
		return
	}
	var out []noticeItem
	for _, it := range a.nt.toasts {
		if it.id != id {
			out = append(out, it)
		}
	}
	a.nt.toasts = out
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

// seedDemoNotices 演示模式:灌入各级别通知历史(约 120 条,截断到 noticeCap)。
func (a *App) seedDemoNotices() {
	a.noticeInit()
	if len(a.nt.items) > 0 {
		return
	}
	now := a.now()
	for _, s := range demo.NoticeSeeds() {
		it := noticeItem{
			id: a.nt.nextID, level: s.Level, title: s.Title, body: s.Body,
			at: now.Add(-s.Ago), read: s.Read,
		}
		a.nt.nextID++
		if s.Actions {
			switch s.Level {
			case "err":
				it.actions = []noticeAction{{"重试", func() { a.toast = "已加入重试队列(演示)" }}}
			case "warn":
				it.actions = []noticeAction{{"打开", func() { a.openSettings("advanced") }}, {"复制", func() { a.copyOut(s.Body, "通知") }}}
			}
		}
		a.nt.items = append(a.nt.items, it)
		if !s.Read {
			a.nt.unread++
		}
	}
	if len(a.nt.items) > noticeCap {
		a.nt.items = a.nt.items[:noticeCap]
	}
}
