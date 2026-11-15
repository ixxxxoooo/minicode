package demo

import "github.com/ixxxxoooo/minicode/internal/workspace"

// routerSrc 是设计稿里 IDE 编辑器显示的 router.rs(工作区版本:已修复)。
const routerSrc = `use std::collections::HashMap;

/// 路由表:路径 → 处理器
pub struct Router {
    routes: HashMap<String, Handler>,
    fallback: Option<Handler>,
}

impl Router {
    /// 注册一条路由,路径先归一化
    pub fn add(&mut self, path: &str, h: Handler) {
        let key = normalize(path);
        self.routes.insert(key, h);
    }

    /// 匹配路径:忽略大小写与末尾斜杠
    pub fn match_path(&self, path: &str) -> Option<&Handler> {
        let key = normalize(path);
        self.routes.get(&key).or(self.fallback.as_ref())
    }
}

fn normalize(path: &str) -> String {
    path.trim_end_matches('/').to_ascii_lowercase()
}
`

// RouterHead 是 router.rs 在 HEAD 里的版本(差异标签的左侧)。
const routerHead = `use std::collections::HashMap;

/// 路由表:路径 → 处理器
pub struct Router {
    routes: HashMap<String, Handler>,
    fallback: Option<Handler>,
}

impl Router {
    /// 注册一条路由,路径先归一化
    pub fn add(&mut self, path: &str, h: Handler) {
        let key = normalize(path);
        self.routes.insert(key, h);
    }

    /// 匹配路径:忽略大小写与末尾斜杠
    pub fn match_path(&self, path: &str) -> Option<&Handler> {
        self.routes.get(path)
    }
}

fn normalize(path: &str) -> String {
    path.trim_end_matches('/').to_ascii_lowercase()
}
`

// HeadOf 返回演示项目里某个文件在 HEAD 的内容(没有记录 = 与工作区相同)。
func HeadOf(path string) (string, bool) {
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

// Sandbox 返回演示用的沙盒项目 router-demo(内存文件系统)。
func Sandbox() *workspace.Mem {
	m := workspace.NewMem("router-demo", map[string]string{
		"src/router.rs": routerSrc,
		"src/main.rs": `mod router;
mod server;

fn main() {
    let app = server::build();
    app.serve("127.0.0.1:8080");
}
`,
		"src/server.rs": `use crate::router::Router;

pub struct Server {
    router: Router,
}

pub fn build() -> Server {
    Server { router: Router::default() }
}
`,
		"src/handlers/mod.rs":   "pub mod users;\n",
		"src/handlers/users.rs": "pub fn list() -> &'static str {\n    \"[]\"\n}\n",
		"tests/router_test.rs":  testAdded + "\n",
		"tests/case_test.rs":    "#[test]\nfn upper_and_lower_match() {\n    assert!(true);\n}\n",
		"docs/routing.md":       "# 路由匹配\n\n路径匹配忽略大小写与末尾斜杠。\n",
		"Cargo.toml": `[package]
name = "router-demo"
version = "0.3.1"
edition = "2021"

[dependencies]
axum = "0.8"
`,
		"CLAUDE.md": "# router-demo\n\n- 用 `cargo test` 跑测试\n- 路径匹配统一走 `normalize`\n",
		"README.md": "# router-demo\n\n一个很小的路由库,用来演示 MiniCode。\n",
	})
	m.Prefer = []string{"src", "tests", "docs"} // 设计稿里的摆放顺序
	return m
}

// GitStatus 是演示项目的工作区状态(与「改动」标签一致)。
func GitStatus() map[string]string {
	return map[string]string{"src/router.rs": "M", "tests/router_test.rs": "M", "tests/case_test.rs": "A"}
}
