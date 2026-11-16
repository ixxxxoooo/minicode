package app

import "testing"

// 取消生成:退出生成态,并让迟到的结果因代数不符被丢弃。
// (a.win == nil 时 a.bg 同步执行,不能用「永不返回」的生成器,直接构造进行中的状态)
func TestScmCancelGenMsg(t *testing.T) {
	a, _ := scmApp(t)
	a.sc.genning = true
	a.sc.genGen = 3
	a.scmCancelGenMsg()
	if a.sc.genning {
		t.Fatal("取消后应退出生成态")
	}
	if a.sc.genGen != 4 {
		t.Fatalf("取消应使进行中的结果作废:genGen=%d", a.sc.genGen)
	}
}
