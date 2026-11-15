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

// Sandbox 返回演示用的沙盒项目 router-demo(内存文件系统)。
func Sandbox() *workspace.Mem {
	files := map[string]string{
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
		// 以下三份给 Markdown 所见即所得的各种状态用(正常 / 含表格只读预览 / 带 YAML 头的规则文件)
		"docs/guide.md":           "# 快速上手\n\n路由库只做一件事:**把路径匹配到处理函数**,忽略大小写与末尾斜杠。\n\n## 安装\n\n- 在 `Cargo.toml` 里加入依赖\n- 运行 `cargo build`\n\n```rust\nlet r = Router::default();\n```\n\n## 待办\n\n- [x] 大小写不敏感\n- [ ] 通配符路径\n\n> 提示:详见 [路由匹配](routing.md)。\n",
		"docs/api.md":             "# 接口\n\n| 方法 | 路径 | 说明 |\n| --- | --- | --- |\n| GET | /users | 用户列表 |\n| POST | /users | 新建用户 |\n\n以上接口都返回 JSON。\n",
		".cursor/rules/style.mdc": "---\ndescription: 代码风格\nglobs: src/**/*.rs\nalwaysApply: false\n---\n\n# 风格\n\n- 公开函数写文档注释\n- 不用 `unwrap()`\n",
		"Cargo.toml": `[package]
name = "router-demo"
version = "0.3.1"
edition = "2021"

[dependencies]
axum = "0.8"
`,
		"CLAUDE.md": "# router-demo\n\n- 用 `cargo test` 跑测试\n- 路径匹配统一走 `normalize`\n",
		"README.md": "# router-demo\n\n一个很小的路由库,用来演示 MiniCode。\n",
	}
	for k, v := range gitSandboxExtraFiles() {
		files[k] = v
	}
	for k, v := range editorSandboxFiles() {
		if _, ok := files[k]; !ok {
			files[k] = v
		}
	}
	m := workspace.NewMem("router-demo", files, editorEmptyDirs()...)
	m.Prefer = []string{"src", "tests", "docs", "demo", "samples", "edge", "data", "assets", "bulk", "deep"}
	return m
}

// GitStatus 是演示项目的工作区状态(与「改动」标签一致)。
func GitStatus() map[string]string {
	st := GitStatusLite()
	for k, v := range ExtraGitStatus() {
		st[k] = v
	}
	return st
}
