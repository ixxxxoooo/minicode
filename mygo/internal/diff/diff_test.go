package diff

import "testing"

func TestLinesProto(t *testing.T) {
	old := "    pub fn match_path() {\n        self.routes.get(path)\n    }"
	nw := "    pub fn match_path() {\n        let key = normalize(path);\n        self.routes.get(&key)\n    }"
	ls := Lines(old, nw, 41, 41)
	want := []struct {
		k      Kind
		o, n   int
		prefix string
	}{{Same, 41, 41, "    pub"}, {Del, 42, 0, "        self.routes.get(path)"}, {Add, 0, 42, "        let key"}, {Add, 0, 43, "        self.routes.get(&key)"}, {Same, 43, 44, "    }"}}
	if len(ls) != len(want) {
		t.Fatalf("行数 %d: %+v", len(ls), ls)
	}
	for i, w := range want {
		g := ls[i]
		if g.Kind != w.k || g.OldNo != w.o || g.NewNo != w.n {
			t.Errorf("第 %d 行 = %+v,期望 %+v", i, g, w)
		}
	}
	if a, d := Stat(ls); a != 2 || d != 1 {
		t.Errorf("统计 +%d −%d", a, d)
	}
}

func TestLinesEdges(t *testing.T) {
	if got := Lines("", "a\nb", 1, 1); len(got) != 2 || got[0].Kind != Add {
		t.Errorf("纯新增: %+v", got)
	}
	if got := Lines("a\nb", "", 1, 1); len(got) != 2 || got[1].Kind != Del {
		t.Errorf("纯删除: %+v", got)
	}
	if got := Lines("a", "a", 5, 5); len(got) != 1 || got[0].OldNo != 5 {
		t.Errorf("无变化: %+v", got)
	}
	// 同一处改动:先删后增
	got := Lines("x\nold\ny", "x\nnew\ny", 1, 1)
	if got[1].Kind != Del || got[2].Kind != Add {
		t.Errorf("顺序: %+v", got)
	}
}
