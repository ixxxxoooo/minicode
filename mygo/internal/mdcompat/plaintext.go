package mdcompat

import (
	"strings"

	"github.com/egoist/mygo/ui/markdown/parser"
)

// Parse 使用与 UI 相同的默认扩展解析 Markdown。
func Parse(src string) *parser.Document {
	d := parser.Parse(Prepare(src))
	return d
}

// PlainText 从解析树收集可见文本(含代码块、HTML 块、数学块源码),用于回归与容错断言。
func PlainText(d *parser.Document) string {
	var b strings.Builder
	walkBlocks(d, 0, &b)
	return b.String()
}

func walkBlocks(d *parser.Document, parent int, b *strings.Builder) {
	for i := range d.Children(parent) {
		walkBlock(d, i, b)
	}
}

func walkBlock(d *parser.Document, i int, b *strings.Builder) {
	switch d.Block(i).Kind {
	case parser.Paragraph, parser.Heading:
		walkInlineRange(d.Inlines(i), 0, len(d.Inlines(i)), b)
		b.WriteByte('\n')
	case parser.TableCell:
		walkInlineRange(d.Inlines(i), 0, len(d.Inlines(i)), b)
		b.WriteByte('\t')
	case parser.TableRow:
		for j := range d.Children(i) {
			walkBlock(d, j, b)
		}
		b.WriteByte('\n')
	case parser.CodeBlock, parser.HTMLBlock, parser.MathBlock:
		b.WriteString(d.Lines(i))
		b.WriteByte('\n')
	case parser.Table, parser.List, parser.Quote, parser.Item, parser.FootnoteDefinition:
		walkBlocks(d, i, b)
	default:
		walkBlocks(d, i, b)
	}
}

func walkInlineRange(in []parser.Inline, from, to int, b *strings.Builder) {
	for at := from; at < to && at < len(in); at = in[at].Next {
		if in[at].Next <= at {
			break
		}
		n := in[at]
		end := n.Next
		if end > to {
			end = to
		}
		switch n.Kind {
		case parser.Text, parser.Code, parser.HTML, parser.Math:
			b.WriteString(n.Text)
		case parser.SoftBreak:
			b.WriteByte(' ')
		case parser.LineBreak:
			b.WriteByte('\n')
		case parser.FootnoteRef, parser.Mention, parser.Reference:
			b.WriteString(n.Text)
		case parser.Link, parser.Image, parser.Emphasis, parser.Strong, parser.Strikethrough:
			if at+1 < end {
				walkInlineRange(in, at+1, end, b)
			}
		}
	}
}
