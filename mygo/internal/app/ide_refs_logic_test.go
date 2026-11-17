package app

import (
	"strings"
	"testing"
)

func TestRefFlattenFold(t *testing.T) {
	items := []refItem{
		{"a.go", 0, 1, "x"},
		{"a.go", 1, 2, "y"},
		{"b.go", 0, 0, "z"},
	}
	folded := map[string]bool{"a.go": true}
	rows := refFlatten(items, folded)
	if len(rows) != 3 {
		t.Fatalf("折叠 a.go 后应有 3 行(a 组头 + b 组头 + b 的 1 条命中),得 %d", len(rows))
	}
	if rows[0].kind != refRowFile || rows[1].path != "b.go" {
		t.Fatalf("行序不对:%+v", rows)
	}
}

func TestRefLooksLikeDef(t *testing.T) {
	if !refLooksLikeDef("fn normalize(path: &str) -> String {", "normalize") {
		t.Fatal("应识别 fn 定义行")
	}
	if refLooksLikeDef("let key = normalize(path);", "normalize") {
		t.Fatal("调用行不应标为定义")
	}
}

func TestRefIsTestFile(t *testing.T) {
	if !refIsTestFile("tests/router_test.rs") {
		t.Fatal("router_test.rs 应为测试")
	}
	if refIsTestFile("src/router.rs") {
		t.Fatal("router.rs 不是测试")
	}
}

func TestRefPreviewCentersHit(t *testing.T) {
	long := stringsRepeat("x", 120) + "normalize" + stringsRepeat("y", 120)
	prefix := stringsRepeat("x", 120)
	vis, lo, hi := refPreviewBytes(long, len(prefix), len(prefix)+len("normalize"), 60)
	if !strings.Contains(vis, "normalize") {
		t.Fatalf("预览应含命中词:%q", vis)
	}
	if lo < 0 || hi <= lo || !strings.Contains(vis[lo:hi], "normalize") {
		t.Fatalf("命中区间不对 lo=%d hi=%d vis=%q", lo, hi, vis)
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
