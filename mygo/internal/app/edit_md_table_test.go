package app

import (
	"strings"
	"testing"
)

const mdTableSample = "# 接口\n\n前言一段。\n\n| 方法 | 路径 | 说明 |\n| :--- | :-: | ---: |\n| GET | /users | 列表 |\n| POST | /users | 新建 \\| 或更新 |\n\n```\n| 不是 | 表格 |\n| --- | --- |\n```\n\n结尾。\n"

// 分段后各段首尾相接必须恰好是原文,围栏代码块里的 | 不算表格。
func TestMdSplitSegsRoundTrip(t *testing.T) {
	for _, src := range []string{
		mdTableSample,
		"| a |\n| - |\n| 1 |",
		"正文\n\n| a | b |\n|---|---|\n",
		"| a | b |\n|---|---|\n| 1 | 2 |\n\n\n| c |\n|---|\n| 3 |\n\n尾",
		"没有表格\n\n- 列表\n",
	} {
		segs := mdSplitSegs(src)
		md := &mdState{segs: segs}
		if got := md.segsText(); got != src {
			t.Errorf("拼回与原文不一致:\n got=%q\nwant=%q", got, src)
		}
	}
	segs := mdSplitSegs(mdTableSample)
	n := 0
	for _, s := range segs {
		if s.table {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("围栏里的不算表格,应只有 1 张:%d", n)
	}
	if segs[0].table || segs[len(segs)-1].table {
		t.Fatal("首尾必须是正文段")
	}
	// 表格在文档开头 / 结尾时补空正文段
	first := mdSplitSegs("| a |\n| - |\n")
	if len(first) != 3 || first[0].table || !first[1].table || first[2].table {
		t.Fatalf("表格在首尾应补空段:%d", len(first))
	}
}

func TestMdTableParseAndSerialize(t *testing.T) {
	lines := mdLinesKeep("| 方法 | 路径 | 说明 |\n| :--- | :-: | ---: |\n| GET | /users | 列表 |\n| POST | /users | 新建 \\| 或更新 |\n")
	tbl := parseMdTable(lines)
	if tbl.cols() != 3 || len(tbl.rows) != 2 {
		t.Fatalf("解析不对:%+v", tbl)
	}
	if string(tbl.align) != "lcr" {
		t.Fatalf("对齐不对:%q", tbl.align)
	}
	if tbl.rows[1][2] != `新建 \| 或更新` {
		t.Fatalf("转义的竖线应原样保留:%q", tbl.rows[1][2])
	}
	out := tbl.markdown()
	// 序列化后再解析,内容与对齐一致
	again := parseMdTable(mdLinesKeep(out + "\n"))
	if again.cols() != 3 || len(again.rows) != 2 || string(again.align) != "lcr" || again.rows[1][2] != tbl.rows[1][2] {
		t.Fatalf("往返不一致:\n%s", out)
	}
	// 输入框里键入的 | 序列化时要转义
	tbl.rows[0][0] = "a|b"
	if !strings.Contains(tbl.markdown(), `a\|b`) {
		t.Fatalf("未转义的 | 应被转义:\n%s", tbl.markdown())
	}
}

func TestMdTableOps(t *testing.T) {
	tbl := parseMdTable(mdLinesKeep("| a | b |\n|---|---|\n| 1 | 2 |\n"))
	tbl.insertRow(0)
	tbl.insertRow(len(tbl.rows))
	if len(tbl.rows) != 3 || tbl.rows[0][0] != "" || tbl.rows[1][0] != "1" {
		t.Fatalf("插入行不对:%v", tbl.rows)
	}
	tbl.deleteRow(0)
	tbl.insertCol(1)
	if tbl.cols() != 3 || tbl.head[1] != "" || tbl.head[2] != "b" || tbl.rows[0][2] != "2" || len(tbl.align) != 3 {
		t.Fatalf("插入列不对:%v %v", tbl.head, tbl.rows)
	}
	tbl.deleteCol(1)
	tbl.deleteCol(0)
	tbl.deleteCol(0) // 只剩一列时不再删
	if tbl.cols() != 1 {
		t.Fatalf("至少保留一列:%d", tbl.cols())
	}
	tbl.setAlign(0, 'r')
	if !tbl.edited || !strings.Contains(tbl.markdown(), "--:") {
		t.Fatalf("右对齐应写成 --:\n%s", tbl.markdown())
	}
}

// 含表格的文档:打开不改文件;改一个单元格只重写这张表,前后的正文原样不动。
func TestMdSegsEditWriteBack(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("docs/api.md", true)
	tt.Frame()
	tb := a.ide.cur()
	md := tb.md
	if md == nil || md.segs == nil {
		t.Fatalf("应分段:%+v", md)
	}
	before := tb.buf.String()
	if tb.dirty() {
		t.Fatal("只打开不能弄脏")
	}
	var tseg *mdSeg
	for _, s := range md.segs {
		if s.table {
			tseg = s
		}
	}
	tseg.tbl.rows[0][2] = "全部用户"
	tseg.tbl.edited = true
	a.mdWriteSegs(tb, md, tseg)
	got := tb.buf.String()
	if !strings.Contains(got, "全部用户") || !strings.HasPrefix(got, "# 接口\n\n| 方法") || !strings.HasSuffix(got, "\n\n以上接口都返回 JSON。\n") {
		t.Fatalf("写回应只改表格:\n%s", got)
	}
	if !tb.dirty() || md.ver != tb.buf.Version() {
		t.Fatal("写回后应变脏且版本对齐")
	}
	// 排队的结构修改在下一次构建开头执行并写回
	a.mdQueue(md, tseg, "添加行", func() { tseg.tbl.insertRow(len(tseg.tbl.rows)) })
	tt.Frame()
	if n := strings.Count(tb.buf.String(), "\n|"); n != 5 { // 表头 + 分隔 + 3 行
		t.Fatalf("添加行后应有 5 行竖线行,得到 %d:\n%s", n, tb.buf.String())
	}
	_ = before
}
