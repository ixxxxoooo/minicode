// Package core 是与界面无关的数据模型:时间线条目、事件 / 命令、会话状态归约器、后台任务。
//
// 数据流:后端(演示 / 真实 claude)→ Event → State.Apply(归约)→ 界面只读 State;
// 界面的所有交互都转成 Command 发给后端。因此演示与真实后端共用同一套界面,Agent / IDE 两种形态也共用同一个 State。
//
// State 只在界面主线程上读写,没有锁:后端线程通过 Window.Update 把事件投递回主线程再归约。
package core

// PermissionMode 是 Claude Code 的六种权限模式,对应 `--permission-mode`。
type PermissionMode int

const (
	ModeDefault           PermissionMode = iota // 默认:每次危险操作都询问
	ModeAcceptEdits                             // 自动接受文件编辑
	ModePlan                                    // 计划模式:只读,先出计划
	ModeAuto                                    // 自动模式:由分类器决定
	ModeDontAsk                                 // 不再询问(未预先允许的直接拒绝)
	ModeBypassPermissions                       // 绕过全部权限检查
)

// AllModes 按界面循环切换(⇧Tab)的顺序排列。
var AllModes = [...]PermissionMode{ModeDefault, ModeAcceptEdits, ModePlan, ModeAuto, ModeDontAsk, ModeBypassPermissions}

var modeCLI = [...]string{"default", "acceptEdits", "plan", "auto", "dontAsk", "bypassPermissions"}
var modeLabel = [...]string{"逐项确认", "自动接受编辑", "计划模式", "自动模式", "不再询问", "跳过权限确认"}

// CLI 返回传给 `claude --permission-mode` 的参数值。
func (m PermissionMode) CLI() string { return modeCLI[m] }

// Label 返回界面显示名。
func (m PermissionMode) Label() string { return modeLabel[m] }

// Next 循环到下一个模式。
func (m PermissionMode) Next() PermissionMode { return AllModes[(int(m)+1)%len(AllModes)] }

// ParseMode 从 CLI 取值解析,未知值返回 false。
func ParseMode(s string) (PermissionMode, bool) {
	for _, m := range AllModes {
		if m.CLI() == s {
			return m, true
		}
	}
	return ModeDefault, false
}

// Effort 是思考强度,对应 `--effort`。
type Effort int

const (
	EffortLow Effort = iota
	EffortMedium
	EffortHigh
	EffortXHigh
	EffortMax
)

// AllEfforts 由低到高。
var AllEfforts = [...]Effort{EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax}

var effortCLI = [...]string{"low", "medium", "high", "xhigh", "max"}
var effortLabel = [...]string{"低", "中", "高", "超高", "最大"}
var effortHint = [...]string{
	"最快最省,适合简单的小改动和问答",
	"速度与质量均衡,日常默认",
	"更深入地推理,适合多文件改动",
	"很深的推理,更慢更贵,适合复杂排查",
	"最大思考量,最慢最贵,留给最难的问题",
}

func (e Effort) CLI() string   { return effortCLI[e] }
func (e Effort) Label() string { return effortLabel[e] }

// Hint 一句话说明:速度、花费与适用场景。
func (e Effort) Hint() string { return effortHint[e] }

// ModelInfo 是可选模型。
type ModelInfo struct {
	ID          string // 传给 --model 的 id 或别名
	Name        string // 界面显示名
	Description string // 一句话说明
}

// DefaultModels 是内置的模型别名列表(真实模式下由 claude 初始化响应覆盖)。
func DefaultModels() []ModelInfo {
	return []ModelInfo{
		{"opus", "Opus 5.5", "最强推理,适合复杂任务"},
		{"sonnet", "Sonnet 5.5", "速度与能力均衡"},
		{"haiku", "Haiku 5.5", "最快,适合简单任务"},
	}
}
