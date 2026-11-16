package app

import (
	"fmt"
	"time"
)

// 形态切换时的会话流「钉宽」。
//
// 对话栏带尺寸过渡(morph),过渡的每一帧库都会按中间宽度把整个子树重排一遍;会话流不是虚拟列表,
// 长会话里几百段文字每帧都要按新宽度重新断行(CoreText),一帧 60–600 ms,界面线程被看门狗记成停滞。
// 做法:记下每个形态稳定时会话流内容列的宽度,切换期间把内容列直接钉在终态宽度——
// 外框照常做宽度动画(对话栏自带裁剪),文字只在第一帧按终态宽度断行一次,之后每帧命中布局缓存。
//
// @author ixxxxoooo

// flowPin 是某个形态稳定时量到的宽度。
type flowPin struct {
	sig   string  // 量宽时的布局签名(窗口宽、各栏宽与开合),对不上就不用
	paneW float32 // 对话栏宽(决定宽 / 中 / 窄三档)
	flowW float32 // 会话流内容列宽(决定文字断行)
}

// morphSettle 是切换结束后再等多久才重新量宽(过渡 440 ms,留出余量)。
const morphSettle = 600 * time.Millisecond

// pinSig 返回当前布局的签名:窗口宽、各栏宽、本形态的开合与方案。
// 返回值:签名字符串,相同即可认为会话流宽度不变。
func (a *App) pinSig() string {
	return fmt.Sprint(a.winW, a.sideW(), a.wsRW(), a.dockW(), a.currentLayMem(), a.tk.Scheme.ID)
}

// morphPinning 报告此刻是否处在形态切换的过渡期(只在真实窗口里生效,测试要显式打开 pinInTest)。
// 返回值:true = 过渡进行中,可以钉宽。
func (a *App) morphPinning() bool {
	if a.win == nil && !a.pinInTest {
		return false
	}
	if a.ly.resizing != "" || a.morphUntil.IsZero() {
		return false
	}
	return time.Now().Before(a.morphUntil)
}

// flowPinFor 取当前形态可用的钉宽。
// 返回值:宽度记录;ok = false 表示不在过渡期或没有可信的记录(退回逐帧重排)。
func (a *App) flowPinFor() (pin flowPin, ok bool) {
	if !a.morphPinning() {
		return flowPin{}, false
	}
	pin, has := a.flowPinBy[a.mode]
	if !has || pin.flowW <= 0 || pin.paneW <= 0 || pin.sig != a.pinSig() {
		return flowPin{}, false
	}
	return pin, true
}

// rememberFlowPin 在界面稳定时记下当前形态的宽度,供下次切回来时钉宽。
// 参数:paneW 对话栏上一帧的宽;flowW 会话流内容列上一帧的宽(都取自 Bounds,为 0 表示还没布局过)。
func (a *App) rememberFlowPin(paneW, flowW float32) {
	if paneW <= 0 || flowW <= 0 || a.ly.resizing != "" {
		return
	}
	if !a.morphAt.IsZero() && time.Since(a.morphAt) < morphSettle {
		return // 过渡中量到的是中间宽度
	}
	if a.flowPinBy == nil {
		a.flowPinBy = map[Mode]flowPin{}
	}
	sig := a.pinSig()
	if old := a.flowPinBy[a.mode]; old.sig == sig && old.paneW == paneW && old.flowW == flowW {
		return
	}
	a.flowPinBy[a.mode] = flowPin{sig: sig, paneW: paneW, flowW: flowW}
}
