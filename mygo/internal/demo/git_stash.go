package demo

import "github.com/ixxxxoooo/minicode/internal/vcs"

func gitRouterStashes() []vcs.Stash {
	return []vcs.Stash{
		{Ref: "stash@{0}", Msg: "WIP on main: 路由实验", When: "2 小时前", Files: 3},
		{Ref: "stash@{1}", Msg: "index on feat/auth: 只暂存了 Cargo.toml 与 handlers", When: "昨天", Files: 2},
		{Ref: "stash@{2}", Msg: "On main: 未跟踪 bulk 目录试水(含未跟踪文件)", When: "周一", Files: 15},
		{Ref: "stash@{3}", Msg: "autostash: 切换分支前自动储藏 — 这是一条很长很长的说明,用来测列表截断与 tooltip 是否容纳完整文案", When: "3 天前", Files: 8},
		{Ref: "stash@{4}", Msg: "WIP: 合并冲突中间态(请勿 apply)", When: "上周", Files: 5},
	}
}
