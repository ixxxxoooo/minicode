package cmdtitle

import "testing"

func TestTitle(t *testing.T) {
	cases := map[string]string{
		"cd /private/tmp/x && ls -la":                                 "列出目录",
		"cat internal/gui/open.go | head -60":                         "查看 open.go",
		`grep -rIl -iE "trafficLight" src`:                            "搜索 trafficLight",
		"git status --short":                                          "查看 Git 状态",
		"cd /a/b && go build ./... && go test ./...":                  "Go 编译 等 2 步",
		"S=/tmp/x; sleep 12; echo hi":                                 "等待 12 秒 等 2 步",
		"ls && git status --short | head":                             "列出目录 等 2 步",
		"python3 scripts/gen.py --out x":                              "运行 gen.py",
		"npm run build":                                               "npm run build",
		"foobar --weird":                                              "foobar --weird",
		"cd /x":                                                       "cd /x",
		"curl -s https://postman-echo.com/get?name=test -o /dev/null": "请求 postman-echo.com",
	}
	for in, want := range cases {
		if got := Title(in); got != want {
			t.Errorf("Title(%q) = %q, 想要 %q", in, got, want)
		}
	}
}
