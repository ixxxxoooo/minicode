package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// DemoWorkspaceSpec 供 app.seedDemoRegistry 注册额外工作区(主代理最小接线)。
type DemoWorkspaceSpec struct {
	ID, Name string
	Paths    []string
}

// DemoWorkspaces 演示模式可注册的扩展工作区清单。
func DemoWorkspaces() []DemoWorkspaceSpec {
	paths := []string{}
	for _, f := range WorkspaceFoldersLarge() {
		paths = append(paths, f.Path)
	}
	return []DemoWorkspaceSpec{
		{ID: "ws-large", Name: "多仓库 · 压测 12 文件夹", Paths: paths},
	}
}

// wsExtraFolders 全栈工作区额外成员:空仓库 / 无提交 / 同步失败 / 慢同步 / 远落后等。
func wsExtraFolders() []WsFolder {
	syncFail := repoSyncFail()
	syncSlow := repoSyncSlow()
	far := repoFarBehind()
	empty := repoEmpty()
	unborn := repoUnborn()
	return []WsFolder{
		{Name: "repo-empty", Path: "/demo/repo-empty", FS: memFS("repo-empty", map[string]string{"README.md": "# empty-demo\n"}), Repo: empty, Git: map[string]string{}},
		{Name: "repo-unborn", Path: "/demo/repo-unborn", FS: memFS("repo-unborn", map[string]string{"README.md": "# first commit pending\n"}), Repo: unborn, Git: map[string]string{"README.md": "U"}},
		{Name: "sync-fail", Path: "/demo/sync-fail", FS: memFS("sync-fail", map[string]string{"src/lib.go": "package lib\n// local edits\n"}), Repo: syncFail,
			Git: map[string]string{"src/lib.go": "M"}, Head: func(p string) (string, bool) { return syncFail.Show("HEAD", p) }},
		{Name: "sync-slow", Path: "/demo/sync-slow", FS: memFS("sync-slow", map[string]string{"pkg/a.go": "package a\n// syncing\n"}), Repo: syncSlow,
			Git: map[string]string{"pkg/a.go": "M"}, Head: func(p string) (string, bool) { return syncSlow.Show("HEAD", p) }},
		{Name: "far-behind", Path: "/demo/far-behind", FS: memFS("far-behind", map[string]string{"README.md": "# old\n"}), Repo: far, Git: map[string]string{}},
	}
}

// WorkspaceFoldersLarge 十二个文件夹:压测 SCM 列表 / 树 / 总览行。
func WorkspaceFoldersLarge() []WsFolder {
	base := WorkspaceFolders()
	extra := wsExtraFolders()
	out := append([]WsFolder(nil), base...)
	out = append(out, extra...)
	// 补足到 12:用轻量占位仓库
	for len(out) < 12 {
		i := len(out) + 1
		name := fmt.Sprintf("svc-%02d", i)
		path := fmt.Sprintf("/demo/%s", name)
		repo := vcs.NewMem()
		repo.SetState([]vcs.Change{{Path: "main.go", Status: 'M', Add: i}}, []vcs.Commit{
			{Hash: fmt.Sprintf("%06x", i*999), Subject: "init " + name, Author: "Lin", When: "昨天", Refs: []string{"HEAD", "main"}},
		}, nil, map[string]string{"main.go": "package main\n"}, i%4)
		repo.SetAheadBehind(i%4, i%7)
		head := repo
		out = append(out, WsFolder{
			Name: name, Path: path,
			FS:   memFS(name, map[string]string{"main.go": fmt.Sprintf("package main\n// svc %d\n", i)}),
			Repo: head, Git: map[string]string{"main.go": "M"},
			Head: func(p string) (string, bool) { return head.Show("HEAD", p) },
		})
	}
	return out[:12]
}
