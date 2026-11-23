package claude

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// usageBody 造一份会话记录:流式的同 id 多行、无 id、<synthetic>、非助手行、未知模型都有。
func usageBody(n int) string {
	var b strings.Builder
	t0 := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		ts := t0.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano)
		fmt.Fprintf(&b, `{"type":"user","timestamp":"%s","message":{"content":"第 %d 问"}}`+"\n", ts, i)
		id := fmt.Sprintf("m%d", i)
		if i%9 == 4 {
			id = "" // 没有 message.id:每行各算一条
		}
		model := "claude-sonnet-5-5"
		switch i % 7 {
		case 3:
			model = "<synthetic>"
		case 5:
			model = "glm-4"
		}
		for k := 1; k <= 1+i%3; k++ { // 流式:同一条消息写 1~3 行,后出现的覆盖先出现的
			fmt.Fprintf(&b, `{"type":"assistant","timestamp":"%s","cwd":"/w/p%d","message":{"id":"%s","model":"%s","usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d}}}`+"\n",
				ts, i%3, id, model, 100*k+i, 10*k, 1000*k)
		}
	}
	return b.String()
}

// fullScan 用全新的扫描器整份重读,作为参照。
func fullScan(dir string) []UsageRec {
	var fresh UsageScanner
	return fresh.Scan(dir)
}

func mustEqual(t *testing.T, step string, got, want []UsageRec) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s:增量结果与整份重读不一致(%d 条 vs %d 条)", step, len(got), len(want))
	}
}

// 流式写入:按随机位置切块追加(常常把一行从中间切开),每追加一次,增量扫描都必须与整份重读完全一致。
func TestUsageIncrementalMatchesFullRead(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "-w-p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(proj, "s1.jsonl")
	other := filepath.Join(proj, "s2.jsonl") // 另一个不变的文件
	if err := os.WriteFile(other, []byte(usageBody(5)), 0o644); err != nil {
		t.Fatal(err)
	}
	body := usageBody(60)
	rng := rand.New(rand.NewSource(7))
	var sc UsageScanner
	fh, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	for pos, step := 0, 0; pos < len(body); step++ {
		n := 1 + rng.Intn(400)
		if pos+n > len(body) {
			n = len(body) - pos
		}
		if _, err := fh.WriteString(body[pos : pos+n]); err != nil {
			t.Fatal(err)
		}
		pos += n
		got, st := sc.ScanStats(dir)
		mustEqual(t, fmt.Sprintf("第 %d 次追加(到 %d 字节)", step, pos), got, fullScan(dir))
		if step > 0 && st.Full != 0 {
			t.Fatalf("第 %d 次追加不该整份重读:%+v", step, st)
		}
		if st.BytesRead > int64(n)+4096 { // 只读新增部分(外加最多一个半行)
			t.Fatalf("第 %d 次追加读了 %d 字节,新增只有 %d", step, st.BytesRead, n)
		}
	}
	// 结尾没有换行的最后一行:与整份重读一致(整份重读同样会解析它)
	if _, err := fh.WriteString(`{"type":"assistant","timestamp":"2026-11-21T00:00:00Z","cwd":"/w","message":{"id":"last","model":"claude-opus-5-5","usage":{"input_tokens":7,"output_tokens":1}}}`); err != nil {
		t.Fatal(err)
	}
	got, _ := sc.ScanStats(dir)
	mustEqual(t, "结尾半行", got, fullScan(dir))
	// 什么都没变:直接返回上次的结果,不重新合并
	again, st := sc.ScanStats(dir)
	if st.Changed || st.BytesRead != 0 || st.Unchanged != 2 {
		t.Fatalf("没有变化时应全部命中缓存:%+v", st)
	}
	if &again[0] != &got[0] {
		t.Fatal("没有变化时应返回同一个切片")
	}
}

// 文件被改写 / 截断 / 替换 / 删除:都要退回整份重读(或移除),结果与参照一致。
func TestUsageIncrementalFallbacks(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "-w-p")
	_ = os.MkdirAll(proj, 0o755)
	path := filepath.Join(proj, "s1.jsonl")
	body := usageBody(20)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var sc UsageScanner
	sc.Scan(dir)

	// 同一个文件被截断后又继续写到比原来还长(类似回退会话后继续对话):
	// 原来读到的位置上已是别的内容,尾部核对不上 → 整份重读
	// 改的是用量本身(token 数变了、字节布局也变了):若误走增量,结果会和整份重读不一致
	rewritten := strings.Replace(usageBody(30), `"cache_read_input_tokens":`, `"cache_read_input_tokens":9`, -1)
	if err := os.WriteFile(path, []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}
	got, st := sc.ScanStats(dir)
	mustEqual(t, "截断后又继续写", got, fullScan(dir))
	if st.Full != 1 {
		t.Fatalf("内容被改写应整份重读:%+v", st)
	}

	// 截断变短:整份重读
	if err := os.WriteFile(path, []byte(usageBody(3)), 0o644); err != nil {
		t.Fatal(err)
	}
	got, st = sc.ScanStats(dir)
	mustEqual(t, "截断", got, fullScan(dir))
	if st.Full != 1 {
		t.Fatalf("变短应整份重读:%+v", st)
	}

	// 换成另一个文件(先写临时文件再改名,原子替换):整份重读
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(usageBody(3)+usageBody(4)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
	got, st = sc.ScanStats(dir)
	mustEqual(t, "原子替换", got, fullScan(dir))
	if st.Full != 1 {
		t.Fatalf("文件被替换应整份重读:%+v", st)
	}

	// 删除:记录随之消失
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got, st := sc.ScanStats(dir); len(got) != 0 || !st.Changed {
		t.Fatalf("文件删除后不应再有记录:%d 条 %+v", len(got), st)
	}
}
