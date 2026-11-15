package search

import (
	"context"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func fixture() *workspace.Mem {
	return workspace.NewMem("p", map[string]string{
		"src/router.rs":     "pub fn match_path(&self) {}\n// 忽略 match_path 的大小写\nlet MATCH_PATH = 1;\n",
		"src/lib.rs":        "mod router;\n",
		"tests/router_t.rs": "r.match_path(\"/Users\");\nr.matchXpath();\n",
		"target/debug/x.rs": "match_path in target\n",
		"node_modules/a.js": "match_path\n",
		"assets/bin.dat":    "ab\x00match_path",
		"notes/中文.md":       "你好 match_path 世界\n",
	})
}

func TestRunBasics(t *testing.T) {
	res, err := Run(context.Background(), fixture(), Query{Text: "match_path"})
	if err != nil {
		t.Fatal(err)
	}
	// router.rs 3 处(含大写,默认不区分大小写)、router_t.rs 1 处、中文.md 1 处;target / node_modules / 二进制被跳过
	if res.Total != 5 || len(res.Files) != 3 {
		t.Fatalf("应 5 处 / 3 个文件,实际 %d / %d:%+v", res.Total, len(res.Files), res.Files)
	}
	// 中文行的列按字符计
	for _, f := range res.Files {
		if f.Path == "notes/中文.md" && (f.Matches[0].Col != 3 || f.Matches[0].End != 13) {
			t.Fatalf("字符列不对:%+v", f.Matches[0])
		}
	}
	// 区分大小写 + 全词
	res, _ = Run(context.Background(), fixture(), Query{Text: "match_path", Case: true, Word: true})
	if res.Total != 4 {
		t.Fatalf("区分大小写应 4 处,实际 %d", res.Total)
	}
	// 正则
	res, _ = Run(context.Background(), fixture(), Query{Text: `match.path`, Regex: true})
	if res.Total != 6 { // 多了 matchXpath
		t.Fatalf("正则应 6 处,实际 %d", res.Total)
	}
	// 非法正则报错
	if _, err := Run(context.Background(), fixture(), Query{Text: "(", Regex: true}); err == nil {
		t.Fatal("非法正则应报错")
	}
}

func TestGlobFilters(t *testing.T) {
	q := Query{Text: "match_path", Include: "*.rs"}
	if res, _ := Run(context.Background(), fixture(), q); len(res.Files) != 2 {
		t.Fatalf("只含 .rs 应 2 个文件:%d", len(res.Files))
	}
	q = Query{Text: "match_path", Include: "src/**,tests/**", Exclude: "*_t.rs"}
	if res, _ := Run(context.Background(), fixture(), q); len(res.Files) != 1 || res.Files[0].Path != "src/router.rs" {
		t.Fatalf("包含 + 排除不对:%+v", res.Files)
	}
	q = Query{Text: "match_path", Include: "{src,notes}/**"} // 花括号里的逗号不拆
	if res, _ := Run(context.Background(), fixture(), q); len(res.Files) != 2 {
		t.Fatalf("花括号 glob 应 2 个文件:%d", len(res.Files))
	}
	// 不使用忽略规则:target 与 node_modules 也搜
	if res, _ := Run(context.Background(), fixture(), Query{Text: "match_path", NoIgnore: true}); len(res.Files) != 5 {
		t.Fatalf("关闭忽略应搜到 5 个文件:%d", len(res.Files))
	}
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, fixture(), Query{Text: "x"}); err == nil {
		t.Fatal("已取消应返回错误")
	}
	_ = time.Second
}

func TestReplace(t *testing.T) {
	fs := fixture()
	q := Query{Text: "match_path"}
	res, _ := Run(context.Background(), fs, q)
	// 只替换 router.rs 的第 0 行与第 1 行,忽略第 2 行(大写那处)
	var picks []Match
	for _, f := range res.Files {
		if f.Path == "src/router.rs" {
			picks = f.Matches[:2]
		}
	}
	n, err := Replace(fs, q, "find_route", picks)
	if err != nil || n != 2 {
		t.Fatalf("应替换 2 处:%d %v", n, err)
	}
	got, _ := fs.Read("src/router.rs")
	want := "pub fn find_route(&self) {}\n// 忽略 find_route 的大小写\nlet MATCH_PATH = 1;\n"
	if got != want {
		t.Fatalf("替换结果不对:\n%q\n%q", got, want)
	}
	// 其它文件不动
	if o, _ := fs.Read("tests/router_t.rs"); o != "r.match_path(\"/Users\");\nr.matchXpath();\n" {
		t.Fatal("没被选中的文件不应改动")
	}
}

func TestReplaceRegexAndCase(t *testing.T) {
	fs := workspace.NewMem("p", map[string]string{"a.txt": "foo(1) foo(2)\nFoo FOO foo\n"})
	q := Query{Text: `foo\((\d)\)`, Regex: true}
	res, _ := Run(context.Background(), fs, q)
	n, _ := Replace(fs, q, "bar[$1]", res.Files[0].Matches)
	if got, _ := fs.Read("a.txt"); n != 2 || got != "bar[1] bar[2]\nFoo FOO foo\n" {
		t.Fatalf("捕获组替换不对 n=%d %q", n, got)
	}
	q = Query{Text: "foo", PreserveCase: true}
	res, _ = Run(context.Background(), fs, q)
	var picks []Match
	for _, m := range res.Files[0].Matches {
		picks = append(picks, m)
	}
	Replace(fs, q, "bar", picks)
	if got, _ := fs.Read("a.txt"); got != "bar[1] bar[2]\nBar BAR bar\n" {
		t.Fatalf("保留大小写不对:%q", got)
	}
}

func TestPreview(t *testing.T) {
	m := Match{Text: "        pub fn match_path(&self) -> Option<&Handler> {", Col: 15, End: 25}
	pre, hit, post := Preview(m, 40)
	if pre != "pub fn " || hit != "match_path" || len([]rune(post)) > 24 {
		t.Fatalf("预览不对:%q %q %q", pre, hit, post)
	}
	long := Match{Text: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa hit bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Col: 47, End: 50}
	pre, hit, post = Preview(long, 30)
	if hit != "hit" || pre[:3] != "…" || post[len(post)-3:] != "…" {
		t.Fatalf("长行应前后截断:%q %q %q", pre, hit, post)
	}
}
