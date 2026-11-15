package app

import "testing"

const rustSrc = `use std::collections::HashMap;

pub struct Router {
    routes: HashMap<String, Handler>,
}

impl Router {
    pub fn add(&mut self, path: &str) {
        let key = normalize(path);
    }

    pub fn match_path(&self, path: &str) -> Option<&Handler> {
        let key = normalize(path);
        self.routes.get(&key)
    }
}

fn normalize(path: &str) -> String {
    path.to_lowercase()
}
`

func TestParseSymbolsRust(t *testing.T) {
	ss := parseSymbols("src/router.rs", rustSrc)
	names := map[string]symbol{}
	for _, s := range ss {
		names[s.Name] = s
	}
	for _, n := range []string{"Router", "impl Router", "add", "match_path", "normalize"} {
		if _, ok := names[n]; !ok {
			t.Fatalf("缺少符号 %q:%+v", n, ss)
		}
	}
	m := names["match_path"]
	if m.Line != 11 || m.End != 14 {
		t.Errorf("match_path 行范围 %d-%d,想要 11-14", m.Line, m.End)
	}
	if ss[m.Parent].Name != "impl Router" {
		t.Errorf("match_path 的外层应是 impl Router,得到 %q", ss[m.Parent].Name)
	}
	chain := enclosing(ss, 13)
	if len(chain) != 2 || ss[chain[0]].Name != "impl Router" || ss[chain[1]].Name != "match_path" {
		t.Errorf("第 13 行的符号链不对:%v", chain)
	}
	if n := names["Router"]; n.End != 4 {
		t.Errorf("struct Router 应在第 4 行结束,得到 %d", n.End)
	}
}

func TestParseSymbolsOthers(t *testing.T) {
	py := parseSymbols("a.py", "class A:\n    def f(self):\n        pass\n\ndef g():\n    pass\n")
	if len(py) != 3 || py[1].Parent != 0 || py[0].End != 2 || py[2].Parent != -1 {
		t.Errorf("python 符号不对:%+v", py)
	}
	md := parseSymbols("a.md", "# 一\n正文\n## 二\n文\n# 三\n")
	if len(md) != 3 || md[0].End != 3 || md[1].Parent != 0 {
		t.Errorf("markdown 符号不对:%+v", md)
	}
	g := parseSymbols("a.go", "package a\n\ntype T struct {\n\tx int\n}\n\nfunc (t T) M() {\n}\n")
	if len(g) != 2 || g[0].Kind != "struct" || g[1].Name != "M" {
		t.Errorf("go 符号不对:%+v", g)
	}
}

func TestWordHelpers(t *testing.T) {
	w, a, b := wordAt("    let key = normalize(path);", 17)
	if w != "normalize" || a != 14 || b != 23 {
		t.Errorf("wordAt = %q %d %d", w, a, b)
	}
	if got := wordHits("normalize(a) + my_normalize(b) + normalize(c)", "normalize"); len(got) != 2 || got[0] != 0 {
		t.Errorf("wordHits 应只命中整词:%v", got)
	}
}
