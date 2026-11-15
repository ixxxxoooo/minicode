package demo

import (
	"sort"
	"sync"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// Scenario 是一个演示会话场景:侧栏里的一行 + 打开它时回放 / 播放的剧本。
// 各 scn_*.go 文件在 init 里调 Register 登记,互不冲突;侧栏与 Session() 都读这张表。
type Scenario struct {
	ID    string // 会话 id,全局唯一;与 projects.go 里已有的 id 重名时,只替换剧本、不重复出现在侧栏
	Title string
	// Project 为空 = 归入 scenario-lab(场景实验室);也可指定 router-demo / web-dashboard / docs-site。
	Project string
	Label   string // 侧栏时间文案,默认「刚刚」
	Group   string // 今天 / 昨天 / 更早,默认「今天」
	Clock   string // 方案 B 行尾时间,默认同 Label
	// Live 为真时按节奏播放(带 After 延时,演示进行中的状态);否则打开即一次性回放。
	Live bool
	// Auto 非空时启动后自动发出这条提示。
	Auto string
	// Build 在打开会话时调用一次,返回剧本。
	Build func() []Step
}

var (
	regMu     sync.Mutex
	scenarios []Scenario
)

// Register 登记一个演示场景(在 init 里调用)。
func Register(s Scenario) {
	regMu.Lock()
	defer regMu.Unlock()
	scenarios = append(scenarios, s)
}

// Scenarios 返回全部已登记的场景(按 Group → 登记顺序稳定排序)。
func Scenarios() []Scenario {
	regMu.Lock()
	defer regMu.Unlock()
	out := append([]Scenario(nil), scenarios...)
	rank := map[string]int{"今天": 0, "昨天": 1, "更早": 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[groupOf(out[i])] < rank[groupOf(out[j])] })
	return out
}

func groupOf(s Scenario) string {
	if s.Group == "" {
		return "今天"
	}
	return s.Group
}

func findScenario(id string) (Scenario, bool) {
	regMu.Lock()
	defer regMu.Unlock()
	for _, s := range scenarios {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

// sessionInfo 把场景转成侧栏的一行。
func (s Scenario) sessionInfo() core.SessionInfo {
	label := s.Label
	if label == "" {
		label = "刚刚"
	}
	clock := s.Clock
	if clock == "" {
		clock = label
	}
	return core.SessionInfo{ID: s.ID, Title: s.Title, Label: label, Group: groupOf(s), Clock: clock}
}
