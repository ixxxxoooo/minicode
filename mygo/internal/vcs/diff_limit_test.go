package vcs

import "testing"

func TestParseDiffMaxBytes(t *testing.T) {
	if ParseDiffMaxBytes("512 KB") != 512*1024 {
		t.Fatalf("512 KB")
	}
	if ParseDiffMaxBytes("2 MB") != 2*1024*1024 {
		t.Fatalf("2 MB")
	}
	if !ExceedsDiffLimit(1024*1024, "512 KB") {
		t.Fatal("应超限")
	}
}
