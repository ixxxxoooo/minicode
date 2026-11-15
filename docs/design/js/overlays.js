/* 设计稿 v3 · 浮层与面板规范页:任务坞、通知中心、更新与关于、选择器、MCP / 上下文面板、会话流补充、对话框与菜单、空状态与错误
   每一块都标出它用到的令牌;所有浮层只用 --bg-raised + --sh-2 / --sh-3 + --r-lg,不引入新的视觉语言。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk } = U;
  const { spec, sec, head } = P.h;
  const g2 = (...x) => `<div class="grid2">${x.join('')}</div>`, g3 = (...x) => `<div class="grid3">${x.join('')}</div>`;
  const mi = (n, t, r = '', on = '') => `<div class="mi ${on ? 'on' : ''}">${n ? ic(n, 's12') : ''}<span class="grow ell">${t}</span>${r ? `<span class="t3 fs-cap">${r}</span>` : ''}</div>`;

  /* ───────── 任务坞与后台任务 ───────── */
  const dockRow = (n, t, d, time, o = {}) => `<div class="row" style="height:auto;padding:5px 8px;align-items:flex-start"><span style="margin-top:3px">${o.stale ? ic('alert', 's12 c-warn') : o.done ? ic('check', 's12 c-ok') : '<span class="spin"></span>'}</span>
    <div class="grow"><div class="f ai g3"><span class="m ell" style="color:var(--tx-1)">${t}</span>${tag(n)}</div><div class="t3 fs-cap ell ${o.done || o.stale ? '' : 'shimmer'}">${d}</div></div><span class="aux">${time}</span>${o.done ? '' : ib('stop', 'sm', '停止')}</div>`;
  const dockPills = () => `<div class="tdock">${[['bot', '子代理', 2, 1], ['terminal', '后台命令', 1], ['eye', '监听', 1], ['layers', '工作流', 0], ['check', '已结束', 3]].map(([i, t, n, on]) => chip(`${t} <b class="num">${n}</b>`, i, on ? 'on' : 'line')).join('')}</div>`;
  const taskSec = sec('任务坞与后台任务', '输入框上方一排固定的类别胶囊;点哪个就在上方悬浮展开那一类,再点同一个或点外部收起。',
    g3(spec('任务坞 · 展开子代理', ['.tdock', '.pop', '--sh-2'], `<div class="pop" style="padding:4px;margin-bottom:6px">${dockRow('子代理', '审查 handlers 的错误处理', '搜索 "unwrap()"', '48 秒')}${dockRow('子代理', '调研 axum 的路由匹配', '读取 routing/mod.rs', '1 分 20 秒')}</div>${dockPills()}`),
      spec('停滞提示 · 后台命令', ['--warn', '.shimmer'], `<div class="pop" style="padding:4px">${dockRow('后台命令', 'cargo watch -x test', '等待文件变化…', '6 分')}${dockRow('子代理', '整理发布说明', '已 3 分钟没有新进展,可能卡住了', '5 分', { stale: 1 })}</div>`),
      spec('已结束 · 通知行', ['.note-line', '--tx-3'], `<div class="fc g1">${F.note('check', '后台命令 <b>cargo watch</b> 已结束(退出码 0)')}${F.note('alert', '后台命令 <b>npm run dev</b> 已停止')}${F.note('check', '监听 <b>target/dev.log</b> 触发了 3 次')}</div><div class="sep"></div>${dockRow('子代理', '调研 axum 的路由匹配', '总结:建议在中间件里归一化路径', '1 分 20 秒', { done: 1 })}`)));

  /* ───────── 通知中心 · 更新 · 关于 ───────── */
  const nt = (lv, t, d, acts = '', time = '刚刚') => `<div class="nt ${lv}">${ic(lv === 'err' ? 'errc' : lv === 'warn' ? 'alert' : lv === 'ok' ? 'okc' : 'info', 's12 ' + { err: 'c-err', warn: 'c-warn', ok: 'c-ok', info: 'c-info' }[lv])}<div class="grow"><div class="m">${t}</div><div class="t3 fs-cap" style="line-height:1.45">${d}</div>${acts ? `<div class="f g2" style="margin-top:6px">${acts}</div>` : ''}</div><span class="t4 fs-micro">${time}</span></div>`;
  const noticeSec = sec('通知 · 更新 · 关于', 'IDE 里所有提示走通知中心:右下角弹出并记入历史,状态栏铃铛显示未读数。Agent 窗口保持顶部轻提示。',
    g3(spec('通知面板(状态栏铃铛)', ['.nt', '--sh-2', '.cnt.ac'], `<div class="pop" style="padding:0;overflow:hidden"><div class="f ai g3" style="padding:8px 10px;box-shadow:inset 0 -1px var(--bd-1)"><b class="grow">通知</b>${cnt(3, 'ac')}${btn('全部清除', 'sm ghost')}</div>
        ${nt('warn', '未安装 rust-analyzer', '语言智能暂不可用。', btn('复制安装命令', 'sm', 'copy'))}${nt('info', '发现新版本 0.2.0', '已自动下载,重启后生效。', btn('重启更新', 'sm soft'), '3 分钟')}${nt('err', '保存失败', 'router.rs:磁盘上的文件已被外部修改。', btn('重新载入', 'sm') + btn('覆盖保存', 'sm ghost'), '12 分钟')}</div>`),
      spec('右下角弹出(自动消失)', ['.a-toast', '--dur-slow'], `<div class="fc g3" style="align-items:flex-end"><div class="pop a-toast" style="width:260px;padding:0">${nt('ok', '已提交', 'fix(router): 路径匹配忽略大小写', '', '')}</div><div class="pop a-toast" style="width:260px;padding:0;animation-delay:.12s">${nt('info', '已切换到分支 feat/auth', '', '', '')}</div></div>`),
      spec('更新胶囊:四种状态', ['.meter', '--ac', '.ring'], `<div class="fc g3">${[['download', '下载更新 42%', 42], ['shieldCk', '校验中…', 100], ['cube', '解包中…', 100], ['refresh', '重启更新', 0]].map(([i, t, p]) => `<div class="upd ${p === 0 ? 'ready' : ''}">${ic(i, 's12')}<span class="grow">${t}</span>${p ? `<span class="meter" style="width:54px"><i style="width:${p}%"></i></span>` : ''}</div>`).join('')}<div class="t3 fs-cap">位置:Agent 左下角;IDE 状态栏右侧「↓ 更新 42%」。</div></div>`)));
  const aboutCard = () => `<div class="dlg" style="width:300px;align-items:center;text-align:center">${U.brand.logo(56)}<div class="fs-head b">MiniCode</div><div class="t3 fs-cap">版本 0.2.0(构建 4f2a91c)<br>构建于 3 天前 · 2026-11-12 09:41</div><div class="f g2">${btn('检查更新', '', 'refresh')}${btn('复制版本信息', 'ghost', 'copy')}</div><div class="t4 fs-micro">© 2026 MiniCode · MIT</div></div>`;

  /* ───────── 选择器与面板 ───────── */
  const pickSec = sec('选择器与面板', '顶部居中的快速选择浮层(分支、历史会话),以及输入框上的 MCP / 上下文弹层。上下文弹层由输入框右侧圆环(点击展开,再点 / 点外部 / Esc 收起)与 IDE 状态栏「上下文」共用同一份内容;≥85% 圆环变警告色、压缩升为主按钮,≥95% 变错误色。',
    g3(spec('分支选择器', ['.pop', '--r-xl', '.mi'], `<div class="pop" style="padding:0;overflow:hidden"><div class="pal-in" style="height:36px;font-size:var(--fs-body)">${ic('branch', 's12 t3')}<span class="caret">feat</span></div><div style="padding:4px">
        <div class="cap">本地</div>${mi('check', 'main', '当前', 1)}${mi('branch', 'feat/auth', '2 小时前')}${mi('branch', 'feat/ui-redesign', '刚刚')}<div class="cap">远程</div>${mi('globe', 'origin/dev', '昨天')}<div class="sep"></div>${mi('plus', '创建并切换到 “feat”', '⏎')}</div></div>`),
      spec('历史会话选择器', ['.pop', '.row'], `<div class="pop" style="padding:0;overflow:hidden"><div class="pal-in" style="height:36px;font-size:var(--fs-body)">${ic('history', 's12 t3')}<span class="t4">搜索历史会话</span></div><div style="padding:4px"><div class="cap">今天</div>${mi('chat', '完整修复一个 Bug:路由大小写匹配', '刚刚', 1)}${mi('chat', '权限请求:修改配置与执行命令', '5 分钟')}<div class="cap">昨天</div>${mi('chat', 'Markdown 与代码高亮:发布说明', '18:20')}${mi('chat', '补充集成测试并修复 CI', '11:02')}<div class="cap">更早</div>${mi('chat', '重构 handlers 模块的错误处理', '周一')}</div></div>`),
      spec('MCP 弹层(输入框)', ['.tog', '.dot', '--tool-mcp'], `<div class="pop" style="padding:0"><div style="padding:8px 10px;box-shadow:inset 0 -1px var(--bd-1)"><div class="f ai"><b class="grow">已启用 3 / 4</b><span class="t3 fs-cap">工具定义占上下文 3.2k · 1.6%</span></div></div><div style="padding:4px">
        ${[['github', '14 个工具 · 3.2k token', 'ok', 1], ['filesystem', '8 个工具 · 按需加载', 'ok', 1], ['figma', '需要授权', 'warn', 1], ['postgres', '连接失败:ECONNREFUSED', 'err', 0]].map(([n, d, s, on]) => `<div class="row" style="height:36px">${ic('chevR', 's12 t4')}<span class="dot ${s}"></span><div class="grow"><div class="m">${n}</div><div class="t3 fs-micro">${d}</div></div>${tog(on)}</div>`).join('')}<div class="sep"></div>${mi('sliders', '管理 MCP…')}</div></div>`),
      spec('上下文弹层 · 常态', ['.ctx-pop', '.ctxbar', '.tick', '.meter'], `<div class="pop ctx-pop">${F.ctxBody()}</div>`),
      spec('上下文弹层 · 接近阈值', ['.ring.warn', '.ctx-hint', '.btn.pri'], `<div class="pop ctx-pop">${F.ctxBody({ state: 'warn' })}</div>`),
      spec('上下文弹层 · 压缩中', ['.btn.loading', '.meter.indet'], `<div class="pop ctx-pop">${F.ctxBody({ state: 'busy' })}</div>`),
      spec('上下文弹层 · 明细加载中', ['.skel'], `<div class="pop ctx-pop">${F.ctxBody({ state: 'load' })}</div>`),
      spec('上下文弹层 · 无账户限额', ['.ctx-none'], `<div class="pop ctx-pop">${F.ctxBody({ state: 'none' })}</div>`),
      spec('上下文弹层 · 限额查询失败', ['.ctx-hint.err'], `<div class="pop ctx-pop">${F.ctxBody({ state: 'err' })}</div>`),
      spec('输入框 · 点圆环展开', ['.ctx-chip', '.a-pop-up', '--sh-2'], `<div style="padding-top:330px;position:relative">${F.composer({ focus: true, ctx: true })}</div>`),
      spec('关于窗口', ['.dlg', '--sh-3', '--r-xl'], `<div class="f jc">${aboutCard()}</div>`),
      spec('Agent 左下角', ['.upd', '.chip', '.cnt'], `<div class="fc g3" style="width:228px;background:var(--bg-chrome);border-radius:var(--r-md);padding:8px">
        <div class="f ai g2">${ib('timer', 'tip-u', '自动化')}${ib('canvas', 'tip-u', '画布')}${ib('globe', 'tip-u', '浏览器')}${ib('plug', 'tip-u', '自定义')}<span class="grow"></span>${ib('gear', 'tip-u', '设置')}</div><div class="upd">${ic('download', 's12')}<span class="grow">下载更新 42%</span></div><div class="f ai g2 t3 fs-cap"><span class="tag ac">开发版</span><span class="grow"></span>0.2.0</div></div>`)));

  /* ───────── 会话流补充 ───────── */
  const msgSec = sec('会话流补充', '长消息折叠、编辑、回合折叠、回退文件、图表卡与附件;都沿用步骤行栅格与卡片层级。',
    g3(spec('长消息 · 折叠为 6 行', ['.msg-u', '.fade', '--dur-slow'], `<div class="msg-u clamp">请按下面的清单逐项处理:<br>1. 把 normalize 抽到单独的模块<br>2. 补 4 条大小写用例<br>3. 修复 CI 里失败的 clippy 警告<br>4. 更新 README 的路由示例<br>5. 把 fallback 逻辑改成显式配置<br>6. 清理未使用的依赖<br>7. 补充发布说明<div class="fade"></div></div><div class="f ai g2 t3 fs-cap" style="margin-top:6px">${ic('chevD', 's12')}展开 · 共 14 行</div>`),
      spec('编辑已发送的消息', ['.dockcard', '.on-out'], `<div class="dockcard" style="box-shadow:0 0 0 1px var(--bd-3),var(--sh-2)"><div class="cmp-in v" style="padding:0">把 normalize 抽到单独的模块,并补 4 条大小写用例</div><div class="f ai g3"><span class="t3 fs-cap">发送后会回退这一轮之后的文件改动</span><span class="grow"></span>${btn('取消', 'sm ghost')}${btn('发送', 'sm pri')}</div></div>`),
      spec('回合折叠 · 已处理', ['.step', '--tx-3'], `${F.step('group', '1 分 13 秒', '3 步 · 2 个文件', '', { verb: '已处理', body: `<div class="nest">${F.step('read', 'router.rs', 'src/', '86 行')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span>')}</div>` })}<div class="t4 fs-cap" style="padding:6px 0 0 22px">点开查看本回合的全部步骤</div>`),
      spec('回退文件预览', ['.dlg', '.dl'], `<div class="dlg" style="width:auto;gap:8px"><b>回退到这条消息之前?</b><div class="t3 fs-cap">以下 2 个文件会恢复到发送这条消息之前的内容:</div><div class="card" style="padding:2px 4px;background:var(--bg-sunken);box-shadow:inset 0 0 0 1px var(--bd-1)">${['router.rs|src|4|1', 'router_test.rs|tests|12|0'].map(r => r.split('|')).map(([n, d, a, b]) => `<div class="row">${ic('code', 's12')}<span class="grow ell">${n} <span class="t4">${d}</span></span><span class="c-ok mono fs-micro">+${a}</span><span class="c-err mono fs-micro">−${b}</span></div>`).join('')}</div><div class="f g2 jc" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('只回退对话', '')}${btn('回退文件', 'pri', 'undo')}</div></div>`),
      spec('图表卡(Mermaid)', ['.cv-card', '--bg-sunken', '.dg'], `<div class="cv-card" style="margin:0"><div class="cv-ch">${ic('layers', 's12')}<span class="grow">请求经过路由表</span>${ib('max', 'sm', '全屏查看')}${ib('code', 'sm', '查看源码')}</div>${W.diagram()}</div>`),
      spec('附件与引用', ['.ref', '--info-soft'], `<div class="fc g3"><div class="f g2 wrap">${F.ref('router.rs')}${F.ref('router_test.rs')}${F.ref('报错截图.png', 'image')}${F.ref('src/', 'folder')}</div><div class="f g3"><div class="thumb-img">${ic('image', 's20 t4')}</div><div class="thumb-img">${ic('image', 's20 t4')}<span class="x">${ic('x', 's12')}</span></div></div><div class="t3 fs-cap">图片点击进入全屏,Esc 关闭</div></div>`)));

  /* ───────── 对话框与菜单 ───────── */
  const ctx = (items) => `<div class="pop" style="width:100%">${items.map(i => i === '-' ? '<div class="sep"></div>' : mi(i[0], i[1], i[2] || '')).join('')}</div>`;
  const dlgSec = sec('对话框与菜单', '确认框用于不可逆或会打断运行的操作;右键菜单对标 VS Code,同一个行高与圆角令牌。',
    g3(spec('确认框 · 切换到演示模式', ['.dlg', '--sh-3', '.scrim'], `<div class="dlg" style="width:auto"><div class="f ai g3"><span class="dlg-i">${ic('swap')}</span><b class="fs-title">切换到演示模式?</b></div><div class="t3" style="line-height:1.55">MiniCode 会重启。有 <b>1 个终端</b>正在运行命令,重启后会被结束。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('重启并切换', 'pri')}</div></div>`),
      spec('确认框 · 未保存的修改', ['.dlg', '.btn.danger'], `<div class="dlg" style="width:auto"><div class="f ai g3"><span class="dlg-i warn">${ic('alert')}</span><b class="fs-title">保存对 router.rs 的修改?</b></div><div class="t3" style="line-height:1.55">不保存的话,这些修改会丢失。</div><div class="f g2" style="justify-content:flex-end">${btn('不保存', 'danger ghost')}${btn('取消', 'ghost')}${btn('保存', 'pri')}</div></div>`),
      spec('确认框 · 删除会话', ['.dlg', '--err'], `<div class="dlg" style="width:auto"><div class="f ai g3"><span class="dlg-i err">${ic('trash')}</span><b class="fs-title">删除这个会话?</b></div><div class="t3" style="line-height:1.55">会话记录会从磁盘移除,无法恢复。</div><div class="f g2" style="justify-content:flex-end">${btn('取消', 'ghost')}${btn('删除', 'danger', 'trash')}</div></div>`),
      spec('编辑器右键菜单', ['.pop', '.mi', 'kbd'], ctx([['code', '转到定义', 'F12'], ['code', '转到实现'], ['search', '查找所有引用', '⇧F12'], '-', ['edit', '重命名符号', 'F2'], ['sliders', '格式化文档', '⇧⌥F'], ['wand', '重构…'], '-', ['branch', 'Git Blame'], ['history', '显示文件历史'], ['cmd', '命令面板…', '⇧⌘P']])),
      spec('标签右键菜单', ['.pop', '.sep'], ctx([['x', '关闭', '⌘W'], ['x', '关闭其他'], ['x', '关闭已保存的'], '-', ['cols', '向右拆分', '⌘\\'], ['cols', '向下拆分'], '-', ['pin', '固定标签'], ['folder', '在访达中显示'], ['copy', '复制路径']])),
      spec('文件树右键菜单', ['.pop', '--r-lg'], ctx([['file', '新建文件…'], ['folder', '新建文件夹…'], '-', ['spark', '让 Claude 处理这个文件'], ['at', '引用到对话', '⌥⌘K'], '-', ['edit', '重命名…', '⏎'], ['copy', '复制'], ['trash', '删除', '⌘⌫']]))));

  /* ───────── 空状态与错误 ───────── */
  const empty = (i, t, d, a = '') => `<div class="empty">${ic(i, 's20 t4')}<b>${t}</b><div class="t3 fs-cap" style="line-height:1.5">${d}</div>${a}</div>`;
  const emptySec = sec('空状态、加载与错误', '空白不留白:每个空状态都说明发生了什么、下一步能做什么,并给出一个主操作。',
    g3(spec('没有项目', ['.empty', '--tx-4'], empty('folder', '还没有项目', '添加一个本地文件夹,Claude 会在里面读写代码、跑命令。', btn('添加项目', 'pri', 'plus'))),
      spec('找不到 claude', ['.notice', '--err-soft'], `<div class="notice err" style="flex-direction:column;gap:8px"><div class="f ai g3">${ic('errc', 'c-err')}<b>没有找到 claude 命令行</b></div><div class="t2" style="line-height:1.55">MiniCode 直连本机的 <code class="mono">claude</code>。安装后重新检测,或在「高级」里手动指定路径。</div><div class="f g2">${btn('复制安装命令', 'sm', 'copy')}${btn('指定路径…', 'sm ghost')}${btn('重新检测', 'sm ghost', 'refresh')}</div></div>`),
      spec('连接中断 · 自动重连', ['.notice', '--warn-soft', '.shimmer'], `<div class="notice warn">${ic('alert', 'c-warn')}<div class="grow"><b>与 claude 的连接已断开</b><div class="t2 shimmer">正在重新连接…(第 2 次)</div></div>${btn('立即重试', 'sm', 'refresh')}</div><div class="sep"></div>${empty('search', '没有匹配的结果', '换个关键字,或在「···」里放宽包含 / 排除规则。')}`),
      spec('加载中 · 骨架', ['.skel', '--dur-loop', '.shimmer'], `<div class="fc g3"><div class="skel" style="width:60%"></div><div class="skel"></div><div class="skel" style="width:82%"></div><div class="f g3 ai"><div class="skel" style="width:28px;height:28px;border-radius:50%"></div><div class="grow fc g2"><div class="skel" style="width:40%"></div><div class="skel"></div></div></div></div>`),
      spec('没有改动', ['.empty'], empty('check', '没有未提交的改动', '工作区与最近一次提交一致。')),
      spec('权限被拒绝', ['.step.deny', '--tx-3'], `${F.step('bash', 'rm -rf target', '', '已拒绝', { state: 'deny' })}${F.note('shield', '你拒绝了这条命令,Claude 会换一种方式继续。')}`)));

  /** 浮层与面板规范页 */
  P.overlays = () => `<div class="ds">${head('浮层与面板', '工作台之外的所有「叠上去」的东西:任务、通知、选择器、对话框、菜单与空状态。它们共用同一套层级阴影与圆角令牌。')}
    ${taskSec}${noticeSec}${pickSec}${msgSec}${dlgSec}${emptySec}</div>`;
})();
