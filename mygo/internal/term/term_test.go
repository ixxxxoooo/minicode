package term

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

func text(s *Screen) string {
	var b strings.Builder
	for _, l := range s.Lines {
		for _, r := range l {
			b.WriteString(r.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func waitFor(t *testing.T, tm *Term, want string) *Screen {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		s := tm.Snapshot(100)
		if strings.Contains(text(s), want) {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("等不到 %q,画面:\n%s", want, text(tm.Snapshot(100)))
	return nil
}

func start(t *testing.T) *Term {
	t.Helper()
	tm, err := Start(Options{Shell: "/bin/sh", Cwd: t.TempDir(), Cols: 60, Rows: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tm.Close)
	return tm
}

// 输入命令、读到输出、颜色与退出码都正确。
func TestTermRunsCommands(t *testing.T) {
	tm := start(t)
	tm.SendText("printf 'hel\\033[31mlo\\033[0m wor%s' ld")
	tm.SendKey(uv.KeyEnter, "", 0)
	s := waitFor(t, tm, "hello world")
	var red bool
	for _, l := range s.Lines {
		for _, r := range l {
			if r.Text == "lo" && r.Fg.Kind == 1 && r.Fg.Idx == 1 {
				red = true
			}
		}
	}
	if !red {
		t.Fatalf("没有红色的 lo 段:%+v", s.Lines)
	}

	tm.SendText("exit 3")
	tm.SendKey(uv.KeyEnter, "", 0)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if ex, msg := tm.Exited(); ex {
			if !strings.Contains(msg, "exit status 3") {
				t.Fatalf("退出说明 %q", msg)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("shell 没有退出")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// 工作目录、Ctrl-C 中断、改大小后 stty 能读到新尺寸。
func TestTermCwdResizeInterrupt(t *testing.T) {
	dir := t.TempDir()
	tm, err := Start(Options{Shell: "/bin/sh", Cwd: dir, Cols: 60, Rows: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tm.Close)
	tm.SendText("pwd -P")
	tm.SendKey(uv.KeyEnter, "", 0)
	waitFor(t, tm, strings.TrimPrefix(dir, "/private")[len(dir)-12:]) // 目录名末尾(/var 与 /private/var 都能匹配)

	tm.Resize(100, 20)
	tm.SendText("stty size")
	tm.SendKey(uv.KeyEnter, "", 0)
	waitFor(t, tm, "20 100")
	if c, r := tm.Size(); c != 100 || r != 20 {
		t.Fatalf("大小 %d×%d", c, r)
	}

	tm.SendText("sleep 30")
	tm.SendKey(uv.KeyEnter, "", 0)
	time.Sleep(300 * time.Millisecond)
	tm.Interrupt()
	tm.SendText("echo after-int")
	tm.SendKey(uv.KeyEnter, "", 0)
	waitFor(t, tm, "after-int")
}

// 关闭后进程组被结束,可重复关闭。
func TestTermClose(t *testing.T) {
	tm := start(t)
	tm.SendText("sleep 60")
	tm.SendKey(uv.KeyEnter, "", 0)
	time.Sleep(200 * time.Millisecond)
	tm.Close()
	tm.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if ex, _ := tm.Exited(); ex {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("关闭后进程没有退出")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// 画面没变化时快照复用同一个对象。
func TestSnapshotCached(t *testing.T) {
	tm := start(t)
	time.Sleep(300 * time.Millisecond)
	a := tm.Snapshot(50)
	b := tm.Snapshot(50)
	if a != b {
		t.Fatal("画面没变,快照应复用")
	}
}
