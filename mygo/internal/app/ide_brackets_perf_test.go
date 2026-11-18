package app

import (
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui/editor"
)

// localeJSON 是性能测试用的大 JSON 文件。本机路径不写进仓库:由环境变量 MINI_LOCALE_JSON 指定,没设置就跳过。
var localeJSON = os.Getenv("MINI_LOCALE_JSON")

func openLocaleBracketHL(t *testing.T) (*editor.Buffer, *bracketHL) {
	t.Helper()
	raw, err := os.ReadFile(localeJSON)
	if err != nil {
		t.Skip("本地语言文件不可用:", err)
	}
	if len(raw) < 100 {
		t.Skip("语言文件过短")
	}
	buf := editor.NewBuffer(string(raw))
	lang := wrapBracketLang(editor.LanguageFor("zh-Hans.json"))
	buf.SetLanguage(lang)
	base := editor.NewTokenHighlighter(lang.Tokenizer, buf.Snapshot())
	bh := &bracketHL{base: base, text: buf.Snapshot()}
	return buf, bh
}

// depthAtLegacy 复现优化前逻辑:基础 token 未 final 时不写入 depth,每行反复从头扫 — scanBracket 路径下近似 O(n²)。
func depthAtLegacy(h *bracketHL, i int) int {
	if i >= h.text.LineCount() {
		return 0
	}
	if i < len(h.depth) {
		return h.depth[i]
	}
	if len(h.depth) == 0 {
		h.depth = append(h.depth, 0)
	}
	d := h.depth[len(h.depth)-1]
	cacheable := true
	for j := len(h.depth) - 1; j < i; j++ {
		line := h.text.Line(j)
		toks, fin := h.base.Tokens(j, line, h.scratch[:0])
		h.scratch = toks[:0]
		_, d = splitBrackets(line, toks, d, nil, false)
		cacheable = cacheable && fin
		if cacheable {
			h.depth = append(h.depth, d)
		}
	}
	return d
}

// 模拟 matchBracket / scanBracket:对每一行取 Tokens(会经 depthAt 累计深度)。
func warmAllLineTokens(bh *bracketHL, n int) time.Duration {
	start := time.Now()
	for i := 0; i < n; i++ {
		line := bh.text.Line(i)
		bh.Tokens(i, line, nil)
	}
	return time.Since(start)
}

// TestBracketDepthLargeJSON 打开真实 locale JSON(约 1356 行),未定稿时连续扫行应在百毫秒级而非秒级。
func TestBracketDepthLargeJSON(t *testing.T) {
	buf, bh := openLocaleBracketHL(t)
	n := buf.LineCount()
	// 光标靠后:单次 depthAt 要扩到末尾
	line := buf.Line(n - 1)
	if _, ok := bh.depthAt(n - 1); !ok {
		t.Fatal("depthAt 应成功")
	}
	t0 := time.Now()
	bh.Tokens(n-1, line, nil)
	one := time.Since(t0)
	// 全文件逐行(模拟 scanBracket 最坏路径)
	_, bh2 := openLocaleBracketHL(t)
	all := warmAllLineTokens(bh2, n)
	const maxAll = 200 * time.Millisecond
	if all > maxAll {
		t.Fatalf("逐行 Tokens %d 行耗时 %v,超过阈值 %v", n, all, maxAll)
	}
	t.Logf("行数=%d 单行 depthAt+Tokens=%v 全文件逐行=%v", n, one, all)
}

func TestBracketDepthLegacyVsFixed(t *testing.T) {
	_, bhFixed := openLocaleBracketHL(t)
	n := min(bhFixed.text.LineCount(), 400) // 旧逻辑在更大文件上过慢,400 行足以看出数量级

	var legacy bracketHL
	legacy.base = bhFixed.base
	legacy.text = bhFixed.text

	t0 := time.Now()
	for i := 0; i < n; i++ {
		depthAtLegacy(&legacy, i)
	}
	legacyDur := time.Since(t0)

	_, bh := openLocaleBracketHL(t)
	t1 := time.Now()
	for i := 0; i < n; i++ {
		bh.depthAt(i)
	}
	fixedDur := time.Since(t1)

	if fixedDur*4 > legacyDur { // 新实现应明显更快(至少 4×)
		t.Fatalf("depthAt 未明显加速:legacy=%v fixed=%v (n=%d)", legacyDur, fixedDur, n)
	}
	t.Logf("depthAt 对比 n=%d legacy=%v fixed=%v 倍数≈%.1f", n, legacyDur, fixedDur, float64(legacyDur)/float64(max(fixedDur, time.Nanosecond)))
}

func BenchmarkBracketDepthLocaleJSON(b *testing.B) {
	raw, err := os.ReadFile(localeJSON)
	if err != nil {
		b.Skip(err)
	}
	buf := editor.NewBuffer(string(raw))
	lang := wrapBracketLang(editor.LanguageFor("zh-Hans.json"))
	buf.SetLanguage(lang)
	n := buf.LineCount()

	b.Run("fixed", func(b *testing.B) {
		for b.Loop() {
			base := editor.NewTokenHighlighter(lang.Tokenizer, buf.Snapshot())
			bh := &bracketHL{base: base, text: buf.Snapshot()}
			var total time.Duration
			for i := 0; i < n; i++ {
				t0 := time.Now()
				bh.Tokens(i, buf.Line(i), nil)
				total += time.Since(t0)
			}
			b.ReportMetric(float64(total)/float64(n)/float64(time.Microsecond), "us/line")
		}
	})

	b.Run("legacy_depth", func(b *testing.B) {
		for b.Loop() {
			base := editor.NewTokenHighlighter(lang.Tokenizer, buf.Snapshot())
			h := &bracketHL{base: base, text: buf.Snapshot()}
			var total time.Duration
			for i := 0; i < min(n, 200); i++ {
				t0 := time.Now()
				depthAtLegacy(h, i)
				total += time.Since(t0)
			}
			b.ReportMetric(float64(total)/float64(min(n, 200))/float64(time.Microsecond), "us/line")
		}
	})
}
