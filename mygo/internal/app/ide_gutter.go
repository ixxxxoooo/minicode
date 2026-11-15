package app

import (
	"encoding/hex"
	"hash/fnv"

	"github.com/egoist/mygo/ui/editor"
)

// syncDiffBase 保证编辑器行号槽的 Git 竖条跟上 HEAD(库用 DiffBase 与缓冲区对比绘制)。
func (a *App) syncDiffBase(tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff {
		return
	}
	if a.ide.headOf == nil {
		tb.st.DiffBase = nil
		return
	}
	head, ok := a.ide.headOf(tb.path)
	if !ok {
		tb.st.DiffBase = nil
		tb.headKey = ""
		return
	}
	sum := hashText(head)
	if tb.headKey == sum && tb.st.DiffBase != nil {
		return
	}
	tb.headKey = sum
	tb.st.DiffBase = editor.NewBuffer(head)
	tb.st.DiffBase.SetLanguage(tb.buf.Language())
}

func hashText(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:12]
}
