package demo

// HeadOf 返回演示项目里某个文件在 HEAD 的内容(没有记录 = 与工作区相同)。
func HeadOf(path string) (string, bool) {
	head := gitRouterHeadContent()
	if s, ok := head[path]; ok {
		return s, true
	}
	switch path {
	case "src/router.rs":
		return routerHead, true
	case "tests/router_test.rs":
		return "use router_demo::router::Router;\n", true
	case "tests/case_test.rs":
		return "", true
	}
	return "", false
}
