package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/update"
)

// 偏好「自动下载更新」默认开;关了才进入「只检查不下载」。
func TestUpdAutoOnDefaultsToTrue(t *testing.T) {
	a, _ := newTestApp(t)
	if !a.updAutoOn() {
		t.Fatal("默认应自动下载更新")
	}
	a.prefs.Set(pAboutAuto, false)
	if a.updAutoOn() {
		t.Fatal("关闭偏好后不应自动下载")
	}
}

// 发现新版本但没自动下载:状态是「发现新版本」且带 Manual,并只通知一次。
func TestOnUpdateProgressFoundManual(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Set(pAboutAuto, false)
	a.noticeInit()
	count := func() int { // 演示数据可能已把通知塞满上限,按标题数而不是总数
		n := 0
		for _, it := range a.nt.items {
			if it.title == "发现新版本 9.9.9" {
				n++
			}
		}
		return n
	}
	a.onUpdateProgress(update.Progress{Version: "9.9.9", Found: true})
	if a.updSt.Phase != updFound || !a.updSt.Manual || a.updSt.Version != "9.9.9" {
		t.Fatalf("应进入手动「发现新版本」:%+v", a.updSt)
	}
	if hint, _ := a.updAboutHint(); hint != "发现新版本 9.9.9(未自动下载)" {
		t.Fatalf("关于页文案不对:%q", hint)
	}
	a.onUpdateProgress(update.Progress{Version: "9.9.9", Found: true})
	if got := count(); got != 1 {
		t.Fatalf("同一版本只该通知一次,实际 %d 条", got)
	}
}

// 静默检查:失败、已是最新、检查中信号都不改状态栏;发现新版本才出现。
func TestSilentUpdateCheckStaysQuiet(t *testing.T) {
	a, _ := newTestApp(t)
	for name, p := range map[string]update.Progress{
		"检查中": {Phase: update.PhaseNone, Version: "0.3.0"},
		"失败":  {Err: "dial tcp: no such host", ErrStage: "check"},
		"最新":  {Version: "0.3.0", UpToDate: true},
	} {
		a.updSilent = true
		a.updSt = updState{}
		a.update = ""
		a.onUpdateProgress(p)
		if a.updSt.Phase != updNone || a.update != "" {
			t.Fatalf("静默检查「%s」不该改界面:%+v / %q", name, a.updSt, a.update)
		}
	}
	// 下载阶段的失败不是静默范围:要让用户看见
	a.updSilent = true
	a.onUpdateProgress(update.Progress{Err: "unexpected EOF", ErrStage: "download"})
	if a.updSt.Phase != updFail {
		t.Fatalf("下载失败必须显示:%+v", a.updSt)
	}
	// 静默检查发现新版本且自动下载 → 进入下载,不再静默
	a.updSilent = true
	a.onUpdateProgress(update.Progress{Phase: update.PhaseDownload, Percent: 40, Version: "9.9.9"})
	if a.updSt.Phase != updDownload || a.updSilent {
		t.Fatalf("自动下载应显示进度:%+v silent=%v", a.updSt, a.updSilent)
	}
}

// 正在下载 / 待重启时,新的检查不能把进度冲掉。
func TestCheckUpdateSkipsWhenBusy(t *testing.T) {
	a, _ := newTestApp(t)
	for _, ph := range []updPhase{updDownload, updVerify, updUnpack, updReady} {
		a.updSt = updState{Phase: ph, Version: "9.9.9"}
		a.checkUpdate(false)
		if a.updSt.Phase != ph {
			t.Fatalf("阶段 %v 被新检查覆盖成 %v", ph, a.updSt.Phase)
		}
	}
}

// 手动模式端到端:检查 → 发现(不下载)→ 点下载 → 走到「可重启」。用演示数据源,不碰网络。
func TestManualUpdateFlowWithDemoFetcher(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // 演示下载会在 ~/.minicode/update 下建暂存目录,隔离到临时目录
	a, _ := newTestApp(t)
	a.prefs.Set(pAboutAuto, false)
	a.initUpdater(true)
	a.upd.CurrentVer = "0.0.1"
	a.upd.InstallFunc = nil
	a.upd.CheckOnly = true
	a.upd.Run(t.Context())
	if a.updSt.Phase != updFound || !a.updSt.Manual {
		t.Fatalf("检查后应停在手动「发现新版本」:%+v", a.updSt)
	}
	if a.upd.FoundVersion() == "" {
		t.Fatal("Runner 应记住已发现的版本")
	}
	a.upd.Download(t.Context())
	if a.updSt.Phase != updReady {
		t.Fatalf("下载完应进入「可重启更新」:%+v", a.updSt)
	}
	if a.upd.FoundVersion() != "" {
		t.Fatal("下载开始后应清掉已发现的版本")
	}
}
