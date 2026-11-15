/* 设置页设计稿:三种布局(通用页)+ 外观 / 模型服务商 / 权限 / 用量 */
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter, hl } = K;
  const { C } = window;
  S.group('settings', '设置');

  const NAV = [
    ['客户端', [['通用', 'sliders'], ['外观', 'palette'], ['对话', 'bot'], ['快捷键', 'kbd']]],
    ['Claude Code', [['自定义', 'plug'], ['模型与思考', 'spark'], ['模型服务商', 'server'], ['提示词', 'fileText'], ['权限', 'shield'], ['项目信任', 'shieldCk'], ['环境变量', 'code']]],
    ['工作台', [['编辑器', 'code'], ['终端', 'terminal'], ['浏览器', 'globe'], ['源代码管理', 'branch'], ['代码智能', 'cube']]],
    ['其他', [['网络', 'globe'], ['用量与成本', 'chart'], ['隐私与遥测', 'lock'], ['诊断', 'activity'], ['高级', 'wrench'], ['关于', 'info']]],
  ];
  const nav = (on) => `<div class="fc" style="width:204px;background:var(--bg-chrome);border-right:1px solid var(--bd);flex:none;min-height:0">
    <div class="tb">${K.tl()}</div>
    <div class="f ai g6 s12 tx2" style="padding:0 10px 6px">${ib('chevL', 'sm')}<span class="m" style="color:var(--tx)">返回</span><span class="grow"></span>${kbd('⌘', ',')}</div>
    <div style="padding:0 8px 4px"><div class="inp fill">${i('search', 's12')}<span class="grow">搜索设置</span>${kbd('⌘', 'F')}</div></div>
    <div class="grow" style="overflow:hidden;padding:0 8px 8px">${NAV.map(([g, items]) => `<div class="navg" style="padding-left:8px">${g}</div>${items.map(([n, ic]) => `<div class="navi ${n === on ? 'on' : ''}">${i(ic, 's12')}<span>${n}</span></div>`).join('')}`).join('')}</div></div>`;

  const set = (t, d, ctl, mod) => `<div class="row-set"><div class="l"><b>${t}${mod ? '<span class="dot ac" title="已修改"></span>' : ''}</b><span>${d}</span></div>${ctl}</div>`;
  const grp = (t, rows) => `<div><div class="grp-t">${t}</div><div class="card" style="overflow:hidden">${rows}</div></div>`;
  const generalRows = {
    window: set('右侧面板默认打开', '启动时是否显示改动 / 文件 / 终端面板', tg(1)) + set('融合模式', '关闭时 IDE 是独立窗口(⇧⌘E 打开 / 回到 Agent);开启后在同一窗口原地切换', tg(0)) + set('关闭窗口时', '红色按钮的行为', seg(['隐藏', '退出'], 0)),
    input: set('发送键', '另一个键用于换行(Shift+Enter 始终换行)', seg(['Enter 发送', '⌘Enter 发送'], 0)) + set('排队与引导', '回合进行中发送新消息的默认行为', seg(['排队', '立即引导'], 0)),
    notice: set('任务完成时通知', '回合结束且 MiniCode 不在前台时发送系统通知', tg(1)) + set('菜单栏图标', '在系统菜单栏显示 MiniCode 图标', tg(1)) + set('自动检查更新', '启动时与每 6 小时检查一次', tg(1), 1),
    lang: set('界面语言', '目前仅提供简体中文', `<span class="sel">简体中文${i('chevUD', 's12 tx3')}</span>`),
  };
  const heading = (t, d) => `<div style="margin-bottom:12px"><div class="s16 b">${t}</div><div class="tx3">${d}</div></div>`;

  // A:左导航 + 居中内容
  const setA = () => `<div class="f" style="height:100%">${nav('通用')}
    <div class="grow" style="overflow:hidden;padding:20px 32px"><div style="max-width:660px;margin:0 auto" class="fc g12">
      ${heading('通用', '窗口行为、输入习惯与系统集成')}
      ${grp('窗口', generalRows.window)}${grp('输入', generalRows.input)}${grp('通知与菜单栏', generalRows.notice)}${grp('语言', generalRows.lang)}
    </div></div></div>`;

  // B:顶部分组页签 + 双列卡片
  const setB = () => `<div class="fc" style="height:100%">
    <div class="tb bd" style="background:var(--bg-chrome)">${K.tl()}${ib('chevL', 'sm')}<b>设置</b><div class="grow f jc"><div class="inp fill" style="width:360px">${i('search', 's12')}<span class="grow">搜索设置…</span>${kbd('⌘', 'F')}</div></div><span class="btn sm">恢复默认</span></div>
    <div class="f ai g16" style="height:34px;padding:0 20px;border-bottom:1px solid var(--bd);flex:none">${['客户端', 'Claude Code', '工作台', '其他'].map((t, k) => `<span style="height:34px;display:flex;align-items:center;position:relative;${k === 0 ? 'font-weight:600' : 'color:var(--tx2)'}">${t}${k === 0 ? '<i style="position:absolute;left:0;right:0;bottom:-1px;height:2px;background:var(--ac)"></i>' : ''}</span>`).join('')}</div>
    <div class="f ai g6" style="padding:10px 20px 0">${['通用', '外观', '对话', '快捷键'].map((t, k) => `<span class="chip line ${k === 0 ? 'on' : ''}" style="height:24px">${t}</span>`).join('')}</div>
    <div class="grow" style="padding:14px 20px;overflow:hidden;display:grid;grid-template-columns:1fr 1fr;gap:14px;align-content:start">
      ${grp('窗口', generalRows.window)}${grp('通知与菜单栏', generalRows.notice)}${grp('输入', generalRows.input)}${grp('语言', generalRows.lang)}
    </div></div>`;

  // C:单页滚动 + 目录
  const setC = () => `<div class="f" style="height:100%">
    <div class="fc" style="width:188px;background:var(--bg-chrome);border-right:1px solid var(--bd);flex:none;padding:0 8px"><div class="tb" style="padding:0">${K.tl()}</div>
      <div class="inp fill" style="margin:0 0 8px">${i('search', 's12')}<span class="grow">搜索设置</span></div>
      <div class="fc g2">${[['客户端', 1], ['Claude Code', 0], ['工作台', 0], ['其他', 0]].map(([t, on]) => `<div class="navi ${on ? 'on' : ''}"><span class="grow">${t}</span>${i('chevR', 's12 tx3')}</div>`).join('')}</div>
      <div class="navg" style="padding-left:8px;margin-top:8px">本页</div><div class="fc g2">${['窗口', '输入', '通知与菜单栏', '语言', '外观', '对话', '快捷键'].map((t, k) => `<div class="navi" style="${k === 1 ? 'color:var(--ac-tx);font-weight:500' : ''}">${k === 1 ? '<i style="width:2px;height:12px;background:var(--ac);border-radius:1px;margin-left:-6px"></i>' : ''}${t}</div>`).join('')}</div></div>
    <div class="grow" style="overflow:hidden;padding:18px 36px"><div style="max-width:720px;margin:0 auto" class="fc g12">
      <div class="f ai g8">${ib('chevL', 'sm')}<span class="s16 b">客户端</span><span class="grow"></span><span class="tx3 s11">已修改 1 项</span><span class="btn sm">只看已修改</span></div>
      ${grp('窗口', generalRows.window)}${grp('输入', generalRows.input)}${grp('通知与菜单栏', generalRows.notice)}
      <div class="f ai g8" style="margin-top:4px"><span class="s14 b">外观</span><span class="grow" style="height:1px;background:var(--bd)"></span></div>
      ${grp('主题', set('配色', '跟随系统或固定浅色 / 深色', seg(['跟随系统', '浅色', '深色'], 0)) + set('强调色', '用于选中、按钮与进度', '<span class="f g6">' + ['#E8541A', '#3B6FD8', '#2F9E5B', '#7A5AE0'].map((c, k) => `<i style="width:16px;height:16px;border-radius:50%;background:${c};box-shadow:${k === 0 ? '0 0 0 2px var(--bg-card),0 0 0 3.5px ' + c : 'none'}"></i>`).join('') + '</span>'))}
    </div></div></div>`;

  S.add({
    id: 'settings-general', group: 'settings', title: '设置 · 通用页的三种布局',
    desc: '设置共 22 个分类。现在是左导航 + 满宽内容,行高偏大、右侧留白多,也看不出哪些被改过。',
    notes: ['共性改进:行高 40 → 34;每项右侧控件对齐同一条竖线;被改动的项目标一个强调色小点,可「只看已修改」;设置搜索 ⌘F 在导航顶部。', '方案 B 利用宽屏做双列;方案 C 把所有分类放进一页长滚动,目录随滚动高亮,搜索时直接过滤整页。'],
    variants: [
      { label: '左导航 + 居中内容', tag: '推荐', rec: true, note: '沿用心智,内容限宽 660 更易读;改动最小。', render: setA },
      { label: '顶部分组页签 + 双列', note: '横向利用宽屏,一屏能看到更多项;导航在顶部,层级少一层。', render: setB },
      { label: '单页长滚动 + 目录', note: '像 Zed / Linear:所有设置在一页,目录 scrollspy,搜索即过滤;适合快速浏览,但分类间干扰多。', render: setC },
    ],
  });

  /* ───────── 外观 ───────── */
  const themeCard = (n, bg, bar, tx, on, ac) => `<div class="fc g4" style="width:120px"><div style="height:74px;border-radius:8px;background:${bg};border:${on ? '2px solid var(--ac)' : '1px solid var(--bd)'};padding:8px;display:flex;gap:5px;overflow:hidden"><div style="width:26px;border-radius:4px;background:${bar}"></div><div class="grow fc g4"><i style="height:6px;width:70%;border-radius:2px;background:${tx};opacity:.8"></i><i style="height:6px;width:50%;border-radius:2px;background:${tx};opacity:.4"></i><i style="height:12px;width:80%;border-radius:3px;background:${ac};margin-top:auto"></i></div></div><div class="f ai g4 s11">${on ? i('check', 's12 acc') : ''}<span class="${on ? 'm' : 'tx2'}">${n}</span></div></div>`;
  S.add({
    id: 'settings-appearance', group: 'settings', title: '设置 · 外观',
    desc: '主题、强调色、字号与工具调用详细程度,带实时预览。',
    notes: ['主题做成缩略卡片网格而非下拉;强调色色板 + 自定义;「会话流密度」三档右侧直接给一小段预览。'],
    variants: [{ label: '外观', tag: '推荐', rec: true, note: '', render: () => `<div class="f" style="height:100%">${nav('外观')}<div class="grow" style="overflow:hidden;padding:20px 32px"><div style="max-width:700px;margin:0 auto" class="fc g12">
      ${heading('外观', '主题、强调色与阅读密度')}
      ${grp('主题', `<div class="f g8 wrap" style="padding:12px">${themeCard('跟随系统', 'linear-gradient(90deg,#FAF9F7 50%,#1E1D1B 50%)', 'rgba(128,128,128,.25)', '#888', 0, '#E8541A')}${themeCard('浅色', '#FAF9F7', '#EEE9E1', '#23211D', 1, '#E8541A')}${themeCard('深色', '#1E1D1B', '#2B2A27', '#EEEBE5', 0, '#FF6B2C')}${themeCard('One Light', '#FAFAFA', '#EEF0F3', '#383A42', 0, '#4078F2')}${themeCard('One Dark', '#282C34', '#21252B', '#ABB2BF', 0, '#61AFEF')}</div>`)}
      ${grp('颜色与字号', set('强调色', '选中、按钮与进度的颜色', '<span class="f g6 ai">' + ['#E8541A', '#3B6FD8', '#2F9E5B', '#7A5AE0', '#C2410C'].map((c, k) => `<i style="width:16px;height:16px;border-radius:50%;background:${c};box-shadow:${k === 0 ? '0 0 0 2px var(--bg-card),0 0 0 3.5px ' + c : 'none'}"></i>`).join('') + '<span class="btn sm">自定义…</span></span>') + set('界面字号', '会话流与界面文字大小', '<span class="f ai g8"><span class="tx3 s11">小</span><span class="rel" style="width:120px;height:4px;border-radius:2px;background:var(--bg-active)"><i class="abs" style="left:0;width:40%;top:0;bottom:0;background:var(--ac);border-radius:2px"></i><i class="abs" style="left:calc(40% - 6px);top:-4px;width:12px;height:12px;border-radius:50%;background:#fff;box-shadow:0 0 0 1.5px var(--ac)"></i></span><span class="tx3 s11">大</span><span class="bdg">12 px</span></span>'))}
      ${grp('会话流密度', `<div class="f" style="padding:10px 12px;gap:12px">${[['紧凑', 0, ['读取 3 个文件 · 编辑 router.rs', '运行 cargo test ✓']], ['标准', 1, ['读取 3 个文件', '编辑 router.rs  +4 −1', '运行 cargo test ✓ 12 passed']], ['详细', 0, ['读取 router.rs', '读取 case.rs', '编辑 router.rs  +4 −1', '运行 cargo test ✓ 12 passed · 3.2s']]].map(([n, on, ls]) => `<div class="grow fc g4"><div class="f ai g4 ${on ? 'm' : 'tx2'}">${on ? i('check', 's12 acc') : ''}${n}</div><div style="border:${on ? '1.5px solid var(--ac)' : '1px solid var(--bd)'};border-radius:7px;padding:6px 8px;background:var(--bg-sunk);height:84px" class="fc g2">${ls.map(l => `<div class="s11 tx2 ell">${l}</div>`).join('')}</div></div>`).join('')}</div>`)}
    </div></div></div>` }],
  });

  /* ───────── 模型服务商 ───────── */
  const prov = (n, ic, st, tone, info, act) => `<div class="trow">${`<span class="avatar">${i(ic)}</span>`}<div class="grow"><div class="f ai g6"><b>${n}</b>${bdg(st, tone)}</div><div class="tx3 s11">${info}</div></div>${act}</div>`;
  S.add({
    id: 'settings-providers', group: 'settings', title: '设置 · 模型服务商',
    desc: '官方与第三方服务商、模型别名映射。',
    notes: ['服务商为列表行(头像 + 状态 + 一句话信息 + 操作),映射用紧凑表格;密钥只显示「已填 / 未填」,从不回显。'],
    variants: [{ label: '服务商与映射', tag: '推荐', rec: true, note: '', render: () => `<div class="f" style="height:100%">${nav('模型服务商')}<div class="grow" style="overflow:hidden;padding:20px 32px"><div style="max-width:700px;margin:0 auto" class="fc g12">
      <div class="f ai"><div class="grow">${heading('模型服务商', '为 Claude Code 切换后端并映射模型别名')}</div>${btn('添加服务商', 'pri', 'plus')}</div>
      ${grp('服务商', prov('Claude 官方', 'spark', '使用中', 'ok', '通过本机 claude 登录 · Max 5×', btn('账户', 'sm')) + '<div style="border-top:1px solid var(--bd-soft)"></div>' + prov('DeepSeek', 'cube', '未填 API Key', 'warn', 'https://api.deepseek.com/anthropic · 2 个模型', btn('填写密钥', 'sm soft')) + '<div style="border-top:1px solid var(--bd-soft)"></div>' + prov('智谱 GLM', 'cube', '已就绪', '', 'https://open.bigmodel.cn/api/anthropic · 回合进行中,结束后可切换', `${btn('切换', 'sm')}${ib('more', 'sm')}`))}
      ${grp('模型别名映射(当前:Claude 官方)', `<div class="tbl-h f"><span style="width:140px">别名</span><span class="grow">实际模型</span><span style="width:70px">思考</span></div>${[['Opus', 'claude-opus-4-5', '支持'], ['Sonnet', 'claude-sonnet-5-5', '支持'], ['Haiku', 'claude-haiku-4-5', '不支持']].map(([a, m, t]) => `<div class="trow"><span style="width:140px" class="m">${a}</span><span class="grow mono">${m}</span><span style="width:70px" class="tx2">${t}</span>${ib('edit', 'sm')}</div>`).join('')}`)}
    </div></div></div>` }],
  });

  /* ───────── 权限规则 ───────── */
  S.add({
    id: 'settings-permissions', group: 'settings', title: '设置 · 权限规则',
    desc: '用户 / 项目 / 本地三层的允许、询问、拒绝规则。',
    notes: ['三层来源用页签,规则按动作分组着色(允许绿 / 询问琥珀 / 拒绝红),每条规则一行可直接启停;来源只读的规则显示锁。'],
    variants: [{ label: '规则表', tag: '推荐', rec: true, note: '', render: () => `<div class="f" style="height:100%">${nav('权限')}<div class="grow" style="overflow:hidden;padding:20px 32px"><div style="max-width:700px;margin:0 auto" class="fc g12">
      <div class="f ai"><div class="grow">${heading('权限', '决定 Claude 哪些操作需要你确认')}</div>${seg(['用户', '项目', '本地'], 1)}</div>
      <div class="f ai g8">${i('shield', 's12 tx2')}<span class="tx2">默认模式</span>${seg(['逐项确认', '接受编辑', '自动', '计划'], 1, 'sm')}<span class="grow"></span><div class="inp fill" style="width:200px">${i('search', 's12')}<span>过滤规则…</span></div>${btn('添加规则', 'pri sm', 'plus')}</div>
      ${[['允许', 'ok', [['Bash(cargo test:*)', '项目'], ['Bash(git diff:*)', '用户'], ['Read(./src/**)', '用户']]], ['询问', 'warn', [['Bash(git push:*)', '项目'], ['Edit(./.github/**)', '本地']]], ['拒绝', 'err', [['Read(./.env*)', '用户', 1], ['Bash(rm -rf:*)', '用户', 1]]]].map(([g, t, rs]) => grp(`<span class="${t === 'ok' ? 'okc' : t === 'warn' ? 'warnc' : 'errc'}">● ${g}</span> · ${rs.length}`, rs.map(([r, s, lock]) => `<div class="trow"><span class="mono grow">${r}</span>${bdg(s, '')}${lock ? i('lock', 's12 tx3') : ib('trash', 'sm')}</div>`).join(''))).join('')}
    </div></div></div>` }],
  });

  /* ───────── 用量与成本 ───────── */
  const days = [22, 38, 30, 55, 72, 44, 61, 35, 48, 66, 80, 52, 40, 58];
  S.add({
    id: 'settings-usage', group: 'settings', title: '设置 · 用量与成本',
    desc: '扫描 ~/.claude/projects 得到的 token 与费用统计。',
    notes: ['顶部四个关键数字;主图是按天的堆叠柱(输入 / 输出 / 缓存);下方按模型、按项目两张紧凑表。'],
    variants: [{ label: '用量总览', tag: '推荐', rec: true, note: '', render: () => `<div class="f" style="height:100%">${nav('用量与成本')}<div class="grow" style="overflow:hidden;padding:20px 32px"><div style="max-width:720px;margin:0 auto" class="fc g12">
      <div class="f ai"><div class="grow">${heading('用量与成本', '本机会话统计,不上传')}</div>${seg(['7 天', '30 天', '全部'], 1)}${ib('refresh')}</div>
      <div class="f g10">${[['总花费', '$38.42', '+12% 较上期'], ['总 token', '8.6M', '输入 6.1M · 输出 0.4M'], ['缓存命中', '91%', '节省约 $52'], ['会话数', '47', '日均 3.4']].map(([a, b, c]) => `<div class="card pad grow"><div class="tx3 s11">${a}</div><div class="s20 b" style="margin:2px 0">${b}</div><div class="tx3 s10">${c}</div></div>`).join('')}</div>
      <div class="card pad"><div class="f ai jb" style="margin-bottom:8px"><b>每日 token</b><span class="f ai g10 s11 tx3"><span class="f ai g4"><i style="width:8px;height:8px;border-radius:2px;background:var(--ac)"></i>输入</span><span class="f ai g4"><i style="width:8px;height:8px;border-radius:2px;background:var(--info)"></i>输出</span><span class="f ai g4"><i style="width:8px;height:8px;border-radius:2px;background:var(--bd-strong)"></i>缓存</span></span></div>
        <div class="f g4" style="align-items:flex-end;height:110px">${days.map(h => `<div class="grow fc" style="justify-content:flex-end;height:100%;gap:1px"><i style="height:${h * 0.3}%;background:var(--bd-strong);border-radius:2px 2px 0 0"></i><i style="height:${h * 0.12}%;background:var(--info)"></i><i style="height:${h * 0.35}%;background:var(--ac)"></i></div>`).join('')}</div>
        <div class="f jb tx3 s10" style="margin-top:3px"><span>11-02</span><span>11-08</span><span>11-15</span></div></div>
      ${grp('按模型', `<div class="tbl-h f"><span class="grow">模型</span><span style="width:90px;text-align:right">token</span><span style="width:70px;text-align:right">费用</span><span style="width:120px"></span></div>${[['Sonnet 5.5', '6.4M', '$24.10', 72], ['Opus 4.5', '1.8M', '$13.20', 28], ['Haiku 4.5', '0.4M', '$1.12', 8]].map(([n, t, c, p]) => `<div class="trow"><span class="grow m">${n}</span><span style="width:90px;text-align:right" class="mono">${t}</span><span style="width:70px;text-align:right" class="mono">${c}</span><span style="width:120px">${meter(p)}</span></div>`).join('')}`)}
    </div></div></div>` }],
  });
})();
