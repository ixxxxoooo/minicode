/* 设计稿 v3.1 · 工作区:编辑区组(标签条 + 内容)与底部面板
   一个窗口只有一个工作区,Agent 形态下它停靠在对话栏右侧(按需展开),IDE 形态下它占主位;
   文件、差异、审查、计划、子 Agent、任务输出、合并冲突都是它的「标签」,会话流里点什么就在这里开什么。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, chk, rust } = U;
  Object.assign(ICONS, {
    arrowL: '<path d="M19 12H5M11 6l-6 6 6 6"/>',
    merge: '<circle cx="6" cy="5.5" r="2"/><circle cx="6" cy="18.5" r="2"/><circle cx="18" cy="11" r="2"/><path d="M6 7.5v9M18 13c0 3-4 3-8 3"/>',
    bulb: '<path d="M9 18h6M10 21h4M12 3a6 6 0 0 0-3.5 10.9c.6.5 1 1.2 1 2.1h5c0-.9.4-1.6 1-2.1A6 6 0 0 0 12 3z"/>',
    unfold: '<path d="M8 9l4-4 4 4M8 15l4 4 4-4"/>',
    quote: '<path d="M6 17c0-4 1-6 4-7M14 17c0-4 1-6 4-7"/>',
  });
  const WS = {};

  /** 标签注册表:id → 种类 / 图标 / 标题。种类决定内容与标签上的色相 */
  WS.TAB = {
    'router.rs': { k: 'file', icon: 'code', t: 'router.rs' },
    'router_test.rs': { k: 'file', icon: 'code', t: 'router_test.rs', mod: 1 },
    'Cargo.toml': { k: 'file', icon: 'gear', t: 'Cargo.toml', pv: 1 },
    'main.py': { k: 'file', icon: 'code', t: 'main.py' },
    'diff:router.rs': { k: 'diff', icon: 'diff', t: 'router.rs (HEAD ↔ 工作区)' },
    review: { k: 'review', icon: 'eye', t: '审查 · 本轮改动', bd: 3 },
    plan: { k: 'plan', icon: 'list', t: '计划 · 路由大小写修复' },
    'sub:handlers': { k: 'sub', icon: 'bot', t: '审查 handlers 的错误处理' },
    'task:watch': { k: 'task', icon: 'activity', t: 'cargo watch -x test' },
    conflict: { k: 'conflict', icon: 'merge', t: 'router.rs (合并冲突)', bd: 2 },
  };
  WS.DEFAULT = ['router.rs', 'router_test.rs', 'Cargo.toml', 'review'];

  const etab = (id, on) => { const d = WS.TAB[id];
    return `<div class="etab ${on ? 'on' : ''} ${d.pv ? 'pv' : ''}" data-k="${d.k}" data-act="wtab" data-id="${id}">${ic(d.icon, 's12')}<span class="nm">${d.t}</span>${d.bd ? `<span class="bd">${d.bd}</span>` : ''}${d.mod ? '<span class="mod"></span>' : `<span class="x" data-act="wx" data-id="${id}">${ic('x', 's12')}</span>`}</div>`; };
  /** 标签条:标签 + 右侧工具(向右拆分 / 更多) */
  WS.strip = (tabs, active) => `<div class="etabs">${tabs.map(id => etab(id, id === active)).join('')}<div class="etab-tools">${ib('cols', 'sm', '向右拆分 ⌘\\')}${ib('more', 'sm', '更多')}</div></div>`;

  /* ───────── 内容:差异行 ───────── */
  const dl = (k, no, tx) => `<div class="dl ${k}"><span class="no">${no}</span><span class="tx">${rust(tx)}</span></div>`;
  const dls = (rows) => rows.map(r => dl(...r)).join('');
  const H1 = [['', 11, '    pub fn add(&mut self, path: &str, h: Handler) {'], ['del', 12, '        self.routes.insert(path.to_string(), h);'], ['add', 12, '        let key = normalize(path);'], ['add', 13, '        self.routes.insert(key, h);'], ['', 14, '    }']];
  const H2 = [['', 17, '    pub fn match_path(&self, path: &str) -> Option<&Handler> {'], ['del', 18, '        self.routes.get(path)'], ['add', 18, '        let key = normalize(path);'], ['add', 19, '        self.routes.get(&key).or(self.fallback.as_ref())'], ['', 20, '    }']];
  const hunk = (label, acts = true) => `<div class="rv-hunk"><span class="grow">${label}</span>${acts ? `${btn('评论', 'ghost', 'chat', 'data-act="rv-cm"')}${btn('撤销', 'ghost no', 'undo', 'data-act="rv-hunk" data-do="no"')}${btn('接受', 'ghost ok', 'check', 'data-act="rv-hunk" data-do="ok"')}` : tag('已接受', 'ok')}</div>`;
  const fileHead = (name, dir, a, d, o = {}) => `<div class="rv-fh" data-act="rv-fold">${ic('chevD', 's12 t3 fc-car')}${ic('code', 's12 t3')}<b class="ell">${name}</b><span class="t4 fs-micro ell">${dir}</span><span class="grow"></span>${a ? `<span class="c-ok mono fs-micro">+${a}</span>` : ''}${d ? `<span class="c-err mono fs-micro">−${d}</span>` : ''}<span class="vw" data-act="rv-viewed">${chk(o.viewed ? 'done' : '')}已查看</span></div>`;

  /** 审查标签:Claude 本轮改动的多文件审查(块级接受 / 撤销、逐行评论、已查看) */
  WS.review = () => `<div class="rv-head">${ic('eye', 's12 c-ac')}<b>本轮改动</b><span class="t3">3 个文件</span><span class="c-ok mono">+18</span><span class="c-err mono">−2</span><span class="grow"></span>${seg(['行内', '左右'], 0)}${btn('全部撤销', 'sm ghost', 'undo')}${btn('全部接受', 'sm soft', 'check')}${btn('提交…', 'sm pri', 'commit', 'data-act="open" data-id="diff:router.rs"')}</div>
    <div class="rv-body scroll">
      <div class="rv-file">${fileHead('router.rs', 'src/', 4, 2)}<div class="rv-fb">${hunk('第 11–14 行 · 注册前先归一化路径')}${dls(H1)}${hunk('第 17–20 行 · 匹配前先归一化路径')}${dls(H2)}
        <div class="rv-cm"><div class="who"><span class="avatar" style="width:16px;height:16px;font-size:9px">L</span>你 · 评论第 18 行</div><div class="inp fill" style="height:auto;min-height:24px;padding:4px 8px"><span class="v">原路径还要作为回退保留吗?</span></div>
          <div class="f g2 ai"><span class="t3 fs-cap grow">评论只留在这里,点「发送给 Claude」才会作为下一条消息</span>${btn('取消', 'sm ghost')}${btn('发送给 Claude', 'sm pri', 'send')}</div></div></div></div>
      <div class="rv-file closed">${fileHead('router_test.rs', 'tests/', 12, 0, { open: false })}</div>
      <div class="rv-file closed">${fileHead('case_test.rs', 'tests/', 2, 0, { open: false })}</div>
      <div class="rv-sum">${ic('info', 's12 c-info')}<span class="grow t2">还没有提交。接受 = 保留(不写入 Git);撤销 = 把这一块还原成改动前。</span>${btn('在源代码管理里暂存', 'sm', 'branch', 'data-act="nav" data-nav="scm"')}</div></div>
    <div class="rv-foot"><span class="grow">已查看 0 / 3 · 来自第 3 回合 · 11 月 15 日 18:20</span>${btn('回退到这一回合之前', 'sm ghost', 'rewind')}</div>`;

  /** 计划标签:ExitPlanMode 的全文,头部固定 Build 按钮 */
  WS.plan = () => `<div class="pl-head">${ic('list', 's16 c-warn')}<b class="grow ell">路由大小写修复计划</b>${tag('待批准', 'warn')}${btn('继续规划', 'sm ghost')}${btn('自动模式', 'sm opt-sec')}${btn('开始实施', 'sm pri', 'play')}</div>
    <div class="pl-body scroll"><div class="pl-col">
      <div class="pl-sec"><h4>执行进度</h4><div class="f ai g3" style="margin-bottom:8px"><div class="meter grow"><i style="width:66%"></i></div><span class="t3 num">2 / 3</span></div>${F.todo(2)}</div>
      <div class="pl-sec"><h4>背景</h4><div class="t2" style="line-height:1.65">路由表用原始路径做键,<b>注册</b>与<b>匹配</b>两处都没有归一化,所以 <span class="mono">/Users</span> 找不到 <span class="mono">/users</span>。</div></div>
      <div class="pl-sec"><h4>步骤</h4><ol><li>抽出 <span class="mono">normalize</span>,注册与匹配共用<small>去掉末尾斜杠并转小写,放在 router.rs 底部</small></li><li>匹配失败时回退到 <span class="mono">fallback</span><small>行为与原先一致,避免影响默认路由</small></li><li>补 2 条测试并跑通<small>大小写、末尾斜杠各一条,放进 router_test.rs</small></li></ol></div>
      <div class="pl-sec"><h4>涉及文件</h4><div class="f g2 wrap">${F.ref('router.rs').replace('class="ref"', 'class="ref" data-act="open" data-id="router.rs"')}${F.ref('router_test.rs').replace('class="ref"', 'class="ref" data-act="open" data-id="router_test.rs"')}</div></div>
      <div class="pl-ann">${ic('chat', 's12')}<span class="grow">选中任意一段文字可以「批注」,批注会随「继续规划」一起发给 Claude</span></div></div></div>`;

  /** 子 Agent 标签 = 与主会话流同一套 UI(用户消息展示任务 + 步骤组 + 助手正文 + 结果卡) */
  WS.sub = () => `<div class="sa-head">${ic('bot', 's16 c-ac')}<b class="grow ell">审查 handlers 的错误处理</b>${tag('运行中', 'ac')}<span class="t3 fs-cap nw num">48 秒 · 12 个工具 · 6.2k</span>${btn('引用到对话', 'sm', 'quote')}${btn('复制全部', 'sm ghost', 'copy')}${ib('stop', 'sm', '停止')}</div>
    <div class="sa-body scroll sa-flow"><div class="sa-col flow-col">
      ${F.user('检查 src/handlers 里所有 <span class="mono">unwrap()</span> 与 <span class="mono">expect()</span>,按「可能在请求路径上 panic」排序,不要修改任何文件。')}
      <div class="acts-g">${F.step('think', '列出目录与风险点', '', '3 秒', { open: true, body: F.think() })}${F.step('glob', 'src/handlers/**/*.rs', '', '4 个文件')}${F.step('grep', '"unwrap()"', 'src/handlers', '7 处')}${F.step('read', 'user.rs', 'src/handlers · L1–64', '64 行', { state: 'run' })}</div>
      ${F.toLatest(true)}</div></div>`;

  /** 任务输出标签:后台命令 / 监听的实时输出 */
  WS.task = () => `<div class="tk-head">${ic('activity', 's16 c-ac')}<b class="grow ell mono">cargo watch -x test</b>${tag('运行中', 'ac')}<span class="t3 fs-cap nw num">6 分 12 秒 · pid 48213</span>${ib('refresh', 'sm', '重启')}${ib('stop', 'sm', '停止')}</div>
    <div class="tk-body scroll"><div class="tk-col"><div class="f ai g3 t3 fs-cap"><span class="grow">来自会话「完整修复一个 Bug」· 后台命令</span><span class="f ai g2">${U.tog(1)}跟随输出</span>${btn('在终端面板中打开', 'sm ghost', 'terminal')}</div>
      <div class="tk-log"><span class="d">[Running 'cargo test']</span>
<span class="d">   Compiling</span> router-demo v0.3.1
<span class="d">    Finished</span> test profile in 1.92s
test matches_ignoring_case ... <span class="p">ok</span>
test matches_trailing_slash ... <span class="p">ok</span>
test result: <span class="p">ok</span>. 12 passed; 0 failed
<span class="d">[Finished running. Exit status: 0]</span>
<span class="d">[Running 'cargo test']</span>
<span class="d">   Compiling</span> router-demo v0.3.1 <span class="caret"></span></div></div></div>`;

  /** 合并冲突标签(三方):当前 / 传入并排,下方是结果,每处冲突都有逐块操作 */
  const cfL = (cls, rows) => rows.map(t => `<div class="${cls}">${rust(t)}</div>`).join('');
  WS.conflict = () => `<div class="cf-head">${ic('merge', 's16 c-err')}<b class="grow ell">router.rs</b>${tag('2 处冲突 · 已解决 0', 'err')}${btn('让 Claude 解决', 'sm soft', 'spark')}${btn('标记为已解决', 'sm', 'check')}</div>
    <div class="cf-wrap"><div class="cf-act">${ib('chevL', 'sm', '上一处')}${ib('chevR', 'sm', '下一处')}<span class="grow t2" style="text-align:center">冲突 1 / 2 · 第 17–20 行</span>${btn('接受当前', 'ghost', '', '')}${btn('接受传入', 'ghost')}${btn('接受双方', 'ghost')}${btn('比较', 'ghost', 'diff')}</div>
      <div class="cf-two"><div class="cf-pane"><div class="cf-ph"><span class="dot ok"></span><b class="t2">当前更改</b><span>main · 你</span></div><div class="cf-code">${cfL('', ['    // 匹配路径'])}${cfL('cur', ['    pub fn match_path(&self, path: &str) -> Option<&Handler> {', '        let key = normalize(path);', '        self.routes.get(&key).or(self.fallback.as_ref())'])}${cfL('', ['    }'])}</div></div>
        <div class="cf-pane"><div class="cf-ph"><span class="dot info"></span><b class="t2">传入更改</b><span>feat/auth · Lin</span></div><div class="cf-code">${cfL('', ['    // 匹配路径'])}${cfL('inc', ['    pub fn match_path(&self, path: &str, auth: &Auth) -> Option<&Handler> {', '        let h = self.routes.get(path)?;', '        auth.check(h).then_some(h)'])}${cfL('', ['    }'])}</div></div></div>
      <div class="cf-ph" style="box-shadow:inset 0 1px var(--bd-2),inset 0 -1px var(--bd-1)">${ic('merge', 's12')}<b class="t2">结果</b><span class="grow"></span><span class="t4">可编辑 · 接受操作后会自动重算</span></div>
      <div class="cf-res scroll"><div class="inp" style="margin:0;border:none;border-radius:0;min-height:120px;font-family:var(--mono);font-size:12px;line-height:1.6;padding:8px 10px">    // 匹配路径\n    // ← 选一边,或接受双方后手动编辑\n    fn normalize(path: &str) -> String {\n        path.trim_end_matches('/').to_ascii_lowercase()\n    }</div></div></div>`;

  /** 工作区主体:按标签种类分发 */
  WS.body = (id, o) => { const k = (WS.TAB[id] || {}).k;
    return k === 'diff' ? IDE.diffView(o) : k === 'review' ? WS.review() : k === 'plan' ? WS.plan() : k === 'sub' ? WS.sub() : k === 'task' ? WS.task() : k === 'conflict' ? WS.conflict() : IDE.fileView(Object.assign({ file: id }, o)); };

  /** 整个工作区(标签条 + 内容 + 底部面板)。o.tabs / o.active / o.panel / 编辑器状态 */
  WS.html = (o = {}) => { const tabs = o.tabs || WS.DEFAULT, active = o.active || tabs[0];
    return `<div class="ws-in" data-tabs="${tabs.join(',')}" data-active="${active}">${WS.strip(tabs, active)}<div class="ws-body">${WS.body(active, o)}</div>${IDE.bpanel(o.panel)}</div>`; };

  window.WS = WS;
})();
