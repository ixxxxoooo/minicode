package app

import "testing"

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
