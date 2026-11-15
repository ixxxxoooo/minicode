package app

import (
	"sort"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

const welcomeRecentN = 3 // 设计稿示意 2 条,取 3:够回到上次的工作,又不把起手卡挤出屏幕

type welcomeRecent struct {
	ID, Title, When string
}

// welcomeRecentRows 取最近 N 条会话(跳过已归档),按更新时间倒序。
// 标题写的是「在 <项目> 里做点什么」,所以优先只列当前项目的会话;当前项目还没有会话时才跨项目取。
func (a *App) welcomeRecentRows() []welcomeRecent {
	type row struct {
		s core.SessionInfo
	}
	var all, here []row
	for _, p := range a.projects {
		cur := a.isCurProject(p)
		for _, s := range p.Sessions {
			if s.Archived || s.ID == "" {
				continue
			}
			all = append(all, row{s})
			if cur {
				here = append(here, row{s})
			}
		}
	}
	if len(here) > 0 {
		all = here
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].s.Updated > all[j].s.Updated
	})
	if len(all) > welcomeRecentN {
		all = all[:welcomeRecentN]
	}
	out := make([]welcomeRecent, 0, len(all))
	now := a.now()
	for _, r := range all {
		when := r.s.Label
		if when == "" && r.s.Updated > 0 {
			when = relTime(now, time.UnixMilli(r.s.Updated))
		}
		if when == "" {
			when = r.s.Clock
		}
		out = append(out, welcomeRecent{ID: r.s.ID, Title: core.ShortTitle(r.s.Title), When: when})
	}
	return out
}
