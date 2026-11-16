package app

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var provLog = diag.L("provider")

// modelSwitchToastNote 会话内换模型/服务商时 toast 的统一下半句。
func modelSwitchToastNote() string {
	return "。新会话与下一条消息起生效,正在进行的回合不受影响"
}

// provState 是「模型服务商」的界面状态与数据(设置页与输入框的模型选择器共用)。
// 真实模式读写 ~/.minicode/providers.json 与 provider-keys.json;演示模式只在内存里,绝不落盘。
type provState struct {
	loaded bool
	store  claudecfg.Store
	keys   map[string]string // 真实 = 磁盘上的密钥;演示 = 占位值(只表示「有没有」)
	busy   map[string]int    // 测试连接中的服务商 id

	// 编辑区(新增与编辑共用)
	editing  bool
	isNew    bool
	draft    claudecfg.Provider
	keyDraft string // 新输入的密钥;留空 = 保持原值
	keyShow  bool
	errName  string
	errURL   string
	shakeAt  time.Time
	editSeq  int // 每次打开编辑区加一,用来丢弃过期的异步结果
	// 模型清单
	fetching    bool
	fetchErr    string
	unsupported bool
	filter      string // 过滤框
	manual      string // 手动添加框
	manualErr   string
	focusManual bool
	// 输入框模型选择器里被折叠的服务商组
	folded map[string]bool
	// 映射下拉的搜索词
	selQ string
}

// pvLoad 首次使用时读入数据。
func (a *App) pvLoad() *provState {
	p := &a.set.pv
	if p.loaded {
		return p
	}
	p.loaded = true
	p.busy = map[string]int{}
	p.folded = map[string]bool{}
	a.loadCfgFolded(p)
	if a.demoWs() {
		p.store = demo.DemoProviderStore()
		p.keys = demo.DemoProviderKeys()
		return p
	}
	st, err := claudecfg.LoadStore()
	if err != nil {
		provLog.Warn("读取服务商配置失败", "err", err)
	}
	p.store = st
	p.keys, _ = claudecfg.LoadKeys()
	if p.keys == nil {
		p.keys = map[string]string{}
	}
	p.store.SanitizePicks()
	return p
}

// pvStore 返回当前的服务商配置(按值,调用方不要改)。
func (a *App) pvStore() claudecfg.Store { return a.pvLoad().store }

// pvHasKey 报告服务商是否已填密钥。
func (a *App) pvHasKey(id string) bool { return strings.TrimSpace(a.pvLoad().keys[id]) != "" }

// pvPersist 把配置写盘(演示 / 从属窗口不写)。
func (a *App) pvPersist() {
	p := a.pvLoad()
	if a.demoWs() || a.secondary {
		return
	}
	if err := claudecfg.SaveStore(p.store); err != nil {
		provLog.Error("保存服务商配置失败", "err", err)
		a.toast = "保存服务商配置失败:" + err.Error()
	}
}

// pvSetKey 保存密钥(空 = 删除);演示只改内存里的「有没有」。
func (a *App) pvSetKey(id, key string) {
	p := a.pvLoad()
	key = strings.TrimSpace(key)
	if key == "" {
		delete(p.keys, id)
	} else if a.demoWs() {
		p.keys[id] = "demo-key"
	} else {
		p.keys[id] = key
	}
	if a.demoWs() || a.secondary {
		return
	}
	if err := claudecfg.SetProviderKey(id, key); err != nil {
		provLog.Error("保存服务商密钥失败", "id", id, "err", err)
		a.toast = "保存密钥失败:" + err.Error()
	}
}

// pvActive 返回当前服务商:id 空 = Claude 官方。
func (a *App) pvActive() (id string, p *claudecfg.Provider) {
	st := a.pvStore()
	id = st.ActiveID()
	if id == "" {
		return "", nil
	}
	return id, st.Get(id)
}

// pvActivate 切换当前服务商(provider 空 = 回到 Claude 官方),model 写进「默认」槽位。
// 持久化并把新环境推给当前会话:空闲时下一条消息起生效;回合进行中只存配置,不打断。
func (a *App) pvActivate(provider, model string) {
	p := a.pvLoad()
	p.store.Activate(provider, model)
	a.pvPersist()
	a.pvPushEnv()
	if provider == "" {
		provLog.Info("切换到 Claude 官方登录")
		return
	}
	provLog.Info("切换服务商", "provider", provider, "model", model)
}

// pvActiveModel 第三方服务商生效时它的默认模型 id;官方登录(或没选)返回空串。
func (a *App) pvActiveModel() string {
	id, p := a.pvActive()
	if id == "" || p == nil {
		return ""
	}
	return strings.TrimSpace(a.pvStore().Mapping.Default)
}

// pvPushEnv 把当前映射对应的环境变量发给活动会话的后端(演示后端会忽略)。
// 后端对这条命令的处理是:结束当前进程、下一条消息用新环境恢复会话,所以回合进行中不发:
// 记成「待推送」,下一条消息发出前补发(此前回合进行中切换服务商会被永远丢掉,新消息还跑在旧环境上)。
func (a *App) pvPushEnv() {
	if a.demoWs() {
		return
	}
	if a.st.Phase == core.PhaseRunning {
		a.pvEnvPending = true
		return
	}
	a.pvEnvPending = false
	p := a.pvLoad()
	var env []core.EnvVar
	for _, e := range p.store.Env(p.keys) {
		env = append(env, core.EnvVar{Key: e.Key, Value: e.Value})
	}
	a.send(core.CmdSetEnv{Env: env})
}

// pvModelLabel 是输入框与状态栏的模型名:官方沿用 claude 报的名称;第三方显示「服务商 · 模型」。
// 返回 (服务商名, 模型 ID);服务商名为空表示官方。
func (a *App) pvModelLabel() (string, string) {
	id, p := a.pvActive()
	if id == "" || p == nil {
		return "", ""
	}
	m := strings.TrimSpace(a.pvStore().Mapping.Default)
	return p.Name, m
}

// pvOfficialPick 运行配置弹层里官方分组记住的模型;无效时退回当前 st.Model。
func (a *App) pvOfficialPick() string {
	st := a.pvStore()
	if st.Picks != nil {
		if m := strings.TrimSpace(st.Picks[claudecfg.PickOfficial]); m != "" {
			for _, mi := range a.st.Models {
				if mi.ID == m {
					return m
				}
			}
		}
	}
	if a.st.Model != "" {
		return a.st.Model
	}
	return "sonnet"
}

// pvSetOfficialPick 记住官方模型选择(演示只改内存)。
func (a *App) pvSetOfficialPick(model string) {
	p := a.pvLoad()
	p.store.SetPick(claudecfg.PickOfficial, model)
	a.pvPersist()
}

// pvModelChipText 第三方运行配置胶囊:短文案 + tooltip「服务商 · 模型 ID」。
func (a *App) pvModelChipText() (short, tip string) {
	prov, m := a.pvModelLabel()
	if prov == "" {
		return "", ""
	}
	if m == "" {
		return "未选模型", prov + " · —"
	}
	tip = prov + " · " + m
	short = m
	if len([]rune(m)) > 16 {
		short = provAbbr(prov)
	}
	return short, tip
}
