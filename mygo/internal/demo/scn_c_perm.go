package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "perm", Project: "router-demo", Title: "待处理 · 权限请求全集", Label: "5 分钟",
		Auto:  "再试一条权限:运行 npm test",
		Build: buildPermGallery,
	})
	Register(Scenario{
		ID: "c-perm-live", Title: "待处理 · 权限停在待处理(Live)", Label: "刚刚", Live: true,
		Build: buildPermLivePending,
	})
	Register(Scenario{
		ID: "c-perm-chain", Title: "待处理 · 同回合连续权限", Label: "刚刚",
		Build: buildPermChain,
	})
}

func buildPermGallery() []Step {
	s := NewScript("perm", "待处理 · 权限请求全集")
	uid := s.User("展示各类权限卡:已处理与嵌套")
	s.Say("下面依次回放 Bash / Edit / Write / MCP / WebFetch / 删除文件,以及允许 / 始终允许 / 拒绝三种结果。")

	// Bash · 允许一次
	bid := s.ID("b")
	bash := core.NewToolCall(bid, "Bash", "运行 cargo test router").WithInput(map[string]any{"command": "cargo test router"})
	promotePerm(s, "", bid, bash)
	resolvePerm(s, "", bid, "allow", true, &core.TerminalBlock{Command: "cargo test router", Output: cargoDone, ExitCode: core.Int(0)})

	// Edit · diff 预览 · 本会话始终允许
	eid := s.ID("e")
	edit := core.NewToolCall(eid, "Edit", "编辑 src/router.rs").WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")
	promotePerm(s, "", eid, edit)
	resolvePerm(s, "", eid, "always", true, &core.DiffBlock{Path: "src/router.rs", Old: oldMatch, New: newMatch, Line: 41})

	// Write
	wid := s.ID("w")
	write := core.NewToolCall(wid, "Write", "写入 config/local.json").WithInput(map[string]any{"file_path": "config/local.json"})
	promotePerm(s, "", wid, write)
	resolvePerm(s, "", wid, "allow", true, &core.TextBlock{Text: "已写入 128 字节"})

	// MCP
	mid := s.ID("m")
	mcp := core.NewToolCall(mid, "mcp__github__create_issue", "创建 GitHub Issue").
		WithInput(map[string]any{"title": "路由大小写", "body": "演示 MCP 权限"})
	promotePerm(s, "", mid, mcp)
	resolvePerm(s, "", mid, "allow", true, &core.TextBlock{Text: "Issue #42"})

	// WebFetch
	fid := s.ID("f")
	fetch := core.NewToolCall(fid, "WebFetch", "抓取文档").WithInput(map[string]any{"url": "https://docs.rs/axum/latest/axum/"})
	promotePerm(s, "", fid, fetch)
	resolvePerm(s, "", fid, "allow", true, &core.TextBlock{Text: "<html>…</html>"})

	// 删除文件 · 拒绝
	did := s.ID("d")
	del := core.NewToolCall(did, "Trash", "删除 tmp/cache.bin")
	del.Kind = core.KindDelete
	del.WithInput(map[string]any{"file_path": "tmp/cache.bin"})
	promotePerm(s, "", did, del)
	resolvePerm(s, "", did, "deny", false)

	// 子 Agent 内嵌权限(待处理样式:已允许)
	sub := core.NewSubagent(s.ID("sa"), "Explore", "在子 Agent 内请求权限", core.SubRunning)
	stampSub(sub, 8*time.Second)
	s.Add(0, core.EvAppend{Item: sub})
	nid := s.ID("n")
	nb := core.NewToolCall(nid, "Bash", "子 Agent 内执行 whoami").WithInput(map[string]any{"command": "whoami"})
	promotePerm(s, sub.ID, nid, nb)
	resolvePerm(s, sub.ID, nid, "allow", true, &core.TextBlock{Text: "demo-user"})

	s.Say("权限演示结束。发消息或点自动提示可再触发一条交互式 Bash 权限。")
	finishLab(s, uid, 3)
	s.Add(0, core.EvMcpServers{Servers: demoMcpServers()})
	s.Add(0, core.EvUsage{Usage: demoContextUsage()})
	return s.Steps()
}

func buildPermLivePending() []Step {
	s := NewScript("c-perm-live", "待处理 · 权限停在待处理(Live)")
	uid := s.User("在权限卡处暂停,不要继续")
	s.Think("准备请求危险命令权限。", 2)
	id := s.ID("x")
	tool := core.NewToolCall(id, "Bash", "运行 rm -rf ./target/demo").WithInput(map[string]any{"command": "rm -rf ./target/demo"})
	promotePerm(s, "", id, tool)
	// 故意不 resolve、不 EndTurn,Live 播放停在这里
	s.Add(ms(400), core.EvTurnStats{ID: uid, Stats: *DefaultStats(1)})
	_ = uid
	return s.Steps()
}

func buildPermChain() []Step {
	s := NewScript("c-perm-chain", "待处理 · 同回合连续权限")
	uid := s.User("一条消息里连续出现三个权限")
	s.Say("将依次请求读文件、改文件、跑命令,中途不结束回合。")

	rid := s.ID("r")
	read := core.NewToolCall(rid, "Read", "读取 Cargo.toml").WithInput(map[string]any{"file_path": "Cargo.toml"})
	promotePerm(s, "", rid, read)
	resolvePerm(s, "", rid, "allow", true, &core.TextBlock{Text: "[package]\nname = \"router-demo\""})

	eid := s.ID("e")
	edit := core.NewToolCall(eid, "Edit", "编辑 src/lib.rs").WithInput(map[string]any{"file_path": "src/lib.rs"})
	promotePerm(s, "", eid, edit)
	resolvePerm(s, "", eid, "always", true, &core.DiffBlock{Path: "src/lib.rs", Old: "// old", New: "// new", Line: 1})

	bid := s.ID("b")
	bash := core.NewToolCall(bid, "Bash", "运行 cargo fmt").WithInput(map[string]any{"command": "cargo fmt --check"})
	promotePerm(s, "", bid, bash)
	// 第三条故意留在待处理
	s.Add(ms(200), core.EvNotice{Level: core.NoticeInfo, Text: "第三条权限停在待处理,可当面板上操作。"})
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 20}})
}
