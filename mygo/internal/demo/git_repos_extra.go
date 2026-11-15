package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// repoEmpty 干净仓库(有提交、无改动)。
func repoEmpty() *vcs.Mem {
	m := vcs.NewMem()
	m.SetState(nil, []vcs.Commit{
		{Hash: "c0ffee0", Subject: "chore: baseline", Author: "Lin", When: "昨天", Refs: []string{"HEAD", "main"}},
	}, nil, map[string]string{"README.md": "# empty-demo\n"}, 0)
	m.SetBranch("main", 0)
	return m
}

// repoUnborn 已初始化、尚无任何提交。
func repoUnborn() *vcs.Mem {
	m := vcs.NewMem()
	m.SetState([]vcs.Change{{Path: "README.md", Status: 'U', Add: 2}}, nil, nil, nil, 0)
	m.SetCurrent("main")
	return m
}

// repoSyncFail 同步失败(演示重试)。
func repoSyncFail() *vcs.Mem {
	m := vcs.NewMem()
	m.SetState([]vcs.Change{{Path: "src/lib.go", Status: 'M', Add: 2}}, []vcs.Commit{
		{Hash: "ab12cd3", Subject: "init", Author: "你", When: "1 小时", Refs: []string{"HEAD", "feat/enterprise/integration-gateway"}},
	}, nil, map[string]string{"src/lib.go": "package lib\n"}, 2)
	m.SetBranch("feat/enterprise/integration-gateway", 0)
	m.SetFail("Pull", "网络超时:无法连接 origin", true)
	return m
}

// repoSyncSlow 正在同步(长延迟)。
func repoSyncSlow() *vcs.Mem {
	m := vcs.NewMem()
	m.SetState([]vcs.Change{{Path: "pkg/a.go", Status: 'M', Add: 1}}, []vcs.Commit{
		{Hash: "slow001", Subject: "wip", Author: "王明", When: "刚刚", Refs: []string{"HEAD", "main"}},
	}, nil, map[string]string{"pkg/a.go": "package a\n"}, 0)
	m.SetDelay(3 * time.Second)
	m.SetAheadBehind(0, 0)
	return m
}

// repoFarBehind 落后很多、只落后不领先。
func repoFarBehind() *vcs.Mem {
	m := vcs.NewMem()
	m.SetState(nil, []vcs.Commit{
		{Hash: "be90001", Subject: "legacy", Author: "Wen", When: "去年", Refs: []string{"HEAD", "main"}},
	}, nil, map[string]string{"README.md": "# old\n"}, 0)
	m.SetAheadBehind(0, 87)
	m.SetBranch("main", 87)
	return m
}

func memFS(name string, files map[string]string) *workspace.Mem {
	fs := workspace.NewMem(name, files)
	fs.Prefer = []string{"src", "pkg", "demo"}
	return fs
}
