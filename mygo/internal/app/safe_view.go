package app

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 界面分区的故障隔离(设计稿 ia「程序健壮性」):某一块界面在构建时 panic,
// 只让这一块显示「出错了」卡片(重试 / 复制错误信息),日志与 crashes/ 里留调用栈,
// 其余区域、会话和后台任务不受影响,不再整个应用闪退。
//
// 注意:测试(a.win == nil)里不吞 panic,否则会把真实缺陷藏起来;只有 a.guardAlways 打开时才拦截。

var guardLog = diag.L("guard")

// guardKind 决定出错后这一块怎么显示。
type guardKind int

const (
	guardCard   guardKind = iota // 整块换成「出错了」卡片
	guardChip                    // 标题栏 / 状态栏:一枚红色小条,点一下重试
	guardSilent                  // 浮层:不画,5 秒后自动重试,每次出错提示一次
)

// viewFault 是一次界面构建故障。
type viewFault struct {
	name  string
	msg   string
	stack string
	at    time.Time
}

// guard 在保护下构建一块界面;key 唯一标识这一块(同一个 key 出错后不再每帧重试)。
func (a *App) guard(c *ui.Context, name, key string, kind guardKind, build func()) {
	if f := a.faults[key]; f != nil {
		if kind == guardSilent && time.Since(f.at) > 5*time.Second {
			delete(a.faults, key) // 浮层:冷却后自动重试一次
		} else {
			a.faultView(c, f, key, kind)
			return
		}
	}
	if a.win == nil && !a.guardAlways { // 测试:原样抛出
		build()
		return
	}
	func() {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			diag.Report("界面:"+name, v) // 在 defer 里调用,调用栈才带 panic 现场
			f := &viewFault{name: name, msg: fmt.Sprint(v), at: time.Now(), stack: string(debug.Stack())}
			if a.faults == nil {
				a.faults = map[string]*viewFault{}
			}
			a.faults[key] = f
			guardLog.Error("界面构建出错,已隔离这一块", "区块", name, "key", key, "error", f.msg)
			if kind == guardSilent {
				a.toast = "「" + name + "」出错了,已记录日志"
			}
			a.frameSoon() // 下一帧起显示出错卡片
		}()
		build()
	}()
}

// faultView 画出错后的占位。
func (a *App) faultView(c *ui.Context, f *viewFault, key string, kind guardKind) {
	t := a.tk
	retry := func() {
		delete(a.faults, key)
		guardLog.Info("重试界面块", "区块", f.name)
		a.redraw()
	}
	switch kind {
	case guardSilent:
		return
	case guardChip:
		ui.Row(c).Key("fault-" + key).Grow(1).Gap(6).AlignItems(ui.Center).PaddingX(8).Height(theme.Px(22)).TextColor(t.Err).FontSize(theme.FsCap).Children(func() {
			kit.Ic(c, "alert", kit.S12)
			ui.Text(c, f.name+"出错了").SingleLine()
			kit.Btn(c, "重试", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(retry)
		})
		return
	}
	msg := f.msg
	if r := []rune(msg); len(r) > 180 {
		msg = string(r[:180]) + "…"
	}
	ui.Column(c).Key("fault-" + key).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Padding(24).Children(func() {
		kit.Ic(c, "alert", kit.S20).TextColor(t.Err)
		ui.Text(c, "「"+f.name+"」出错了").FontSize(theme.FsHead).FontWeight(600)
		ui.Text(c, "已写入日志,其它功能和会话不受影响。点「重试」重新加载这一块;反复出错请把错误信息发给开发者。").
			TextColor(t.Tx3).LineHeight(1.5).MaxWidth(420).TextAlign(ui.Center)
		ui.Text(c, msg).Font(theme.CodeFont()).FontSize(theme.Px(11)).TextColor(t.Tx3).MaxWidth(480).TextAlign(ui.Center)
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "重试", kit.BtnOpt{Variant: kit.BtnPri, Icon: "refresh"}).OnClick(retry)
			kit.Btn(c, "复制错误信息", kit.BtnOpt{Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() {
				a.copyOut(f.name+": "+f.msg+"\n\n"+strings.TrimSpace(f.stack), "错误信息")
			})
		})
	})
}
