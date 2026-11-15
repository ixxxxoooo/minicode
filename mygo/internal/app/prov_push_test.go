package app

import (
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

func TestPvPushEnvSkipsWhenRunning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MINI_HOME", home)
	a, _ := newTestApp(t)
	// 演示模式(内存注册表)下 pvPushEnv 直接返回,这里换成落盘注册表走真实路径
	a.SetRegistry(projects.Open(filepath.Join(t.TempDir(), "ws.json")))
	be := &sinkBE{}
	a.Attach(be)
	p := a.pvLoad()
	p.store = claudecfg.Store{
		Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"m1"}}},
		Mapping:   claudecfg.Mapping{Enabled: true, Provider: "ds", Default: "m1"},
	}
	p.keys["ds"] = "k"
	a.st.Phase = core.PhaseRunning
	a.pvPushEnv()
	if len(be.cmds) != 0 {
		t.Fatalf("回合进行中不应 CmdSetEnv:%v", be.cmds)
	}
	a.st.Phase = core.PhaseIdle
	a.pvPushEnv()
	if len(be.cmds) != 1 {
		t.Fatalf("空闲应推送环境:%v", be.cmds)
	}
	if _, ok := be.cmds[0].(core.CmdSetEnv); !ok {
		t.Fatalf("应是 CmdSetEnv:%T", be.cmds[0])
	}
}
