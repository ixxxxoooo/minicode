package app

import "testing"

// 图表卡应渲染出图本体(文字层)与失败回退的源码卡,而不是只剩标题行。
func TestMermaidCardsRender(t *testing.T) {
	_, tt := sessionApp(t, "mermaid-samples")
	tt.Frame()
	for _, want := range []string{"请求", "路由", "网络 [50]", "Idle", "暂不支持 gantt 图表"} {
		if !tt.HasText(want) {
			t.Fatalf("Mermaid 场景缺少文字 %q", want)
		}
	}
}
