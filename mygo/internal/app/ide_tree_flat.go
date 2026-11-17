package app

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// treeLine 是文件树虚拟列表的一行(与 treeDir 递归展开结果一致)。
type treeLine struct {
	entry   workspace.Entry
	compact bool
	tail    string // 紧凑文件夹链尾
	label   string
	depth   int
	newKind string // new / newdir / rename
}

func (a *App) treeOpenSig() string {
	d := a.ide
	if d.open == nil {
		return ""
	}
	keys := make([]string, 0, len(d.open))
	for k, on := range d.open {
		if on {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	o := aSigOr(a.fo.kind, a.fo.path)
	return strings.Join(keys, "\n") + "|" + d.fltKey + "|" + itoa(bool01(d.rootShut)) + "|" + o + "|" + d.fs.Name()
}

func aSigOr(kind, path string) string {
	if kind == "" {
		return ""
	}
	return kind + ":" + path
}

func bool01(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *App) treeLineCache() []treeLine {
	d := a.ide
	sig := a.treeOpenSig()
	if sig == d.treeLineSig && d.treeLineBuf != nil {
		return d.treeLineBuf
	}
	d.treeLineBuf = a.buildTreeLines()
	d.treeLineSig = sig
	return d.treeLineBuf
}

func (a *App) buildTreeLines() []treeLine {
	d := a.ide
	if d.rootShut {
		return nil
	}
	var out []treeLine
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if o := a.fo; (o.kind == "new" || o.kind == "newdir") && o.dir == dir && !(dir == "" && a.multiRoot()) {
			out = append(out, treeLine{depth: depth, newKind: o.kind})
		}
		for _, e := range d.listing(dir) {
			if e.Dir && a.fo.kind != "rename" && a.fo.path != e.Path {
				if tail, label, compact := d.compactFolderTail(e.Path); compact {
					out = append(out, treeLine{entry: e, compact: true, tail: tail, label: label, depth: depth})
					if d.open[tail] {
						walk(tail, depth)
					}
					continue
				}
			}
			if a.fo.kind == "rename" && a.fo.path == e.Path {
				out = append(out, treeLine{entry: e, depth: depth, newKind: "rename"})
				if e.Dir && d.open[e.Path] {
					walk(e.Path, depth+1)
				}
				continue
			}
			out = append(out, treeLine{entry: e, depth: depth})
			if e.Dir && d.open[e.Path] {
				walk(e.Path, depth+1)
			}
		}
	}
	if a.multiRoot() {
		for _, e := range d.listing("") {
			if e.Dir {
				out = append(out, treeLine{entry: e, depth: 0})
				if d.open[e.Path] {
					walk(e.Path, 1)
				}
			}
		}
	} else {
		walk("", 0)
	}
	return out
}

func treeLinePath(ln treeLine) string {
	if ln.compact {
		return ln.tail
	}
	if ln.newKind != "" && ln.entry.Path == "" {
		return ""
	}
	return ln.entry.Path
}

func (a *App) treeLineAt(c *ui.Context, ln treeLine, row int) {
	switch ln.newKind {
	case "new", "newdir":
		a.fileOpInput(c, ln.depth, ln.newKind == "newdir", "file")
		return
	case "rename":
		a.fileOpInput(c, ln.depth, ln.entry.Dir, fileIcon(ln.entry.Name))
		return
	}
	if ln.compact {
		a.treeCompactRow(c, ln.tail, ln.label, ln.depth, row)
		return
	}
	a.treeRow(c, ln.entry, ln.depth, row)
}
