package app

import (
	"slices"

	"github.com/egoist/mygo/ui/editor"
)

// fileFolds 记住各文件最后一次的折叠行号(关标签后再开同一文件时恢复,同会话内有效)。

func foldMemKey(tb *ideTab) string {
	if tb == nil || tb.diff || tb.isVirt() || tb.path == "" {
		return ""
	}
	return tb.path
}

func (d *ide) saveFolds(tb *ideTab) {
	k := foldMemKey(tb)
	if k == "" || tb.st == nil || tb.st.NoFolding {
		return
	}
	folds := tb.st.Folded()
	if d.fileFolds == nil {
		d.fileFolds = map[string][]int{}
	}
	if len(folds) == 0 {
		delete(d.fileFolds, k)
		return
	}
	cp := slices.Clone(folds)
	slices.Sort(cp)
	d.fileFolds[k] = cp
}

func (d *ide) restoreFolds(tb *ideTab) {
	k := foldMemKey(tb)
	if k == "" || tb.st == nil || tb.st.NoFolding {
		return
	}
	folds, ok := d.fileFolds[k]
	if !ok || len(folds) == 0 {
		return
	}
	for _, line := range folds {
		if line < 0 || line >= tb.buf.LineCount() {
			continue
		}
		tb.st.Fold(line)
	}
}

// unfoldLineForNav 把光标挪到 line 并走编辑器的 reveal(会展开隐藏该行的折叠区)。
func unfoldLineForNav(tb *ideTab, line int) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	if line < 0 {
		line = 0
	}
	if line >= tb.buf.LineCount() {
		line = tb.buf.LineCount() - 1
	}
	tb.st.GoTo(editor.Pos{Line: line, Col: 0})
}
