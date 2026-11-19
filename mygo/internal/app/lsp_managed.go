package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// 语言服务:托管 Node 与语言包的「下载安装」任务(设计稿 lsp-spec「托管 Node 与语言包」)。
// 只在用户点「下载安装」后才联网;同一时刻只跑一个任务,其余排队;可取消,失败留在行上等重试。

var lspMgrLog = diag.L("lsp")

// pLspMirror 是下载源偏好:空 / official = 官方源,cn = 国内镜像。
const pLspMirror = "lsp.mirror"

// lspJob 是一个服务器(或一组共用一个包的服务器)的下载安装任务。
type lspJob struct {
	sp      *lsp.Spec
	stage   string // 当前阶段,如「下载 Node v24.11.1」「安装 pyright@1.1.414」
	done    int64
	total   int64 // ≤ 0 = 不知道总量
	running bool  // false 且 err == "" = 排队中
	err     string
	cancel  context.CancelFunc
}

// pct 是下载百分比;未知返回 -1。
func (j *lspJob) pct() int {
	if j.total <= 0 {
		return -1
	}
	return int(min(j.done*100/j.total, 100))
}

// lspJobKey:同一个包提供的多个服务器(HTML / CSS / JSON)共用一个任务。
func lspJobKey(sp *lsp.Spec) string {
	if sp.NPMGroup != "" {
		return sp.NPMGroup
	}
	return sp.ID
}

func lspSpecByID(id string) *lsp.Spec {
	for i := range lsp.Specs {
		if lsp.Specs[i].ID == id {
			return &lsp.Specs[i]
		}
	}
	return nil
}

func (a *App) lspMirror() lsp.Mirror { return lsp.ParseMirror(a.prefs.Str(pLspMirror, "")) }

func (a *App) setLspMirror(m lsp.Mirror) {
	if m == lsp.MirrorOfficial {
		a.prefs.Reset(pLspMirror) // 默认值:清掉偏好
	} else {
		a.prefs.Set(pLspMirror, string(m))
	}
	lspMgrLog.Info("切换语言包下载源", "mirror", string(m))
}

func (a *App) lspJobOf(sp *lsp.Spec) *lspJob { return a.lspJobs[lspJobKey(sp)] }

// lspInstallManaged 开始(或排队)下载安装;已在装 / 排队中的忽略,失败的重来一遍。
func (a *App) lspInstallManaged(sp *lsp.Spec) {
	if sp == nil || !sp.Installable() {
		return
	}
	key := lspJobKey(sp)
	if j := a.lspJobs[key]; j != nil && j.err == "" {
		return
	}
	if a.lspJobs == nil {
		a.lspJobs = map[string]*lspJob{}
	}
	if _, had := a.lspJobs[key]; !had {
		a.lspJobOrder = append(a.lspJobOrder, key)
	}
	a.lspJobs[key] = &lspJob{sp: sp, stage: "排队中…"}
	lspMgrLog.Info("开始下载安装语言包", "server", sp.Name, "mirror", string(a.lspMirror()))
	a.lspJobKick()
}

// lspJobKick 没有在跑的任务时,启动排队最早的一个。
func (a *App) lspJobKick() {
	if a.demoWs() { // 演示:不排队(预置的「下载中」那行是静态的,不会结束,不能挡住别的)
		for _, k := range a.lspJobOrder {
			if j := a.lspJobs[k]; j != nil && !j.running && j.err == "" {
				a.lspJobRun(j)
			}
		}
		return
	}
	var next *lspJob
	for _, k := range a.lspJobOrder {
		j := a.lspJobs[k]
		if j == nil {
			continue
		}
		if j.running {
			return
		}
		if next == nil && j.err == "" {
			next = j
		}
	}
	if next != nil {
		a.lspJobRun(next)
	}
}

func (a *App) lspJobRun(j *lspJob) {
	j.running, j.stage = true, "准备中…"
	if a.demoWs() {
		a.lspJobDemo(j)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	sp, key, mirror := j.sp, lspJobKey(j.sp), a.lspMirror()
	a.bg("lsp.install", func() func() {
		prog := func(stage string, done, total int64) {
			a.onUI(func() {
				if a.lspJobs[key] == j {
					j.stage, j.done, j.total = stage, done, total
				}
			})
		}
		logf := func(line string) { a.onUI(func() { a.outAppend(outLSP, sp.Name+": "+line) }) }
		var err error
		if sp.Managed() { // 依赖 Node:托管 Node + 固定版本的 npm 包
			err = sp.InstallManaged(ctx, mirror, prog, logf)
		} else { // 其余:应用在后台替用户执行清单里的安装命令(不开终端)
			err = sp.InstallByCommand(ctx, mirror, prog, logf)
		}
		return func() { a.lspJobDone(j, err, ctx.Err() != nil) }
	})
}

// lspJobDone 在界面线程处理任务结果。
func (a *App) lspJobDone(j *lspJob, err error, canceled bool) {
	key := lspJobKey(j.sp)
	if a.lspJobs[key] != j {
		return
	}
	j.running = false
	if j.cancel != nil {
		j.cancel()
	}
	switch {
	case canceled || errors.Is(err, context.Canceled):
		delete(a.lspJobs, key)
		a.outAppend(outLSP, j.sp.Name+" 下载安装已取消")
		a.flash("已取消 " + j.sp.Name + " 的下载安装")
	case err != nil:
		j.err = err.Error()
		lspMgrLog.Warn("语言包下载安装失败", "server", j.sp.Name, "err", err)
		a.outAppend(outLSP, j.sp.Name+" 下载安装失败:"+err.Error())
		a.flash(j.sp.Name + " 下载安装失败,可在设置里重试或切换下载源")
	default:
		delete(a.lspJobs, key)
		a.outAppend(outLSP, j.sp.Name+" 已下载安装")
		a.flash("已安装 " + j.sp.Name)
		a.lspAfterChange(j.sp)
	}
	a.lspJobKick()
}

// lspJobCancel 取消正在下载的任务(取消后由 lspJobDone 清掉);排队中的直接移除。
func (a *App) lspJobCancel(sp *lsp.Spec) {
	j := a.lspJobOf(sp)
	if j == nil {
		return
	}
	if j.cancel != nil && j.running {
		j.cancel()
		return
	}
	delete(a.lspJobs, lspJobKey(sp))
}

// lspJobDismiss 关掉失败行上的错误(不重试)。
func (a *App) lspJobDismiss(sp *lsp.Spec) {
	if j := a.lspJobOf(sp); j != nil && j.err != "" {
		delete(a.lspJobs, lspJobKey(sp))
	}
}

// lspAfterChange 装好 / 卸载后:重新检测设置页的行,并让用到它的会话重新连接(没装的会话变成「未安装」,装好的会启动)。
func (a *App) lspAfterChange(sp *lsp.Spec) {
	a.set.lspScanned = false
	var hit []*lspSess
	for _, s := range a.lspH.sess {
		if lspJobKey(s.spec) == lspJobKey(sp) {
			hit = append(hit, s)
		}
	}
	for _, s := range hit {
		a.lspRetry(s)
	}
}

// lspUninstallManaged 删掉托管目录(共用目录的服务器一起没掉);Node 运行时保留。
func (a *App) lspUninstallManaged(sp *lsp.Spec) {
	if a.demoWs() {
		for i := range a.set.lspRows {
			if r := &a.set.lspRows[i]; lspSpecByID(r.ID) != nil && lspJobKey(lspSpecByID(r.ID)) == lspJobKey(sp) {
				r.OK, r.Path, r.MgrPath, r.Stale = false, "", false, false
			}
		}
		a.flash("已卸载 " + sp.Name)
		return
	}
	a.bg("lsp.uninstall", func() func() {
		err := sp.UninstallManaged()
		return func() {
			if err != nil {
				a.flash("卸载失败:" + err.Error())
				lspMgrLog.Warn("卸载托管语言包失败", "server", sp.Name, "err", err)
				return
			}
			a.outAppend(outLSP, sp.Name+" 已卸载")
			a.flash("已卸载 " + sp.Name)
			a.lspAfterChange(sp)
		}
	})
}

// lspShowLog 打开底部「输出 → 语言服务」(设置页在 IDE 之外时只切好状态,提示去哪里看)。
func (a *App) lspShowLog() {
	a.ide.bottomOpen, a.ide.bottomTab = true, "output"
	a.out.ch = outLSP
	a.flash("日志在 IDE 底部「输出 → 语言服务」")
}

// ───────── 演示模式:模拟一次下载安装(没有联网,供设计稿对照 / 截图 / 走一遍状态) ─────────

// lspJobDemo 用定时器走完「下载 Node → 安装包 → 完成」,过程里界面状态与真实一致。
func (a *App) lspJobDemo(j *lspJob) {
	key := lspJobKey(j.sp)
	tick := func(d time.Duration, f func()) {
		time.AfterFunc(d, func() {
			a.onUI(func() {
				if a.lspJobs[key] == j && j.running {
					f()
				}
			})
		})
	}
	if !j.sp.Managed() { // 执行安装命令的:没有可量的进度,只有「执行 …」一个阶段
		j.stage, j.done, j.total = "执行 "+j.sp.Install, 0, 0
		tick(2200*time.Millisecond, func() {
			for i := range a.set.lspRows {
				r := &a.set.lspRows[i]
				if sp := lspSpecByID(r.ID); sp != nil && lspJobKey(sp) == key {
					r.OK, r.MgrPath, r.Path = true, false, "/opt/homebrew/bin/"+sp.Bins[0]
				}
			}
			a.lspJobDone(j, nil, false)
		})
		return
	}
	const totalMB = 49 << 20
	j.stage, j.done, j.total = "下载 Node "+lsp.ManagedNodeVersion, 0, totalMB
	for i := 1; i <= 8; i++ {
		i := i
		tick(time.Duration(i)*250*time.Millisecond, func() { j.done = totalMB * int64(i) / 8 })
	}
	tick(2300*time.Millisecond, func() { j.stage, j.done, j.total = "安装 "+strings.Join(j.sp.NPM, " "), 0, 0 })
	tick(3600*time.Millisecond, func() {
		for i := range a.set.lspRows {
			r := &a.set.lspRows[i]
			if sp := lspSpecByID(r.ID); sp != nil && lspJobKey(sp) == key {
				r.OK, r.MgrPath, r.Stale = true, true, false
				r.Path = "~/.minicode/lsp/" + key + "/node_modules/.bin/" + sp.Bins[0]
				r.Node = "托管 " + lsp.ManagedNodeVersion
			}
		}
		a.lspJobDone(j, nil, false)
	})
}

// fmtMB 把字节数写成「21 MB」。
func fmtMB(n int64) string { return fmt.Sprintf("%d MB", (n+(1<<20)/2)>>20) }

// lspJobText 是行上的进度文字:阶段 +(有总量时)百分比与大小。
func lspJobText(j *lspJob) string {
	if !j.running {
		return "排队中,等前一个安装完成…"
	}
	if p := j.pct(); p >= 0 {
		return fmt.Sprintf("%s · %d%%(%s / %s)", j.stage, p, fmtMB(j.done), fmtMB(j.total))
	}
	return j.stage + "…"
}
