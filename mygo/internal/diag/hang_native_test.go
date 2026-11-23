package diag

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 回归:卡死栈原先叫 hang-<时间>.txt,字典序排在所有「时间开头」的记录之后,清理时永远删不到它,
// 反而把真正的崩溃记录挤掉。现在所有记录都以时间开头,清理按时间保留最新的。
func TestPruneCrashesKeepsNewestAcrossKinds(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"20261120-101835-hang-54184.txt",
		"20261120-101836-02-hang-54184.json",
		"20261121-090000-hang-native-1.txt",
		"20261122-120000-01-unclean-exit-9.json",
		"20261123-140626-01-panic-18287.json",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pruneCrashes(dir, 2)
	ents, _ := os.ReadDir(dir)
	var got []string
	for _, e := range ents {
		got = append(got, e.Name())
	}
	if len(got) != 2 || got[0] != names[3] || got[1] != names[4] {
		t.Fatalf("应只留最新的两份,得到 %v", got)
	}
}

// 真的调一次 /usr/bin/sample(对测试进程自己),确认采样文件落盘、内容是调用图;5 分钟内第二次被限流。
func TestNativeSampleWritesCallGraph(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("只有 macOS 有 /usr/bin/sample")
	}
	if _, err := os.Stat("/usr/bin/sample"); err != nil {
		t.Skip("本机没有 /usr/bin/sample")
	}
	if os.Getenv("CI") != "" {
		t.Skip("CI 机器上的采样权限不确定:只在本机跑")
	}
	setup(t)
	lastNativeSample.Store(0)
	nativeSample(500 * time.Millisecond)
	nativeSample(500 * time.Millisecond) // 紧接着第二次:应被限流,不再起第二个 sample

	var files []string
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		files, _ = filepath.Glob(filepath.Join(CrashDir(), "*-hang-native-*.txt"))
		if len(files) > 0 && strings.Contains(readLog(t), "已采样主线程停滞时的原生调用栈") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if len(files) != 1 {
		t.Fatalf("应恰好生成 1 份原生采样(第二次被限流),得到 %v;日志:\n%s", files, readLog(t))
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Call graph") {
		t.Fatalf("采样文件里应有调用图,开头是:%.300s", b)
	}
}
