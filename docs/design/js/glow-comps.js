/* 设计稿 v4.3 · 流光主题的玻璃组件库:每个组件在「流光」与「流光·夜」两块样本台上各画一遍,含状态矩阵
   组件清单对齐「组件」页(P.comps)与会话流 / 输入区 / 浮层:按钮、图标按钮、分段、开关、复选、输入、胶囊与徽标、标签条、列表行、卡片、
   会话流卡、输入区(任务胶囊 / 待办条 / 排队卡 / 输入框)、菜单与右键菜单、浮层(信息卡 / 提示 / 通知 / 对话框)、进度与加载、横幅。
   玻璃的规则见「流光主题」页;这里只做组件的样子,可以直接悬停 / 点击。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, chk, inp } = U;
  const { head, sec } = P.h;
  const THEMES2 = [['glow', '流光 · 浅'], ['glow-night', '流光 · 夜']];

  /** 两块样本台:同一段组件 html 在浅 / 夜壁纸上各画一次 */
  const pair = (fn, pad = true) => `<div class="gl-pair">${THEMES2.map(([t, n]) => `<div class="gl-tile" data-theme="${t}" data-scheme="b"><div class="gl-lb">${n}</div><div class="${pad ? 'gc-pad' : ''}">${fn(t)}</div></div>`).join('')}</div>`;
  /** 状态矩阵:cols 是列名,rows 是 [行名, 各列 html] */
  const mx = (cols, rows) => `<div class="gc-mx" style="--n:${cols.length}"><span></span>${cols.map(c => `<span class="h">${c}</span>`).join('')}${rows.map(([n, cs]) => `<span class="r">${n}</span>${cs.map(h => `<span class="c">${h}</span>`).join('')}`).join('')}</div>`;
  const dash = '<span class="t4">—</span>';
  const S6 = ['默认', '悬停', '按下', '聚焦', '禁用', '加载'];

  /* ───────── 控件 ───────── */
  const buttons = () => mx(S6, [
    ['默认', [btn('保存'), btn('保存', 'is-hover'), btn('保存', 'is-active'), btn('保存', 'is-focus'), btn('保存', 'is-disabled'), btn('保存', 'loading')]],
    ['主要', [btn('应用', 'pri'), btn('应用', 'pri is-hover'), btn('应用', 'pri is-active'), btn('应用', 'pri is-focus'), btn('应用', 'pri is-disabled'), btn('应用', 'pri loading')]],
    ['柔和', [btn('审查', 'soft'), btn('审查', 'soft is-hover'), btn('审查', 'soft is-active'), btn('审查', 'soft is-focus'), btn('审查', 'soft is-disabled'), dash]],
    ['幽灵', [btn('重试', 'ghost'), btn('重试', 'ghost is-hover'), btn('重试', 'ghost is-active'), btn('重试', 'ghost is-focus'), btn('重试', 'ghost is-disabled'), dash]],
    ['危险', [btn('删除', 'danger'), btn('删除', 'danger is-hover'), dash, btn('删除', 'danger is-focus'), btn('删除', 'danger is-disabled'), dash]],
    ['图标', [ib('copy'), ib('copy', 'is-hover'), ib('copy', 'is-active'), ib('copy', 'is-focus'), ib('copy', 'is-disabled'), ib('refresh')]],
    ['发送', [ib('send', 'send', '发送'), ib('send', 'send is-hover'), dash, dash, dash, ib('stop', 'stop', '停止')]]]);
  const toggles = () => `<div class="gc-stack">${seg(['Agent', 'IDE'], 0, 'lg')}${seg(['关', '低', '中', '高'], 2)}<div class="f g5 ai">${tog(1)}${tog(0)}${tog(1, 'style="opacity:.42"')}<span class="t3 fs-cap">开 / 关 / 禁用</span></div><div class="f g5 ai">${chk()}${chk('now')}${chk('done')}<span class="t3 fs-cap">待办 / 进行 / 完成</span></div></div>`;
  const inputs = () => `<div class="gc-stack" style="width:100%"><label class="inp" style="width:100%">${ic('search', 's12')}<span class="grow t3">搜索会话、文件与命令</span>${kbd('⌘', 'K')}</label><label class="inp focus" style="width:100%">${ic('search', 's12')}<span class="v">match_path</span></label><label class="inp fill" style="width:100%">${ic('filter', 's12')}<span class="t3">筛选文件</span></label><label class="inp" style="width:100%;box-shadow:inset 0 0 0 1px var(--err),0 0 0 3px color-mix(in srgb,var(--err) 16%,transparent)">${ic('alert', 's12')}<span class="v">无效的路径</span></label><div class="fs-cap c-err">路径不能包含空字符</div></div>`;
  const badges = () => `<div class="gc-stack"><div class="f g3 wrap ai">${chip('main', 'branch')}${chip('MCP 3', 'plug', 'on')}${chip('子代理 2', 'bot', 'line')}${chip('后台命令 1', 'terminal', 'line')}</div><div class="f g3 wrap ai">${tag('默认')}${tag('强调', 'ac')}${tag('通过', 'ok')}${tag('失败', 'err')}${tag('待批准', 'warn')}${tag('信息', 'info')}${cnt(3)}${cnt(12, 'ac')}</div><div class="f g4 ai">${kbd('⌘', 'K')}${kbd('⇧', '⌘', 'F')}${kbd('esc')}<span class="avatar">L</span><span class="dot ok"></span><span class="dot warn"></span><span class="dot err"></span></div></div>`;
  const tabsRow = () => `<div class="gc-stack" style="width:100%"><div class="tabs" style="padding:0;width:100%">${['改动', '文件', '终端', '计划'].map((t, k) => `<span class="tab ${k ? '' : 'on'}">${t}${k ? '' : cnt(3)}</span>`).join('')}</div><div class="f g3 ai t3 fs-cap">${ic('folder', 's12')}src${ic('chevR', 's12')}${ic('code', 's12')}router.rs${ic('chevR', 's12')}<b class="t1">match_path</b></div></div>`;
  const rows = () => `<div class="fc" style="width:100%;gap:2px"><div class="grp-h">今天</div><div class="row on">${ic('chat', 's12')}<span class="grow ell">完整修复一个 Bug:路由大小写匹配</span><span class="aux">刚刚</span></div><div class="row is-hover">${ic('chat', 's12')}<span class="grow ell">权限请求:修改配置与执行命令</span><span class="aux">5 分钟</span></div><div class="row">${ic('chat', 's12')}<span class="grow ell">文件读写:梳理并整理路由层</span><span class="aux swap">1 小时</span><span class="acts">${ib('more', 'sm')}</span></div><div class="row is-focus">${ic('code', 's12')}<span class="ell">router.rs</span><span class="grow"></span><span class="aux">M</span></div></div>`;

  /* ───────── 卡片与会话流 ───────── */
  const cards = () => `<div class="gc-stack" style="width:100%"><div class="f g4" style="width:100%"><div class="card grow" style="padding:10px 12px"><b>卡片</b><div class="t3 fs-cap">静止:软阴影</div></div><div class="card is-hover grow" style="padding:10px 12px"><b>悬停</b><div class="t3 fs-cap">上浮 1px</div></div><div class="card grow" style="padding:10px 12px;box-shadow:var(--ring),inset 0 0 0 1px var(--ac-line)"><b>选中</b><div class="t3 fs-cap">强调色描边</div></div></div>
    <div class="card" style="width:100%;padding:10px 12px"><div class="f ai g4">${ic('list', 's12')}<b class="grow">路由大小写修复计划</b>${tag('已批准', 'ok')}</div><div class="t2" style="margin-top:6px;line-height:1.6">1. 抽出 <code>normalize</code>,注册与匹配共用<br>2. 匹配失败时回退到 fallback</div></div>
    <div class="card" style="width:100%;padding:8px 12px"><div class="f ai g4">${ic('file', 's12')}<span class="grow">读取 3 个文件</span><span class="t3 fs-cap">src/router.rs</span></div></div></div>`;
  const dock = () => `<div class="gc-stack" style="width:100%"><div class="f g3">${chip('子代理 2', 'bot', 'line')}${chip('后台命令 1', 'terminal', 'line')}</div><div class="pop" style="width:100%;padding:0"><div class="f ai g4" style="padding:7px 11px"><span class="ring" style="--p:33"></span><b>待办 1 / 3</b><span class="t3 grow">编辑</span>${ic('chevR', 's12')}</div></div><div class="cmp" style="width:100%"><div class="cmp-in">描述任务,@ 引用文件,/ 调用命令</div><div class="cmp-bar"><span class="cfg">${ic('plus', 's12')}</span><span class="cfg">${ic('shield', 's12')}逐项确认<i></i>Sonnet 5.5</span><span class="grow"></span><span class="t3 fs-cap">38%</span>${ib('send', 'send', '发送')}</div></div></div>`;

  /* ───────── 菜单与浮层 ───────── */
  const menu = () => `<div class="f g5 wrap" style="align-items:flex-start"><div class="pop" style="width:206px"><div class="cap">权限模式</div><div class="mi on">${ic('shield', 's12')}<span class="grow">逐项确认</span>${ic('check', 's12')}</div><div class="mi">${ic('edit', 's12')}<span class="grow">自动接受编辑</span>${kbd('⇧', '⇥')}</div><div class="mi is-disabled" style="opacity:.42">${ic('bot', 's12')}<span class="grow">计划模式</span></div><div class="sep"></div><div class="mi">${ic('sliders', 's12')}设置…</div><div class="mi c-err">${ic('trash', 's12')}清除会话</div></div>
    <div class="pop" style="width:196px"><div class="mi">${ic('code', 's12')}<span class="grow">打开</span>${kbd('↵')}</div><div class="mi is-hover">${ic('edit', 's12')}<span class="grow">重命名…</span>${kbd('F2')}</div><div class="mi">${ic('pin', 's12')}置顶</div><div class="sep"></div><div class="mi">${ic('copy', 's12')}复制路径</div><div class="mi">${ic('folder', 's12')}<span class="grow">在访达中显示</span>${ic('chevR', 's12')}</div></div></div>`;
  const overlays = () => `<div class="gc-stack" style="width:100%"><div class="f g5 wrap ai">${btn('悬停看提示', 'sm', '', 'data-tip="延迟 350ms 出现"')}<div class="pop f g4" style="padding:9px 11px;align-items:flex-start;width:268px">${ic('okc', 'c-ok')}<div class="grow"><b>已下载 0.2.0</b><div class="t3 fs-cap">重启后生效</div></div>${btn('重启更新', 'sm soft')}</div></div>
    <div class="pop" style="width:100%;padding:12px 14px"><b class="fs-title">删除这个会话?</b><div class="t2" style="margin:4px 0 12px">删除后无法恢复,它的后台任务也会一并停止。</div><div class="f g3" style="justify-content:flex-end">${btn('取消')}${btn('删除', 'danger')}</div></div>
    <div class="pop hover-card" style="width:100%"><div class="f ai g3"><b>match_path</b><span class="t3 mono">fn</span></div><div class="doc">按路径匹配处理器;忽略大小写与末尾斜杠。</div></div></div>`;
  const progress = () => `<div class="gc-stack" style="width:100%"><div class="f g5 ai"><span class="spin"></span><span class="ring" style="--p:38"></span><span class="ring" style="--p:86;--ac:var(--warn)"></span><div class="grow"><div class="meter"><i style="width:62%"></i></div></div></div><div class="shimmer m">正在生成提交说明…</div><div class="pop f g4" style="width:100%;padding:8px 11px;align-items:center">${ic('alert', 'c-warn')}<span class="grow">有 3 项更改需要重启后生效</span>${btn('稍后', 'sm ghost')}${btn('立即重启', 'sm pri')}</div></div>`;

  P['glow-comps'] = () => `<div class="ds" style="max-width:1240px">${head('流光 · 玻璃组件', '「流光」与「流光·夜」下每个组件的样子,含状态矩阵;全部可以直接悬停、点击。玻璃只用在控件层与浮层,内容(文字、代码)始终在不透明或高对比的面上;所有描边是浅色高光边而不是灰线,所有抬起感来自软阴影。')}
    ${sec('按钮 · 状态矩阵', '默认按钮是薄玻璃(悬停升为「选中」玻璃);主要按钮是着色玻璃 + 强调色光晕;柔和按钮是强调色 16% 的玻璃;幽灵按钮没有底,悬停才出现。按下一律缩到 96%。', pair(buttons))}
    ${sec('分段 · 开关 · 复选', '分段的槽是略深的凹槽,滑块是实色抬起的白块;开关打开时是着色玻璃。', pair(toggles))}
    ${sec('输入', '输入框是薄玻璃 + 内阴影;聚焦是强调色外环;错误是红色外环加一行说明。', pair(inputs))}
    ${sec('胶囊 · 标签 · 徽标 · 键帽', '带描边的胶囊是一小块可交互玻璃;标签与计数是实色小块(信息密度高,不做玻璃)。', pair(badges))}
    ${sec('标签条 · 面包屑', '标签的下划线从中间向两侧展开。', pair(tabsRow))}
    ${sec('列表行', '选中是更亮的玻璃 + 高光边;悬停是 8.5% 墨色;键盘聚焦是强调色外环。', pair(rows))}
    ${sec('卡片 · 会话流卡', '卡片 = 玻璃卡片色 + 软阴影,悬停上浮 1px;选中是强调色描边。会话流里的计划卡、工具卡是同一种。', pair(cards))}
    ${sec('输入区', '任务胶囊、待办条、输入框浮在会话流之上(会话流延伸到它们下面,滚动内容从玻璃后面穿过);背后有一层向上渐隐,避免正文直接透到小字下面。', pair(dock))}
    ${sec('菜单 · 右键菜单', '菜单面板是厚玻璃 + 高光边 + 顶部光泽;选中项是更亮的玻璃;危险项用错误色;禁用项 42% 透明;子菜单箭头在右侧。右键菜单与按钮弹出的「···」菜单是同一个组件。', pair(menu))}
    ${sec('浮层 · 提示 · 通知 · 对话框', '提示是深色半透明玻璃;通知、对话框、信息卡都是厚玻璃。', pair(overlays))}
    ${sec('进度 · 加载 · 横幅', '转圈、环、进度条都用强调色;横幅是玻璃条,带动作按钮。', pair(progress))}</div>`;
})();
