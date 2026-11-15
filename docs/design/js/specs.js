/* 设计稿 v3 · 规范页:会话流与工具、组件、设计令牌、动效。每一块都标出它用到的令牌名
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk, TOOLS, THEMES } = U;
  const P = {};
  const tok = (...n) => n.map(x => `<span class="tok">${x}</span>`).join('');
  const spec = (label, toks, body, cls = '') => `<div class="spec ${cls}"><div class="lb"><span>${label}</span><span class="grow"></span>${tok(...toks)}</div>${body}</div>`;
  const sec = (title, desc, body) => `<section class="sec"><h2>${title}</h2><p>${desc}</p>${body}</section>`;
  const head = (t, d) => `<h1>${t}</h1><div class="lede">${d}</div>`;

  /* ───────── 会话流与工具 ───────── */
  const EX = { think: ['定位大小写敏感的来源', '', '4 秒'], read: ['router.rs', 'src/ · L1–86', '86 行'], grep: ['"match_path"', 'src/', '5 处'], glob: ['**/*_test.rs', '', '3 个文件'], group: ['2 个文件 · 1 次搜索', '', ''],
    edit: ['router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−1</span>'], write: ['case_test.rs', 'tests/', '<span class="c-ok">+28</span>'], bash: ['cargo test router', '', '2.4 秒'], bg: ['cargo watch -x test', '', '已启动'], monitor: ['target/dev.log', '', '3 条'],
    fetch: ['docs.rs/axum/routing', '', '12 KB'], search: ['axum case insensitive route', '', '8 条结果'], mcp: ['github · create_issue', '', '#128'], agent: ['代码审查', 'handlers/', '1 分 20 秒'], todo: ['2 / 3 完成', '', ''], plan: ['路由大小写修复计划', '', '待批准'] };
  P.flow = () => `<div class="ds">${head('会话流与工具', '会话流里的每一行都是「步骤行」:图标列 16px 贴左,文字起于 22px。工具按「族」取色相,只染图标;状态靠动效表达,不靠颜色堆叠。')}
    ${sec('工具令牌', '每种工具 = 图标 + 动词 + 工具族。新增工具只要在注册表里加一行,颜色、流光、展开方式自动继承。',
      `<div class="spec" style="padding:6px 14px"><div class="tool-tb t3 fs-cap m" style="height:26px"><span>工具</span><span>色相令牌</span><span>运行中 · 流光 + 转圈</span><span>完成</span></div>
      ${Object.keys(TOOLS).map(k => `<div class="tool-tb"><span class="f ai g3"><span class="si" data-tool="${k}" style="color:var(--tone);display:grid">${ic(TOOLS[k].icon)}</span><b>${TOOLS[k].cn}</b></span><span>${tok('--tool-' + TOOLS[k].fam)}</span><div>${F.step(k, EX[k][0], EX[k][1], '', { state: 'run' })}</div><div>${F.step(k, EX[k][0], EX[k][1], EX[k][2])}</div></div>`).join('')}</div>`)}
    ${sec('步骤行的状态', '完成是默认态,不加任何标记;只有「运行中 / 失败 / 被拒绝」才有视觉变化。点击带箭头的行可展开。',
      `<div class="grid2">${spec('运行中', ['.shimmer', '--dur-loop'], F.step('bash', 'cargo build --release', '', '', { state: 'run' }) + F.step('group', '项目代码', '', '', { state: 'run', verb: '正在探索' }) + '<div class="ticker" data-ticks="read|router.rs;grep|match_path;read|router_test.rs;glob|**/*.toml"></div>')}
      ${spec('完成 · 失败 · 被拒绝', ['--err', '--tx-3'], F.step('read', 'server.rs', 'src/', '212 行') + F.step('bash', 'cargo clippy', '', '退出码 101', { state: 'err' }) + F.step('bash', 'rm -rf target', '', '已拒绝', { state: 'deny' }))}
      ${spec('展开:差异', ['.fold', '--dur-slow', '--diff-add'], F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span><span class="c-err">−1</span>', { open: true, body: F.diff(false) }))}
      ${spec('展开:终端(运行中贴底跟随)', ['--bg-sunken', '.caret'], F.step('bash', 'cargo test router', '', '', { state: 'run', open: true, body: F.term(true) }))}
      ${spec('思考 · 待办', ['--tool-think', '.chk'], F.step('think', '定位大小写敏感的来源', '', '4 秒', { open: true, body: F.think() }) + F.step('todo', '2 / 3 完成', '', '', { open: true, body: F.todo(2) }))}
      ${spec('子 Agent · 并行组', ['--tool-agent', '.ticker'], F.step('agent', '并行 2 个', '', '1 个运行中', { verb: '子 Agent' }) + `<div class="nest" style="gap:6px;padding-top:4px">${F.sub('代码审查', '审查 handlers 的错误处理', true, '<span>48 秒</span><span>7 次工具</span><span>12.4k token</span>')}${F.sub('调研', 'axum 的路由匹配实现', false, '<span>1 分 20 秒</span><span>11 次工具</span><span>18.9k token</span>')}</div>`)}</div>`)}
    ${sec('交互卡', '需要你决定的事就地替换输入框:权限、提问。处理完输入框原位回来,视线不用移动。计划卡留在正文里。',
      `<div class="grid2">${spec('权限请求', ['.a-pop-up', '.a-glow', '--ac-line'], F.perm())}${spec('提问', ['.opt', '--ac-soft'], F.ask())}
      ${spec('计划 · 生成中', ['.shimmer', '.caret'], F.plan(true))}${spec('计划 · 待批准', ['--tool-plan'], F.plan(false))}</div>`)}
    ${sec('消息与提示', '用户消息、助手正文、回合操作条,以及三级提示(信息是一行灰字,警告与错误才成卡)。',
      `<div class="grid2">${spec('用户消息', ['--fs-read', '.ref'], F.user(`看一下 ${F.ref('router.rs')} 和 ${F.ref('报错截图.png', 'image')},修好并补上测试。`))}
      ${spec('提示分级', ['--warn-soft', '--err-soft'], `<div class="fc g3">${F.note('info', '已切换到「计划模式」')}${F.notice('warn', '上下文已用 92%,建议压缩后继续。', btn('压缩', 'sm'))}${F.notice('err', '<b>请求失败</b> · 529 服务过载', btn('重试', 'sm', 'refresh'))}</div>`)}
      ${spec('助手正文 + 操作条', ['--lh-read', '.tfoot'], F.md() + F.tfoot())}
      ${spec('输入框:运行中 + 任务坞 + 排队', ['.cmp', '.queue', '--ring'], `${F.tdock()}${F.queue()}${F.composer({ focus: true, running: true, text: `把 ${F.ref('router.rs')} 里的 normalize 挪到单独的模块` })}`)}
      ${spec('运行配置弹层(权限模式 + 模型 + 思考强度)', ['.cfg-pop', '.a-pop-up', '--sh-2'], F.cfgPop())}${spec('输入框:常态', ['--r-lg', '--ac-line'], F.composer())}</div>`)}</div>`;

  /* ───────── 组件 ───────── */
  P.comps = () => `<div class="ds">${head('组件', '所有组件只引用语义令牌;高度只有 20 / 22 / 24 / 26 四档,保证任何两个控件并排都能对齐。可以直接点、悬停、按下。')}
    <div class="grid3" style="margin-top:20px">
    ${spec('按钮', ['--h-ctl', '--r-sm'], `<div class="f g3 wrap ai">${btn('主要', 'pri')}${btn('柔和', 'soft')}${btn('默认')}${btn('幽灵', 'ghost')}${btn('危险', 'danger')}</div><div class="f g3 wrap ai" style="margin-top:8px">${btn('开始实施', 'pri sm', 'play')}${btn('审查', 'sm', 'eye')}${btn('重试', 'sm ghost', 'refresh')}</div>`)}
    ${spec('图标按钮', ['--fill-1', '--fill-2'], `<div class="f g2 ai">${ib('panelL', '', '导航栏')}${ib('search', 'on', '选中')}${ib('more')}${ib('copy', 'sm')}${ib('x', 'sm')}<span class="vsep" style="margin:0 6px"></span>${ib('send', 'send', '发送')}${ib('stop', 'stop', '停止')}</div>`)}
    ${spec('胶囊与徽标', ['.chip', '.tag', '.cnt'], `<div class="f g2 wrap ai">${chip('main', 'branch')}${chip('MCP 3', 'plug', 'on')}${chip('子代理 2', 'bot', 'line')}</div><div class="f g3 wrap ai" style="margin-top:8px">${tag('默认')}${tag('强调', 'ac')}${tag('通过', 'ok')}${tag('失败', 'err')}${tag('待批准', 'warn')}${tag('信息', 'info')}${cnt(3)}${cnt(12, 'ac')}</div>`)}
    ${spec('分段与开关', ['--ease-spring', '--dur-base'], `<div class="f g4 ai wrap">${seg(['Agent', 'IDE'], 0, 'lg')}${seg(['关', '低', '中', '高'], 2)}${tog(1)}${tog(0)}</div>`)}
    ${spec('输入', ['--ring', '--ac-line'], `<div class="fc g3"><div class="inp">${ic('search', 's12')}<span class="grow">搜索会话…</span>${kbd('⌘', 'K')}</div><div class="inp focus">${ic('search', 's12')}<span class="v caret">match_path</span></div><div class="inp fill">${ic('filter', 's12')}<span>筛选文件</span></div></div>`)}
    ${spec('标签条', ['--h-tab', '--ac'], `<div class="tabs" style="padding:0">${['改动', '文件', '终端', '计划'].map((t, k) => `<span class="tab ${k ? '' : 'on'}" data-act="tab">${t}${k ? '' : cnt(3)}</span>`).join('')}</div><div class="t3 fs-cap" style="margin-top:8px">下划线从中间向两侧展开,不做位移追踪。</div>`)}
    ${spec('菜单 / 弹层', ['--sh-2', '--r-lg', '.a-pop'], `<div class="pop" style="width:200px"><div class="cap">权限模式</div><div class="mi on">${ic('shield', 's12')}<span class="grow">逐项确认</span>${ic('check', 's12')}</div><div class="mi">${ic('edit', 's12')}<span class="grow">自动接受编辑</span>${kbd('⇧', '⇥')}</div><div class="sep"></div><div class="mi">${ic('sliders', 's12')}设置…</div></div>`)}
    ${spec('列表行', ['--h-row', '--r-sm'], `<div class="fc"><div class="row on"><span class="grow ell">完整修复一个 Bug:路由大小写匹配</span><span class="aux">刚刚</span></div><div class="row"><span class="grow ell">权限请求:修改配置与执行命令</span><span class="aux swap">5 分钟</span><span class="acts">${ib('more', 'sm')}</span></div><div class="row tr gm" style="--d:1"><span style="width:12px"></span>${ic('code', 's12')}<span class="ell">router.rs</span><span class="g">M</span></div></div><div class="t3 fs-cap" style="margin-top:6px">会话、文件、菜单同一个行高令牌;悬停时时间换成操作。</div>`)}
    ${spec('进度与状态', ['.spin', '.ring', '.meter'], `<div class="f g5 ai"><span class="spin"></span><span class="ring" style="--p:38"></span><span class="ring" style="--p:86;--ac:var(--warn)"></span><div class="grow"><div class="meter"><i style="width:62%"></i></div></div></div><div class="f g4 ai" style="margin-top:10px">${chk()}${chk('now')}${chk('done')}<span class="dot ok"></span><span class="dot warn"></span><span class="dot err"></span><span class="shimmer m">正在生成提交说明…</span></div>`)}
    ${spec('键帽与提示', ['kbd', '[data-tip]'], `<div class="f g4 ai wrap">${kbd('⌘', 'K')}${kbd('⇧', '⌘', 'F')}${kbd('esc')}${btn('悬停看提示', 'sm', '', 'data-tip="延迟 350ms 出现"')}</div>`)}
    ${spec('通知', ['.a-toast', '--sh-2'], `<div class="pop f g4" style="padding:9px 11px;align-items:flex-start">${ic('okc', 'c-ok')}<div class="grow"><b>已下载 0.2.0</b><div class="t3 fs-cap">重启后生效</div></div>${btn('重启更新', 'sm soft')}</div>`)}
    ${spec('卡片层级', ['--sh-1', '--sh-2', '--sh-3'], `<div class="f g4"><div class="card grow" style="height:44px;display:grid;place-items:center">卡片</div><div class="pop grow" style="height:44px;display:grid;place-items:center">弹层</div><div class="grow" style="height:44px;display:grid;place-items:center;border-radius:var(--r-lg);background:var(--bg-raised);box-shadow:var(--sh-3)">浮窗</div></div>`)}
    </div></div>`;

  /* ───────── 设计令牌 ───────── */
  const sw = (v, name) => `<div class="sw"><i style="background:var(${v})"></i><span>${name}</span><code>${v}</code></div>`;
  const swg = (title, items) => `<div class="spec"><div class="lb">${title}</div><div class="sw-g">${items.map(([v, n]) => sw(v, n)).join('')}</div></div>`;
  P.tokens = (st) => `<div class="ds">${head('设计令牌', '三层:原始色板(每套主题 14 个颜色)→ 语义令牌(组件只认这一层)→ 方案倍率(圆角 / 间隙 / 行高)。换主题、换方案都不动组件。')}
    ${sec('主题', '六套主题,点一下整份设计稿跟着换。每套只定义 5 个表面 / 墨色 + 1 个强调色 + 7 个色相,其余全部由 color-mix 推导。', `<div class="spec">${W.themeCards(st.theme)}</div>`)}
    ${sec('色彩 · 语义层', '下面是当前主题算出来的值。', `<div class="grid3">
      ${swg('表面', [['--bg-chrome', '窗口底'], ['--bg', '内容区'], ['--bg-raised', '卡片 / 弹层'], ['--bg-sunken', '代码 / 凹槽']])}
      ${swg('文字', [['--tx-1', '正文'], ['--tx-2', '次要'], ['--tx-3', '说明'], ['--tx-4', '占位 / 时间']])}
      ${swg('描边与填充', [['--bd-1', '分隔'], ['--bd-2', '面板边'], ['--bd-3', '强调边'], ['--fill-1', '悬停'], ['--fill-2', '选中'], ['--fill-3', '按下']])}
      ${swg('强调', [['--ac', '强调'], ['--ac-soft', '柔底'], ['--ac-line', '焦点线'], ['--ac-tx', '强调文字']])}
      ${swg('状态', [['--ok', '成功'], ['--err', '失败'], ['--warn', '警告'], ['--info', '信息'], ['--ok-soft', '成功底'], ['--err-soft', '失败底'], ['--warn-soft', '警告底'], ['--info-soft', '信息底']])}
      ${swg('工具族', [['--tool-explore', '探索'], ['--tool-edit', '修改'], ['--tool-exec', '执行'], ['--tool-web', '联网'], ['--tool-mcp', '扩展'], ['--tool-agent', '子 Agent'], ['--tool-plan', '计划'], ['--tool-think', '思考']])}</div>`)}
    ${sec('字体 · 间距 · 圆角 · 尺寸', '字号 7 档、间距 8 档、圆角 5 档(乘方案倍率)。', `<div class="grid2">
      ${spec('字号', ['--fs-*'], [['--fs-display', 26, '今天想做点什么?'], ['--fs-head', 18, '页面标题'], ['--fs-title', 14, '卡片标题 / 正文小标题'], ['--fs-read', 13, '会话正文与输入框'], ['--fs-body', 12, '界面默认字号'], ['--fs-cap', 11, '说明与状态栏'], ['--fs-micro', 10, '时间与计数']].map(([v, px, t]) => `<div class="f ai g4" style="height:${Math.max(22, px + 10)}px"><span style="font-size:var(${v});font-weight:${px > 13 ? 600 : 400}" class="grow ell">${t}</span>${tok(v)}<span class="t4 num" style="width:22px;text-align:right">${px}</span></div>`).join(''))}
      ${spec('间距', ['--sp-1…8'], [2, 4, 6, 8, 12, 16, 24, 32].map((px, k) => `<div class="f ai g4" style="height:20px"><span style="width:${px}px;height:10px;border-radius:2px;background:var(--ac)"></span><span class="grow"></span>${tok('--sp-' + (k + 1))}<span class="t4 num" style="width:22px;text-align:right">${px}</span></div>`).join(''))}
      ${spec('圆角(当前方案倍率)', ['--r-k'], `<div class="f g4 ai">${['xs', 'sm', 'md', 'lg', 'xl'].map(r => `<div class="fc ai g2"><div style="width:52px;height:40px;border-radius:var(--r-${r});background:var(--ac-soft);box-shadow:inset 0 0 0 1px var(--ac-line)"></div>${tok('--r-' + r)}</div>`).join('')}</div>`)}
      ${spec('尺寸', ['--h-*', '--w-*'], [['--h-row', '列表行 22 / 24 / 26(随方案)'], ['--h-ctl', '控件 24 · 小号 20'], ['--h-tab', '标签条 30'], ['--h-bar', '标题栏 36'], ['--h-status', '状态栏 22'], ['--w-side', '导航栏 228(轨道 212)'], ['--w-insp', '检查器 304'], ['--w-dock', 'IDE 对话栏 352'], ['--w-col', '会话内容上限 712']].map(([v, t]) => `<div class="f ai g4" style="height:20px"><span class="grow">${t}</span>${tok(v)}</div>`).join(''))}</div>`)}</div>`;

  /* ───────── 动效 ───────── */
  const track = (name, val, d, e) => `<div class="mrow"><span class="m">${name}</span><span class="t3 mono">${val}</span><div class="track" style="--d:${d};--e:${e}"><i></i></div></div>`;
  P.demos = [
    ['行进入 rise', '--dur-base · --ease-out · 逐行延迟 30ms', () => `<div class="stagger" style="width:100%">${F.step('read', 'router.rs', 'src/', '86 行')}${F.step('grep', '"match_path"', '', '5 处')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span>')}</div>`],
    ['弹层 pop', '--dur-base · --ease-out · 自锚点缩放 96%', () => `<div class="pop a-pop" style="width:170px"><div class="mi on">Sonnet 5.5</div><div class="mi">Opus 5.5</div><div class="mi">Haiku 5.5</div></div>`],
    ['就地变形 pop-up + glow', '输入框 → 权限卡,呼吸描边提示待处理', () => `<div style="width:100%">${F.perm()}</div>`],
    ['流光 shimmer', '--dur-loop 线性循环,只用于「正在进行」', () => `<div style="width:100%">${F.step('bash', 'cargo build --release', '', '', { state: 'run' })}<div class="ticker" data-ticks="read|router.rs;grep|match_path;edit|server.rs"></div></div>`],
    ['折叠 fold', '--dur-slow · --ease-io · 点行展开 / 收起', () => `<div style="width:100%">${F.step('todo', '2 / 3 完成', '', '', { body: F.todo(2) })}</div>`, 1],
    ['勾选 check', '--dur-base · --ease-spring · 对勾描边', () => `<div class="f g5 ai">${chk('done')}${chk('done')}${chk('done')}<span class="t3">点击重放</span></div>`],
    ['滑块 thumb', '--dur-base · --ease-spring · 轻微回弹', () => `<div class="f g5 ai">${seg(['Agent', 'IDE'], 0, 'lg')}${tog(1)}</div>`, 1],
    ['通知 toast', '--dur-slow · --ease-out · 自右侧滑入', () => `<div class="pop a-toast f g4 ai" style="padding:8px 11px">${ic('okc', 'c-ok')}<b>已保存</b><span class="t3">router.rs</span></div>`],
    ['布局变形 morph', '--dur-morph · --ease-morph · 点击切换', () => `<div class="mini" data-act="mini" data-mode="agent"><i class="s"></i><i class="e"></i><i class="c"></i><i class="r"></i></div>`, 1],
  ];
  P.demo = (k) => `<div class="demo ${P.demos[k][3] ? '' : 'replay'}" ${P.demos[k][3] ? '' : `data-act="demo" data-i="${k}"`}><div class="lb"><span>${P.demos[k][0]}</span><span class="grow"></span>${P.demos[k][3] ? '' : ic('refresh', 's12')}</div><div class="demo-st">${P.demos[k][2]()}</div><div class="t3 fs-cap">${P.demos[k][1]}</div></div>`;
  P.motion = () => `<div class="ds">${head('动效', '动效只做三件事:说明「它从哪来」、提示「正在进行」、确认「已经完成」。时长 6 档、曲线 5 条,组件里不出现裸数字。')}
    ${sec('时长与曲线', `所有过渡都是「一个时长令牌 + 一条曲线令牌」。${btn('重放', 'sm soft', 'play', 'data-act="tracks" style="margin-left:6px"')}`,
      `<div class="grid2"><div class="spec tracks">${[['--dur-instant', '80ms', '按压 / 勾选'], ['--dur-fast', '120ms', '悬停 / 收起'], ['--dur-base', '180ms', '弹层 / 行进入 / 滑块'], ['--dur-slow', '260ms', '折叠 / 抽屉'], ['--dur-morph', '440ms', 'Agent ⇄ IDE']].map(([n, v, u]) => track(`${n}<span class="t3 fs-cap" style="font-weight:400;margin-left:6px">${u}</span>`, v, `var(${n})`, 'var(--ease-out)')).join('')}</div>
      <div class="spec tracks">${[['--ease-out', '进入'], ['--ease-in', '离开'], ['--ease-io', '位移'], ['--ease-spring', '滑块 / 勾选'], ['--ease-morph', '布局变形']].map(([n, u]) => track(`${n}<span class="t3 fs-cap" style="font-weight:400;margin-left:6px">${u}</span>`, '900ms', '900ms', `var(${n})`)).join('')}</div></div>`)}
    ${sec('动效模式', '组件只引用模式名(a-rise / a-pop / shimmer…),不直接写关键帧。点卡片重放。', `<div class="grid3">${P.demos.map((_, k) => P.demo(k)).join('')}</div>`)}</div>`;

  window.P = P;
})();
