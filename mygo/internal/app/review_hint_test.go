package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 审查页:文件标题里的目录要相对项目根(不露绝对前缀);改动已提交后底部说明要如实写。
func TestReviewDispDirAndHint(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/work/proj"
	if got := a.fileDispDir("/work/proj/main.go"); got != "" {
		t.Errorf("根目录下的文件不该显示目录:%q", got)
	}
	if got := a.fileDispDir("/work/proj/pkg/sub/x.go"); got != "pkg/sub" {
		t.Errorf("应为相对目录:%q", got)
	}
	// 没有仓库 / 有未提交改动:仍是「还没有提交」
	if got := a.reviewHintText(); !strings.Contains(got, "还没有提交") {
		t.Errorf("无仓库时文案不对:%q", got)
	}
	a.repo = vcs.NewMem()
	a.sc.log = []vcs.Commit{{Hash: "abc123"}}
	a.sc.changes = []vcs.Change{{Path: "main.go"}}
	if got := a.reviewHintText(); !strings.Contains(got, "还没有提交") {
		t.Errorf("有未提交改动时文案不对:%q", got)
	}
	a.sc.changes = nil
	if got := a.reviewHintText(); !strings.Contains(got, "已经提交") {
		t.Errorf("工作区干净时应写已提交:%q", got)
	}
}
