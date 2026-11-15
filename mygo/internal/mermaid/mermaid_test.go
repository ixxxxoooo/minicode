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
	_, err := Render("gantt\n title x")
	if err == nil {
		t.Fatal("应不支持 gantt")
	}
	re, ok := err.(RenderError)
	if !ok || re.Kind != ErrUnsupported {
		t.Fatalf("错误类型 %v", err)
	}
}

func TestKindTitle(t *testing.T) {
	if k, ok := KindTitle("graph TD\nA-->B"); !ok || k != "流程图" {
		t.Fatalf("kind %q ok=%v", k, ok)
	}
}
