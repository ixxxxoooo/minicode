// Package canvas 是「画布」文档的模型、版本与大纲(与界面无关)。
// 画布是一份由块组成的结构化文档(标题 / 段落 / 图表 / 表格 / 待办),Claude 与用户都可以改它;
// 每次改动落成一个版本,可以回到旧版本(回退本身也是一个新版本,不丢历史)。
package canvas

import (
	"fmt"
	"strings"
	"time"
)

// Kind 是块的种类。
type Kind int

const (
	Heading Kind = iota
	Para
	DiagramBlock
	TableBlock
	TodoBlock
)

// Node 是流程图里的一个方框(坐标是 520×150 画布里的位置)。
type Node struct {
	X, Y, W float32
	Label   string
	Hot     bool
}

// Diagram 是一张流程图:方框 + 折线箭头。
type Diagram struct {
	Caption string
	Nodes   []Node
	Edges   [][][2]float32 // 每条边是一串折点,末端画箭头
}

// Table 是表格。
type Table struct {
	Head []string
	Rows [][]string
}

// Todo 是一条待办:0 未开始 / 1 进行中 / 2 已完成。
type Todo struct {
	Text  string
	State int
}

// Block 是文档的一个块。
type Block struct {
	ID      string
	Kind    Kind
	Level   int    // 标题级别(1 / 2)
	Lead    bool   // 段落是导语(稍大、灰)
	Anchor  string // 非空时这个块出现在大纲里(导语没有标题,用它起名,如「背景」)
	Text    string
	Diagram *Diagram
	Table   *Table
	Todos   []Todo
}

// Doc 是一份文档。
type Doc struct {
	Title  string
	Status string // 草稿 / 定稿
	Blocks []Block
}

// Clone 深拷贝(版本快照要彼此独立)。
func (d Doc) Clone() Doc {
	out := d
	out.Blocks = make([]Block, len(d.Blocks))
	for i, b := range d.Blocks {
		nb := b
		if b.Diagram != nil {
			dg := *b.Diagram
			dg.Nodes = append([]Node(nil), b.Diagram.Nodes...)
			dg.Edges = make([][][2]float32, len(b.Diagram.Edges))
			for j, e := range b.Diagram.Edges {
				dg.Edges[j] = append([][2]float32(nil), e...)
			}
			nb.Diagram = &dg
		}
		if b.Table != nil {
			tb := Table{Head: append([]string(nil), b.Table.Head...)}
			for _, r := range b.Table.Rows {
				tb.Rows = append(tb.Rows, append([]string(nil), r...))
			}
			nb.Table = &tb
		}
		nb.Todos = append([]Todo(nil), b.Todos...)
		out.Blocks[i] = nb
	}
	return out
}

// Equal 两份文档内容是否相同。
func (d Doc) Equal(o Doc) bool { return d.signature() == o.signature() }

func (d Doc) signature() string {
	var b strings.Builder
	b.WriteString(d.Title + "|" + d.Status)
	for _, k := range d.Blocks {
		fmt.Fprintf(&b, "\x00%d:%d:%v:%s", k.Kind, k.Level, k.Lead, k.Text)
		if k.Diagram != nil {
			fmt.Fprintf(&b, ":dg%d:%d", len(k.Diagram.Nodes), len(k.Diagram.Edges))
			for _, n := range k.Diagram.Nodes {
				b.WriteString(":" + n.Label)
			}
		}
		if k.Table != nil {
			b.WriteString(":" + strings.Join(k.Table.Head, ","))
			for _, r := range k.Table.Rows {
				b.WriteString(";" + strings.Join(r, ","))
			}
		}
		for _, t := range k.Todos {
			fmt.Fprintf(&b, ":%s=%d", t.Text, t.State)
		}
	}
	return b.String()
}

// OutlineItem 是大纲的一项。
type OutlineItem struct {
	ID    string
	Text  string
	Level int // 0 = 文档标题,1 = 二级标题
}

// Outline 由标题块生成大纲:第一项总是文档标题。
func (d Doc) Outline() []OutlineItem {
	out := []OutlineItem{{ID: "title", Text: d.Title}}
	for _, b := range d.Blocks {
		switch {
		case b.Kind == Heading && b.Level >= 2:
			out = append(out, OutlineItem{ID: b.ID, Text: b.Text, Level: 1})
		case b.Anchor != "":
			out = append(out, OutlineItem{ID: b.ID, Text: b.Anchor, Level: 1})
		}
	}
	return out
}

// Version 是一个历史版本。
type Version struct {
	N    int
	By   string // 你 / Claude
	Note string
	At   time.Time
	Doc  Doc
}

// Store 保存当前文档与全部版本(新的在前)。只在界面线程使用。
type Store struct {
	Doc      Doc
	Versions []Version
}

// NewStore 以 doc 作为 v1 建立存储。
func NewStore(doc Doc, by, note string, at time.Time) *Store {
	s := &Store{Doc: doc}
	s.Versions = []Version{{N: 1, By: by, Note: note, At: at, Doc: doc.Clone()}}
	return s
}

// Dirty 当前文档相对最新版本是否有未提交的修改。
func (s *Store) Dirty() bool { return !s.Doc.Equal(s.Versions[0].Doc) }

// Commit 把当前文档落成新版本;没有修改时不产生版本,返回 false。
func (s *Store) Commit(by, note string, at time.Time) bool {
	if !s.Dirty() {
		return false
	}
	s.Versions = append([]Version{{N: s.Versions[0].N + 1, By: by, Note: note, At: at, Doc: s.Doc.Clone()}}, s.Versions...)
	return true
}

// Rewind 回到第 n 版:文档换成那一版的内容,并记一个新版本(历史不丢)。
func (s *Store) Rewind(n int, at time.Time) error {
	for _, v := range s.Versions {
		if v.N == n {
			s.Doc = v.Doc.Clone()
			s.Versions = append([]Version{{N: s.Versions[0].N + 1, By: "你", Note: fmt.Sprintf("回到 v%d", n), At: at, Doc: s.Doc.Clone()}}, s.Versions...)
			return nil
		}
	}
	return fmt.Errorf("没有 v%d", n)
}

// Block 按 ID 找块。
func (d *Doc) Block(id string) *Block {
	for i := range d.Blocks {
		if d.Blocks[i].ID == id {
			return &d.Blocks[i]
		}
	}
	return nil
}

// Ago 把时间说成相对 now 的人话:刚刚 / N 分钟前 / N 小时前 / 昨天 / M 月 D 日。
func Ago(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	case d < 48*time.Hour:
		return "昨天"
	}
	return fmt.Sprintf("%d 月 %d 日", int(t.Month()), t.Day())
}

// Markdown 导出成 Markdown(图表输出为 mermaid 围栏,待办输出为任务列表)。
func (d Doc) Markdown() string {
	var b strings.Builder
	b.WriteString("# " + d.Title + "\n\n")
	for _, k := range d.Blocks {
		switch k.Kind {
		case Heading:
			b.WriteString(strings.Repeat("#", max(k.Level, 1)+0) + " " + k.Text + "\n\n")
		case Para:
			b.WriteString(k.Text + "\n\n")
		case DiagramBlock:
			b.WriteString("```mermaid\nflowchart LR\n")
			for i, n := range k.Diagram.Nodes {
				fmt.Fprintf(&b, "  n%d[%s]\n", i, n.Label)
			}
			b.WriteString("```\n\n")
		case TableBlock:
			b.WriteString("| " + strings.Join(k.Table.Head, " | ") + " |\n|" + strings.Repeat(" --- |", len(k.Table.Head)) + "\n")
			for _, r := range k.Table.Rows {
				b.WriteString("| " + strings.Join(r, " | ") + " |\n")
			}
			b.WriteString("\n")
		case TodoBlock:
			for _, t := range k.Todos {
				mark := " "
				if t.State == 2 {
					mark = "x"
				}
				b.WriteString("- [" + mark + "] " + t.Text + "\n")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
