package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func TestTimelinePreviewReusesTab(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.ide.openFile("src/router.rs", true)
	tt.Frame()
	g := a.gitInfoFor("src/router.rs")
	tt.Frame()
	if g == nil || len(g.log) < 2 {
		t.Fatal("演示仓库应有文件历史")
	}
	a.openTimelineCommit(g.log[0], "src/router.rs", false)
	tt.Frame()
	tt.Frame()
	id := timelineDiffID("src/router.rs")
	if a.ide.active != id {
		t.Fatalf("应在时间线预览标签,得到 %q", a.ide.active)
	}
	if tb := a.ide.tab(id); tb == nil || !tb.preview || tb.revHash != g.log[0].Hash {
		t.Fatalf("预览标签状态不对:%+v", tb)
	}
	nTabs := len(a.ide.tabs)
	a.openTimelineCommit(g.log[1], "src/router.rs", false)
	tt.Frame()
	tt.Frame()
	if len(a.ide.tabs) != nTabs {
		t.Fatalf("预览对比应替换而非新开标签:%d → %d", nTabs, len(a.ide.tabs))
	}
	if tb := a.ide.tab(id); tb == nil || tb.revHash != g.log[1].Hash {
		t.Fatalf("应切换到第二条提交:%+v", tb)
	}
}

func TestTimelineFollowsDiffTab(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.ide.openFile("src/router.rs", true)
	tt.Frame()
	g := a.gitInfoFor("src/router.rs")
	tt.Frame()
	if g == nil || len(g.log) == 0 {
		t.Fatal("缺少历史")
	}
	cm := g.log[0]
	a.openTimelineCommit(cm, "src/router.rs", false)
	tt.Frame()
	tt.Frame()
	if a.timelineSubject() != "src/router.rs" {
		t.Fatalf("应对齐源文件,得到 %q", a.timelineSubject())
	}
	if a.timelinePickHash() != cm.Hash {
		t.Fatalf("应高亮当前提交 %s,得到 %q", cm.Short(), a.timelinePickHash())
	}
}

func TestTimelinePickHeadDiff(t *testing.T) {
	a, _ := demoIDE(t)
	a.openDiffKind(dkHead, "src/router.rs", false)
	if a.timelinePickHash() != "__local__" {
		t.Fatalf("HEAD 对比应高亮本地更改行,得到 %q", a.timelinePickHash())
	}
}

func TestSideSecHeightClamp(t *testing.T) {
	a, _ := demoIDE(t)
	h := a.sideSecHeight(lySideOutline, sideOutlineDef, 400)
	if h < sideSecMin || h > sideSecMax {
		t.Fatalf("高度应在范围内:%v", h)
	}
	hi := a.sideSecMax(200)
	if hi > sideSecMax {
		t.Fatalf("窄窗上限过大:%v", hi)
	}
}

func TestTimelineVsWorkLabel(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	var cm vcs.Commit
	for _, x := range a.sc.log {
		if x.Hash != "" {
			cm = x
			break
		}
	}
	a.openTimelineVsWork(cm, "src/router.rs", false)
	tt.Frame()
	tt.Frame()
	tb := a.ide.tab(timelineDiffID("src/router.rs"))
	if tb == nil || tb.label == "" {
		t.Fatal("应有与工作区对比标签")
	}
}
