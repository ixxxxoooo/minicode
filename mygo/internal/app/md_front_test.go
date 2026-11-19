package app

import (
	"reflect"
	"testing"
)

func TestParseFrontEntries(t *testing.T) {
	cases := []struct {
		name  string
		front string
		ok    bool
		want  []fmEntry
	}{
		{
			name:  "技能头:name / description / 逗号清单",
			front: "---\nname: bw-exec-query\ndescription: 通过工具查询:新老蓝鲸。\nallowed-tools: Bash, Read\n---\n",
			ok:    true,
			want: []fmEntry{
				{Key: "name", Text: "bw-exec-query"},
				{Key: "description", Text: "通过工具查询:新老蓝鲸。"},
				{Key: "allowed-tools", Kind: fmList, List: []string{"Bash", "Read"}},
			},
		},
		{
			name:  "块清单 + 行内清单 + 布尔 + 引号",
			front: "---\nglobs:\n  - src/**\n  - \"docs/*.md\"\ntags: [a, 'b']\nalwaysApply: true\ntitle: \"带引号\"\n---\n",
			ok:    true,
			want: []fmEntry{
				{Key: "globs", Kind: fmList, List: []string{"src/**", "docs/*.md"}},
				{Key: "tags", Kind: fmList, List: []string{"a", "b"}},
				{Key: "alwaysApply", Kind: fmBool, Text: "true"},
				{Key: "title", Text: "带引号"},
			},
		},
		{
			name:  "块标量:> 折成一段,| 保留换行",
			front: "---\ndescription: >\n  第一行\n  第二行\nnotes: |\n  甲\n  乙\n---\n",
			ok:    true,
			want: []fmEntry{
				{Key: "description", Text: "第一行 第二行"},
				{Key: "notes", Text: "甲\n乙"},
			},
		},
		{
			name:  "普通标量跨行",
			front: "---\ndescription: 很长的\n  一句话\n---\n",
			ok:    true,
			want:  []fmEntry{{Key: "description", Text: "很长的 一句话"}},
		},
		{name: "不是键值:整段退回源码", front: "---\n随便一句话\n---\n", ok: false},
		{name: "开头就是缩进行", front: "---\n  a: 1\n---\n", ok: false},
		{name: "只有注释 / 空", front: "---\n# 只有注释\n---\n", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseFrontEntries(tc.front)
			if ok != tc.ok {
				t.Fatalf("ok = %v,期望 %v(got=%+v)", ok, tc.ok, got)
			}
			if tc.ok && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func TestMdSplitFrontKeepsBody(t *testing.T) {
	front, body := mdSplitFront("---\nname: x\n---\n# 标题\n\n正文\n")
	if front != "---\nname: x\n---\n" || body != "# 标题\n\n正文\n" {
		t.Fatalf("front=%q body=%q", front, body)
	}
	// 正文里的 --- 不是头
	if f, _ := mdSplitFront("正文\n---\nname: x\n---\n"); f != "" {
		t.Fatalf("不在开头的 --- 不应被当成 YAML 头: %q", f)
	}
}
