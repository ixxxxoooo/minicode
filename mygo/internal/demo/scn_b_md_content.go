package demo

import (
	"fmt"
	"strings"
)

// mdFullBody 是会话 md 的助手正文:覆盖设计稿里的 Markdown 全元素样例。
func mdFullBody() string {
	var b strings.Builder
	b.WriteString("# 正文 · Markdown 全元素样例\n\n")
	b.WriteString("本节集中展示会话流助手正文能渲染的 Markdown 结构(对标设计稿「发布说明」页的扩展版)。\n\n")

	b.WriteString("## 标题层级\n\n")
	b.WriteString("### 三级标题\n\n")
	b.WriteString("#### 四级标题\n\n")

	b.WriteString("## 列表\n\n")
	b.WriteString("无序:\n\n- 苹果\n- 香蕉\n  - 嵌套 A\n  - 嵌套 B\n    - 再嵌套\n- 橙子\n\n")
	b.WriteString("有序:\n\n1. 拉取代码\n2. 运行测试\n3. 打 tag\n   1. 子步骤 1\n   2. 子步骤 2\n4. 发布\n\n")
	b.WriteString("任务清单:\n\n- [x] 更新 CHANGELOG\n- [x] 校对截图\n- [ ] 同步翻译\n- [ ] 提交商店素材\n\n")

	b.WriteString("## 行内样式\n\n")
	b.WriteString("这是 **粗体**、*斜体*、~~删除线~~、`行内代码`、")
	b.WriteString("[外部链接](https://example.com/docs)、脚注示例[^note]与 ")
	b.WriteString("裸 URL:")
	b.WriteString(" https://example.com/api/v2/projects/1234567890abcdef/repositories/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/path/to/resource?query=verylongparameterthatshouldnotbreakacrosswords&lang=zh-CN#section-anchor\n\n")
	b.WriteString("CJK 与 emoji 混排:中文 English 日本語 🎉✨🔧 不意外断行。\n\n")
	b.WriteString("[^note]: 脚注正文:若渲染器不支持,应优雅降级为普通段落。\n\n")

	b.WriteString("## 引用与分割线\n\n")
	b.WriteString("> 升级前请备份 `~/.minicode/settings.json`。\n>\n> 多行引用第二段。\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## 表格\n\n")
	b.WriteString("| 左对齐 | 居中 | 右对齐 | 单位 |\n")
	b.WriteString("| :--- | :---: | ---: | --- |\n")
	b.WriteString("| 启动 | 冷启动 | 0.42s | s |\n")
	b.WriteString("| 内存 | 常驻 | 118MB | MB |\n\n")
	b.WriteString("很宽的表格(应出现横向滚动):\n\n")
	b.WriteString("| 阶段 | 负责人 | 状态 | 备注 | 依赖项 | 风险 | 预计工时 | 实际工时 | 链接 |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for i := 1; i <= 4; i++ {
		fmt.Fprintf(&b, "| 里程碑 %d | 团队 %c | 进行中 | 细节说明占位 | 任务 A/B | 低 | %dh | %dh | [看板](https://example.com) |\n",
			i, 'A'+i-1, i*8, i*6)
	}
	b.WriteString("\n")

	b.WriteString("## 围栏代码 · 多语言\n\n")
	b.WriteString("```rust\nfn normalize(path: &str) -> String {\n    path.trim_end_matches('/').to_lowercase()\n}\n```\n\n")
	b.WriteString("```go\nfunc main() {\n    fmt.Println(\"你好, MiniCode\")\n}\n```\n\n")
	b.WriteString("```typescript\nexport type Route = { path: string; handler: Handler };\n```\n\n")
	b.WriteString("```python\ndef greet(name: str) -> str:\n    return f\"你好,{name}\"\n```\n\n")
	b.WriteString("```json\n{\"ok\": true, \"version\": \"2.0.0\"}\n```\n\n")
	b.WriteString("```yaml\nservice:\n  name: minicode\n  port: 8080\n```\n\n")
	b.WriteString("```shell\nset -euo pipefail\ncargo test --workspace\n```\n\n")
	b.WriteString("```sql\nSELECT id, title FROM sessions ORDER BY updated_at DESC LIMIT 10;\n```\n\n")
	b.WriteString("```diff\n- self.routes.get(path)\n+ self.routes.get(&normalize(path))\n```\n\n")
	b.WriteString("无语言标注(纯文本围栏):\n\n```\nplain text fence\nno highlighter\n```\n\n")

	b.WriteString("### 超过 12 行:限高折叠\n\n")
	b.WriteString(longGoFence(18))

	b.WriteString("### 四个反引号的外层围栏\n\n")
	b.WriteString("````markdown\n")
	b.WriteString("内层围栏应原样保留:\n\n")
	b.WriteString("```rust\nfn main() {}\n```\n")
	b.WriteString("````\n\n")

	b.WriteString("## Mermaid\n\n")
	b.WriteString("### 流程图\n\n")
	b.WriteString("```mermaid\nflowchart LR\n  UI[界面] --> Core[core.State]\n  Core --> Claude[claude 后端]\n```\n\n")
	b.WriteString("### 时序图\n\n")
	b.WriteString("```mermaid\nsequenceDiagram\n  participant U as 用户\n  participant A as App\n  U->>A: 发送\n  A-->>U: 流式事件\n```\n\n")
	b.WriteString("### 状态图\n\n")
	b.WriteString("```mermaid\nstateDiagram-v2\n  [*] --> 草稿\n  草稿 --> 已发布 : 发布\n  已发布 --> [*]\n```\n\n")
	b.WriteString("### 饼图\n\n")
	b.WriteString("```mermaid\npie title 耗时占比\n  \"渲染\" : 30\n  \"网络\" : 50\n  \"其他\" : 20\n```\n\n")
	b.WriteString("### 解析失败(应显示原因 + 源码)\n\n")
	b.WriteString("```mermaid\nflowchart TD\n  A --> \n  ???\n```\n\n")

	b.WriteString("## 数学与 HTML 片段\n\n")
	b.WriteString("行内公式 `$E=mc^2$` 与块级 `$$\\sum_{i=1}^{n} i$$` 若未接 LaTeX 渲染,应原样显示。\n\n")
	b.WriteString("<picture>\n  <source media=\"(prefers-color-scheme: dark)\" srcset=\"docs/assets/lockup-dark.png\">\n  <img src=\"docs/assets/lockup-light.png\" alt=\"MiniCode\" height=\"96\">\n</picture>\n\n")
	b.WriteString("<details><summary>HTML details</summary><p>内联 <strong>HTML</strong> 片段。</p></details>\n\n")

	b.WriteString("## 收尾\n\n")
	b.WriteString("以上元素应在一屏流内连续可见,便于对照设计稿检查间距、字号与代码折叠。\n")
	return b.String()
}
