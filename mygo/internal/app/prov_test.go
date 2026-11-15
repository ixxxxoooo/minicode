package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
)

// 服务商:新建自定义、校验、获取模型列表、勾选、映射下拉可选范围、切换;演示模式绝不写盘。
func TestProviderFlowDemo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MINI_HOME", home)
	a, tt := newTestApp(t)
	if !a.demoWs() {
		t.Skip("测试应用不是演示模式")
	}
	a.openSettings("models")
	tt.Frame()
	p := a.pvLoad()
	if len(p.store.Providers) < 3 {
		t.Fatalf("演示数据应有多个服务商:%d", len(p.store.Providers))
	}

	// 新建自定义:缺字段时校验失败并留在编辑区
	a.pvOpenEditor(&claudecfg.Provider{}, true)
	tt.Frame()
	a.pvSaveDraft(false)
	if p.errName == "" || p.errURL == "" || !p.editing {
		t.Fatalf("空表单应报错并保持编辑:name=%q url=%q editing=%v", p.errName, p.errURL, p.editing)
	}
	p.draft.Name, p.draft.BaseURL = "我的网关", "ftp://bad"
	a.pvSaveDraft(false)
	if p.errURL == "" {
		t.Fatal("非 http(s) 地址应报错")
	}

	// 先获取模型列表(演示桩),勾选一个,再保存
	p.draft.BaseURL = "https://gw.example.com/anthropic" // 演示桩:example.com = 401 失败态
	p.keyDraft = "k"
	a.pvFetch()
	tt.Frame()
	if p.fetchErr == "" || p.fetching {
		t.Fatalf("演示的失败态应给出错误原因:%q", p.fetchErr)
	}
	p.draft.BaseURL = "https://gw.test.local/anthropic"
	a.pvFetch()
	tt.Frame()
	if len(p.draft.Available) == 0 {
		t.Fatalf("演示获取应返回模型:err=%q unsupported=%v", p.fetchErr, p.unsupported)
	}
	first := p.draft.Available[0]
	p.draft.ToggleModel(first)
	n := len(p.store.Providers)
	a.pvSaveDraft(false)
	if p.editing || len(p.store.Providers) != n+1 {
		t.Fatalf("保存后应新增一项并关闭编辑区:editing=%v n=%d", p.editing, len(p.store.Providers))
	}
	id := p.store.NewID("我的网关") // 已占用 → 说明保存时用了同一套 id 规则
	if id == "provider" {
		t.Fatalf("id 生成异常:%q", id)
	}
	var added *claudecfg.Provider
	for i := range p.store.Providers {
		if p.store.Providers[i].Name == "我的网关" {
			added = &p.store.Providers[i]
		}
	}
	if added == nil || !added.HasModel(first) || !a.pvHasKey(added.ID) {
		t.Fatalf("新服务商应带着勾选的模型与密钥:%+v", added)
	}

	// 映射下拉只能选已勾选的模型;切换后输入框显示「服务商 · 模型」
	a.pvActivate(added.ID, first)
	tt.Frame()
	if name, model := a.pvModelLabel(); name != "我的网关" || model != first {
		t.Fatalf("标签应是服务商+模型:%q %q", name, model)
	}
	if got := a.pvStore().Get(added.ID).Models; len(got) != 1 || got[0] != first {
		t.Fatalf("映射可选范围应只含已勾选模型:%v", got)
	}
	a.pvActivate("", "")
	if name, _ := a.pvModelLabel(); name != "" {
		t.Fatal("回到官方后不应再显示服务商名")
	}

	// 配置弹层(会话里直接选服务商)能画出来
	a.togglePop("cfg")
	tt.Frame()

	// 演示模式:不写任何文件
	for _, f := range []string{"providers.json", "provider-keys.json"} {
		if _, err := os.Stat(filepath.Join(home, f)); err == nil {
			t.Errorf("演示模式不应写 %s", f)
		}
	}
}

// 没有钥匙 / 没有模型的服务商不能被切过去,选择器里置灰并给原因。
func TestProviderBlockedInPicker(t *testing.T) {
	t.Setenv("MINI_HOME", t.TempDir())
	a, _ := newTestApp(t)
	if !a.demoWs() {
		t.Skip("测试应用不是演示模式")
	}
	st := a.pvStore()
	p := a.pvLoad()
	blocked := 0
	for _, g := range st.PickerGroups(p.keys) {
		if g.Blocked != "" {
			blocked++
		}
	}
	if blocked == 0 {
		t.Fatal("演示数据应包含不可用的服务商(缺密钥 / 清单为空)")
	}
}
