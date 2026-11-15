// Package term 是内置终端的内核(与界面无关):用伪终端(PTY)起用户的 shell,输出交给
// VT 仿真器(charmbracelet/x/vt)解析成字符网格,界面只读「快照」、把按键编码后写回。
//
// 并发模型:读 PTY 的 goroutine 把字节喂给仿真器并通知界面重绘(合并成约 60 fps);
// 仿真器自己产生的应答(光标位置报告、按键编码)由另一个 goroutine 写回 PTY;
// 界面线程只调 Snapshot / Resize / 输入方法。快照按「画面版本号」缓存,没变化就不重建。
package term

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("term")

// ScrollbackLines 是保留的历史行数。
const ScrollbackLines = 5000

// Options 是起一个终端的参数。
type Options struct {
	Shell      string   // 空 = $SHELL,再退回 /bin/zsh
	Cwd        string   // 工作目录
	Cols, Rows int      // 初始大小(字符数)
	Env        []string // 额外环境变量(KEY=VALUE)
	Scrollback int      // 历史行数;0 = ScrollbackLines
	LoginShell bool     // 以登录 shell 启动(-l),默认 true
	Notify     func()   // 画面变化 / 进程退出时调用(任意 goroutine,已合并节流)
}

// Term 是一个运行中(或已退出)的终端。
type Term struct {
	em   *vt.SafeEmulator
	ptmx *os.File
	cmd  *exec.Cmd

	notify  func()
	pending atomic.Bool // 已有一次待发的通知
	ver     atomic.Uint64

	mu       sync.Mutex
	cols     int
	rows     int
	title    string
	exited   bool
	exitText string
	closed   bool
	cache    *Screen
	cacheVer uint64
	cacheSB  int

	mouseModes  atomic.Uint32 // 鼠标上报模式是否开启(见 mouse.go)
	cmdRunning  atomic.Bool   // OSC 133:命令是否在运行
	cmdBlocks   []CmdBlock    // 命令块装饰(行号 + 成败)
	blockScroll int           // 上次记块时的回看行数,变小说明被清掉了
}

// Start 起 shell。登录 shell(-l)才有完整的 PATH(从访达启动的应用只继承很短的环境)。
func Start(o Options) (*Term, error) {
	if o.Cols < 2 {
		o.Cols = 80
	}
	if o.Rows < 1 {
		o.Rows = 24
	}
	shell := o.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/zsh"
	}
	args := []string{}
	if o.LoginShell {
		args = append(args, "-l")
	}
	cmd := exec.Command(shell, args...)
	if o.Cwd != "" {
		cmd.Dir = o.Cwd
	}
	cmd.Env = environ(o.Env)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(o.Cols), Rows: uint16(o.Rows)})
	if err != nil {
		return nil, fmt.Errorf("启动终端失败:%w", err)
	}
	scroll := o.Scrollback
	if scroll <= 0 {
		scroll = ScrollbackLines
	}
	t := &Term{em: vt.NewSafeEmulator(o.Cols, o.Rows), ptmx: ptmx, cmd: cmd, notify: o.Notify, cols: o.Cols, rows: o.Rows}
	t.em.SetScrollbackSize(scroll)
	cb := vt.Callbacks{Title: func(s string) {
		t.mu.Lock()
		t.title = s
		t.mu.Unlock()
	}}
	cb.EnableMode = func(m ansi.Mode) { t.onMouseMode(m, true) }
	cb.DisableMode = func(m ansi.Mode) { t.onMouseMode(m, false) }
	t.em.SetCallbacks(cb)
	t.installShellIntegration(shell)
	log.Info("终端已启动", "shell", shell, "cwd", o.Cwd, "pid", cmd.Process.Pid, "size", fmt.Sprintf("%dx%d", o.Cols, o.Rows))

	go t.readLoop()
	go t.replyLoop()
	go t.waitLoop()
	return t, nil
}

// environ 给子进程的环境:继承当前环境,补上终端标识与 UTF-8 语言,再叠加额外变量。
func environ(extra []string) []string {
	env := os.Environ()
	set := func(k, v string) {
		for i, e := range env {
			if strings.HasPrefix(e, k+"=") {
				env[i] = k + "=" + v
				return
			}
		}
		env = append(env, k+"="+v)
	}
	has := func(k string) bool {
		for _, e := range env {
			if strings.HasPrefix(e, k+"=") {
				return true
			}
		}
		return false
	}
	set("TERM", "xterm-256color")
	set("COLORTERM", "truecolor")
	set("TERM_PROGRAM", "MiniCode")
	if !has("LANG") && !has("LC_ALL") {
		set("LANG", "en_US.UTF-8")
	}
	for _, kv := range extra {
		if k, v, ok := strings.Cut(kv, "="); ok {
			set(k, v)
		}
	}
	return env
}

// readLoop 把 shell 的输出喂给仿真器。
func (t *Term) readLoop() {
	defer diag.Recover("term.read")
	buf := make([]byte, 32*1024)
	for {
		n, err := t.ptmx.Read(buf)
		if n > 0 {
			_, _ = t.em.Write(buf[:n])
			t.ver.Add(1)
			t.poke()
		}
		if err != nil {
			return // 子进程退出或终端已关(EIO / 已关闭)
		}
	}
}

// replyLoop 把仿真器要发给 shell 的字节(按键编码、光标位置报告等应答)写回 PTY。
func (t *Term) replyLoop() {
	defer diag.Recover("term.reply")
	buf := make([]byte, 4096)
	for {
		n, err := t.em.Read(buf)
		if n > 0 {
			if _, werr := t.ptmx.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// waitLoop 等 shell 退出并记下结果。
func (t *Term) waitLoop() {
	defer diag.Recover("term.wait")
	err := t.cmd.Wait()
	t.mu.Lock()
	t.exited = true
	switch {
	case t.closed:
		t.exitText = ""
	case err == nil:
		t.exitText = "进程已结束(退出码 0)"
	default:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.exitText = fmt.Sprintf("进程已结束(%s)", ee.ProcessState.String())
		} else {
			t.exitText = "进程已结束:" + err.Error()
		}
	}
	text := t.exitText
	t.mu.Unlock()
	log.Info("终端进程退出", "result", text)
	t.ver.Add(1)
	t.poke()
}

// poke 通知界面重绘;16 ms 内的多次变化合并成一次。
func (t *Term) poke() {
	if t.notify == nil || !t.pending.CompareAndSwap(false, true) {
		return
	}
	time.AfterFunc(16*time.Millisecond, func() {
		t.pending.Store(false)
		t.notify()
	})
}

// Exited 进程是否已结束,以及结束说明。
func (t *Term) Exited() (bool, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exited, t.exitText
}

// Title 是程序用 OSC 设置的标题(shell 通常设成当前目录或命令)。
func (t *Term) Title() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.title
}

// Size 当前字符网格大小。
func (t *Term) Size() (cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cols, t.rows
}

// Resize 改大小:通知内核(shell 收到 SIGWINCH)并重排仿真器画面。
func (t *Term) Resize(cols, rows int) {
	if cols < 2 || rows < 1 {
		return
	}
	t.mu.Lock()
	if t.closed || (cols == t.cols && rows == t.rows) {
		t.mu.Unlock()
		return
	}
	t.cols, t.rows = cols, rows
	t.mu.Unlock()
	_ = pty.Setsize(t.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	t.em.Resize(cols, rows)
	t.ver.Add(1)
}

// SendKey 发一个按键(仿真器按当前模式编码:应用光标键、修饰键等)。
func (t *Term) SendKey(code rune, text string, mod uv.KeyMod) {
	t.em.SendKey(uv.KeyPressEvent{Code: code, Text: text, Mod: mod})
}

// SendText 发一段文字(输入法提交、普通字符)。
func (t *Term) SendText(s string) { t.em.SendText(s) }

// Paste 粘贴:程序开了括号粘贴模式时会加首尾标记。
func (t *Term) Paste(s string) { t.em.Paste(s) }

// Interrupt 向前台进程发 Ctrl-C。
func (t *Term) Interrupt() { t.SendKey('c', "", uv.ModCtrl) }

// Close 结束 shell(整个进程组,免得留下孤儿进程)并释放资源。可重复调用。
func (t *Term) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	if t.cmd.Process != nil {
		pid := t.cmd.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGHUP)
		time.AfterFunc(800*time.Millisecond, func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
	}
	_ = t.ptmx.Close()
	// 不调仿真器的 Close(它改 closed 字段,与应答 goroutine 里读它构成数据竞争):
	// 直接关输入管道,应答 goroutine 读到 EOF 就退出。
	if pw, ok := t.em.InputPipe().(*io.PipeWriter); ok {
		_ = pw.CloseWithError(io.EOF)
	}
	log.Info("终端已关闭")
}

// ---- 快照 ----

// Color 是一个终端颜色。Kind:0 默认色(界面用主题色)/ 1 调色板索引(0-15 用主题的 ANSI 色)/ 2 真彩色。
type Color struct {
	Kind    uint8
	Idx     uint8
	R, G, B uint8
}

// Run 是一行里样式相同的一段文字。
type Run struct {
	Text      string
	Fg, Bg    Color
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
	Reverse   bool
	Strike    bool
}

// Line 是一行(尾部空白已去掉)。
type Line []Run

// Screen 是某一刻的画面:历史行 + 当前屏幕。
type Screen struct {
	Lines        []Line
	Cols, Rows   int
	LineBase     int // Lines[0] 在全局历史里的绝对行号
	ScreenTop    int // Lines 里当前屏幕第一行的下标(之前是历史)
	CursorX      int
	CursorY      int // 相对 Lines 的行下标
	CursorHidden bool
	Alt          bool // 备用屏幕(vim / less / htop 等全屏程序)
	Ver          uint64
}

// Snapshot 取画面;history 是最多带上的历史行数(备用屏幕下不带)。画面没变且参数相同时复用上次的结果。
func (t *Term) Snapshot(history int) *Screen {
	ver := t.ver.Load()
	t.mu.Lock()
	if t.cache != nil && t.cacheVer == ver && t.cacheSB == history {
		s := t.cache
		t.mu.Unlock()
		return s
	}
	cols, rows := t.cols, t.rows
	t.mu.Unlock()

	s := &Screen{Cols: cols, Rows: rows, Ver: ver, Alt: t.em.IsAltScreen()}
	if !s.Alt && history > 0 {
		n := t.em.ScrollbackLen()
		from := max(0, n-history)
		s.LineBase = from
		for y := from; y < n; y++ {
			s.Lines = append(s.Lines, t.line(cols, func(x int) *uv.Cell { return t.em.ScrollbackCellAt(x, y) }))
		}
	}
	s.ScreenTop = len(s.Lines)
	for y := 0; y < rows; y++ {
		y := y
		s.Lines = append(s.Lines, t.line(cols, func(x int) *uv.Cell { return t.em.CellAt(x, y) }))
	}
	pos := t.em.CursorPosition()
	s.CursorX, s.CursorY = pos.X, s.ScreenTop+pos.Y
	s.CursorHidden = t.em.CursorHidden()

	t.mu.Lock()
	t.cache, t.cacheVer, t.cacheSB = s, ver, history
	t.mu.Unlock()
	return s
}

// line 把一行格子合并成样式段;尾部的默认样式空格去掉(但光标所在列由界面另画)。
func (t *Term) line(cols int, at func(x int) *uv.Cell) Line {
	var out Line
	var cur Run
	var sb strings.Builder
	flush := func() {
		if sb.Len() > 0 {
			cur.Text = sb.String()
			out = append(out, cur)
			sb.Reset()
		}
	}
	for x := 0; x < cols; x++ {
		c := at(x)
		var r Run
		text := " "
		if c != nil {
			if c.Width == 0 && c.Content == "" {
				continue // 宽字符的后半格
			}
			if c.Content != "" {
				text = c.Content
			}
			r = runOf(c.Style)
		}
		if sb.Len() > 0 && !sameStyle(cur, r) {
			flush()
		}
		cur = r
		sb.WriteString(text)
	}
	flush()
	// 去掉尾部只有空格的默认样式段
	for len(out) > 0 {
		last := &out[len(out)-1]
		if last.Bg.Kind != 0 || last.Reverse || last.Underline {
			break
		}
		trimmed := strings.TrimRight(last.Text, " ")
		if trimmed == "" {
			out = out[:len(out)-1]
			continue
		}
		last.Text = trimmed
		break
	}
	return out
}

func sameStyle(a, b Run) bool {
	a.Text, b.Text = "", ""
	return a == b
}

func runOf(s uv.Style) Run {
	return Run{
		Fg: colorOf(s.Fg), Bg: colorOf(s.Bg),
		Bold: s.Attrs&uv.AttrBold != 0, Dim: s.Attrs&uv.AttrFaint != 0, Italic: s.Attrs&uv.AttrItalic != 0,
		Reverse: s.Attrs&uv.AttrReverse != 0, Strike: s.Attrs&uv.AttrStrikethrough != 0,
		Underline: s.Underline != uv.UnderlineNone,
	}
}

func colorOf(c color.Color) Color {
	switch v := c.(type) {
	case nil:
		return Color{}
	case ansi.BasicColor:
		return Color{Kind: 1, Idx: uint8(v)}
	case ansi.IndexedColor:
		return Color{Kind: 1, Idx: uint8(v)}
	}
	r, g, b, _ := c.RGBA()
	return Color{Kind: 2, R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
}
