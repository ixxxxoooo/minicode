package app

import "testing"

func TestCrumbUsesFileTypeIcons(t *testing.T) {
	if treeFolderIcon(".git") != "branch" {
		t.Fatalf(".git 文件夹")
	}
	if fileIcon("router.rs") != "code" {
		t.Fatalf("rs 应用 code 图标")
	}
	if fileIcon("Cargo.toml") != "gear" {
		t.Fatalf("toml 应用 gear")
	}
}
