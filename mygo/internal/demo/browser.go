package demo

import "github.com/ixxxxoooo/minicode/internal/browser"

// BrowserSource 是演示用的网页数据源:两篇固定文章,其余地址都「尚未接入网页抓取」。
type BrowserSource struct{}

// BrowserHome 是演示浏览器的起始页。
const BrowserHome = "docs.rs/axum/latest/axum/routing"

var routing = browser.Page{
	URL: BrowserHome, Site: "docs.rs", ReadMin: 6,
	Sections: []browser.Section{
		{Heading: "Routing", Body: "Routing 决定请求交给哪个处理器。`Router` 用路径模式匹配请求,并把它转发给对应的 `Handler`。"},
		{Heading: "基本用法", Level: 1, Body: "```rust\nlet app = Router::new()\n    .route(\"/\", get(root))\n    .route(\"/users\", post(create_user));\n```"},
		{Heading: "路径匹配", Level: 1, Body: "路径区分大小写;末尾斜杠需要显式处理。常见做法是在中间件里统一归一化。\n\n- 捕获参数:`/users/:id`\n- 通配:`/files/*path`\n- 嵌套:`nest(\"/api\", api)`"},
		{Heading: "捕获参数", Level: 2, Body: "用 `Path<T>` 提取器取出冒号后的片段,类型不符会返回 400。"},
		{Heading: "通配与嵌套", Level: 2, Body: "`nest` 会把前缀从路径里剥掉再交给子路由;通配参数保留完整余下路径。"},
	},
	Summary: []string{"Router 把路径模式映射到处理器", "路径区分大小写,末尾斜杠需显式处理", "支持捕获参数、通配与嵌套路由"},
	Links: []browser.Link{
		{Title: "axum::Router", Host: "docs.rs", URL: BrowserHome},
		{Title: "tower::Service", Host: "docs.rs", URL: "docs.rs/tower/latest/tower/trait.Service.html"},
		{Title: "路径匹配的 RFC", Host: "rfc-editor.org"},
		{Title: "示例仓库", Host: "github.com"},
	},
}

var service = browser.Page{
	URL: "docs.rs/tower/latest/tower/trait.Service.html", Site: "docs.rs", ReadMin: 4,
	Sections: []browser.Section{
		{Heading: "Trait Service", Body: "`Service` 把「请求 → 响应」抽象成一个可以异步调用的东西,中间件就是包住另一个 `Service` 的 `Service`。"},
		{Heading: "核心方法", Level: 1, Body: "```rust\nfn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>>;\nfn call(&mut self, req: Request) -> Self::Future;\n```"},
	},
	Summary: []string{"Service 是异步的请求 → 响应函数", "先 poll_ready 确认可用,再 call", "中间件通过包装 Service 组合"},
	Links:   []browser.Link{{Title: "axum::Router", Host: "docs.rs", URL: BrowserHome}},
}

// Fetch 实现 browser.Source。
func (BrowserSource) Fetch(u string) (browser.Page, error) {
	switch u {
	case routing.URL:
		return routing, nil
	case service.URL:
		return service, nil
	}
	return browser.Page{}, browser.ErrNoFetch
}
