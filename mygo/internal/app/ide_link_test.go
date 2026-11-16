package app

import "testing"

func TestDisplayColsAndByteAtCell(t *testing.T) {
	// 行首一个 Tab(宽 4)+ "foo(" ,单词 foo 的字节区间 [1,4)
	line := "\tfoo(x)"
	c0, c1 := displayCols(line, 1, 4, 4)
	if c0 != 4 || c1 != 7 {
		t.Fatalf("Tab 行显示列 = %d,%d,想要 4,7", c0, c1)
	}
	if b := byteAtCell(line, 5, 4); b != 2 {
		t.Fatalf("显示列 5 应落在字节 2,实际 %d", b)
	}
	// 宽字符占两格:"中a" 里 a 的显示列是 2
	c0, c1 = displayCols("中a", 3, 4, 4)
	if c0 != 2 || c1 != 3 {
		t.Fatalf("宽字符行显示列 = %d,%d,想要 2,3", c0, c1)
	}
	if b := byteAtCell("中a", 1, 4); b != 0 {
		t.Fatalf("落在宽字符后半格应算该字符,实际 %d", b)
	}
	// 指针在行尾之外 → 行长
	if b := byteAtCell("ab", 9, 4); b != 2 {
		t.Fatalf("行尾之外应返回行长,实际 %d", b)
	}
}
