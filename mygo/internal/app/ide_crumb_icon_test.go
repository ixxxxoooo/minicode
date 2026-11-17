package app

import (
	"testing"
	"time"
)

func TestCrumbUsesFileTypeIcons(t *testing.T) {
	if treeFolderIcon(".git") != "branch" {
		t.Fatalf(".git 文件夹")
	}
	if fileIconID("main.go") != "badge:GO" {
		t.Fatalf("go 应与树徽标一致")
	}
	if fileIconID("router.rs") != "badge:RS" {
		t.Fatalf("rs 应与树徽标一致")
	}
	if fileIconID("config.json") != "badge:{}" {
		t.Fatalf("json")
	}
	if fileIconID("notes.txt") != "fileText" {
		t.Fatalf("未知扩展名")
	}
}

// 回归:fileMenuIcon 曾在持锁时再调用同样加锁的 menuFileBadgeIcon,主线程自死锁(点面包屑下拉就卡死)。
func TestFileMenuIconNoDeadlock(t *testing.T) {
	a, _ := newTestApp(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, n := range []string{"main.go", "a.json", "b.rs", "c.md", "notes.txt", "main.go"} {
			_ = a.fileMenuIcon(n)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("fileMenuIcon 死锁")
	}
}
