package app

import (
	"sync"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
)

// blockDiff 是一个差异块算好的行级差异与增删行数。
type blockDiff struct {
	lines    []diff.Line
	add, del int
	// 校验字段:块一般不可变,但万一内容被改动,不会读到旧结果。
	oldLen, newLen, line int
}

// blockDiffMax 缓存上限;超过就整体清空重来(会话里的差异块数量远低于它,清空只是兜底防膨胀)。
const blockDiffMax = 2048

var blockDiffCache = struct {
	sync.Mutex
	m map[*core.DiffBlock]*blockDiff
}{m: map[*core.DiffBlock]*blockDiff{}}

// blockDiffOf 返回差异块的行级差异(带缓存)。
// 改动卡、编辑器叠层、状态栏、会话流差异卡每帧都要它;diff.Lines 是 O(n·m) 的 LCS,
// 在长会话里一次重算要几十毫秒(日志「会话流行构建偏慢 chg:u30 31.86ms」),
// 所以按块指针缓存。返回的切片只读,调用方不要修改。
func blockDiffOf(d *core.DiffBlock) *blockDiff {
	blockDiffCache.Lock()
	defer blockDiffCache.Unlock()
	if e, ok := blockDiffCache.m[d]; ok && e.oldLen == len(d.Old) && e.newLen == len(d.New) && e.line == d.Line {
		return e
	}
	if len(blockDiffCache.m) >= blockDiffMax {
		blockDiffCache.m = map[*core.DiffBlock]*blockDiff{}
	}
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	ad, de := diff.Stat(ls)
	e := &blockDiff{lines: ls, add: ad, del: de, oldLen: len(d.Old), newLen: len(d.New), line: d.Line}
	blockDiffCache.m[d] = e
	return e
}

// blockLines 是 blockDiffOf(d).lines 的简写。
func blockLines(d *core.DiffBlock) []diff.Line { return blockDiffOf(d).lines }
