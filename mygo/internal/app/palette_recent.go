package app

import "strings"

// 命令面板「最近使用」:欢迎列表(无查询)最顶部列出最近执行过的 5 条命令,按时间倒序。
// 记录存在偏好 palette.recent(标题以换行连接,最新在前),重启后仍在。
//
// @author ixxxxoooo

const (
	pPalRecent    = "palette.recent" // 最近执行的命令标题,换行分隔,最新在前
	palRecentKeep = 20               // 偏好里最多留几条(显示只取前 5,留多一些是为了某些命令暂时置灰时能递补)
	palRecentShow = 5                // 欢迎列表里显示几条
	palRecentGrp  = "最近使用"
)

// palRecentTitles 读出记录的命令标题(最新在前)。
// 返回:标题切片;没有记录时为 nil。
func (a *App) palRecentTitles() []string {
	raw := a.prefs.Str(pPalRecent, "")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

// palRecordRecent 把刚执行的命令记到最前面(已有则提前,不重复),超出上限的丢弃。
// 参数:it 刚执行的面板项;只记「命令」组、有标题、且不是「转到第 N 行」这类带动态标题的项。
func (a *App) palRecordRecent(it palItem) {
	if it.group != "命令" || it.title == "" || strings.HasPrefix(it.title, "转到第 ") || strings.Contains(it.title, "\n") {
		return
	}
	out := []string{it.title}
	for _, t := range a.palRecentTitles() {
		if t != it.title {
			out = append(out, t)
		}
	}
	if len(out) > palRecentKeep {
		out = out[:palRecentKeep]
	}
	a.prefs.Set(pPalRecent, strings.Join(out, "\n"))
}

// palRecentItems 取欢迎列表顶部的「最近使用」项,并返回它们在 cmds 里的下标集合(用来从普通命令组去重)。
// 参数:cmds 全部命令。只取当前存在且可执行(未置灰)的命令,最多 palRecentShow 条,最新在前。
// 返回:最近项(group 已改为「最近使用」)、被占用的 cmds 下标。
func (a *App) palRecentItems(cmds []palItem) ([]palItem, map[int]bool) {
	titles := a.palRecentTitles()
	if len(titles) == 0 {
		return nil, nil
	}
	idx := map[string]int{}
	for i, cm := range cmds {
		if _, dup := idx[cm.title]; !dup {
			idx[cm.title] = i
		}
	}
	var out []palItem
	used := map[int]bool{}
	for _, t := range titles {
		i, ok := idx[t]
		if !ok || cmds[i].off || used[i] {
			continue
		}
		it := cmds[i]
		it.group = palRecentGrp
		out = append(out, it)
		used[i] = true
		if len(out) >= palRecentShow {
			break
		}
	}
	return out, used
}
