package diag

import "testing"

// Report 记录 panic 但不拦截:外层仍能 recover 到同一个值。
func TestReportDoesNotSwallow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	defer func() {
		if v := recover(); v != "boom" {
			t.Fatalf("应继续 panic 出同一个值,得到 %v", v)
		}
	}()
	func() {
		defer func() {
			if v := recover(); v != nil {
				Report("测试", v)
				panic(v)
			}
		}()
		panic("boom")
	}()
}
