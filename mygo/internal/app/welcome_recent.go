package app

import (
	"sort"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

const welcomeRecentN = 5

type welcomeRecent struct {
	ID, Title, When string
}

// welcomeRecentRows 跨项目取最近 N 条会话(跳过已归档),按更新时间倒序。
func (a *App) welcomeRecentRows() []welcomeRecent {
	type row struct {
		s core.SessionInfo
	}
	var all []row
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.Archived || s.ID == "" {
				continue
			}
			all = append(all, row{s})
		}
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
