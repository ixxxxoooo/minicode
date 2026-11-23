package app

import (
	"path"
	"regexp"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/symbols"
)

var (
	reGoTest   = regexp.MustCompile(`^func (Test|Benchmark|Fuzz)\w+`)
	reRustTest = regexp.MustCompile(`#\[test\]`)
	rePyTest   = regexp.MustCompile(`^\s*def (test_\w+)`)
)

// testsForSymbol 从本地符号索引推断某声明旁的测试数(0 = 不显示测试胶囊)。
func (a *App) testsForSymbol(tb *ideTab, sym symbols.Symbol) int {
	if tb == nil || sym.Kind != "fn" {
		return 0
	}
	name := sym.Name
	switch symbols.Lang(tb.path) {
	case "go":
		return a.countGoTests(tb.path, name)
	case "rust":
		return a.countRustTests(tb, name)
	case "py":
		return a.countPyTests(tb, name)
	case "ts":
		return a.countTSTests(tb, name)
	}
	return 0
}

func (a *App) countGoTests(file, fn string) int {
	cands := []string{"Test" + titleCase(fn), "Test" + fn}
	testFile := file
	if !strings.HasSuffix(file, "_test.go") {
		base := strings.TrimSuffix(path.Base(file), ".go")
		testFile = path.Join(path.Dir(file), base+"_test.go")
	}
	src, ok := a.ide.fileText(testFile)
	if !ok {
		return 0
	}
	n := 0
	for _, ln := range strings.Split(src, "\n") {
		if !reGoTest.MatchString(ln) {
			continue
		}
		for _, c := range cands {
			if strings.Contains(ln, c) {
				n++
				break
			}
		}
	}
	return n
}

func (a *App) countRustTests(tb *ideTab, fn string) int {
	src, ok := a.ide.fileText(tb.path)
	if !ok {
		return 0
	}
	lines := strings.Split(src, "\n")
	n := 0
	for i, ln := range lines {
		if !reRustTest.MatchString(ln) {
			continue
		}
		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			if strings.Contains(lines[j], "fn "+fn) || strings.Contains(lines[j], "fn test_"+strings.ToLower(fn)) {
				n++
				break
			}
		}
	}
	return n
}

func (a *App) countPyTests(tb *ideTab, fn string) int {
	src, ok := a.ide.fileText(tb.path)
	if !ok {
		return 0
	}
	n := 0
	lowFn := strings.ToLower(fn)
	for _, ln := range strings.Split(src, "\n") {
		if rePyTest.MatchString(ln) && strings.Contains(strings.ToLower(ln), lowFn) {
			n++
		}
	}
	return n
}

func (a *App) countTSTests(tb *ideTab, fn string) int {
	src, ok := a.ide.fileText(tb.path)
	if !ok {
		return 0
	}
	n := 0
	lowFn := strings.ToLower(fn)
	for _, ln := range strings.Split(src, "\n") {
		low := strings.ToLower(ln)
		if (strings.Contains(low, "it(") || strings.Contains(low, "test(")) && strings.Contains(low, lowFn) {
			n++
		}
	}
	return n
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
