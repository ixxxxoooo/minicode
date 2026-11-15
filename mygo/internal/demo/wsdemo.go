package demo

import (
	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// WsFolder 是演示工作区里的一个文件夹:文件系统、仓库(nil = 不是 Git 仓库)与树上的改动标记。
type WsFolder struct {
	Name string
	Path string
	FS   workspace.FS
	Repo vcs.Repo
	Git  map[string]string
	Head func(p string) (string, bool)
}

// WorkspaceFolders 是演示模式「全栈工作区」的成员:
// router-demo(主,main,3 项改动)、web-dashboard(feat/charts,2 项改动,领先 2)、
// docs-site(main,干净,落后 1)、design-assets(不是仓库)。
func WorkspaceFolders() []WsFolder {
	dash := workspace.NewMem("web-dashboard", map[string]string{
		"src/app.ts":              "import { render } from './charts';\n\nrender(document.body);\n",
		"src/charts.ts":           "export function render(root: HTMLElement) {\n  root.innerHTML = '<canvas id=\"c\"></canvas>';\n  drawLine('c');\n}\n\nfunction drawLine(id: string) {}\n",
		"src/components/Card.tsx": "export const Card = () => null;\n",
		"package.json":            "{\n  \"name\": \"web-dashboard\",\n  \"version\": \"0.4.0\"\n}\n",
		"README.md":               "# web-dashboard\n\n指标看板前端。\n",
	})
	dash.Prefer = []string{"src"}
	dashRepo := vcs.NewMem()
	dashRepo.SetState(
		[]vcs.Change{
			{Path: "src/charts.ts", Status: 'M', Add: 6, Del: 2},
			{Path: "src/components/Card.tsx", Status: 'A', Add: 1},
		},
		[]vcs.Commit{
			{Hash: "9c1d2e3", Subject: "feat(charts): 折线图支持缩放", Author: "你", When: "1 小时", Parents: []string{"7a6b5c4"}, Refs: []string{"HEAD", "feat/charts"}},
			{Hash: "7a6b5c4", Subject: "feat(charts): 引入 canvas 渲染", Author: "你", When: "昨天", Parents: []string{"3e2f1a0"}},
			{Hash: "3e2f1a0", Subject: "chore: 初始化看板", Author: "Lin", When: "周一", Refs: []string{"main"}},
		},
		map[string][]vcs.Change{
			"9c1d2e3": {{Path: "src/charts.ts", Status: 'M', Add: 12, Del: 1}},
			"7a6b5c4": {{Path: "src/charts.ts", Status: 'A', Add: 20}},
			"3e2f1a0": {{Path: "package.json", Status: 'A', Add: 4}},
		},
		map[string]string{
			"src/charts.ts":           "export function render(root: HTMLElement) {\n  root.innerHTML = '';\n}\n",
			"src/components/Card.tsx": "",
		}, 2)
	dashRepo.SetBranch("feat/charts", 0)

	docs := workspace.NewMem("docs-site", map[string]string{
		"docs/index.md":   "# 文档站\n\n欢迎。\n",
		"docs/routing.md": "# 路由\n\n路径匹配忽略大小写。\n",
		"config.yaml":     "title: router-demo\n",
	})
	docsRepo := vcs.NewMem()
	docsRepo.SetState(nil,
		[]vcs.Commit{
			{Hash: "b1c2d3e", Subject: "docs: 补充路由章节", Author: "Wen", When: "昨天", Parents: []string{"a0b1c2d"}, Refs: []string{"HEAD", "main"}},
			{Hash: "a0b1c2d", Subject: "chore: 初始化文档站", Author: "Wen", When: "周一", Refs: []string{"origin/main"}},
		},
		map[string][]vcs.Change{"b1c2d3e": {{Path: "docs/routing.md", Status: 'A', Add: 3}}}, nil, 0)
	docsRepo.SetBranch("main", 1)

	assets := workspace.NewMem("design-assets", map[string]string{
		"logo.svg":   "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"32\" height=\"32\"/>\n",
		"palette.md": "# 色板\n\n- 主色 #D97757\n",
	})

	router := Sandbox()
	return []WsFolder{
		{Name: "router-demo", Path: "/demo/router-demo", FS: router, Repo: Repo(), Git: GitStatus(), Head: HeadOf},
		{Name: "web-dashboard", Path: "/demo/web-dashboard", FS: dash, Repo: dashRepo,
			Git:  map[string]string{"src/charts.ts": "M", "src/components/Card.tsx": "A"},
			Head: func(p string) (string, bool) { return dashRepo.Show("HEAD", p) }},
		{Name: "docs-site", Path: "/demo/docs-site", FS: docs, Repo: docsRepo, Git: map[string]string{},
			Head: func(p string) (string, bool) { return docsRepo.Show("HEAD", p) }},
		{Name: "design-assets", Path: "/demo/design-assets", FS: assets},
	}
}

// Found 是「添加项目」对话框里「最近 · 发现」列表的一项(演示数据)。
type Found struct {
	Name, Path, Branch string // Branch 为空 = 不是 Git 仓库
}

// Discovered 演示用的可添加项目。
func Discovered() []Found {
	return []Found{
		{"api-gateway", "~/Code/api-gateway", "main"},
		{"mobile-app", "~/Code/mobile-app", "feat/login"},
		{"notes", "~/Documents/notes", ""},
	}
}
