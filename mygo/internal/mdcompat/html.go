package mdcompat

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// ParseDetails 从 HTML 块中提取首层 <details> 的 summary 与内部文本。
func ParseDetails(raw string) (summary, body string, ok bool) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", "", false
	}
	var find func(*html.Node) (*html.Node, bool)
	find = func(n *html.Node) (*html.Node, bool) {
		if n.Type == html.ElementNode && n.Data == "details" {
			return n, true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if d, yes := find(c); yes {
				return d, true
			}
		}
		return nil, false
	}
	dn, yes := find(doc)
	if !yes {
		return "", "", false
	}
	for c := dn.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "summary" {
			summary = strings.TrimSpace(textContent(c))
		}
	}
	var bodyB strings.Builder
	for c := dn.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data != "summary" {
			bodyB.WriteString(textContent(c))
			if c.NextSibling != nil {
				bodyB.WriteByte('\n')
			}
		}
	}
	body = strings.TrimSpace(bodyB.String())
	return summary, body, summary != "" || body != ""
}

// VisibleTextFromHTML 从 HTML 片段提取全部文本节点(不丢内容)。
func VisibleTextFromHTML(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	out := strings.TrimSpace(b.String())
	if out == "" {
		return strings.TrimSpace(raw)
	}
	return out
}

func textContent(n *html.Node) string {
	var b bytes.Buffer
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
