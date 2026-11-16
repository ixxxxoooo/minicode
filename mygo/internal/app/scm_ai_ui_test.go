package app

import "testing"

func TestScmCancelGenMsg(t *testing.T) {
	a, _ := scmApp(t)
	a.SetTextGenerator(func(dir, prompt string) (string, error) {
		<-make(chan struct{}) // 永不返回,模拟进行中的请求
		return "", nil
	})
	a.scmGenMsg()
	if !a.sc.genning {
		t.Fatal("应进入生成态")
	}
	a.scmCancelGenMsg()
	if a.sc.genning {
		t.Fatal("取消后应退出生成态")
	}
}
