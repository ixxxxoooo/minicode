package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/update"
)

// 检查更新阶段的各类失败要翻成「标题 + 一句原因 + 下一步」,不能一律是红色「下载失败」。
func TestFailedUpdState(t *testing.T) {
	cases := []struct {
		p     update.Progress
		phase updPhase
		title string
	}{
		{update.Progress{Err: `Get "x": context canceled`, ErrStage: "check"}, updNetErr, ""},
		{update.Progress{Err: "appcast HTTP 404", ErrStage: "check"}, updNetErr, "检查失败"},
		{update.Progress{Err: "dial tcp: lookup github.com: no such host", ErrStage: "check"}, updNetErr, ""},
		{update.Progress{Err: "校验和不一致", ErrStage: "verify"}, updFail, "校验失败"},
		{update.Progress{Err: "unexpected EOF", ErrStage: "download"}, updFail, ""},
		{update.Progress{Err: "解包失败: x", ErrStage: "unpack"}, updFail, "安装准备失败"},
	}
	for _, c := range cases {
		st := failedUpdState(c.p)
		if st.Phase != c.phase || st.Title != c.title || st.Msg == "" {
			t.Errorf("%+v → %+v,期望阶段 %v 标题 %q 且有原因", c.p, st, c.phase, c.title)
		}
	}
}

// Runner.Run 是同步的:返回时检查已经结束,调用方 defer cancel() 不会把检查提前取消。
func TestRunnerRunIsSynchronous(t *testing.T) {
	var got []update.Progress
	r := &update.Runner{
		Fetch:      update.DemoFetcher{Version: "9.9.9"},
		CurrentVer: "9.9.9",
		OnProgress: func(p update.Progress) { got = append(got, p) },
	}
	r.Run(t.Context())
	if len(got) == 0 || !got[len(got)-1].UpToDate {
		t.Fatalf("同步返回时应已走到「已是最新」,实际 %+v", got)
	}
}

// 检查更新刚开始时界面应显示「检查更新…」,而不是「发现新版本」。
func TestRunnerRunStartsAsChecking(t *testing.T) {
	var first *update.Progress
	r := &update.Runner{
		Fetch:      update.DemoFetcher{Version: "9.9.9"},
		CurrentVer: "9.9.9",
		OnProgress: func(p update.Progress) {
			if first == nil {
				first = &p
			}
		},
	}
	r.Run(t.Context())
	if first == nil || first.Phase != update.PhaseNone {
		t.Fatalf("开头应发 PhaseNone(检查中),实际 %+v", first)
	}
	a := &App{}
	a.onUpdateProgress(*first)
	if a.updSt.Phase != updChecking {
		t.Fatalf("界面应进入「检查中」,实际阶段 %v", a.updSt.Phase)
	}
}
