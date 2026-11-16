package app

import (
	"hash/fnv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 等待动画风格(设计稿 resp-time 页「等待动画风格」):字符帧与动词表移植自主分支 appearance.rs,
// 沿用 Claude Code 的「· ✢ ✳ ✶ ✻ ✽」字符动画。

const pSpinner = "appearance.spinner" // 0 经典 / 1 趣味 / 2 Claude Code(默认)/ 3 诗意 / 4 极客 / 5 极简

const glyphFrameMs = 150 // 每一帧停留(与 Claude Code 一致)

type spinnerDef struct {
	name   string
	frames []string
	verbs  []string
}

var ccFrames = []string{"·", "✢", "✳", "✶", "✻", "✽", "✻", "✶", "✳", "✢"}

var spinnerDefs = []spinnerDef{
	{"经典", ccFrames, []string{"思考中", "琢磨中", "推敲中", "构思中", "斟酌中", "酝酿中", "梳理中", "盘算中"}},
	{"趣味", ccFrames, []string{"摸鱼中", "炼丹中", "薅头发中", "掐指一算中", "憋大招中", "搬砖中", "喂猫中", "冲咖啡中"}},
	{"Claude Code", ccFrames, []string{"沉思中", "踱步中", "冥想中", "酿造中", "蹦跶中", "琢磨透中", "神游中", "凝思中"}},
	{"诗意", []string{"◜", "◠", "◝", "◞", "◡", "◟"}, []string{"凝神中", "遐思中", "斟字酌句中", "静待花开中", "拾掇思绪中", "望月沉吟中", "研墨中", "听风中"}},
	{"极客", []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, []string{"编译宇宙中", "梳理依赖中", "调优中", "遍历语法树中", "回收内存中", "预热缓存中", "握手中", "解析意图中"}},
	{"极简", []string{"●", "◉", "○", "◉"}, []string{"工作中"}},
}

// spinner 当前风格。
func (a *App) spinner() spinnerDef {
	i := 2
	if a != nil && a.prefs != nil {
		i = a.prefs.Int(pSpinner, 2)
	}
	if i < 0 || i >= len(spinnerDefs) {
		i = 2
	}
	return spinnerDefs[i]
}

// spinnerVerb 稳定地挑一个动词:同一个 seed(会话 id + 回合开始时刻)永远得到同一个词,回合内不会乱跳。
func spinnerVerb(d spinnerDef, seed string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return d.verbs[int(h.Sum32())%len(d.verbs)] + "…"
}

// glyphFrame 当前帧:用全局时钟取模,所以界面里所有同风格的转圈同步;动效关闭时固定第一帧。
func glyphFrame(d spinnerDef) string {
	if !kit.MotionOn() {
		return d.frames[0]
	}
	return d.frames[int(time.Now().UnixMilli()/glyphFrameMs)%len(d.frames)]
}

// glyph 画一个动画字符(14px 宽的列,居中);并请求下一帧(限频 ~10 帧 / 秒,不让整个会话流每帧重排)。
func (a *App) glyph(c *ui.Context, col ui.Color) ui.Element {
	d := a.spinner()
	if kit.MotionOn() {
		a.frameIn(100 * time.Millisecond)
	}
	return ui.Text(c, glyphFrame(d)).FontSize(14).Width(14).TextAlign(ui.Center).TextColor(col).SingleLine().Shrink(0)
}

// frameIn 请求 d 之后重绘一次(低频动画用;frameSoon 是 16 ms)。
func (a *App) frameIn(d time.Duration) {
	if a.win == nil || a.frameInPending {
		return
	}
	a.frameInPending = true
	time.AfterFunc(d, func() {
		a.win.Update(func() { a.frameInPending = false })
	})
}

// activeTodoVerb 有进行中的待办时,状态行的动词换成它的进行时(对标 Claude Code 的状态行);没有返回空串。
func (a *App) activeTodoVerb() string {
	for i := len(a.st.Items) - 1; i >= 0; i-- {
		td, ok := a.st.Items[i].(*core.Todos)
		if !ok {
			continue
		}
		for _, it := range td.Items {
			if it.Status == core.TodoInProgress {
				if it.ActiveForm != "" {
					return it.ActiveForm + "…"
				}
				return it.Content + "…"
			}
		}
		return ""
	}
	return ""
}
