package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUsageScanAndReport(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "-tmp-demo")
	_ = os.MkdirAll(proj, 0o755)
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ts := now.Add(-time.Hour).Format(time.RFC3339Nano)
	old := now.AddDate(0, 0, -3).Format(time.RFC3339Nano)
	body := `{"type":"user","message":{"content":"hi"}}
{"type":"assistant","timestamp":"` + ts + `","cwd":"/tmp/demo","message":{"id":"m1","model":"claude-sonnet-5-5","usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","timestamp":"` + ts + `","cwd":"/tmp/demo","message":{"id":"m1","model":"claude-sonnet-5-5","usage":{"input_tokens":1000000,"output_tokens":100000,"cache_read_input_tokens":3000000}}}
{"type":"assistant","timestamp":"` + old + `","cwd":"/tmp/demo","message":{"id":"m2","model":"glm-4","usage":{"input_tokens":50,"output_tokens":5}}}
`
	if err := os.WriteFile(filepath.Join(proj, "s1.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var sc UsageScanner
	recs := sc.Scan(dir)
	if len(recs) != 2 {
		t.Fatalf("同一 message.id 的多行应合并为一条:%d 条", len(recs))
	}
	rep := BuildReport(recs, 0, now, func(id string) (string, string) { return "示例会话", "/tmp/demo" })
	// sonnet:1M×3 + 0.1M×15 + 3M×0.3 = 3 + 1.5 + 0.9 = 5.4
	if d := rep.Total - 5.4; d > 1e-6 || d < -1e-6 {
		t.Fatalf("费用估算应为 5.4,得到 %v", rep.Total)
	}
	if len(rep.Days) != 7 || rep.Sessions != 1 || len(rep.TopSessions) != 1 || rep.TopSessions[0].Title != "示例会话" {
		t.Fatalf("报告不对:%+v", rep)
	}
	if rep.CacheHit != 75 { // 3M 命中 / (1M + 3M) 输入
		t.Fatalf("缓存命中率应为 75%%:%d", rep.CacheHit)
	}
	if len(rep.Models) < 1 || rep.Models[0].Name != "Sonnet 5.5" {
		t.Fatalf("模型名应写成「Sonnet 5.5」:%+v", rep.Models)
	}
	if modelLabel("claude-opus-4-1-20250805") != "Opus 4.1" {
		t.Fatalf("日期后缀应去掉:%s", modelLabel("claude-opus-4-1-20250805"))
	}
	// 文件没变:缓存命中,结果一致
	if again := sc.Scan(dir); len(again) != 2 {
		t.Fatal("缓存后结果应一致")
	}
}
