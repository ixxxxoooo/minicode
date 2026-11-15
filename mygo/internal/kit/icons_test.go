package kit

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// 源码里写的图标名必须都在图标表里:不存在的名字会退回画成 ✕,看上去像「图标显示异常」。
func TestIconNamesExist(t *testing.T) {
	pats := []*regexp.Regexp{
		regexp.MustCompile(`\bIc\(c, "([\w-]+)"`),
		regexp.MustCompile(`\bIb\(c, "([\w-]+)"`),
		regexp.MustCompile(`\bIcon: "([\w-]+)"`),
		regexp.MustCompile(`\bicon: "([\w-]+)"`),
		regexp.MustCompile(`MenuIcon\("([\w-]+)"\)`),
	}
	root := filepath.Join("..")
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".go" || regexp.MustCompile(`_test\.go$`).MatchString(p) {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, re := range pats {
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				if _, ok := iconPaths[m[1]]; !ok {
					t.Errorf("%s 用了不存在的图标 %q", p, m[1])
				}
			}
		}
		return nil
	})
}
