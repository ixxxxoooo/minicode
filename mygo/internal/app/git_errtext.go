package app

import "strings"

// friendlyGitErr 把 git 的英文报错压成一句带「下一步」的中文提示,给 toast / 提交框用。
// 完整原文仍写进输出面板与日志,所以这里只管好读:认不出的取第一行并截断,不再把整段多行英文糊在界面上。
func friendlyGitErr(raw string) string {
	low := strings.ToLower(raw)
	has := func(ss ...string) bool {
		for _, s := range ss {
			if strings.Contains(low, strings.ToLower(s)) {
				return true
			}
		}
		return false
	}
	switch {
	case has("no configured push destination", "no remote repository specified", "does not appear to be a git repository", "no such remote"):
		return "还没有配置远程仓库。在终端运行 git remote add origin <地址> 后再试"
	case has("has no upstream branch", "no tracking information"):
		return "当前分支还没有上游分支。在终端运行 git push -u origin <分支名> 一次即可"
	case has("authentication failed", "could not read username", "could not read password", "permission denied (publickey)", "invalid username or password"):
		return "认证失败。请检查远程仓库的账号凭据或 SSH 密钥"
	case has("could not resolve host", "failed to connect", "connection timed out", "connection refused", "network is unreachable", "unable to access"):
		return "连不上远程仓库。请检查网络或代理后重试"
	case has("non-fast-forward", "[rejected]", "fetch first", "updates were rejected"):
		return "远端有本地没有的提交,推送被拒绝。先拉取再推送"
	case has("please tell me who you are", "unable to auto-detect email", "empty ident name"):
		return "还没有设置 Git 用户。在终端运行 git config --global user.name 与 user.email"
	case has("nothing to commit", "no changes added to commit"):
		return "没有可提交的内容"
	case has("index.lock"):
		return "仓库正被另一个 Git 进程占用(.git/index.lock)。稍后重试,确认没有 Git 在运行后可删除该文件"
	}
	// 认不出:取第一条非空行,去掉常见前缀,过长截断
	for _, ln := range strings.Split(raw, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		for _, p := range []string{"fatal: ", "error: ", "fatal:", "error:"} {
			ln = strings.TrimPrefix(ln, p)
		}
		if r := []rune(ln); len(r) > 120 {
			ln = string(r[:120]) + "…"
		}
		return ln + "(完整信息见输出面板)"
	}
	return "未知错误(完整信息见输出面板)"
}
