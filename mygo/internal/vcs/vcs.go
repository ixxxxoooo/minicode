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
	Push() error
}

// Change 是一个文件的改动。
type Change struct {
	Path     string
	Status   byte // M 修改 / A 新增 / D 删除 / R 重命名 / U 未跟踪
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

// ErrInvalidName 分支名不合法。
var ErrInvalidName = errors.New("分支名不合法")

var badName = regexp.MustCompile(`(^[-/.])|[\s~^:?*\[\\]|\.\.|@\{|//|\.lock$|[/.]$`)

// ValidName 按 git check-ref-format 的主要规则判断分支名是否合法。
func ValidName(n string) bool {
	return n != "" && n != "@" && !badName.MatchString(n) && !strings.ContainsAny(n, "\x00\x7f")
}
