// 演示数据索引(侧栏 · 项目与会话)
//
//   - 基础四项目 + 下方 ExtraProjects:导航栏「会话」列表(方案 A/C 按项目折叠;方案 B 只看当前项目)。
//   - 120 会话项目 mega-sessions:展开后见时间分组(今天/昨天/更早/去年标题混排);Count=120,方案 A 每项目只预览 4 条。
//   - 超长名/路径项目 longpath-demo、同名不同路径 dup-router-a/b:项目行名称与路径展示。
//   - gone-demo 路径不在 Exists 白名单:添加项目对话框会报找不到;侧栏仍显示该行(失效态 UI 未接)。
//   - plain-notes 非 Git:侧栏「非仓库」标签(isRepoDir)。
//   - archive-vault 仅归档会话:需展开底部「已归档 N」。
//   - 空项目 empty-lab:只在 seed 的 registry.Projects,不在本表(侧栏补一行无会话项目)。
package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ExtraProjects 返回附加的侧栏项目(合并进 Projects,位于 scenario 合并之前)。
func ExtraProjects() []core.ProjectInfo {
	out := []core.ProjectInfo{
		megaProject(),
		longPathProject(),
		dupProjectA(),
		dupProjectB(),
		goneProject(),
		plainNotesProject(),
		archiveVaultProject(),
		statusLabProject(),
	}
	return out
}

func sess(id, title, label, group, clock string) core.SessionInfo {
	if group == "" {
		group = "今天"
	}
	if clock == "" {
		clock = label
	}
	return core.SessionInfo{ID: id, Title: title, Label: label, Group: group, Clock: clock}
}

func megaProject() core.ProjectInfo {
	const n = 120
	ss := make([]core.SessionInfo, 0, n)
	titles := []string{
		"修复路由大小写", "补充集成测试", "梳理 README", "重构错误处理",
		"🔧 工具链升级", "发布说明草稿", "【紧急】线上 502", "调研 actix 路由",
		"这是一段非常非常非常非常非常非常非常非常非常非常长的会话标题用来测试侧栏省略号与搜索命中",
		"代码审查 · handlers", "MCP 连通性排查", "画布架构图调整",
	}
	groups := []struct {
		g, pref string
		cnt     int
	}{
		{"今天", "刚刚", 18},
		{"今天", "5 分钟", 12},
		{"昨天", "昨天", 22},
		{"更早", "周一", 28},
		{"更早", "去年", 40},
	}
	ix := 0
	for _, blk := range groups {
		for j := 0; j < blk.cnt; j++ {
			title := titles[ix%len(titles)]
			if j%7 == 3 {
				title = fmt.Sprintf("%s · #%d", title, ix+1)
			}
			label := blk.pref
			clock := blk.pref
			if blk.g == "今天" && j > 0 {
				label = fmt.Sprintf("%d 分钟", (j%50)+1)
				clock = label
			}
			if blk.g == "更早" && blk.pref == "去年" {
				label = fmt.Sprintf("2025-%02d", (j%11)+1)
				clock = label
			}
			ss = append(ss, sess(fmt.Sprintf("mega-%03d", ix+1), title, label, blk.g, clock))
			ix++
		}
	}
	return core.ProjectInfo{
		ID: "mega-sessions", Name: "mega-sessions", Path: "/demo/mega-sessions",
		Count: n, Sessions: ss,
	}
}

func longPathProject() core.ProjectInfo {
	name := "enterprise-platform-国区-🚀-monorepo"
	path := "/demo/这是一个层级很深的演示路径/华东区域/平台组/frontend/enterprise-platform-国区-🚀-monorepo"
	return core.ProjectInfo{
		ID: "longpath-demo", Name: name, Path: path, Count: 2,
		Sessions: []core.SessionInfo{
			sess("long-s1", "对齐设计令牌与主题", "2 小时", "今天", "2 小时"),
			sess("long-s2", "脚手架:pnpm workspace", "昨天", "昨天", "18:40"),
		},
	}
}

func dupProjectA() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "dup-router-a", Name: "router-demo", Path: "/demo/router-demo-copy-a", Count: 1,
		Sessions: []core.SessionInfo{sess("dup-a-1", "分叉实验:路径 A", "3 小时", "今天", "3 小时")},
	}
}

func dupProjectB() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "dup-router-b", Name: "router-demo", Path: "/demo/router-demo-copy-b", Count: 1,
		Sessions: []core.SessionInfo{sess("dup-b-1", "分叉实验:路径 B", "周一", "更早", "周一")},
	}
}

func goneProject() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "gone-demo", Name: "旧笔记本上的仓库", Path: "/Volumes/OldDisk/Projects/removed-repo", Count: 1,
		Sessions: []core.SessionInfo{
			sess("gone-s1", "最后一次会话(路径已失效)", "10月", "更早", "10月12日"),
		},
	}
}

func plainNotesProject() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "plain-notes", Name: "plain-notes", Path: "/demo/plain-notes", Count: 2,
		Sessions: []core.SessionInfo{
			sess("notes-1", "整理会议纪要", "刚刚", "今天", "刚刚"),
			sess("notes-2", "旅行清单", "昨天", "昨天", "21:00"),
		},
	}
}

func archiveVaultProject() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "archive-vault", Name: "archive-vault", Path: "/demo/archive-vault", Count: 3,
		Sessions: []core.SessionInfo{
			sess("vault-1", "季度复盘(已归档)", "上月", "更早", "9月30日"),
			sess("vault-2", "旧版迁移记录", "去年", "更早", "2024-12-01"),
			sess("vault-3", "实验性分支探索", "10月", "更早", "10月5日"),
		},
	}
}

// statusLabProject 收纳侧栏状态相关的演示会话(运行/待处理/出错等剧本在 misc_scenarios 登记)。
func statusLabProject() core.ProjectInfo {
	return core.ProjectInfo{
		ID: "status-lab", Name: "status-lab", Path: "/demo/status-lab", Count: 5,
		Sessions: []core.SessionInfo{
			sess("side-run", "演示 · 运行中(后台保活)", "刚刚", "今天", "刚刚"),
			sess("side-wait", "演示 · 等待权限", "2 分钟", "今天", "2 分钟"),
			sess("side-err", "演示 · 刚出错", "8 分钟", "今天", "8 分钟"),
			sess("side-unread", "演示 · 已完成未读", "半小时", "今天", "半小时"),
			sess("side-ok", "演示 · 普通已完成", "1 小时", "今天", "1 小时"),
		},
	}
}

// RegistryManualProjects 手动添加、无 Claude 会话的项目路径(进 registry.Projects)。
func RegistryManualProjects() []string {
	return []string{
		"/demo/empty-lab",
	}
}

// NonRepoPaths 演示里应显示「非仓库」标签的目录。
func NonRepoPaths() map[string]bool {
	return map[string]bool{
		"/demo/plain-notes": true, "/demo/empty-lab": true, "/demo/design-assets": true,
		"/demo/mega-sessions": true,
	}
}

// ExistsDemoPath 演示模式路径存在性(用于 projects.Store.Exists)。
func ExistsDemoPath(p string, known map[string]bool) bool {
	if strings.HasPrefix(p, "/demo/") {
		if p == goneProject().Path {
			return false
		}
		return true
	}
	return known[p]
}
