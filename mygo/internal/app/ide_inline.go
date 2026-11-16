package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 行内编辑块与 Claude 改动行的审查条。
type inlinePatch struct {
	tab    string
	start  int
	end    int
	old    string
	new    string
	prompt string
	busy   bool
}

func (a *App) submitInlineEdit(tb *ideTab, prompt string) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	s := tb.st.Selection()
	lo, hi := s.Anchor, s.Head
	if hi < lo {
		lo, hi = hi, lo
	}
	if lo == hi {
		a.toast = "先选中要修改的代码"
		return
	}
	old := tb.buf.Slice(lo, hi)
	if a.inl == nil {
		a.inl = &inlinePatch{}
	}
	*a.inl = inlinePatch{tab: tb.id, start: lo, end: hi, old: old, prompt: prompt, busy: true}
	a.closeOv()
	a.toast = "正在生成行内修改…"
	promptText := inlineEditPrompt(old, prompt)
	ctx := a.beginClaudeJob()
	tbID := tb.id
	a.bg("claude.inline", func() func() {
		defer a.endClaudeJob()
		var newText string
		var err error
		if a.proj == "" {
			newText = demoInlineRewrite(old, prompt)
		} else {
			newText, err = claude.RunPromptModelCtx(ctx, a.proj, promptText, "plan", haikuInline, 90*time.Second)
			if err != nil {
				claudeEditLog.Warn("行内编辑失败", "path", tb.path, "err", err)
				newText = demoInlineRewrite(old, prompt)
			}
		}
		newText = stripCodeFence(newText)
		if newText == "" {
			newText = old
		}
		return func() {
			if a.inl == nil || a.inl.tab != tbID || !a.inl.busy {
				return
			}
			a.inl.busy = false
			a.inl.new = newText
			a.toast = "已生成行内修改预览 · Tab 或点「接受」写回"
		}
	})
}

func demoInlineRewrite(src, prompt string) string {
	p := strings.ToLower(prompt)
	if strings.Contains(p, "忽略大小写") {
		return strings.ReplaceAll(src, "ToLower", "EqualFold")
	}
	if strings.Contains(p, "注释") {
		lines := strings.Split(src, "\n")
		for i, l := range lines {
			if strings.TrimSpace(l) != "" && !strings.HasPrefix(strings.TrimSpace(l), "//") {
				lines[i] = "// " + l
			}
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func (a *App) acceptInline() {
	p := a.inl
	if p == nil || p.tab == "" || p.busy {
		return
	}
	tb := a.ide.tab(p.tab)
	if tb == nil || tb.buf == nil || tb.st == nil {
		a.inl = nil
		return
	}
	tb.buf.Edit(editor.Edit{Start: p.start, End: p.end, Text: p.new})
	tb.st.Select(p.start, p.start+len(p.new))
	a.inl = nil
	a.toast = "已接受行内修改"
}

func (a *App) revertInline() {
	if a.inl != nil && a.inl.busy {
		a.cancelClaudeJob()
	}
	a.inl = nil
	a.toast = "已撤消行内修改预览"
}

// inlinePatchUI 在选区下方画预览条。
func (a *App) inlinePatchUI(c *ui.Context, tb *ideTab) {
	p := a.inl
	if p == nil || p.tab != tb.id {
		return
	}
	t := a.tk
	s := tb.st.Selection()
	pos := tb.buf.Pos(s.Head)
	x, y := cursorXY(tb, pos.Line, pos.Col)
	row := ui.Row(c).Key("inl-bar").Absolute().Left(x).Top(y+28).Padding(6, 8).Gap(6).AlignItems(ui.Center).
		Background(t.Raised).Radius(t.RMd()).Border(1, t.Ac).FontSize(theme.FsCap)
	row.Children(func() {
		if p.busy {
			ui.Text(c, "正在请求 Claude…").TextColor(t.Ac)
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.revertInline)
			return
		}
		ui.Text(c, "行内修改预览").TextColor(t.Ac)
		kit.Btn(c, "接受", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(a.acceptInline)
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.revertInline)
		ui.Text(c, ellipsize(p.new, 40)).TextColor(t.Tx3).Font("monospace")
	})
	if !p.busy {
		tb.st.TextBackgrounds = append(tb.st.TextBackgrounds, editor.TextBackground{
			Start: p.start, End: p.end, Color: t.Ac.Alpha(.22),
		})
	}
}

func ellipsize(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// reviewHover 光标在 Claude 改动行时,行尾显示接受 / 撤销。
func (a *App) reviewHover(c *ui.Context, tb *ideTab) {
	lines := a.claudeLines(tb)
	if len(lines) == 0 {
		return
	}
	line := tb.st.CursorPos().Line
	var hit *struct {
		f chFile
		i int
	}
	for _, f := range a.changedFiles() {
		if a.wsRel(f.path) != tb.path {
			continue
		}
		for i, b := range f.blocks {
			if a.rv.hunk[hunkKey(f.path, i)] != "" {
				continue
			}
			lo, hi := hunkRange(b)
			if lo > 0 && line >= lo-1 && line <= hi-1 {
				hit = &struct {
					f chFile
					i int
				}{f, i}
				break
			}
		}
	}
	if hit == nil {
		return
	}
	t := a.tk
	_, y := cursorXY(tb, line, 0)
	// 审查条贴在光标行上方,不盖住行尾的追溯文字与下面的提交卡;第一行放不下才退到行下方
	a.ide.rvBarBelow = false
	barTop := y - edLineH*edScale - 24
	if barTop < 2 {
		barTop = y + 2
		a.ide.rvBarBelow = true
	}
	bar := ui.Row(c).Key("rv-hov").Absolute().Right(a.mmWidth()+mmRulW+edTrackW+20).Top(barTop).Height(22).Gap(6).PaddingX(8).
		Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd1).FontSize(theme.FsMicro)
	bar.Children(func() {
		ui.Text(c, "Claude 改动").TextColor(t.Ac)
		kit.Btn(c, "接受", kit.BtnOpt{Small: true}).OnClick(func() {
			a.rv.init()
			a.rv.hunk[hunkKey(hit.f.path, hit.i)] = "ok"
			a.toast = "已接受这一块改动"
			log.Info("行内审查接受", "path", tb.path, "hunk", hit.i)
		})
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			a.revertHunk(hit.f, hit.i)
			a.toast = "已撤销这一块改动"
		})
	})
}
