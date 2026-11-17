/* 设计稿 v3 · 设置页(重新规划后的信息架构:5 组 19 个分类)
   原则:设置只放「偏好」;功能内容归功能页(自定义 / 用量);一项配置只有一个家,其它入口一律是跳转。
   所有控件都接了真实交互(见 behave.js):搜索、已修改与还原、需重启条、快捷键录制与冲突检测、连通性检测、检查更新……
   旧分类 → 新分类的对照见规范页「信息架构」。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, inp, field, area, esc, THEMES, SCHEMES } = U;

  /** 导航分组:[组名, [分类 id…]] */
  const GROUPS = [
    ['客户端', ['general', 'notify', 'shortcuts']],
    ['外观', ['appearance', 'typography', 'motion', 'brand']],
    ['Claude Code', ['models', 'chat', 'permissions', 'runtime']],
    ['工作台', ['editor', 'lsp', 'terminal', 'scm', 'browser']],
    ['其他', ['privacy', 'advanced', 'about']],
  ];
  const ICON = {
    general: 'sliders', notify: 'bell', shortcuts: 'kbd',
    appearance: 'palette', typography: 'align', motion: 'spark', brand: 'grid',
    models: 'cpu', chat: 'chat', permissions: 'shield', runtime: 'cube',
    editor: 'code', lsp: 'brain', terminal: 'terminal', scm: 'branch', browser: 'globe',
    privacy: 'lock', advanced: 'wrench', about: 'info',
  };
  /** 旧分类 id → 新分类 id(地址栏 / 旧记忆里的 setTab 仍能落到正确位置) */
  const ALIAS = {
    customize: 'general', model: 'models', providers: 'models', prompts: 'chat', trust: 'permissions', perm: 'permissions',
    env: 'runtime', network: 'runtime', intel: 'lsp', usage: 'general', diagnostics: 'privacy',
  };

  /* ───────── 行索引:渲染时顺带记录,搜索用 ───────── */
  let REC = null, PEND = [];
  /** 设置行。t:标题;d:说明;ctl:右侧控件;o.restart:改动后需重启 */
  const R = (t, d, ctl = '', o = {}) => {
    if (REC && t) PEND.push({ t, d: d || '' });
    return `<div class="set-row" data-k="${esc(t)}" ${o.restart ? 'data-restart="1"' : ''}><div class="grow"><div class="m">${t}${o.restart ? ' ' + tag('需重启', 'warn') : ''}</div>${d ? `<div class="d">${d}</div>` : ''}</div>${ctl}${t ? `<button class="ib sm rst" data-act="set-reset" data-tip="还原">${ic('undo')}</button>` : ''}</div>`;
  };
  /** 分组卡。title:小标题;rows:若干行 */
  const G = (title, ...rows) => {
    if (REC) { PEND.forEach(p => REC.push({ ...p, g: title })); PEND = []; }
    return `<div class="set-grp">${title ? `<div class="gt">${title}</div>` : ''}<div class="card">${rows.join('')}</div></div>`;
  };
  const stp = (v, u = '', o = {}) => `<span class="stp" data-min="${o.min ?? 0}" data-max="${o.max ?? 999}" data-step="${o.step || 1}"><button data-act="step" data-d="-1" aria-label="减少">−</button><b class="num">${v}</b>${u ? `<span class="t3">${u}</span>` : ''}<button data-act="step" data-d="1" aria-label="增加">+</button></span>`;
  /** 设置行下面的预览块(见 settings-pv.js);脚本未加载时为空 */
  const PVX = (k) => (typeof W !== 'undefined' && W.pv && W.pv[k] ? W.pv[k]() : '');
  const note = (t) => `<div class="t3 fs-cap" style="padding:0 4px;line-height:1.5">${t}</div>`;
  /** 会实时改变界面的分段(密度、圆角……):data-key 交给 behave.js 的 live 处理 */
  const segL = (items, on, key) => seg(items, on, '', 'live').replace('<div class="seg ', `<div data-key="${key}" class="seg `);
  const accents = ['--ac-theme', '--c-info', '--c-ok', '--c-cyan', '--c-vio', '--c-pink', '--c-err', '--c-warn'];
  const goto = (label, screen, cust = '') => btn(label, 'sm', 'arrow', `data-act="screen" data-screen="${screen}" ${cust ? `data-cust="${cust}"` : ''}`);

  /** 外观 → 应用图标 / 菜单栏图标:样式卡、外观、配色、Dock 与菜单栏行为(控件与预览在 brand.js,改动即时生效) */
  const brandGroups = () => { const B = U.brand;
    return `${G('应用图标',
      `<div style="padding:10px 12px 2px"><div class="usage-card" style="min-height:0;padding:12px"><div class="f ai g2"><b>动态预览</b><span class="t3 fs-cap">Dock 与菜单栏用的是同一个样式:选一次,两处同时换</span></div>${B.live()}</div></div>`,
      `<div style="padding:8px 12px 2px">${B.previewDocks()}</div>`,
      R('图标样式', '五选一;应用图标与菜单栏图标同时换成同一种,立即生效。鼠标放在卡片上能看到菜单栏图标动起来的样子;默认「提示符」', `<b class="t2" data-bi-name="app">${B.styleName()}</b>`),
      `<div style="padding:0 12px 12px">${B.pickApp()}</div>`,
      R('图标外观', '对应 macOS 26 的默认 / 深色 / 透明 / 着色;「自动」随系统深浅在浅色与深色之间选', B.segOpt('icoMode', ['自动', '浅色', '深色', '透明', '着色'])),
      R('配色', '跟随当前主题,或固定为某一套主题的色板(图标不再随主题变)', B.tintPick()),
      R('Dock 角标', '图标右上角的红色数字', B.segOpt('dockBadge', ['关', '待处理数', '运行中数'])),
      R('需要确认时 Dock 弹跳', '权限请求或提问到达、而 MiniCode 不在前台时', B.segOpt('dockBounce', ['不弹', '一次', '直到处理'])),
      R('Dock 图标显示进度', '下载更新、导出诊断包等超过 2 秒的操作', B.togOpt('dockProgress')),
      R('同时更换访达里的图标', '默认只改 Dock、⌘Tab 与应用内;打开后把所选图标写进 MiniCode.app 的自定义图标(访达、启动台、关于本机);关闭即还原。应用更新后第一次启动会自动重新套用', B.togOpt('fileIcon')),
      R('当前生效', '改完设置会立刻套用,不需要重启;哪一处没变,点右边「重新应用」', `<div class="fc g1" style="align-items:flex-end"><span class="fs-cap"><span class="t4">Dock</span> <span class="t3">已套用</span></span><span class="fs-cap"><span class="t4">访达</span> <span class="t3">已更换 / 未开启 / 失败:原因</span></span><span class="fs-cap"><span class="t4">菜单栏</span> <span class="t3">已显示 · 提示符</span></span>${btn('重新应用', 'sm', 'refresh', 'data-act="brand-reapply"')}</div>`))}
      ${note('运行时换的图标只在 MiniCode 运行期间有效;通知中心与「保留在 Dock」的那一格仍是包内默认图标。打开「同时更换访达里的图标」后,访达与启动台也会换;该开关会让 .app 在 codesign --verify --strict 下报「resource fork」,但不影响启动与更新。')}
      ${G('菜单栏图标',
      R('在菜单栏显示图标', '关闭后可在本类重新打开;⌘ 拖出菜单栏也等于关闭', B.togOpt('trayOn')),
      R('图标样式', '与应用图标是同一个样式(单色模板图,系统按菜单栏明暗自动着色,点开时自动反白);要换样式请在上方「应用图标」里选,两处同时变', `<b class="t2" data-bi-name="tray">${B.styleName()}</b>`),
      R('随会话状态变化', '运行中动画,以及待确认(!)/ 已完成 / 出错(×)/ 有更新的徽标;关闭后始终是静态图标', B.togOpt('trayDyn')),
      R('运行中动画', '标记的片段依次明暗(提示符是光标闪烁),或整体呼吸;开启「减弱动态效果」时静止', B.segOpt('trayAnim', ['逐段点亮', '呼吸', '无'])),
      R('彩色徽标', '徽标改用系统语义色(橙 / 绿 / 红 / 蓝),形状不变;关闭则保持纯单色模板', B.togOpt('trayColor')),
      R('在图标旁显示数字', '待处理数(权限 + 提问),为 0 时隐藏', B.togOpt('trayCount')),
      R('单击图标', '「弹出菜单」:单击弹出菜单;「打开窗口」:单击打开主窗口,右键弹出菜单', B.segOpt('trayClick', ['弹出菜单', '打开窗口'])))}
      ${note('完整的状态、动画与各场景(Dock、⌘Tab、通知、安装盘……)见规范页「图标与品牌」。')}`; };

  /** 界面布局三选一:点击立即切换整个查看器的方案(data-act="layout") */
  const layCards = (cur) => `<div class="f g3">${SCHEMES.map(s => `<div class="lay-card ${s.id === cur ? 'on' : ''}" data-act="layout" data-id="${s.id}" tabindex="0">
    <div class="lay-prev" data-lay="${s.id}"><i class="rl"></i><i class="sd"></i><i class="mn"><b></b><b></b><b></b></i><i class="rt"></i></div>
    <div class="f ai jb"><b>${s.name}</b><span class="t4 fs-micro">${s.en}</span></div><div class="t3 fs-cap" style="line-height:1.4">${s.tagline}</div></div>`).join('')}</div>`;

  /* ───────── 供交互层增删行用的模板 ───────── */
  const ruleRow = (k, t) => `<div class="set-row a-rise" data-rm style="min-height:30px">${tag({ allow: '允许', ask: '询问', deny: '拒绝' }[k], { allow: 'ok', ask: 'warn', deny: 'err' }[k])}<span class="mono grow">${esc(t)}</span>${ib('trash', 'sm', '删除', 'data-act="del-row"')}</div>`;
  const envRow = (n, l) => `<div class="set-row a-rise" data-rm style="min-height:32px"><span class="mono m" style="width:230px">${esc(n)}</span><span class="mono t3 grow env-v" data-mask="1">••••••••</span>${tag(l)}${ib('eye', 'sm', '显示 / 遮盖', 'data-act="env-eye"')}${ib('trash', 'sm', '删除', 'data-act="del-row"')}</div>`;
  const provRow = (n, u, on, s) => (typeof PV !== 'undefined' ? PV.provRow({ id: 'tmp', name: n, url: u, status: s === '已连通' ? 'ok' : s ? 'warn' : '', hasKey: 1, selected: ['m'], fixed: 0 })
    : `<div class="set-row" data-prov data-rm><span class="si-box">${ic('server')}</span><div class="grow"><div class="m f ai g3">${n}<span class="prov-res">${s ? tag(s, s === '已连通' ? 'ok' : 'warn') : ''}</span></div><div class="d mono">${u}</div></div><span class="prov-st">${on ? tag('当前', 'ac') : btn('切换', 'sm', '', 'data-act="prov-use"')}</span>${btn('测试', 'sm ghost', 'activity', 'data-act="prov-test"')}${ib('trash', 'sm', '删除', 'data-act="del-row"')}</div>`);

  /* ───────── 各分类 ───────── */
  const TABS = {
    general: { t: '通用', d: '启动、窗口、输入与界面语言。', body: () => `
      ${G('启动与窗口',
        R('启动时的形态', 'Agent 与 IDE 是同一个窗口的两种形态;「上次」沿用关闭时的形态', seg(['Agent', 'IDE', '上次'], 2)),
        R('恢复上次的会话与布局', '项目、会话、右栏标签、面板开合、编辑器标签;关闭时不恢复(走欢迎页)。Go: general.restore + last-session.json(未测试)', tog(1)),
        R('关闭窗口时', '红色按钮:隐藏=OnClose PreventDefault+Hide,退出=关窗;⌘W 先关所在的标签,不在多标签区域时只隐藏窗口、从不退出(见「窗口形态 · 关闭 ⌘W」);⌘Q 仍走 OnBeforeQuit(未测试)', seg(['隐藏', '退出'], 0)),
        R('开标签时自动展开右侧工作区', 'Agent 形态下在会话里点计划 / 文件 / 审查时展开工作区;各形态的面板开合各自记忆', tog(1)))}
      ${note('想同时看另一个项目?⇧⌘N 新开一个完整窗口,它同样可以在 Agent / IDE 之间切换。')}
      ${G('输入', R('发送时附带 IDE 上下文', '把当前文件、光标行、选中的代码和打开的文件名告诉 Claude(随消息发出,不显示在会话流里;相同内容不重复发;选区限 4000 字)。关闭后只发你输入的文字', tog(1)), R('发送键', '另一个键用于换行(Shift+Enter 始终换行)', seg(['Enter', '⌘ Enter'], 0)), R('回合进行中再发消息', 'Enter = 排队;⌘Enter = 立即引导', seg(['排队', '立即引导'], 0)), R('粘贴长文本时折叠为附件', 'general.paste:>2000 字写入 ~/.minicode/paste 并插 @ 引用,发送时展开(无 chip 展开按钮,未测)', tog(1)))}
      ${G('语言', R('界面语言', '目前仅提供简体中文', chip('简体中文', '', 'line')))}` },

    notify: { t: '通知', d: '系统通知、提示音与菜单栏入口。', body: () => `
      ${G('系统通知',
        R('任务完成时通知', '回合结束且 MiniCode 不在前台时发送系统通知', tog(1)),
        R('前台也通知', 'MiniCode 在前台时回合结束 / 出错也弹系统通知并响提示音(general.notifyFront,默认开;未测试)', tog(1)),
        R('需要确认时提醒', '权限请求、提问在后台出现时弹通知', tog(1)),
        R('暂停通知', 'general.pauseNotify:暂停后不发通知、不播提示音;限时暂停在菜单栏图标菜单里', tog(0)),
        R('通知声音', 'general.sound 控制 UN 通知 / osascript 是否带声(未测试)', seg(['静音', '轻提示'], 1)))}
      ${G('菜单栏', R('菜单栏图标与托盘', '是否显示、样式、状态徽标与动画在「应用与菜单栏图标」', btn('前往', 'sm', 'arrow', 'data-act="settab" data-tab="brand"')))}` },

    appearance: { t: '主题与布局', d: '界面布局、主题、强调色与密度。所有选项都落在设计令牌上,即改即生效。', body: (st) => `
      ${G('界面布局', `<div style="padding:10px 12px 12px"><div class="d t3 fs-cap" style="margin-bottom:8px">三种布局共用同一份令牌与组件,只在「面板怎么分隔、导航放在哪、步骤怎么串」上不同。点一下立即切换,Agent 与 IDE 形态同步变化。</div>${layCards(st.scheme)}</div>`)}
      ${G('主题', `<div style="padding:10px 12px">${W.themeCards(st.theme)}</div>`, R('跟随系统深浅色', '浅色用「宣纸」,深色用「墨夜」', tog(0)))}
      ${G('强调色', R('强调色', '按钮、选中态、焦点环、选区、链接等使用的主题色;每个主题单独记忆,并按明暗自动调整明度', `<span class="f g2">${accents.map((a, k) => `<span class="swatch ${k ? '' : 'on'}" data-act="accent" data-v="${k ? a : ''}" tabindex="0" style="background:var(${a})"></span>`).join('')}</span>`))}
      ${G('密度与形状', R('界面密度', '行高令牌 --h-row:紧凑 −2 / 标准 / 宽松 +2(在当前布局的基础上)', segL(['紧凑', '标准', '宽松'], 1, 'density')), R('圆角', '圆角倍率 --r-k:利落 ×0.55 / 适中 / 圆润 ×1.35', segL(['利落', '适中', '圆润'], 1, 'radius')))}` },

    typography: { t: '字体与尺寸', d: '界面缩放、会话正文字号与内容宽度。', body: (st) => `
      ${G('界面与正文',
        PVX('typo'),
        R('字号预设', '同时调整界面大小与会话正文字号;滑块偏离后自动显示「自定义」(appearance.fontPreset)', seg(['紧凑', '标准', '舒适', '大字', '自定义'], 1)),
        R('界面大小', '侧栏、列表、菜单、按钮、文件树等整体缩放(appearance.uiScale)', stp(100, '%', { min: 90, max: 150, step: 5 })),
        R('会话正文字号', '会话流与 Markdown 预览正文(appearance.fontScale)', stp(100, '%', { min: 85, max: 150, step: 5 })),
        R('正文行距', '会话流、用户消息与输入框的行距(1.45 / 1.6 / 1.75);不影响编辑器与终端(appearance.lineHeight)', seg(['紧凑', '标准', '宽松'], 1)))}
      ${G('内容宽度',
        R('会话内容宽度', '时间线、排队卡、任务坞与输入框同宽;窄屏自动收窄,占满上限 1400(appearance.flowWidth)', seg(['窄', '紧凑', '标准', '宽', '特宽', '占满'], 2)),
        R('设置页内容宽度', '设置主栏内容列;窗口较窄时自动铺满(appearance.setWidth)', seg(['窄', '标准', '宽', '特宽', '占满'], 1)),
        PVX('width'))}
      ${G('文件树', R('彩色文件图标', '按文件类型上色;关闭后全部用灰色线条图标', tog(1, 'data-live="fileIcons"')))}
      ${note('排版规范(字号阶梯、预设、档位)见「规范 → 排版与宽度」页。')}` },

    motion: { t: '动效与反馈', d: '动画总开关、形态切换、等待指示与滚动条。', body: () => `
      ${G('动效',
        R('动效', '流光、滑入与布局变形;关闭后所有时长令牌归零(也跟随系统「减弱动态效果」)', tog(1, 'data-live="motion"')),
        R('形态切换动画', 'Agent ⇄ IDE 的布局变形,约 0.44 秒;「简化」只淡入淡出', seg(['流畅', '简化', '无'], 0)),
        R('列表落下速度', '导航栏换分段 / 换形态 / 换项目时,列表行自上而下逐行落下(appearance.navDrop);「标准」= .cascade:每行约 0.26 秒、间隔 0.03 秒;慢 ×1.8,快 ×0.55', seg(['慢', '标准', '快', '无'], 1)),
        R('流光样式', '「正在进行」的文字扫光', seg(['流光', '呼吸', '无'], 0)),
        PVX('shimmer'),
        R('等待字符动画', '状态行与回合进行中的字符(appearance.spinner)', seg(['经典', '趣味', 'Claude', '诗意', '极客', '极简'], 2)),
        PVX('spinner'),
        R('等待指示样式', '步骤行 / 子代理 / 任务坞 / 状态栏的进行中图标;按钮内 loading 仍用小圆环', seg(['沙漏', '圆环', '点阵', '脉冲'], 0)),
        PVX('wait'),
        R('滚动条', '自动隐藏 / 跟随系统 / 始终显示', seg(['跟随系统', '自动隐藏', '始终显示'], 1)))}` },

    brand: { t: '应用与菜单栏图标', d: 'Dock 应用图标与菜单栏托盘。', body: () => brandGroups() },

    shortcuts: { t: '快捷键', d: '点一个快捷键就开始录制,按下新的组合即可;冲突会当场提示。', body: () => {
      const K = (n, k) => R(n, '', `<span class="keycap" data-act="keyrec" data-def="${k}" tabindex="0">${kbd(...k.split(' '))}</span>`);
      return `<div class="f g3">${inp('搜索快捷键', { id: 'kbq', icon: 'search', cls: 'fill', style: 'flex:1' })}${btn('恢复默认', '', 'undo', 'data-act="kb-reset"')}</div>
      ${G('全局', K('命令面板', '⌘ K'), K('新对话', '⌘ N'), K('新窗口', '⇧ ⌘ N'), K('设置', '⌘ ,'), K('循环权限模式', '⇧ ⇥'))}
      ${G('形态与面板', K('切换 Agent / IDE', '⌘ E'), K('显示 / 隐藏导航栏', '⌘ B'), K('显示 / 隐藏右侧栏(Agent = 工作区,IDE = 对话)', '⌥ ⌘ B'), K('底部面板', '⌘ J'), K('源代码管理', '⌃ ⇧ G'))}
      ${G('编辑器', K('当前文件查找', '⌘ F'), K('项目内搜索', '⇧ ⌘ F'), K('转到符号', '⇧ ⌘ O'), K('向右拆分', '⌘ \\'), K('全部保存', '⌥ ⌘ S'), K('行注释', '⌘ /'), K('重新打开关闭的标签', '⇧ ⌘ T'))}
      ${G('对话', K('立即引导', '⌘ ⏎'), K('中断回合', 'esc'), K('编辑上一条消息', '↑'))}
      <div class="kb-warn" hidden></div>`; } },

    models: { t: '模型与服务商', d: '默认模型与思考强度、上下文压缩,以及第三方模型服务商。密钥单独存放在 0600 权限的文件里。', body: () => `
      ${G('默认', R('默认模型', '新会话的起始模型;运行中可在输入框运行配置或状态栏里按服务商选模型', `<span class="f g2">${['默认', 'Opus', 'Sonnet', 'Haiku'].map((m, k) => chip(m, '', k === 2 ? 'on' : 'line', 'data-act="chip-one" tabindex="0"')).join('')}</span>`), R('默认思考强度', '', seg(['关', '低', '中', '高'], 2)), R('默认开启扩展思考', '写 ~/.claude/settings.json alwaysThinkingEnabled(未测试)', tog(1)))}
      ${G('上下文', R('自动压缩', 'models.compact 空闲时 CmdCompact(未测试)', tog(1)), R('压缩触发阈值', 'models.compactAt;环与弹层同色阶(未测试)', stp(85, '%', { min: 50, max: 95, step: 5 })), R('输出风格', '', chip('默认', '', 'line')))}
      ${typeof PV !== 'undefined' ? PV.providersSection() + PV.mappingBlock() : ''}` },

    chat: { t: '对话与提示词', d: '会话流的显示与回合行为,以及追加到系统提示词的内容。', body: () => `
      ${G('会话样式', R('会话样式', '对话:你的消息靠右成气泡,回复靠左;通栏:你的消息是与正文同宽、四角圆角的整行卡片,上下排列。「跟随布局」按界面布局方案', segL(['跟随布局', '对话', '通栏'], 0, 'flowStyle')), R('工具调用详细程度', '精简:相邻调用合并成一行摘要;标准:只合并连续的读取 / 搜索;详细:差异与命令输出直接显示在行下', seg(['精简', '标准', '详细'], 1)), PVX('tools'))}
      ${G('显示', R('思考内容默认展开', '关闭时只显示「思考 · 4 秒」一行', tog(0)), R('回合结束后折叠过程', '把本回合的工具步骤折成「已处理 1 分 13 秒」', tog(1)), R('用户消息吸顶', '长会话里滚动时,当前这一轮的提问固定在会话顶部;下一轮顶上来时它被顶走,始终最多一张。点击提问卡可就地编辑', tog(1, 'data-live="stickyUser"')), R('显示回合用量', '回复下方的模型 · 用时 · token · 费用', tog(1)), R('工具组默认折叠', '连续的读取 / 搜索收进一行;失败时自动展开', tog(1)))}
      ${G('新会话', R('默认权限模式', '新会话的起始模式;在输入框里切换后会自动记住,下次新会话沿用(「跳过权限确认」不记)', seg(['逐项确认', '自动接受编辑', '计划'], 0)), R('回合结束提示音', 'chat.ding 回合正常结束时 afplay Tink(未测试)', tog(0)))}
      ${G('排队与引导', R('排队消息带图片与文件引用', 'chat.queueAtt 关时排队只发纯文本(未测试)', tog(1)), R('被中断后暂停队列', 'chat.queuePause 关时出错后继续 drain(未测试)', tog(1)))}
      ${G('追加系统提示词', `<div style="padding:10px 12px">${area('所有回答用中文,注释也用中文。\n提交信息使用 Conventional Commits。\n不要读取或输出任何密钥。', { id: 'sys', rows: 4 })}<div class="f ai g3" style="margin-top:6px"><span class="t3 fs-cap grow">以 --append-system-prompt 传给 claude,新会话生效;多根工作区说明会自动合并</span><span class="t4 fs-cap num" data-count="sys">62 字</span></div></div>`)}
      ${G('记忆文件(CLAUDE.md)', R('记忆文件已移到「自定义 → 规则与记忆」', '在那里编辑文件、查看加载顺序,并开关「加载 CLAUDE.md」', goto('前往', 'customize', 'rules')))}` },

    permissions: { t: '权限与信任', d: '允许、询问与拒绝的规则,以及哪些项目可以运行钩子与 MCP。', body: () => {
      const grp = (id, title, kind, rows) => `<div class="set-grp" id="rg-${id}"><div class="gt">${title}</div><div class="card" data-list>${rows.map(t => ruleRow(kind, t)).join('')}</div></div>`;
      const P = (n, p, trust, hook, mcp, err) => `<div class="set-row"${err ? ' data-trust-err' : ''} style="min-height:52px"><span class="si-box">${ic('folder')}</span><div class="grow"><div class="m">${n}</div><div class="d mono">${p}</div>${err ? `<div class="t4 c-err fs-cap">${err}</div>` : ''}</div><span class="f ai g2 t3 fs-cap">钩子 ${tog(hook)}</span><span class="f ai g2 t3 fs-cap">MCP ${tog(mcp)}</span><span class="f ai g2 t3 fs-cap">信任 ${tog(trust)}</span></div>`;
      return `${G('策略', R('配置层', '规则写入哪一层', seg(['用户', '项目', '本地'], 1)), R('默认权限模式', 'permissions.defaultMode', seg(['逐项确认', '自动接受编辑', '计划'], 0)), R('禁止绕过权限模式', 'permissions.disableBypassPermissionsMode', tog(0)), R('Bash 沙盒', 'sandbox.enabled:在受限环境里执行 Bash 命令', tog(0), { restart: 1 }), R('额外可访问目录', 'permissions.additionalDirectories', field('~/Projects/shared', 200, { mono: 1 })))}
      ${G('添加规则', `<div class="set-row"><label class="inp grow">${ic('plus', 's12')}<input data-in="rule" placeholder="示例:Bash(git status)、Read(src/**)" spellcheck="false"></label>${seg(['允许', '询问', '拒绝'], 0)}${btn('添加', 'sm pri', '', 'data-act="rule-add"')}</div>`)}
      ${grp('allow', '始终允许', 'allow', ['Bash(git status)', 'Bash(cargo test:*)', 'Read(src/**)'])}${grp('ask', '每次询问', 'ask', ['Bash(git push:*)'])}${grp('deny', '始终拒绝', 'deny', ['Read(.env)', 'Bash(rm -rf:*)'])}
      ${G('项目信任', P('router-demo', '~/Projects/router-demo', 1, 1, 1, ''), P('web-dashboard', '~/Projects/web-dashboard', 1, 1, 0, ''), P('docs-site', '~/Projects/docs-site', 0, 0, 0, ''), `<div class="set-row t3 fs-cap" style="padding:12px">合并 ~/.claude.json 与项目注册表 · 撤销信任需确认 · 空态 / 读失败 / 写失败行内提示</div>`, btn('刷新列表', 'sm', 'refresh', 'data-act="toast" data-msg="刷新信任列表"'))}
      ${note('陌生项目带来的钩子与 MCP 默认不运行,信任后才生效。')}`; } },

    runtime: { t: '运行环境', d: 'MiniCode 如何找到并启动 claude:连接、可执行文件、代理与网络、环境变量。', body: () => {
      const nc = (h, ok = 1, ms = '') => `<div class="set-row" data-nc="${ok}" data-ms="${ms}"><span class="nc-ic">${ic(ok ? 'okc' : 'errc', 's12 ' + (ok ? 'c-ok' : 'c-err'))}</span><span class="grow mono">${h}</span><span class="nc-v fs-cap ${ok ? 'c-ok' : 'c-err'}">${ok ? ms + ' ms' : '超时'}</span></div>`;
      const E = (n, l) => envRow(n, l);
      return `${G('连接', R('预热 Claude 连接', 'chat.warm:打开会话时 = 切到该会话(去抖 0.35 秒)就启动 claude 并握手;输入时 = 只在开始打字时启动;不预热 = 第一条消息才启动', seg(['打开会话时', '输入时', '不预热'], 0)), R('空闲后释放连接', 'chat.idle:闲置超过时间就结束 claude 进程省内存(会话保留,再发送自动 --resume);窗口在后台时用 ⅓ 的时间', seg(['5 分钟', '10 分钟', '30 分钟', '不释放'], 1)))}
      ${G('claude 命令行', R('可执行文件', 'MiniCode 直连本机的 claude(stream-json 协议),不经过 Node', `<span class="f g2 ai">${field('/opt/homebrew/bin/claude', 210, { mono: 1 })}${btn('检测', 'sm', 'refresh', 'data-act="claude-detect"')}</span>`), R('版本', '', `<span class="claude-ver">${tag('2.1.318', 'ok')}</span>`))}
      ${G('代理与网络', R('代理模式', '跟随系统:沿用继承的 HTTP(S)_PROXY;直连:清空代理变量', seg(['跟随系统', '直连', '自定义'], 0)), R('代理地址', '写入 HTTP_PROXY、HTTPS_PROXY、ALL_PROXY', field('http://127.0.0.1:7890', 200, { mono: 1 })), R('绕过代理', 'NO_PROXY:这些主机直连', field('localhost,127.0.0.1', 200, { mono: 1 })), R('请求超时(秒)', 'API_TIMEOUT_MS;留空为 claude 默认', field('', 90, { ph: '默认' })))}
      <div class="set-grp" id="net-grp"><div class="gt">连通性</div><div class="card">${nc('api.anthropic.com', 1, 128)}${nc('api.deepseek.com', 1, 212)}${nc('github.com', 0)}<div class="set-row">${btn('重新检测', 'sm', 'refresh', 'data-act="net-check"')}<span class="t3 fs-cap">依次访问以上主机,最长等待 5 秒</span></div></div></div>
      <div class="set-grp" id="env-grp"><div class="gt">环境变量 <span class="t4" style="font-weight:400">只显示变量名,值已遮盖;点眼睛临时显示</span></div><div class="card" data-list>${E('ANTHROPIC_API_KEY', '用户')}${E('HTTPS_PROXY', '用户')}${E('ENABLE_TOOL_SEARCH', '项目')}${E('API_TIMEOUT_MS', '本地')}</div></div>
      ${G('新增或覆盖', `<div class="set-row"><label class="inp grow"><input data-in="env-k" placeholder="变量名,如 HTTPS_PROXY" spellcheck="false"></label><label class="inp grow"><input data-in="env-v" placeholder="值" spellcheck="false"></label>${seg(['用户', '项目', '本地'], 0)}${btn('添加', 'sm pri', '', 'data-act="env-add"')}</div>`)}`; } },

    editor: { t: '编辑器', d: '代码编辑器的字体、显示、格式化与 Git 追溯。', body: () => {
      return `${G('字体与缩进', R('字号', '', stp(12, 'px', { min: 9, max: 24 })), R('缩进', '', seg(['2 空格', '4 空格', 'Tab'], 1)), R('自动保存', '', seg(['关', '延时', '失焦'], 1)))}
      ${G('迷你地图', R('显示迷你地图', '', tog(1)), R('滑块', '视口滑块何时显示', seg(['悬停', '常驻'], 0)), R('宽度', '宽一点能看清更多列', seg(['窄', '中', '宽'], 1)), R('Git 改动刻度', '', tog(1)))}
      ${G('显示', R('行号', '', tog(1)), R('代码折叠', '', tog(1)), R('自动换行', '', tog(0)), R('括号配对高亮', '', tog(1)), R('括号配对着色', '', tog(1)), R('括号配对竖线', '', tog(1)), R('语法高亮', '', tog(1)))}
      ${G('Markdown 与格式化', R('Markdown 默认视图', '打开 .md / .markdown / .mdc 时的样子;标签右上角可随时切换,每个文件记住自己的选择', seg(['预览', '源码'], 0)), R('保存时格式化', '语言服务器或内置格式化器能处理的文件,保存前先整理一遍', tog(0)), R('保存时去掉行尾空白', '没有格式化器的文件也生效', tog(0)))}
      ${G('Git 追溯', R('行内追溯', '光标行末尾显示作者与时间', tog(1)), R('追溯栏', '行号左侧逐行显示提交', tog(0)))}`; } },

    lsp: { t: '语言服务', d: '语言服务器(LSP)与工作区诊断、忽略规则。', body: () => {
      const S = (n, lang, path, inst, ok, en) => `<div class="set-row" data-lsp="${ok}" style="min-height:56px"><span class="lsp-ic">${ic(ok ? 'okc' : 'alert', ok ? 'c-ok' : 'c-warn')}</span><div class="grow"><div class="m">${n}</div><div class="t3 fs-cap">${lang}</div><div class="t3 fs-cap ${ok ? '' : 'c-warn'}">${ok ? path : '未安装 · ' + inst}</div></div><span class="f ai g2 t3 fs-cap">启用 ${tog(en)}</span>${ok ? '' : `${btn('复制', 'sm', 'copy', 'data-act="copy-install"')}${btn('在终端安装', 'sm', 'terminal', 'data-act="toast" data-msg="在终端安装"')}`}</div>`;
      return `${G('语言服务', R('启用语言服务', '悬停、补全、转到定义、诊断', tog(1)))}
      ${G('语言服务器', S('rust-analyzer', 'Rust · .rs', '~/…/rust-analyzer', '', 1, 1), S('gopls', 'Go · .go', '~/go/bin/gopls', '', 1, 1), S('Pyright', 'Python · .py', '', 'npm install -g pyright', 0, 1), S('typescript-language-server', 'TS / JS · .ts', '', 'npm i -g typescript-language-server', 0, 0), `<div class="set-row">${btn('重新检测', 'sm', 'refresh', 'data-act="lsp-scan"')}<span class="t3 fs-cap">后台检测 PATH · 演示 MINI_SET_TRUST 不影响本块</span></div>`)}
      ${G('忽略规则', `<div style="padding:10px 12px">${area('target\nnode_modules\n.git\ndist', { mono: 1, id: 'ign', rows: 4 })}<div class="d t3 fs-cap" style="margin-top:6px">同时作用于文件树、搜索与文件监视</div></div>`)}`; } },

    terminal: { t: '终端', d: '内置终端的外观与 Shell。', body: () => `
      ${G('外观', R('字号', '终端文字大小,行高自动按比例换算', stp(12, 'px', { min: 9, max: 24 })), R('光标形状', '', seg(['块', '竖线', '下划线'], 0)), R('回看行数', '滚轮可回看的历史输出行数;对之后新开的终端生效', field('10000', 90)), R('标签排列', '终端 ≥2 时横排在面板头或竖排在右侧;面板 &lt;560 或窄档强制横向', seg(['横向', '竖向', '自动'], 2, '', 'term.tabs')))}
      ${G('Shell', R('Shell 程序', '留空使用环境变量 $SHELL;失焦或回车应用', field('/bin/zsh', 160, { mono: 1 })), R('登录 shell', '', tog(1)), R('在项目目录启动', '', tog(1)))}` },

    scm: { t: '源代码管理', d: '改动面板、差异与 Worktree。', body: () => `
      ${G('改动面板', R('Diff 视图', '改动、提交详情里差异的默认呈现;差异块右上角也能随时切换', seg(['左右', '统一'], 0)), R('Diff 大小上限', '超过这个大小的文件不计算 diff,避免卡顿', field('512 KB', 100)), R('AI 提交说明', '用 Haiku 根据改动生成', tog(1)), R('自动获取远端', '每 5 分钟 git fetch', tog(0)), R('多仓库:全部展开', '多根工作区里每个仓库各显示一整块更改与提交框(默认只显示活动仓库 + 总览)', tog(0)))}
      ${G('Worktree', R('存放位置', '', field('~/.minicode/worktrees', 200, { mono: 1 })), `<div class="set-row" data-rm><span class="si-box">${ic('branch')}</span><div class="grow"><div class="m">feat/auth</div><div class="d mono">~/.minicode/worktrees/router-demo-auth</div></div>${ib('trash', 'sm', '删除', 'data-act="del-row"')}</div><div class="set-row"><span class="si-box">${ic('plus')}</span><div class="grow"><div class="d mono">新分支名(已存在则直接检出),例如 feat/login</div><div class="d" style="color:var(--err)">分支名不合法:不能含空格、~ ^ : ? * [ \\ 或以 / 开头结尾</div></div>${ib('plus', 'sm', '新建 worktree')}</div>`)}` },

    browser: { t: '浏览器', d: '内置阅读模式浏览器的抓取参数。浏览器页右上角的齿轮会直接跳到这里。', body: () => `${G('抓取', R('请求超时', '单个页面的最长等待时间', stp(20, '秒', { min: 5, max: 120 })), R('User-Agent', '留空使用内置标识;个别站点会按它返回不同内容', field('', 220, { ph: '默认' })), R('默认进入阅读模式', '内置浏览器目前只有阅读模式,此开关暂无作用', tog(1)))}` },

    privacy: { t: '隐私与诊断', d: '默认不上报任何数据;性能、崩溃与卡顿记录都在本机。', body: () => {
      const spark = (d) => `<svg class="spk" viewBox="0 0 120 26" preserveAspectRatio="none"><polyline points="${d}" /></svg>`;
      return `${G('数据上报', R('崩溃报告', '崩溃时是否允许上传匿名堆栈', tog(0)), R('使用统计', '匿名的功能使用次数', tog(0)), R('凭据', 'MiniCode 不读取钥匙串;服务商 API Key 单独存放在 0600 权限的文件里', tag('本地', 'ok')))}
      ${G('性能(最近 5 分钟)', R('内存', '本进程常驻内存', `<b class="num">412 MB</b>`), R('', '', spark('0,20 12,18 24,19 36,14 48,15 60,11 72,12 84,9 96,10 108,7 120,8')), R('CPU', '100% = 一个核心满载', `<b class="num">6%</b>`), R('', '', spark('0,22 12,21 24,22 36,10 48,20 60,22 72,21 84,12 96,22 108,21 120,22')), R('界面构建耗时', '', `<b class="num">平均 1.8 ms</b>`), R('最近一次卡顿', '界面线程停滞超过 3 秒才会记录', '<span class="t3">无</span>'))}
      ${G('日志与诊断包', R('日志级别', '环境变量 MINI_LOG 优先', seg(['错误', '警告', '信息', '调试'], 2)), R('记录目录', 'panic、原生崩溃、卡死、异常退出与系统崩溃报告摘要', btn('在访达中显示', 'sm', 'folder', 'data-act="toast" data-msg="已在访达中显示 ~/.minicode/logs"')), R('诊断包', '家目录显示为 ~,只含配置概况与变量名,不含密钥和环境变量值', btn('导出', 'sm', 'download', 'data-act="diag-export"')))}
      ${G('崩溃与卡顿', `<div class="set-row">${ic('okc', 'c-ok')}<span class="grow t2">最近 30 天没有崩溃记录</span></div>`, R('claude doctor', '检查安装方式、版本、自动更新与配置问题(最多等待 30 秒)', btn('运行', 'sm', 'play', 'data-act="doctor"')))}
      <div class="fold" id="doctor-out"><div class="fold-in"><div class="card" style="padding:10px 12px;display:flex;flex-direction:column;gap:5px"><div class="f ai g3 fs-cap">${ic('okc', 's12 c-ok')}安装方式:原生安装(~/.local/bin/claude)</div><div class="f ai g3 fs-cap">${ic('okc', 's12 c-ok')}版本 2.1.318,已是最新</div><div class="f ai g3 fs-cap">${ic('okc', 's12 c-ok')}自动更新已启用</div><div class="f ai g3 fs-cap">${ic('alert', 's12 c-warn')}~/.claude.json 里有 1 个已失效的项目路径</div></div></div></div>`; } },

    advanced: { t: '高级', d: '数据、运行模式、实验功能与重置。', body: () => `
      ${G('数据', R('数据目录', '设置、服务商、日志与诊断包所在目录', `<span class="f g2 ai">${field('~/.minicode', 150, { mono: 1 })}${btn('打开', 'sm', 'folder', 'data-act="toast" data-msg="已在访达中显示 ~/.minicode"')}</span>`), R('设置文件', '导出 / 导入 settings.json(不含密钥)', `<span class="f g2">${btn('导出', 'sm', 'download', 'data-act="toast" data-msg="已导出 settings.json"')}${btn('导入…', 'sm', 'upload')}</span>`))}
      ${G('运行模式', R('真实 / 演示', '演示用脚本化后端与沙盒项目,不连接 claude;切换后会重启', seg(['真实', '演示'], 0, '', 'runmode'), { restart: 1 }))}
      ${G('实验功能', R('前台命令实时输出', '命令运行期间每 0.7 秒读取任务输出文件,在工具卡里实时显示', tog(1)), R('主线程卡死看门狗', '停滞超过阈值时抓取调用栈,写入诊断记录', tog(1)), R('会话流一次测完行高', '载入长会话时先测量全部行,滚动更稳(首帧稍慢)(Go 版暂不生效:列表由库自行测量)', tog(1)))}
      ${G('重置', R('恢复默认客户端偏好', '主题、发送键、通知、代理等恢复默认;不影响 Claude 配置、栏宽与最近项目', btn('恢复默认…', 'sm danger', '', 'data-act="reset-all"')))}` },

    about: { t: '关于', d: '', body: () => `
      <div class="card" style="padding:22px;display:flex;flex-direction:column;align-items:center;gap:6px;text-align:center">${U.brand.logo(52)}<div class="fs-head b" style="margin-top:6px">MiniCode</div><div class="t3">版本 0.2.0 · 构建 4f2a91c · 3 天前</div>
        <div id="upd-st" class="fs-cap t3" style="min-height:16px;display:flex;align-items:center;gap:6px"><span class="c-ok f ai g2">${ic('okc', 's12')}已是最新</span></div>
        <div id="upd-bar" class="meter" style="width:220px;display:none"><i style="width:0"></i></div>
        <div class="f g3" style="margin-top:6px">${btn('检查更新', '', 'refresh', 'id="upd-btn" data-act="upd-check"')}${btn('复制版本信息', 'ghost', 'copy', 'data-act="toast" data-msg="已复制版本信息"')}</div></div>
      ${G('更新', R('自动下载更新', '发现新版本时自动下载,重启后生效', tog(1)), R('更新通道', '', seg(['稳定', '测试'], 0)))}
      ${G('', R('许可', '', '<span class="t3">MIT</span>'), R('开源致谢', '使用到的第三方库与许可', ic('chevR', 's12 t4')), R('在 GitHub 上查看', '发行说明与问题反馈', ic('ext', 's12 t4')))}` },
  };

  /* ───────── 搜索 ───────── */
  const INDEX = {};
  const buildIndex = () => Object.keys(TABS).forEach(id => { REC = []; PEND = []; try { TABS[id].body({ scheme: 'a', theme: 'paper' }); } catch (e) { /* 取不到就当没有 */ } PEND.forEach(p => REC.push({ ...p, g: '' })); INDEX[id] = REC; REC = null; PEND = []; });
  const re = (q) => new RegExp('(' + q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'ig');
  const hl = (s, q) => esc(s).replace(re(esc(q)), '<mark class="hit">$1</mark>');
  /** 命中的行:{ id, t, d, g } */
  const hits = (q) => { if (!Object.keys(INDEX).length) buildIndex(); const k = q.toLowerCase(), out = [];
    Object.keys(TABS).forEach(id => INDEX[id].forEach(r => { if ((r.t + ' ' + r.d + ' ' + r.g + ' ' + TABS[id].t).toLowerCase().includes(k)) out.push({ id, ...r }); })); return out; };
  const searchView = (q) => {
    const hs = hits(q); if (!hs.length) return `<div class="empty" style="padding:48px 12px">${ic('search', 's20 t4')}<b>没有匹配「${esc(q)}」的设置</b><div class="t3 fs-cap" style="line-height:1.6">换个关键字试试,例如「主题」「代理」「快捷键」「服务商」。<br>功能内容(MCP、Skills、命令……)在侧栏的「自定义」页。</div>${btn('清除搜索', 'sm', 'x', 'data-act="setq-clear"')}</div>`;
    const by = {}; hs.forEach(h => (by[h.id] = by[h.id] || []).push(h));
    return `<div><div class="fs-head b">搜索结果</div><div class="t3">${hs.length} 项,分布在 ${Object.keys(by).length} 个分类里 · 点一项直接跳过去 · Esc 清除</div></div>
      ${Object.keys(by).map(id => `<div class="set-grp"><div class="gt f ai g3">${ic(ICON[id], 's12')}${TABS[id].t}</div><div class="card">${by[id].map(h => `<div class="set-row sr" data-act="setgo" data-tab="${id}" data-k="${esc(h.t)}" style="cursor:pointer"><div class="grow"><div class="m">${hl(h.t, q)}</div>${h.d ? `<div class="d">${hl(h.d, q)}</div>` : ''}</div>${h.g ? `<span class="t4 fs-micro nw">${esc(h.g)}</span>` : ''}${ic('chevR', 's12 t4')}</div>`).join('')}</div></div>`).join('')}`;
  };

  /** 导航项。q 非空时右侧显示命中数 */
  const navi = (id, cur, n) => `<div class="row ${id === cur && n === undefined ? 'on' : ''} ${n === 0 ? 'dim' : ''}" data-act="settab" data-tab="${id}" tabindex="0">${ic(ICON[id], 's12')}<span class="grow">${TABS[id].t}</span>${n ? cnt(n, 'ac') : ''}</div>`;
  /** 右侧正文:查询为空时是当前分类,否则是搜索结果 */
  W.settingsView = (st) => {
    const q = (st.setQ || '').trim(); if (q) return searchView(q);
    const T = TABS[W.settingsCur(st)];
    return `<div><div class="fs-head b">${T.t}</div>${T.d ? `<div class="t3" style="max-width:560px;line-height:1.5">${T.d}</div>` : ''}</div>${T.body(st)}`;
  };
  W.settingsCur = (st) => { const k = ALIAS[st.setTab] || st.setTab; return TABS[k] ? k : 'appearance'; };
  /** 设置页。st.setTab:当前分类(默认外观);st.setQ:搜索词 */
  W.settings = (st) => {
    const cur = W.settingsCur(st), q = (st.setQ || '').trim(), by = {};
    if (q) hits(q).forEach(h => by[h.id] = (by[h.id] || 0) + 1);
    return `<div class="set"><div class="set-nav fc g1"><div class="row set-back" data-act="screen" data-screen="work" tabindex="0" style="margin-bottom:6px;color:var(--tx-2)">${ic('arrowL', 's12')}<span class="grow">返回应用</span>${kbd('esc')}</div>${inp('搜索设置', { id: 'setq', icon: 'search', cls: 'fill', v: esc(st.setQ || ''), style: 'margin-bottom:6px' })}
      <div class="scroll fc g1" style="flex:1;min-height:0">${GROUPS.map(([g, ids]) => `<div class="grp-h" style="margin-top:4px">${g}</div>${ids.map(id => navi(id, cur, q ? (by[id] || 0) : undefined)).join('')}`).join('')}</div></div>
      <div class="set-main scroll" id="set-main"><div class="set-col a-rise" id="set-col">${W.settingsView(st)}</div>
        <div class="rbar" hidden><span class="grow">${ic('refresh', 's12')} <b class="rcount">1</b> 项更改需要重启后生效</span>${btn('稍后', 'sm ghost', '', 'data-act="rbar-later"')}${btn('立即重启', 'sm pri', '', 'data-act="rbar-go"')}</div></div></div>`;
  };
  W.settingsTabs = Object.keys(TABS).length;
  W.settingsApi = { ruleRow, envRow, provRow, hits, buildIndex };
})();