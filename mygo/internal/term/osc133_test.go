package term

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

// 真实 zsh + 集成脚本:跑 `false` 和 `echo hi`,应各记一个命令块,成败正确,
// 且 StartRow 指向「命令输出的第一行」(命令行的下一行)。
func TestOsc133Blocks(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("没有 zsh")
	}
	home := t.TempDir()
	tm, err := Start(Options{Shell: "/bin/zsh", Cwd: home, Cols: 60, Rows: 12, Env: []string{"ZDOTDIR=" + home, "HOME=" + home}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tm.Close)
	tm.SendText(IntegrationCmd("zsh"))
	time.Sleep(600 * time.Millisecond)
	run := func(cmd, wantOut string) {
		tm.SendText(cmd)
		tm.SendKey(uv.KeyEnter, "", 0)
		if wantOut != "" {
			waitFor(t, tm, wantOut)
		}
		time.Sleep(500 * time.Millisecond)
	}
	run("false", "")
	run("echo hello-osc", "hello-osc")
	bl := tm.CommandBlocks()
	if len(bl) < 2 {
		t.Fatalf("应有 2 个命令块,实际 %d:%+v\n画面:\n%s", len(bl), bl, text(tm.Snapshot(100)))
	}
	if bl[0].ExitOK {
		t.Errorf("false 应记为失败:%+v", bl[0])
	}
	if !bl[1].ExitOK {
		t.Errorf("echo 应记为成功:%+v", bl[1])
	}
	// 空回车也会触发 precmd:不能改写上一个已结束的块
	run("", "")
	if bl2 := tm.CommandBlocks(); len(bl2) != len(bl) || bl2[1] != bl[1] {
		t.Errorf("空回车改写了命令块:%+v → %+v", bl, bl2)
	}
	s := tm.Snapshot(100)
	row := bl[1].StartRow - s.LineBase
	if row < 0 || row >= len(s.Lines) {
		t.Fatalf("StartRow %d 超出画面(LineBase %d,共 %d 行)", bl[1].StartRow, s.LineBase, len(s.Lines))
	}
	var sb strings.Builder
	for _, r := range s.Lines[row] {
		sb.WriteString(r.Text)
	}
	if !strings.HasPrefix(sb.String(), "hello-osc") {
		t.Errorf("命令块起始行应是输出第一行,实际 %q\n画面:\n%s", sb.String(), text(s))
	}
}
