package vcs

import (
	"reflect"
	"testing"
)

// 设计稿里的那张图:c1 是个合并(父提交 c4 与 c2),c3 并回主线。
func demoCommits() []Commit {
	return []Commit{
		{Hash: "c0", Parents: []string{"c1"}},
		{Hash: "c1", Parents: []string{"c4", "c2"}},
		{Hash: "c2", Parents: []string{"c3"}},
		{Hash: "c3", Parents: []string{"c4"}},
		{Hash: "c4", Parents: []string{"c5"}},
		{Hash: "c5"},
	}
}

func TestLayoutDemo(t *testing.T) {
	rows := Layout(demoCommits())
	want := []struct {
		node        int
		fork, merge []int
		top, bottom []int
	}{
		{0, nil, nil, nil, []int{0}},
		{0, []int{1}, nil, []int{0}, []int{0, 1}},
		{1, nil, nil, []int{0, 1}, []int{0, 1}},
		{1, nil, []int{0}, []int{0, 1}, []int{0}},
		{0, nil, nil, []int{0}, []int{0}},
		{0, nil, nil, []int{0}, nil},
	}
	for i, w := range want {
		r := rows[i]
		if r.Node != w.node || !reflect.DeepEqual(r.Fork, w.fork) || !reflect.DeepEqual(r.Merge, w.merge) ||
			!reflect.DeepEqual(r.Top, w.top) || !reflect.DeepEqual(r.Bottom, w.bottom) {
			t.Errorf("第 %d 行不对:%+v,想要 %+v", i, r, w)
		}
	}
}

func TestLayoutSharedParent(t *testing.T) {
	// 两个分支头 a、b 指向同一个父 p:b 在新泳道 1 上,并向泳道 0 并过去,p 行只剩一条泳道
	rows := Layout([]Commit{
		{Hash: "a", Parents: []string{"p"}},
		{Hash: "b", Parents: []string{"p"}},
		{Hash: "p"},
	})
	if rows[1].Node != 1 || !reflect.DeepEqual(rows[1].Merge, []int{0}) || !reflect.DeepEqual(rows[1].Bottom, []int{0}) {
		t.Fatalf("b 应在泳道 1 并并入泳道 0:%+v", rows[1])
	}
	if got := rows[2]; got.Node != 0 || len(got.Join) != 0 || got.Width != 1 {
		t.Fatalf("p 行不对:%+v", got)
	}
}

// 多个孩子都在等同一个提交时(分叉后各自推进),汇入曲线出现在父提交那一行。
func TestLayoutJoin(t *testing.T) {
	rows := Layout([]Commit{
		{Hash: "a", Parents: []string{"x"}},
		{Hash: "m", Parents: []string{"y", "x"}}, // x 被泳道 0 与 m 的第二父提交同时等着
		{Hash: "y", Parents: []string{"x"}},
		{Hash: "x"},
	})
	last := rows[3]
	if last.Node != 0 {
		t.Fatalf("x 行应在泳道 0:%+v", last)
	}
}

func TestLayoutLinear(t *testing.T) {
	rows := Layout([]Commit{{Hash: "x", Parents: []string{"y"}}, {Hash: "y", Parents: []string{"z"}}, {Hash: "z"}})
	for i, r := range rows {
		if r.Node != 0 || r.Width > 1 {
			t.Fatalf("线性历史只应有一条泳道,第 %d 行:%+v", i, r)
		}
	}
}
