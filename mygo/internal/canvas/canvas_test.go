package canvas

import (
	"strings"
	"testing"
	"time"
)

func sample() Doc {
	return Doc{Title: "T", Status: "草稿", Blocks: []Block{
		{ID: "h-bg", Kind: Heading, Level: 2, Text: "背景"},
		{ID: "p", Kind: Para, Text: "正文"},
		{ID: "dg", Kind: DiagramBlock, Diagram: &Diagram{Nodes: []Node{{Label: "a"}}, Edges: [][][2]float32{{{0, 0}, {1, 1}}}}},
		{ID: "tb", Kind: TableBlock, Table: &Table{Head: []string{"x"}, Rows: [][]string{{"1"}}}},
		{ID: "td", Kind: TodoBlock, Todos: []Todo{{"a", 0}}},
	}}
}

var t0 = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

func TestOutline(t *testing.T) {
	o := sample().Outline()
	if len(o) != 2 || o[0].ID != "title" || o[0].Level != 0 || o[1].Text != "背景" || o[1].Level != 1 {
		t.Fatalf("%+v", o)
	}
}

func TestCommitAndRewind(t *testing.T) {
	s := NewStore(sample(), "Claude", "初稿", t0)
	if s.Dirty() || s.Commit("你", "无修改", t0) {
		t.Fatal("没改动不应产生版本")
	}
	s.Doc.Block("p").Text = "改过的正文"
	s.Doc.Block("td").Todos[0].State = 2
	if !s.Dirty() || !s.Commit("你", "改正文", t0) || len(s.Versions) != 2 || s.Versions[0].N != 2 {
		t.Fatalf("应产生 v2:%+v", s.Versions)
	}
	// 快照彼此独立:再改当前文档不影响 v2
	s.Doc.Block("td").Todos[0].State = 1
	if s.Versions[0].Doc.Block("td").Todos[0].State != 2 {
		t.Fatal("版本快照被连带修改了(没有深拷贝)")
	}
	// 回到 v1:文档恢复,且历史多一条 v3
	if err := s.Rewind(1, t0); err != nil {
		t.Fatal(err)
	}
	if s.Doc.Block("p").Text != "正文" || len(s.Versions) != 3 || s.Versions[0].N != 3 || s.Versions[0].Note != "回到 v1" {
		t.Fatalf("回退不对:%q %+v", s.Doc.Block("p").Text, s.Versions[0])
	}
	if s.Rewind(9, t0) == nil {
		t.Fatal("不存在的版本应报错")
	}
	// 回退后文档与新版本一致,不算脏
	if s.Dirty() {
		t.Fatal("回退后不应是脏的")
	}
}

func TestAgoAndMarkdown(t *testing.T) {
	now := t0.Add(30 * time.Hour)
	for in, want := range map[time.Duration]string{0: "刚刚", 5 * time.Minute: "5 分钟前", 2 * time.Hour: "2 小时前", 30 * time.Hour: "昨天", 80 * time.Hour: "11 月 12 日"} {
		if got := Ago(now, now.Add(-in)); got != want {
			t.Errorf("%v → %q,想要 %q", in, got, want)
		}
	}
	md := sample().Markdown()
	for _, want := range []string{"# T\n", "## 背景", "```mermaid", "n0[a]", "| x |", "- [ ] a"} {
		if !strings.Contains(md, want) {
			t.Errorf("导出缺少 %q:\n%s", want, md)
		}
	}
}
