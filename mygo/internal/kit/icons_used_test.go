package kit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 源码里用到的图标名必须都在图标表里,否则界面会画成「×」(曾有 merge / rows / msg / reply / warn / hash 漏登记)。
func TestUsedIconsExist(t *testing.T) {
	re := regexp.MustCompile(`(?:\b(?:Ic|Ib|MenuIcon)\(c?,? ?"([A-Za-z0-9]+)"|[iI]con:\s*"([A-Za-z0-9]+)"|\bic:\s*"([A-Za-z0-9]+)")`)
	var miss []string
	_ = filepath.Walk("..", func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			n := m[1] + m[2] + m[3]
			if _, ok := iconPaths[n]; !ok {
				miss = append(miss, p+": "+n)
			}
		}
		return nil
	})
	if len(miss) > 0 {
		t.Fatalf("图标名未登记:\n%s", strings.Join(miss, "\n"))
	}
}
