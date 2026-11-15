package mermaid

import "testing"

func texts(s *Scene) []string {
	var out []string
	for _, it := range s.Items {
		if it.Text != nil {
			out = append(out, it.Text.Text)
		}
	}
	return out
}

func TestFlowBasic(t *testing.T) {
	s, err := Render("graph TD\n  A[开始] --> B{判断}\n  B -->|是| C(处理)\n  B -->|否| D([结束])\n  C --> D")
	if err != nil {
		t.Fatal(err)
	}
	tm := texts(&s)
	for _, w := range []string{"开始", "判断", "是", "否", "处理", "结束"} {
		found := false
		for _, x := range tm {
			if x == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("缺少 %s: %v", w, tm)
		}
	}
	if s.Width < 100 || s.Height < 100 {
		t.Fatalf("尺寸异常 %vx%v", s.Width, s.Height)
	}
}

func TestUnsupported(t *testing.T) {
	for _, src := range []struct {
		src, kind string
	}{
		{"gantt\n title x", "gantt"},
		{"erDiagram\n  A ||--o{ B : has", "erDiagram"},
		{"classDiagram\n  Animal <|-- Duck", "classDiagram"},
	} {
		_, err := Render(src.src)
		if err == nil {
			t.Fatalf("%s 应不支持", src.kind)
		}
		re, ok := err.(RenderError)
		if !ok || re.Kind != ErrUnsupported || re.Arg != src.kind {
			t.Fatalf("%s 错误类型 %v", src.kind, err)
		}
	}
}

func TestClassInFlowchartStillFlow(t *testing.T) {
	s, err := Render("graph TD\n  A[开始] --> B[结束]\n  classDef ok fill:#dfd\n  class A ok")
	if err != nil {
		t.Fatal(err)
	}
	if s.Width < 40 {
		t.Fatalf("流程图 class 样式应有有效布局:%vx%v", s.Width, s.Height)
	}
}

func TestKindTitle(t *testing.T) {
	if k, ok := KindTitle("graph TD\nA-->B"); !ok || k != "流程图" {
		t.Fatalf("kind %q ok=%v", k, ok)
	}
}
