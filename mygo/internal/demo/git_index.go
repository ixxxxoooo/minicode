// 演示仓库索引(源代码管理 / 图谱 / 差异 / 冲突 / 多仓库)
//
// router-demo · 主演示仓库(Repo/Sandbox/GitStatus/HeadOf)
//   保留三项改动(router.rs 暂存、router_test 工作区、case_test 新增)与提交 a3f9c21…f5521ab、stash@{0}
//   变更全集:见 git_changes.go(含 300 文件 bulk、冲突、重命名、二进制等)
//   图谱:220+ 条历史(git_log.go),含合并 / 章鱼合并 / tag / 远程分支 / 多作者 / 时间文案
//   分支:↑3↓2、长分支名、origin/*、feat/* 多级
//   储藏:5 条(git_stash.go)
//   差异内容:HEAD / 暂存 / 工作区三套(git_content.go · HeadOf / SetIndex)
//   冲突:2 处 / 5 处 / diff3 / 损坏标记 / 二进制(git_conflict.go)
//   追溯:src/router.rs 30 次 FileLog + 多作者 blame
//
// 独立小仓库(git_repos_extra.go)
//   repo-empty:干净、有提交
//   repo-unborn:已 init、尚无提交
//   repo-sync-fail / repo-sync-slow / repo-far-behind / repo-long-branch
//
// 工作区(wsdemo.go)
//   WorkspaceFolders:原有 4 成员 + 扩展场景仓库
//   WorkspaceFoldersLarge:12 个文件夹(压测列表 / 树)
//   DemoWorkspaces:供 seedDemoRegistry 注册 ws-large(主代理接线)
//
// 审查页:多仓库时 demoRepoFiles 读 web-dashboard / design-assets;
//   主仓库改动仍来自 fixbug 会话 + demoExtra case_test.

package demo
