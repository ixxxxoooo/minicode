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
}

// ErrInvalidName 分支名不合法。
var ErrInvalidName = errors.New("分支名不合法")

var badName = regexp.MustCompile(`(^[-/.])|[\s~^:?*\[\\]|\.\.|@\{|//|\.lock$|[/.]$`)

// ValidName 按 git check-ref-format 的主要规则判断分支名是否合法。
func ValidName(n string) bool {
	return n != "" && n != "@" && !badName.MatchString(n) && !strings.ContainsAny(n, "\x00\x7f")
}
