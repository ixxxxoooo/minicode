// Package vcs 是版本控制(Git)的统一入口:界面只依赖 Repo 接口,
// 演示模式用内存实现(Mem),真实模式走 git 命令行(CLI)。
package vcs

import (
	"errors"
	"regexp"
	"strings"
)

// Branch 是一个分支。
type Branch struct {
	Name    string
	Remote  bool   // 远程分支(origin/xxx)
	Current bool   // 当前检出的分支
	When    string // 最近提交的相对时间,如「2 小时前」
}

// Repo 是一个仓库。所有方法可在任意 goroutine 调用;界面侧应放到后台线程执行慢操作。
type Repo interface {
	// Current 当前分支名(游离 HEAD 返回短哈希)。
	Current() string
	// Branches 列出本地与远程分支;远程里已有同名本地分支的不重复列出。
	Branches() ([]Branch, error)
	// Switch 切换到分支(远程分支会建立跟踪的本地分支)。
	Switch(name string) error
	// Create 以当前提交为起点新建并切换到分支。
	Create(name string) error

	// Status 工作区状态:暂存区与工作区各一份清单(同一文件可同时出现在两边)。
	Status() ([]Change, error)
	Stage(paths ...string) error
	Unstage(paths ...string) error
	// Discard 放弃工作区改动(未跟踪文件则删除)。
	Discard(path string) error
	// Commit 提交暂存区;amend 为真时修改上一次提交。
	Commit(msg string, amend bool) error
	// Log 最近 n 条提交(所有分支,按时间序,含父提交与引用名)。
	Log(n int) ([]Commit, error)
	// CommitFiles 某次提交涉及的文件。
	CommitFiles(hash string) ([]Change, error)
	// Show 返回某版本里文件的内容;rev 为 "HEAD" 等,不存在时 ok=false。
	Show(rev, path string) (content string, ok bool)
	// AheadBehind 相对上游分支领先 / 落后的提交数(没有上游返回 0, 0)。
	AheadBehind() (ahead, behind int)
	Pull() error
	PullRebase() error
	PullMerge() error
	RebaseAbort() error
	// MergeAbort 中止进行中的合并(git merge --abort)。
	MergeAbort() error
	// OpState 进行中的多步操作:"" / "merge" / "rebase"(按仓库真实状态判断,重启后也能接上)。
	OpState() string
	Push() error
	// PushForce 强制推送(带 lease);仅在用户明确确认后调用。
	PushForce() error

	// Index 暂存区里文件的内容(没有暂存过就是 HEAD 的);StageContent 把内容写成暂存区里的版本(块级暂存的落点)。
	Index(path string) (string, bool)
	StageContent(path, content string) error
	// FileLog 文件的提交历史(跟随重命名);Blame 逐行追溯(下标 = 行号 - 1)。
	FileLog(path string, n int) ([]Commit, error)
	Blame(path string) ([]BlameLine, error)
	// 储藏。
	Stashes() ([]Stash, error)
	StashPush(msg string) error
	StashApply(i int, drop bool) error
	StashDrop(i int) error
	// StashFiles 储藏涉及的文件(name-status)。
	StashFiles(i int) ([]Change, error)
	// StashFileContent 储藏与工作区父提交之间某文件的内容(old=父提交,new=储藏侧)。
	StashFileContent(i int, path string) (old, nw string, ok bool)
	// CompareCommits 两提交之间的文件差异清单(a...b,按 git diff --name-status)。
	CompareCommits(a, b string) ([]Change, error)
	// 提交上的操作(图谱右键菜单)。
	Checkout(hash string) error
	CherryPick(hash string) error
	RevertCommit(hash string) error
	CreateAt(name, hash string) error
	// UndoCommit 撤销上一次提交,改动留在暂存区(git reset --soft HEAD~1)。
	UndoCommit() error
	// ResetKeep 把当前分支回退到某个提交(git reset --keep):之后的提交从分支上移除,
	// 未提交的改动保留;与回退内容冲突时中止并返回原因。不用 --hard,不丢工作区。
	ResetKeep(hash string) error
}

// StatusConflict 是合并冲突文件的状态字母。
const StatusConflict = 'C'

// Change 是一个文件的改动。
type Change struct {
	Path     string
	Status   byte // M 修改 / A 新增 / D 删除 / R 重命名 / U 未跟踪 / C 合并冲突
	Staged   bool // 在暂存区(否则在工作区)
	Add, Del int
}

// Commit 是一条提交。
type Commit struct {
	Hash, Subject, Author, When string
	Parents                     []string
	Refs                        []string // HEAD / 分支 / 远程分支 / 标签
}

// Short 短哈希。
func (c Commit) Short() string {
	if len(c.Hash) > 7 {
		return c.Hash[:7]
	}
	return c.Hash
}

// ErrInvalidHash 提交哈希不合法。
var ErrInvalidHash = errors.New("提交哈希不合法")

var hashRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// ErrInvalidName 分支名不合法。
var ErrInvalidName = errors.New("分支名不合法")

var badName = regexp.MustCompile(`(^[-/.])|[\s~^:?*\[\\]|\.\.|@\{|//|\.lock$|[/.]$`)

// ValidName 按 git check-ref-format 的主要规则判断分支名是否合法。
func ValidName(n string) bool {
	return n != "" && n != "@" && !badName.MatchString(n) && !strings.ContainsAny(n, "\x00\x7f")
}
