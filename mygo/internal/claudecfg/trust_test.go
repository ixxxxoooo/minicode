package claudecfg

import "testing"

func TestMergeTrustPaths(t *testing.T) {
	got := MergeTrustPaths(
		[]string{"/a/foo", "/b/bar"},
		[]string{"/b/bar", "/c/baz"},
		[]string{"/d/qux"},
		"/a/foo",
	)
	if len(got) != 4 {
		t.Fatalf("期望 4 条, 得到 %d: %v", len(got), got)
	}
	if got[0] != "/a/foo" || got[1] != "/b/bar" || got[2] != "/c/baz" || got[3] != "/d/qux" {
		t.Fatalf("排序或去重错误: %v", got)
	}
}

func TestMergeTrustPathsEmpty(t *testing.T) {
	if len(MergeTrustPaths(nil, nil, nil, "")) != 0 {
		t.Fatal("空输入应得空列表")
	}
}
