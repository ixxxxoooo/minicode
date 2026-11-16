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
	// refreshing 是正在后台重读的键:TTL 过期后先继续用旧内容,后台重读完再换,
	// 界面线程不会在每个 TTL 周期里卡一次 `git show`。
	refreshing map[string]bool
}

// peek 与 get 类似,但过期的条目也返回(stale=true)。
func (h *headCache) peek(p, tip string, now time.Time) (content string, ok, hit, stale bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tip != tip {
		h.tip, h.m, h.refreshing = tip, nil, nil
		return "", false, false, false
	}
	e, found := h.m[p]
	if !found {
		return "", false, false, false
	}
	return e.content, e.ok, true, now.Sub(e.at) > headCacheTTL
}

// beginRefresh 登记一次后台重读;已有同键重读在进行时返回 false。
func (h *headCache) beginRefresh(p string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.refreshing[p] {
		return false
	}
	if h.refreshing == nil {
		h.refreshing = map[string]bool{}
	}
	h.refreshing[p] = true
	return true
}

func (h *headCache) endRefresh(p string) {
	h.mu.Lock()
	delete(h.refreshing, p)
	h.mu.Unlock()
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
	h.tip, h.m, h.refreshing = "", nil, nil
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
// 真正的读取交给 load(会起 git 子进程,只能用闭包捕获的值,不能碰 App 状态)。
// 第一次读取同步完成(调用方这一帧就要内容);TTL 过期后继续返回旧内容,并在后台重读一次。
func (a *App) headShow(key string, load func() (string, bool)) (string, bool) {
	tip, now := a.headTip(), a.now()
	c, ok, hit, stale := a.headC.peek(key, tip, now)
	if hit && !stale {
		return c, ok
	}
	if hit && a.win != nil {
		if a.headC.beginRefresh(key) {
			a.bg("head-refresh", func() func() {
				defer a.headC.endRefresh(key)
				end := diag.SpanSlow("vcs", "后台重读HEAD内容", 500*time.Millisecond, "path", key)
				nc, nok := load()
				end()
				a.headC.put(key, tip, nc, nok, a.now())
				return func() { a.frameSoon() }
			})
		}
		return c, ok
	}
	end := diag.SpanSlow("vcs", "读取HEAD内容", 200*time.Millisecond, "path", key)
	c, ok = load()
	end()
	a.headC.put(key, tip, c, ok, now)
	return c, ok
}
