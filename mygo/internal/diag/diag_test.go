package diag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 每个测试用独立的数据目录,并把日志接到该目录。
func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MINI_HOME", dir)
	SetLevel(LevelDebug)
	openLog(LogPath())
	ResetStats()
	return dir
}

func readLog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(LogPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestLogLineFormat(t *testing.T) {
	setup(t)
	L("demo").Info("你好", "k", "a b", "n", 3)
	got := readLog(t)
	if !strings.Contains(got, `INFO  [demo] 你好 k="a b" n=3`) {
		t.Fatalf("日志格式不对: %q", got)
	}
}

func TestLevelFilter(t *testing.T) {
	setup(t)
	SetLevel(LevelWarn)
	L("x").Info("不该出现")
	L("x").Warn("应当出现")
	got := readLog(t)
	if strings.Contains(got, "不该出现") || !strings.Contains(got, "应当出现") {
		t.Fatalf("级别过滤失效: %q", got)
	}
}

func TestBadKey(t *testing.T) {
	if got := kvString([]any{"a", 1, "orphan"}); got != "a=1 !BADKEY=orphan" {
		t.Fatal(got)
	}
}

func TestRotation(t *testing.T) {
	dir := t.TempDir()
	r, err := openRotator(filepath.Join(dir, "t.log"))
	if err != nil {
		t.Fatal(err)
	}
	chunk := []byte(strings.Repeat("x", 1<<20) + "\n")
	for i := 0; i < 40; i++ {
		_, _ = r.Write(chunk)
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != LogFiles {
		t.Fatalf("应保留 %d 份,实际 %d", LogFiles, len(ents))
	}
}

func TestSpanSlowWarns(t *testing.T) {
	setup(t)
	end := SpanSlow("perf", "sleepy", 5*time.Millisecond, "id", 1)
	time.Sleep(15 * time.Millisecond)
	end()
	if !strings.Contains(readLog(t), "WARN  [perf] 慢操作 sleepy") {
		t.Fatal("慢操作没有记 WARN")
	}
	var found bool
	for _, s := range Stats() {
		if s.Name == "perf.sleepy" && s.Count == 1 && s.Slow == 1 && s.P95() >= 15*time.Millisecond {
			found = true
		}
	}
	if !found {
		t.Fatalf("统计不对: %+v", Stats())
	}
}

func TestCrumbsRing(t *testing.T) {
	for i := 0; i < 100; i++ {
		Crumb("step %d", i)
	}
	c := Crumbs()
	if len(c) != 64 || !strings.HasSuffix(c[63], "step 99") {
		t.Fatalf("面包屑环不对: %d %v", len(c), c[len(c)-1])
	}
}

func TestRecoverWritesCrash(t *testing.T) {
	setup(t)
	func() {
		defer Recover("test.where")
		panic("炸了")
	}()
	ents, _ := os.ReadDir(CrashDir())
	if len(ents) != 1 || !strings.Contains(ents[0].Name(), "panic") {
		t.Fatalf("没有崩溃记录: %v", ents)
	}
	b, _ := os.ReadFile(filepath.Join(CrashDir(), ents[0].Name()))
	if !strings.Contains(string(b), "炸了") || !strings.Contains(string(b), "test.where") || !strings.Contains(string(b), "goroutine") {
		t.Fatalf("记录内容不全: %s", b)
	}
}

func TestGuard(t *testing.T) {
	setup(t)
	if Guard("g", func() { panic("x") }) {
		t.Fatal("应返回 false")
	}
	if !Guard("g", func() {}) {
		t.Fatal("应返回 true")
	}
}

func TestStaleMarker(t *testing.T) {
	setup(t)
	_ = os.MkdirAll(RunDir(), 0o755)
	// 一个不存在的 pid(足够大)
	_ = os.WriteFile(filepath.Join(RunDir(), "99999999.json"), []byte(`{"pid":99999999,"started":"s","heartbeat":"h"}`), 0o644)
	if n := ScanStale(); n != 1 {
		t.Fatalf("应补记 1 条,实际 %d", n)
	}
	ents, _ := os.ReadDir(CrashDir())
	if len(ents) != 1 || !strings.Contains(ents[0].Name(), "unclean-exit") {
		t.Fatalf("记录不对: %v", ents)
	}
	if n := ScanStale(); n != 0 {
		t.Fatal("标记应已清除")
	}
}

func TestCleanExitRemovesMarker(t *testing.T) {
	setup(t)
	writeMarker(time.Now())
	if _, err := os.Stat(markerPath(os.Getpid())); err != nil {
		t.Fatal(err)
	}
	CleanExit("test")
	if _, err := os.Stat(markerPath(os.Getpid())); !os.IsNotExist(err) {
		t.Fatal("标记应被删除")
	}
}

func TestParseFrameLine(t *testing.T) {
	setup(t)
	line := "mygo: frame 412 took 40.3 ms: build 3.1 (2 passes), layout 1.2, paint 0.8, present 35.0"
	if !ParseFrameLine(line) {
		t.Fatal("应能解析")
	}
	if ParseFrameLine("something else") {
		t.Fatal("不该解析")
	}
	if !strings.Contains(readLog(t), "慢帧") {
		t.Fatal("40ms 的帧应记慢帧 WARN")
	}
}

func TestWatchdogDetectsStall(t *testing.T) {
	setup(t)
	release := make(chan struct{})
	first := true
	Watchdog(func(done func()) {
		if first {
			first = false
			<-release // 模拟主线程卡住
		}
		done()
	}, 200*time.Millisecond, 400*time.Millisecond)
	time.Sleep(1300 * time.Millisecond)
	close(release)
	time.Sleep(300 * time.Millisecond)
	log := readLog(t)
	if !strings.Contains(log, "主线程停滞") || !strings.Contains(log, "已保存全部 goroutine 栈") || !strings.Contains(log, "主线程恢复响应") {
		t.Fatalf("看门狗日志不全:\n%s", log)
	}
}
