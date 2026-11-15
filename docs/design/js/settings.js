/* 设计稿 v3 · 设置页:四组 22 个分类,全部数据驱动;「外观」里有「界面布局」,点一下就地切换三套布局
   分类、分组与真实产品的 SettingsTab 一一对应。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, kbd, seg, tog, THEMES, SCHEMES } = U;

  /** 导航分组:[组名, [分类 id…]] */
  const GROUPS = [
    ['客户端', ['general', 'appearance', 'chat', 'shortcuts']],
    ['Claude Code', ['customize', 'model', 'providers', 'prompts', 'permissions', 'trust', 'env']],
    ['工作台', ['editor', 'terminal', 'browser', 'scm', 'intel']],
    ['其他', ['network', 'usage', 'privacy', 'diagnostics', 'advanced', 'about']],
  ];
  const ICON = { general: 'sliders', appearance: 'palette', chat: 'chat', shortcuts: 'kbd', customize: 'plug', model: 'cpu', providers: 'server', prompts: 'fileText', permissions: 'shield', trust: 'shieldCk', env: 'key',
    editor: 'code', terminal: 'terminal', browser: 'link', scm: 'branch', intel: 'brain', network: 'globe', usage: 'chart', privacy: 'lock', diagnostics: 'activity', advanced: 'wrench', about: 'info' };

  /* ───────── 部件 ───────── */
  /** 设置行。t:标题;d:说明;ctl:右侧控件 */
  const R = (t, d, ctl = '') => `<div class="set-row"><div class="grow"><div class="m">${t}</div>${d ? `<div class="d">${d}</div>` : ''}</div>${ctl}</div>`;
  /** 分组卡。title:小标题;rows:若干行 */
  const G = (title, ...rows) => `<div class="set-grp">${title ? `<div class="gt">${title}</div>` : ''}<div class="card">${rows.join('')}</div></div>`;
  const fld = (v, w = 180, c = '') => `<span class="fld ${c}" style="width:${w}px">${v}</span>`;
  const stp = (v, u = '') => `<span class="stp"><button>−</button><b class="num">${v}</b>${u ? `<span class="t3">${u}</span>` : ''}<button>+</button></span>`;
  const note = (t) => `<div class="t3 fs-cap" style="padding:0 4px;line-height:1.5">${t}</div>`;
  const keys = (s) => kbd(...s.split(' '));

  /** 界面布局三选一:点击立即切换整个查看器的方案(data-act="layout") */
  const layCards = (cur) => `<div class="f g3">${SCHEMES.map(s => `<div class="lay-card ${s.id === cur ? 'on' : ''}" data-act="layout" data-id="${s.id}">
    <div class="lay-prev" data-lay="${s.id}"><i class="rl"></i><i class="sd"></i><i class="mn"><b></b><b></b><b></b></i><i class="rt"></i></div>
    <div class="f ai jb"><b>${s.name}</b><span class="t4 fs-micro">${s.en}</span></div><div class="t3 fs-cap" style="line-height:1.4">${s.tagline}</div></div>`).join('')}</div>`;
  const accents = ['--c-accent', '--c-info', '--c-ok', '--c-cyan', '--c-vio', '--c-pink', '--c-err', '--c-warn'];

  /* ───────── 各分类 ───────── */
  const TABS = {
    general: { t: '通用', d: '窗口、输入、通知与语言。', body: () => `
      ${G('窗口', R('右侧面板默认打开', '启动时是否显示改动 / 文件 / 终端面板', tog(1)), R('融合模式', '关闭时 IDE 是独立窗口(⇧⌘E 打开或回到 Agent);开启后在同一窗口原地切换', tog(1)), R('启动时恢复上次的会话与布局', '', tog(1)))}
      ${G('输入', R('发送键', '另一个键用于换行(Shift+Enter 始终换行)', seg(['Enter', '⌘ Enter'], 0)), R('回合进行中再发消息', 'Enter = 排队;⌘Enter = 立即引导', seg(['排队', '立即引导'], 0)))}
      ${G('通知与菜单栏', R('任务完成时通知', '回合结束且 MiniCode 不在前台时发送系统通知', tog(1)), R('菜单栏图标', '在系统菜单栏显示 MiniCode 图标', tog(1)), R('需要确认时提醒', '权限请求、提问在后台出现时弹通知', tog(1)))}
      ${G('语言', R('界面语言', '目前仅提供简体中文', chip('简体中文', '', 'line')))}` },

    appearance: { t: '外观', d: '布局、主题、密度与动效。所有选项都落在设计令牌上,即改即生效。', body: (st) => `
      ${G('界面布局', `<div style="padding:10px 12px 12px"><div class="d t3 fs-cap" style="margin-bottom:8px">三种布局共用同一份令牌与组件,只在「面板怎么分隔、导航放在哪、步骤怎么串」上不同。点一下立即切换,Agent 与 IDE 同步变化。</div>${layCards(st.scheme)}</div>`)}
      ${G('主题', `<div style="padding:10px 12px">${W.themeCards(st.theme)}</div>`, R('跟随系统深浅色', '浅色用「宣纸」,深色用「墨夜」', tog(0)))}
      ${G('强调色', R('强调色', '按钮、选中态、焦点环、选区、链接等使用的主题色;每个主题单独记忆,并按明暗自动调整明度', `<span class="f g2">${accents.map((a, k) => `<span class="swatch ${k ? '' : 'on'}" style="background:var(${a})"></span>`).join('')}</span>`))}
      ${G('密度与形状', R('界面密度', '行高令牌 --h-row:紧凑 22 / 标准 24 / 宽松 26', seg(['紧凑', '标准', '宽松'], 1)), R('圆角', '圆角倍率 --r-k', seg(['利落', '适中', '圆润'], 1)))}
      ${G('Agent 会话', R('内容宽度', '会话时间线与输入框的最大宽度;窄屏或分屏时会自动收窄', seg(['窄', '标准', '宽'], 1)), R('工具调用详细程度', '精简只显示动词与对象;详细默认展开输出', seg(['精简', '标准', '详细'], 1)), R('时间线字号', '会话里 Claude 回复正文的字号(85%–150%)', stp(100, '%')))}
      ${G('动效', R('动效', '流光、滑入与布局变形;关闭后所有时长令牌归零', tog(1)), R('流光样式', '「正在进行」的文字扫光', seg(['流光', '呼吸', '无'], 0)), R('等待动画', '转圈的样式', seg(['圆环', '点阵', '脉冲'], 0)), R('滚动条', '「跟随系统」沿用 macOS「显示滚动条」设置', seg(['跟随系统', '自动隐藏', '始终显示'], 1)))}` },

    chat: { t: '对话', d: '会话流里的默认展开与回合行为。', body: () => `
      ${G('显示', R('思考内容默认展开', '关闭时只显示「思考 · 4 秒」一行', tog(0)), R('回合结束后折叠过程', '把本回合的工具步骤折成「已处理 1 分 13 秒」', tog(1)), R('显示回合用量', '回复下方的模型 · 用时 · token · 费用', tog(1)), R('工具组默认折叠', '连续的读取 / 搜索收进一行;失败时自动展开', tog(1)))}
      ${G('新会话', R('默认权限模式', '新会话的起始模式;运行中可随时在输入框切换', seg(['逐项确认', '自动接受编辑', '计划'], 0)), R('默认模型', '', chip('Sonnet 5.5', '', 'line')), R('默认思考强度', '', seg(['关', '低', '中', '高'], 2)))}
      ${G('排队与引导', R('排队消息带图片与文件引用', '', tog(1)), R('被中断后暂停队列', '中断或出错后不自动发送排队消息', tog(1)))}` },

    shortcuts: { t: '快捷键', d: '全局与编辑器常用快捷键(对标 VS Code)。', body: () => {
      const L = [['命令面板', '⌘ K'], ['新对话', '⌘ N'], ['切换 Agent / IDE', '⌘ E'], ['显示 / 隐藏导航栏', '⌘ B'], ['显示 / 隐藏检查器', '⌥ ⌘ B'], ['终端', '⌃ `'], ['设置', '⌘ ,'], ['循环权限模式', '⇧ ⇥'],
        ['当前文件查找', '⌘ F'], ['项目内搜索', '⇧ ⌘ F'], ['转到符号', '⇧ ⌘ O'], ['向右拆分', '⌘ \\'], ['全部保存', '⌥ ⌘ S'], ['行注释', '⌘ /'], ['重新打开关闭的标签', '⇧ ⌘ T']];
      return `<div class="inp fill">${ic('search', 's12')}<span class="grow">搜索快捷键</span></div>${G('', ...L.map(([n, k]) => R(n, '', keys(k))))}`; } },

    customize: { t: '自定义', d: '插件、MCP、Skills、命令、子 Agent、规则与钩子,在「自定义」页集中管理。', body: () => `
      ${G('', ...[['plug', '插件', '4 个已启用', 'plugins'], ['server', 'MCP 服务器', '3 个已启用 · 1 个需要授权', 'mcp'], ['spark', 'Skills', '12 个', 'skills'], ['slash', '斜杠命令', '8 个', 'commands'], ['bot', '子 Agent', '5 个', 'agents'], ['book', '规则与记忆', '2 个文件', 'rules'], ['bolt', '钩子', '3 个', 'hooks']]
        .map(([i, t, d, id]) => `<div class="set-row" data-act="screen" data-screen="customize" data-cust="${id}" style="cursor:pointer"><span class="si-box">${ic(i)}</span><div class="grow"><div class="m">${t}</div><div class="d">${d}</div></div>${ic('chevR', 's12 t4')}</div>`))}
      <div class="f g3">${btn('打开自定义页', 'pri', 'arrow', 'data-act="screen" data-screen="customize"')}${btn('浏览插件市场', '', 'ext')}</div>` },

    model: { t: '模型与思考', d: '默认模型、扩展思考与上下文压缩。', body: () => `
      ${G('常用模型', R('点一下填入并保存', '', `<span class="f g2">${['Fable 5.2', 'Opus 5.5', 'Sonnet 5.5', 'Haiku 5.5'].map((m, k) => chip(m, '', k === 2 ? 'on' : 'line')).join('')}</span>`), R('默认开启扩展思考', 'alwaysThinkingEnabled', tog(1)), R('默认思考强度', '', seg(['关', '低', '中', '高'], 2)))}
      ${G('上下文', R('自动压缩', '接近上限时自动 /compact', tog(1)), R('压缩触发阈值', '占用达到该比例时压缩', stp(85, '%')), R('输出风格', '', chip('默认', '', 'line')))}
      ${G('API', R('当前 API 提供方', '按环境变量判断', tag('Anthropic 官方', 'ok')), R('ANTHROPIC_BASE_URL', '来自「模型服务商」当前启用项', fld('未设置', 220, 't4')))}` },

    providers: { t: '模型服务商', d: '添加兼容 Anthropic 协议的服务商,并把 Opus / Sonnet / Haiku 三个槽位映射到它的模型。密钥单独存放在 0600 权限的文件里。', body: () => {
      const P = (n, u, on, s) => `<div class="set-row"><span class="si-box">${ic('server')}</span><div class="grow"><div class="m f ai g3">${n}${s ? tag(s, s === '已连通' ? 'ok' : 'warn') : ''}</div><div class="d mono">${u}</div></div>${on ? tag('当前', 'ac') : btn('切换', 'sm')}${ib('more', 'sm')}</div>`;
      return `${G('已添加', P('Anthropic 官方', 'https://api.anthropic.com', 1, '已连通'), P('DeepSeek', 'https://api.deepseek.com/anthropic', 0, '已连通'), P('自建网关', 'https://llm.example.com', 0, '未验证'))}
      ${G('模型映射 · DeepSeek', R('Opus 槽位', '', fld('deepseek-reasoner', 190, 'mono')), R('Sonnet 槽位', '', fld('deepseek-chat', 190, 'mono')), R('Haiku 槽位', '', fld('deepseek-chat', 190, 'mono')))}
      ${G('编辑服务商', R('名称', '', fld('DeepSeek')), R('Base URL', 'Anthropic 协议兼容的地址', fld('https://api.deepseek.com/anthropic', 260, 'mono')), R('API Key', '保存后不再显示,留空保持原值', fld('••••••••••••••a3f9', 220, 'mono')), R('鉴权方式', '', seg(['API Key', 'Auth Token'], 0)), R('操作', '', `<span class="f g2">${btn('测试连通', 'sm', 'activity')}${btn('保存', 'sm pri')}</span>`))}
      <div>${btn('添加服务商', '', 'plus')}</div>`; } },

    prompts: { t: '提示词', d: '追加到系统提示词的内容,以及记忆文件的加载。', body: () => `
      ${G('追加系统提示词', `<div style="padding:10px 12px"><div class="ta mono">所有回答用中文,注释也用中文。\n提交信息使用 Conventional Commits。\n不要读取或输出任何密钥。</div><div class="d t3 fs-cap" style="margin-top:6px">以 --append-system-prompt 传给 claude,会与多根工作区说明合并</div></div>`)}
      ${G('记忆文件', R('加载 CLAUDE.md', '关闭时注入 CLAUDE_CODE_DISABLE_CLAUDE_MDS=1,新会话不读任何记忆文件', tog(1)), R('~/.claude/CLAUDE.md', '用户级 · 1.2 KB', btn('编辑', 'sm ghost', 'edit')), R('router-demo/CLAUDE.md', '项目级 · 3.4 KB', btn('编辑', 'sm ghost', 'edit')))}` },

    permissions: { t: '权限', d: '允许、询问与拒绝的规则,写入所选配置层。', body: () => {
      const rule = (k, t) => `<div class="set-row" style="min-height:30px">${tag({ allow: '允许', ask: '询问', deny: '拒绝' }[k], { allow: 'ok', ask: 'warn', deny: 'err' }[k])}<span class="mono grow">${t}</span>${ib('trash', 'sm')}</div>`;
      return `${G('', R('配置层', '规则写入哪一层', seg(['用户', '项目', '本地'], 1)), R('默认权限模式', 'permissions.defaultMode;新会话的默认模式在「对话」页', seg(['逐项确认', '自动接受编辑', '计划'], 0)), R('禁止绕过权限模式', 'permissions.disableBypassPermissionsMode', tog(0)), R('Bash 沙盒', 'sandbox.enabled:在受限环境里执行 Bash 命令', tog(0)), R('额外可访问目录', 'permissions.additionalDirectories', fld('~/Projects/shared', 200, 'mono')))}
      ${G('添加规则', `<div class="set-row"><div class="inp grow">${ic('plus', 's12')}<span class="grow">示例:Bash(git status)、Read(src/**)</span></div>${seg(['允许', '询问', '拒绝'], 0)}${btn('添加', 'sm pri')}</div>`)}
      ${G('始终允许', rule('allow', 'Bash(git status)'), rule('allow', 'Bash(cargo test:*)'), rule('allow', 'Read(src/**)'))}${G('每次询问', rule('ask', 'Bash(git push:*)'))}${G('始终拒绝', rule('deny', 'Read(.env)'), rule('deny', 'Bash(rm -rf:*)'))}`; } },

    trust: { t: '项目信任', d: '陌生项目带来的钩子与 MCP 默认不运行,信任后才生效。', body: () => {
      const P = (n, p, a, b) => `<div class="set-row"><span class="si-box">${ic('folder')}</span><div class="grow"><div class="m">${n}</div><div class="d mono">${p}</div></div><span class="f ai g2 t3 fs-cap">钩子 ${tog(a)}</span><span class="f ai g2 t3 fs-cap">MCP ${tog(b)}</span></div>`;
      return G('项目', P('router-demo', '~/Projects/router-demo', 1, 1), P('web-dashboard', '~/Projects/web-dashboard', 1, 0), P('docs-site', '~/Projects/docs-site', 0, 0)); } },

    env: { t: '环境变量', d: '注入到 claude 进程的环境变量。只显示变量名,值已遮盖。', body: () => {
      const E = (n, l) => `<div class="set-row" style="min-height:32px"><span class="mono m" style="width:230px">${n}</span><span class="mono t3 grow">••••••••</span>${tag(l)}${ib('trash', 'sm')}</div>`;
      return G('已设置', E('ANTHROPIC_API_KEY', '用户'), E('HTTPS_PROXY', '用户'), E('ENABLE_TOOL_SEARCH', '项目'), E('API_TIMEOUT_MS', '本地')) + G('新增或覆盖', `<div class="set-row"><div class="inp grow">变量名</div><div class="inp grow">值</div>${btn('添加', 'sm pri')}</div>`); } },

    editor: { t: '编辑器', d: '代码编辑器的显示、括号与 Git 追溯。', body: () => `
      ${G('字体与缩进', R('字号', '', stp(12, 'px')), R('缩进', '', seg(['2 空格', '4 空格', 'Tab'], 1)), R('自动保存', '', seg(['关', '延时', '失焦'], 1)))}
      ${G('迷你地图', R('显示迷你地图', '', tog(1)), R('滑块', '视口滑块何时显示', seg(['悬停', '常驻'], 0)), R('宽度', '宽一点能看清更多列', seg(['窄', '中', '宽'], 1)), R('Git 改动刻度', '', tog(1)))}
      ${G('显示', R('行号', '', tog(1)), R('代码折叠', '', tog(1)), R('自动换行', '', tog(0)), R('括号配对高亮', '', tog(1)), R('括号配对着色', '', tog(1)), R('括号配对竖线', '', tog(1)), R('语法高亮', '', tog(1)))}
      ${G('Git 追溯', R('行内追溯', '光标行末尾显示作者与时间', tog(1)), R('追溯栏', '行号左侧逐行显示提交', tog(0)))}` },

    terminal: { t: '终端', d: '内置终端的外观与 Shell。', body: () => `
      ${G('外观', R('字号', '终端文字大小,行高自动按比例换算', stp(12, 'px')), R('光标形状', '', seg(['块', '竖线', '下划线'], 0)), R('回看行数', '滚轮可回看的历史输出行数;对之后新开的终端生效', fld('10000', 90)))}
      ${G('Shell', R('Shell 程序', '留空使用环境变量 $SHELL;失焦或回车应用', fld('/bin/zsh', 160, 'mono')), R('登录 shell', '', tog(1)), R('在项目目录启动', '', tog(1)))}` },

    browser: { t: '浏览器', d: '内置阅读模式浏览器的抓取参数。', body: () => `${G('抓取', R('请求超时', '单个页面的最长等待时间', stp(20, '秒')), R('User-Agent', '留空使用内置标识;个别站点会按它返回不同内容', fld('默认', 200, 't4')), R('默认进入阅读模式', '', tog(1)))}` },

    scm: { t: '源代码管理', d: '改动面板、差异与 Worktree。', body: () => `
      ${G('改动面板', R('Diff 视图', '改动、提交详情里差异的默认呈现;差异块右上角也能随时切换', seg(['左右', '统一'], 0)), R('Diff 大小上限', '超过这个大小的文件不计算 diff,避免卡顿', fld('512 KB', 100)), R('AI 提交说明', '用 Haiku 根据改动生成', tog(1)), R('自动获取远端', '每 5 分钟 git fetch', tog(0)))}
      ${G('Worktree', R('存放位置', '', fld('~/.minicode/worktrees', 200, 'mono')), `<div class="set-row"><span class="si-box">${ic('branch')}</span><div class="grow"><div class="m">feat/auth</div><div class="d mono">~/.minicode/worktrees/router-demo-auth</div></div>${ib('trash', 'sm')}</div>`)}` },

    intel: { t: '代码智能', d: '语言服务器(LSP)、诊断与搜索忽略规则。', body: () => {
      const S = (n, st, ok) => `<div class="set-row" style="min-height:32px">${ic(ok ? 'okc' : 'alert', ok ? 'c-ok' : 'c-warn')}<span class="m grow">${n}</span><span class="t3 fs-cap">${st}</span>${ok ? '' : btn('复制安装命令', 'sm', 'copy')}</div>`;
      return `${G('', R('启用语言服务', '悬停、补全、转到定义、诊断', tog(1)))}${G('服务器', S('rust-analyzer', '已就绪 · 索引完成', 1), S('Pyright', '未安装', 0), S('gopls', '未安装', 0), S('typescript-language-server', '未安装', 0))}
      ${G('忽略规则', `<div style="padding:10px 12px"><div class="ta mono">target\nnode_modules\n.git\ndist</div><div class="d t3 fs-cap" style="margin-top:6px">同时作用于文件树、搜索与文件监视</div></div>`)}`; } },

    network: { t: '网络', d: '代理与请求超时,可一键检测连通性。', body: () => `
      ${G('代理', R('代理模式', '跟随系统:沿用继承的 HTTP(S)_PROXY;直连:清空代理变量', seg(['跟随系统', '直连', '自定义'], 0)), R('代理地址', '写入 HTTP_PROXY、HTTPS_PROXY、ALL_PROXY', fld('http://127.0.0.1:7890', 200, 'mono')), R('绕过代理', 'NO_PROXY:这些主机直连', fld('localhost,127.0.0.1', 200, 'mono')))}
      ${G('请求', R('请求超时(秒)', 'API_TIMEOUT_MS;留空为 claude 默认', fld('留空', 90, 't4')))}
      ${G('连通性', R('api.anthropic.com', '', `<span class="c-ok f ai g2">${ic('okc', 's12')}128 ms</span>`), R('api.deepseek.com', '', `<span class="c-ok f ai g2">${ic('okc', 's12')}212 ms</span>`), R('github.com', '', `<span class="c-err f ai g2">${ic('errc', 's12')}超时</span>`), `<div class="set-row">${btn('重新检测', 'sm', 'refresh')}</div>`)}` },

    usage: { t: '用量与成本', d: '本机各会话的 token 与费用估算,以及账户限额。', body: () => {
      const stat = (v, l) => `<div class="stat-c"><b class="num">${v}</b><span class="t3 fs-cap">${l}</span></div>`;
      const bar = (n, p, v) => `<div class="set-row" style="min-height:28px"><span style="width:110px">${n}</span><div class="meter grow" style="height:6px"><i style="width:${p}%"></i></div><span class="num t3" style="width:60px;text-align:right">${v}</span></div>`;
      return `<div class="f g3">${stat('$12.40', '本月费用')}${stat('3.8M', '输入 token')}${stat('212k', '输出 token')}${stat('64', '会话')}</div>
      ${G('按模型', bar('Sonnet 5.5', 72, '$8.90'), bar('Opus 5.5', 21, '$2.60'), bar('Haiku 5.5', 7, '$0.90'))}
      ${G('账户限额', R('5 小时窗口', '38% · 2 小时 10 分后重置', `<div class="meter" style="width:140px;height:6px"><i style="width:38%"></i></div>`), R('每周', '12% · 周四重置', `<div class="meter" style="width:140px;height:6px"><i style="width:12%"></i></div>`), R('更新于 3 分钟前', '只在打开面板或点「刷新」时查询', btn('刷新', 'sm', 'refresh')))}`; } },

    privacy: { t: '隐私与遥测', d: '默认不上报任何数据。', body: () => `
      ${G('数据上报', R('崩溃报告', '崩溃时是否允许上传匿名堆栈', tog(0)), R('使用统计', '匿名的功能使用次数', tog(0)))}
      ${G('本地', R('日志与诊断包', '家目录显示为 ~,只含配置概况与变量名,不含密钥和环境变量值', btn('导出', 'sm', 'download')), R('凭据', 'MiniCode 不读取钥匙串;服务商 API Key 单独存放在 0600 权限的文件里', tag('本地', 'ok')))}` },

    diagnostics: { t: '诊断', d: '性能、崩溃与卡顿记录。', body: () => {
      const spark = (d) => `<svg class="spk" viewBox="0 0 120 26" preserveAspectRatio="none"><polyline points="${d}" /></svg>`;
      return `${G('性能(最近 5 分钟)', R('内存', '本进程常驻内存', `<b class="num">412 MB</b>`), R('', '', spark('0,20 12,18 24,19 36,14 48,15 60,11 72,12 84,9 96,10 108,7 120,8')), R('CPU', '100% = 一个核心满载', `<b class="num">6%</b>`), R('', '', spark('0,22 12,21 24,22 36,10 48,20 60,22 72,21 84,12 96,22 108,21 120,22')), R('界面构建耗时', '', `<b class="num">平均 1.8 ms</b>`), R('最近一次卡顿', '界面线程停滞超过 3 秒才会记录', '<span class="t3">无</span>'))}
      ${G('日志', R('日志级别', '环境变量 MINI_LOG 优先', seg(['错误', '警告', '信息', '调试'], 2)), R('记录目录', 'panic、原生崩溃、卡死、异常退出与系统崩溃报告摘要', btn('在访达中显示', 'sm', 'folder')))}
      ${G('崩溃与卡顿', `<div class="set-row">${ic('okc', 'c-ok')}<span class="grow t2">最近 30 天没有崩溃记录</span></div>`, R('运行 claude doctor', '检查安装方式、版本、自动更新与配置问题(最多等待 30 秒)', btn('运行', 'sm', 'play')))}`; } },

    advanced: { t: '高级', d: '运行参数与重置。', body: () => `
      ${G('MCP 运行参数', R('MCP 工具按需加载', 'ENABLE_TOOL_SEARCH:工具很多时只在需要时加载描述,节省上下文', tog(1)))}
      ${G('客户端', R('claude 可执行文件', '', fld('/opt/homebrew/bin/claude', 220, 'mono')), R('数据目录', '设置、服务商、日志与诊断包所在目录', fld('~/.minicode', 160, 'mono')), R('恢复默认客户端偏好', '主题、发送键、通知、代理等恢复默认;不影响 Claude 配置、栏宽与最近项目', btn('恢复默认', 'sm danger')))}` },

    about: { t: '关于', d: '', body: () => `
      <div class="card" style="padding:22px;display:flex;flex-direction:column;align-items:center;gap:6px;text-align:center"><span class="logo" style="width:52px;height:52px;border-radius:var(--r-xl)">${ic('mark', 's20')}</span><div class="fs-head b" style="margin-top:6px">MiniCode</div><div class="t3">版本 0.2.0 · 构建 4f2a91c · 3 天前</div>
        <div class="f g3" style="margin-top:8px">${btn('检查更新', '', 'refresh')}${btn('复制版本信息', 'ghost', 'copy')}</div></div>
      ${G('', R('更新', '发现新版本时自动下载', `<span class="c-ok f ai g2">${ic('okc', 's12')}已是最新</span>`), R('运行模式', '切换后会重启', seg(['真实', '演示'], 0)), R('许可', '', '<span class="t3">MIT</span>'))}` },
  };

  /** 导航项 */
  const navi = (id, cur) => `<div class="row ${id === cur ? 'on' : ''}" data-act="settab" data-tab="${id}">${ic(ICON[id], 's12')}<span class="grow">${TABS[id].t}</span></div>`;
  /** 设置页。st.setTab:当前分类(默认外观) */
  W.settings = (st) => {
    const cur = TABS[st.setTab] ? st.setTab : 'appearance', T = TABS[cur];
    return `<div class="set"><div class="set-nav fc g1"><div class="inp fill" style="margin-bottom:6px">${ic('search', 's12')}<span class="grow">搜索设置</span></div>
      ${GROUPS.map(([g, ids]) => `<div class="grp-h" style="margin-top:4px">${g}</div>${ids.map(id => navi(id, cur)).join('')}`).join('')}<span class="grow"></span>${btn('返回', 'ghost', 'chevL', 'data-act="screen" data-screen="work"')}</div>
      <div class="set-main"><div class="set-col a-rise"><div><div class="fs-head b">${T.t}</div>${T.d ? `<div class="t3" style="max-width:560px;line-height:1.5">${T.d}</div>` : ''}</div>${T.body(st)}</div></div></div>`;
  };
  W.settingsTabs = Object.keys(TABS).length;
})();
