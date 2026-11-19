package app

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// MCP 详情「编辑配置」:命令 / 参数 / 环境变量 / URL,保存走 DiskStore.SaveMCP。

func (a *App) cuMCPStartEdit(it *customize.Item) {
	s := &a.cu
	s.draft, s.dOf = it.Clone(), it
	s.mcpEdit = true
	s.mcpEnv = mcpEnvDraft(it)
	s.mcpHdr = mcpHdrDraft(it)
	s.err = ""
}

// mcpHdrDraft 把请求头排成「名称: 值」每行一条(按名称排序,顺序稳定)。
func mcpHdrDraft(it *customize.Item) string {
	keys := make([]string, 0, len(it.Headers))
	for k := range it.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k + ": " + it.Headers[k])
	}
	return b.String()
}

func mcpEnvDraft(it *customize.Item) string {
	var b strings.Builder
	if len(it.EnvVals) > 0 {
		for k, v := range it.EnvVals {
			line := k + "=" + v
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(line)
		}
		return b.String()
	}
	for _, k := range it.Env {
		fmtLine := k + "="
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(fmtLine)
	}
	return b.String()
}

func (a *App) cuMCPEditDlg(c *ui.Context, st *customize.Store) {
	s := &a.cu
	if !s.mcpEdit || s.dOf == nil {
		return
	}
	d := &s.draft
	it := s.dOf
	a.autoDlg(c, "mcp-edit", 480, "编辑 MCP 配置 · "+it.Name, func() {
		s.mcpEdit, s.err = false, ""
	}, func() {
		t := a.tk
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "传输").TextColor(t.Tx3).Width(72).Shrink(0)
			on := 0
			if d.Transport == "http" || d.Transport == "sse" {
				on = 1
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "stdio"}, {Text: "http"}}, on, kit.SegOpt{Key: "mcp-tr"}); i >= 0 {
				if i == 0 {
					d.Transport = "stdio"
				} else {
					d.Transport = "http"
				}
			}
		})
		if d.Transport == "http" || d.Transport == "sse" {
			a.cuLbl(c, "URL", func() { a.cuField(c, "mcp-url", &d.Cmd, true, "https://…") })
		} else {
			a.cuLbl(c, "命令", func() { a.cuField(c, "mcp-cmd", &d.Cmd, true, "npx -y @scope/mcp-server") })
			a.cuLbl(c, "额外参数(空格分隔,接在命令后)", func() { a.cuField(c, "mcp-args", &d.Note, true, "无") })
		}
		if d.Transport == "http" || d.Transport == "sse" {
			a.cuLbl(c, "请求头(每行 名称: 值,如 Cookie / Authorization;值里可写 ${环境变量})", func() {
				ui.TextArea(c.Key("mcp-hdr"), &s.mcpHdr).Lines(3, 6).Font(theme.CodeFont()).FontSize(theme.Px(11.5))
			})
		}
		a.cuLbl(c, "环境变量(每行 KEY=VALUE)", func() {
			ui.TextArea(c.Key("mcp-env"), &s.mcpEnv).Lines(3, 6).Font(theme.CodeFont()).FontSize(theme.Px(11.5))
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "配置层").TextColor(t.Tx3).Width(72).Shrink(0)
			on := 0
			for i, sc := range cuScopes[1:] {
				if sc == d.Scope {
					on = i
				}
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "用户"}, {Text: "项目"}, {Text: "本地"}}, on, kit.SegOpt{Key: "mcp-ed-scope"}); i >= 0 {
				d.Scope = cuScopes[1:][i]
			}
		})
		if s.err != "" {
			ui.Text(c, s.err).FontSize(theme.FsCap).TextColor(t.Err)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { s.mcpEdit = false })
			if kit.Btn(c, "保存", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).Clicked() {
				a.cuMCPSaveEdit(st, it)
			}
		})
	})
}

func (a *App) cuMCPSaveEdit(st *customize.Store, it *customize.Item) {
	s := &a.cu
	d := s.draft
	if d.Transport != "http" && d.Transport != "sse" {
		cmd := strings.TrimSpace(d.Cmd)
		if args := strings.TrimSpace(d.Note); args != "" {
			cmd = strings.TrimSpace(cmd + " " + args)
		}
		d.Cmd = cmd
	}
	env, err := customize.ParseEnvLines(s.mcpEnv)
	if err != nil {
		s.err = err.Error()
		return
	}
	d.Env = envKeys(env)
	if d.Transport == "http" || d.Transport == "sse" {
		hdr, err := customize.ParseHeaderLines(s.mcpHdr)
		if err != nil {
			s.err = err.Error()
			return
		}
		d.Headers = hdr
	}
	if err := st.Save(customize.MCP, it, d); err != nil {
		s.err = err.Error()
		return
	}
	if ds := a.cuDisk(); ds != nil {
		if err := ds.SaveMCP(it, d, env); err != nil {
			s.err = "保存到磁盘失败: " + err.Error()
			log.Warn("MCP 保存失败", "name", it.Name, "err", err)
			return
		}
		// 保存后重新检测连接状态:要跑 claude mcp list 子进程(实测 130 ms ~ 1 s,上限 45 s),不能卡在界面线程里。
		// 必须走「后台只取、结果回主线程再写」的 mcpFetchListBG:此前这里在后台线程直接调 ds.RefreshMCPStatus()
		// (注释写明「只在测试里用」),它会在后台改界面正读的条目;连续保存几台就并发几份,实测引发进程静默崩溃。
		a.mcpFetchListBG(true)
	}
	it.EnvVals = env
	*it = d
	s.mcpEdit, s.dOf, s.err = false, nil, ""
	a.toast = "已保存 MCP 配置"
	log.Info("保存 MCP 配置", "name", it.Name)
}

func envKeys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
