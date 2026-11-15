package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diff"
)

func TestHunkStageAndRevert(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	if a.repo == nil {
		t.Fatal("演示应有仓库")
	}
	a.openDiffKind(dkWork, "src/router.rs", true)
	tb := a.ide.tab(diffID(dkWork, "src/router.rs"))
	if tb == nil || tb.dv == nil || !tb.dv.Editable || tb.dk != dkWork {
		t.Fatalf("差异标签没打开 / 应可编辑:%+v", tb)
	}
	hunks := tb.dv.Hunks()
	if len(hunks) == 0 {
		t.Fatal("router.rs 相对暂存区应有改动块")
	}
	work := tb.dv.New.String()
	// 暂存第一块:暂存区的版本 = 旧文本 + 这一块
	a.stageHunk(tb, 0)
	idx, _ := a.repo.Index("src/router.rs")
	if idx != tb.dv.Old.String() || idx == "" || idx == work && len(hunks) > 1 {
		t.Fatalf("暂存这一块后,暂存区应是旧文本 + 该块;得到 %d 字节(工作区 %d)", len(idx), len(work))
	}
	if left := len(tb.dv.Hunks()); left != len(hunks)-1 {
		t.Fatalf("暂存一块后应少一块:%d → %d", len(hunks), left)
	}
	// 还原一块:工作区文件被改回去
	if left := tb.dv.Hunks(); len(left) > 0 {
		before, _ := a.ide.fs.Read("src/router.rs")
		a.revertDiffHunk(tb, 0)
		after, _ := a.ide.fs.Read("src/router.rs")
		if before == after {
			t.Fatal("还原一块后文件内容应变化")
		}
	}
	// 补丁文本
	if hs := tb.dv.Hunks(); len(hs) > 0 {
		if p := a.hunkPatch(tb, hs[0]); !strings.HasPrefix(p, "--- a/src/router.rs") || !strings.Contains(p, "@@") {
			t.Fatalf("补丁格式不对:%q", p)
		}
	}
}

func TestStagedDiffUnstageHunk(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	_ = a.repo.StageContent("src/router.rs", "pub struct Router;\n")
	a.openDiffKind(dkStaged, "src/router.rs", true)
	tb := a.ide.tab(diffID(dkStaged, "src/router.rs"))
	if tb == nil || tb.dv.Editable {
		t.Fatal("暂存区视图应只读")
	}
	if len(tb.dv.Hunks()) == 0 {
		t.Fatal("HEAD ↔ 暂存区应有改动块")
	}
	a.unstageHunk(tb, 0)
	if idx, _ := a.repo.Index("src/router.rs"); idx != tb.dv.New.String() {
		t.Fatal("取消暂存后暂存区应等于还原后的暂存区一侧")
	}
}

// 左右对比排行:删除与新增一一配对,多出的一侧留空,相同行两侧都有。
func TestSplitRows(t *testing.T) {
	ls := diff.Lines("a\nb\nc\n", "a\nB\nB2\nc\n", 1, 1)
	rows := splitRows(ls)
	if len(rows) != 4 {
		t.Fatalf("应排成 4 行:%d", len(rows))
	}
	if rows[0].l == nil || rows[0].r == nil || rows[0].l.Text != "a" {
		t.Fatalf("相同行两侧都有:%+v", rows[0])
	}
	if rows[1].l == nil || rows[1].l.Text != "b" || rows[1].r == nil || rows[1].r.Text != "B" {
		t.Fatalf("删除与新增配对:%+v", rows[1])
	}
	if rows[2].l != nil || rows[2].r == nil || rows[2].r.Text != "B2" {
		t.Fatalf("多出的新增左侧留空:%+v", rows[2])
	}
}

// 窄档临时强制统一,窗口变宽后回到用户选的左右;没点过的标签跟随偏好。
func TestDiffLayoutSurvivesNarrow(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.openDiffKind(dkHead, "src/router.rs", true)
	tb := a.ide.cur()
	if tb == nil || !tb.diff {
		t.Fatal("应打开了差异标签")
	}
	if a.diffLayout(tb, false) != editor.DiffSplit {
		t.Fatal("默认偏好是左右")
	}
	tb.lay = 1 // 用户选了左右
	if a.diffLayout(tb, true) != editor.DiffUnified {
		t.Fatal("窄档应强制统一")
	}
	if a.diffLayout(tb, false) != editor.DiffSplit || tb.lay != 1 {
		t.Fatal("变宽后应回到左右,选择不能被改写")
	}
	tb.lay = 0
	a.prefs.Set("scm.diff", 1)
	if a.diffLayout(tb, false) != editor.DiffUnified {
		t.Fatal("没点过的标签应跟随偏好(统一)")
	}
}

// 在别处暂存 / 取消暂存之后,已打开的差异标签要跟着刷新;工作区一侧正在编辑的内容不被覆盖。
func TestRefreshDiffTabsAfterStage(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.openDiffKind(dkWork, "src/router.rs", true)
	tb := a.ide.tab(diffID(dkWork, "src/router.rs"))
	if tb == nil {
		t.Fatal("差异标签没打开")
	}
	work := tb.dv.New.String()
	// 在标签之外把整份工作区内容暂存(等价于 SCM 里点「暂存」)
	if err := a.repo.StageContent("src/router.rs", work); err != nil {
		t.Fatal(err)
	}
	a.refreshDiffTabs()
	if len(tb.dv.Hunks()) != 0 {
		t.Fatalf("暂存全部后,暂存区 ↔ 工作区应没有差异块,实际 %d 块", len(tb.dv.Hunks()))
	}
	// 用户在工作区一侧改了内容:再刷新不能把它冲掉
	tb.dv.New.Reset(work + "// 我的修改\n")
	a.refreshDiffTabs()
	if !strings.HasSuffix(tb.dv.New.String(), "// 我的修改\n") {
		t.Fatal("正在编辑的工作区一侧被刷新覆盖了")
	}
}
