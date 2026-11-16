package claude

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 命令队列满了也不能丢命令(尤其是提示词),并且保持投递顺序。
func TestSendNeverDropsAndKeepsOrder(t *testing.T) {
	b := &Backend{cmds: make(chan core.Command, 2), done: make(chan struct{})}
	defer b.Close()
	for i := 0; i < 10; i++ {
		b.Send(core.CmdPrompt{Text: string(rune('a' + i))})
	}
	for i := 0; i < 10; i++ {
		select {
		case c := <-b.cmds:
			if got := c.(core.CmdPrompt).Text; got != string(rune('a'+i)) {
				t.Fatalf("第 %d 条顺序不对:%s", i, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("第 %d 条丢失", i)
		}
	}
}

// 环境变量没有变化不能触发任何进程操作;受管键之外的变量原样保留。
func TestSetEnvNoChangeIsNoop(t *testing.T) {
	b := &Backend{cfg: Config{Env: []core.EnvVar{{Key: "ANTHROPIC_BASE_URL", Value: "http://a"}, {Key: "FOO", Value: "1"}}}, reqs: map[string]req{}}
	b.onSetEnv(core.CmdSetEnv{Env: []core.EnvVar{{Key: "ANTHROPIC_BASE_URL", Value: "http://a"}}})
	if len(b.reqs) != 0 || b.envStale {
		t.Fatal("没有变化不应发请求")
	}
	// 有变化但还没有进程:只更新配置,启动时自然带上
	b.onSetEnv(core.CmdSetEnv{Env: []core.EnvVar{{Key: "ANTHROPIC_BASE_URL", Value: "http://b"}}})
	if len(b.reqs) != 0 {
		t.Fatal("没有进程不应发请求")
	}
	var foo, base string
	for _, e := range b.cfg.Env {
		switch e.Key {
		case "FOO":
			foo = e.Value
		case "ANTHROPIC_BASE_URL":
			base = e.Value
		}
	}
	if foo != "1" || base != "http://b" {
		t.Fatalf("配置合并不对:%+v", b.cfg.Env)
	}
}
