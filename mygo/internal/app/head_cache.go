package app

import (
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// HEAD 版本内容缓存。
//
// 编辑器行号竖条(syncDiffBase)与迷你地图每帧都要读文件在 HEAD 的内容,原先每次都起一个
// `git show` 子进程:对未跟踪文件还会反复失败(日志里 1 分钟内上千条「git 失败 show HEAD:…」),
// 并且全部发生在界面线程。这里把结果(含「HEAD 里没有这个文件」)缓存起来:
//   - HEAD 提交变了(提交 / 切分支 / 回退 / 拉取后仓库状态刷新,最新提交哈希变化)就整体作废;
//   - 切项目 / 切活动仓库 / 换文件系统时清空;
//   - 再加一个 TTL 兜底,多根时非活动仓库的 HEAD 变化也会在 TTL 内被发现。
const (
	headCacheTTL = 10 * time.Second
	headCacheMax = 256 // 条目上限,超过就整体清空(只缓存打开过的文件,正常用不到)
)

type headEntry struct {
	content string
	ok      bool
	at      time.Time
}

type headCache struct {
	mu  sync.Mutex
	tip string // 缓存对应的活动仓库最新提交哈希(仓库状态没读完时为空)
	m   map[string]headEntry
}

// get 命中且未过期时返回缓存;tip 与缓存不一致视为 HEAD 已变,先清空。
func (h *headCache) get(p, tip string, now time.Time) (content string, ok, hit bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tip != tip {
		h.tip, h.m = tip, nil
		return "", false, false
	}
	e, found := h.m[p]
	if !found || now.Sub(e.at) > headCacheTTL {
		return "", false, false
	}
	return e.content, e.ok, true
}

func (h *headCache) put(p, tip string, content string, ok bool, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tip != tip { // 读取期间 HEAD 已变,这份结果可能是旧的,不存
		return
	}
	if h.m == nil || len(h.m) >= headCacheMax {
		h.m = map[string]headEntry{}
	}
	h.m[p] = headEntry{content: content, ok: ok, at: now}
}

// reset 清空(切项目 / 切仓库 / 换文件系统)。
func (h *headCache) reset() {
	h.mu.Lock()
	h.tip, h.m = "", nil
	h.mu.Unlock()
}

// headTip 活动仓库当前已知的最新提交哈希;仓库状态还没读完时为空。
func (a *App) headTip() string {
	if a.sc.loaded && len(a.sc.log) > 0 {
		return a.sc.log[0].Hash
	}
	return ""
}

// headShow 带缓存地读取「某仓库里某文件在 HEAD 的内容」。key 用文件树路径(多根时全局唯一),
// 真正的读取交给 load(会起 git 子进程)。
func (a *App) headShow(key string, load func() (string, bool)) (string, bool) {
	tip, now := a.headTip(), a.now()
	if c, ok, hit := a.headC.get(key, tip, now); hit {
		return c, ok
	}
	end := diag.SpanSlow("vcs", "读取HEAD内容", 200*time.Millisecond, "path", key)
	c, ok := load()
	end()
	a.headC.put(key, tip, c, ok, now)
	return c, ok
}
