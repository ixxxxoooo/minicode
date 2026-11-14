/* 其余页面:自定义、画布、自动化、浏览器、命令面板、选择器、通知与对话框 */
(function () {
  const { i, chip, bdg, cnt, kbd, tg, seg, dot, ib, btn, logo, meter, hl } = K;
  const { C } = window;

  // 页面外壳:沿用 Agent 侧栏,右侧是页面
  const shell = (nav, body, head) => `<div class="f" style="height:100%">${C.sbA(-1, { nav })}<div class="grow fc" style="min-width:0">${head}<div class="grow f" style="min-height:0">${body}</div></div></div>`;
  const pageHead = (ic, title, crumb, right = '') => `<div class="page-h">${ib('chevL', 'sm')}${i(ic, 's14 tx2')}<b class="s13">${title}</b>${crumb ? `<span class="tx3">/ ${crumb}</span>` : ''}<span class="grow"></span>${right}${seg(['Agent', 'IDE'], 0)}${ib('panelR')}</div>`;

  /* ───────── 自定义 ───────── */
  S.group('pages', '其他页面');
  const cats = [['plug', '插件', 3], ['server', 'MCP', 5], ['bolt', '技能', 12], ['slash', '命令', 8], ['bot', '子 Agent', 4], ['fileText', '规则', 2], ['code', 'Hooks', 3]];
  const plug = (n, v, ver, scope, desc, on, tags = '') => `<div class="trow" style="align-items:flex-start;padding:8px 12px"><span class="avatar">${i('plug')}</span><div class="grow"><div class="f ai g6"><b>${n}</b><span class="bdg">${ver}</span>${bdg(scope, 'ac')}<span class="tx3 s11">${v}</span></div><div class="tx2" style="margin-top:1px">${desc}</div><div class="f g4" style="margin-top:4px">${tags}</div></div><div class="f ai g6" style="margin-top:2px">${btn('更新', 'sm')}${tg(on)}${ib('more', 'sm')}</div></div>`;
  const custA = () => shell(3, `
    <div class="fc" style="width:176px;border-right:1px solid var(--bd);padding:8px;gap:1px;flex:none">${cats.map(([ic, n, c], k) => `<div class="navi ${k === 0 ? 'on' : ''}">${i(ic, 's12')}<span class="grow">${n}</span><span class="tx3 s10">${c}</span></div>`).join('')}<div class="sep" style="margin:8px 0"></div><div class="navi">${i('refresh', 's12')}重新读取</div><div class="navi">${i('book', 's12')}文档</div></div>
    <div class="grow fc" style="min-width:0;border-right:1px solid var(--bd)">
      <div class="f ai g8" style="height:40px;padding:0 12px;border-bottom:1px solid var(--bd-soft)"><div class="inp fill grow">${i('search', 's12')}<span>搜索插件…</span></div>${seg(['全部', '用户', '项目', '本地'], 0, 'sm')}${btn('安装', 'pri', 'plus')}</div>
      <div class="grow" style="overflow:hidden"><div class="cap">已安装 3</div>
        ${plug('gopls-lsp', 'Go 语言服务', 'v1.0.0', '用户', 'claude-plugins-official', 0)}
        ${plug('rust-analyzer-lsp', 'Rust 语言服务', 'v1.0.0', '用户', '跳转、补全与诊断', 1).replace('trow"', 'trow" style="background:var(--bg-hover);align-items:flex-start;padding:8px 12px"')}
        ${plug('statusline-pro', 'demo-market', 'v1.3.2', '用户', '状态栏显示上下文用量、token 与会话时长', 1, '<span class="bdg">2 个命令</span>')}
        <div class="cap" style="margin-top:6px">插件市场 2</div>
        <div class="trow"><span class="avatar sm">${i('globe', 's12')}</span><div class="grow"><b>claude-plugins-official</b><div class="tx3 mono s11">anthropics/claude-plugins-official</div></div><span class="tx3 s11">11-01 更新</span>${btn('更新', 'sm')}</div>
      </div></div>
    <div class="fc" style="width:300px;flex:none;padding:12px;gap:10px"><div class="f ai g8"><span class="avatar">${i('plug')}</span><div class="grow"><b class="s13">rust-analyzer-lsp</b><div class="tx3 s11">v1.0.0 · claude-plugins-official</div></div></div>
      <div class="f g6">${btn('停用', 'sm')}${btn('更新', 'sm')}${btn('卸载', 'sm danger')}</div>
      <div class="tx2" style="line-height:1.6">Rust 语言服务:跳转、补全与诊断。启用后 Claude 可通过 LSP 工具查询符号定义与引用。</div>
      <div class="grp-t" style="margin-top:4px">包含</div><div class="fc g4">${[['bolt', 'LSP 服务器 · rust-analyzer'], ['slash', '0 个命令'], ['code', '0 个 Hooks']].map(([ic, t]) => `<div class="f ai g6 tx2">${i(ic, 's12')}${t}</div>`).join('')}</div>
      <div class="grp-t" style="margin-top:4px">作用范围</div>${seg(['用户', '项目', '本地'], 0, 'sm')}
    </div>`, pageHead('plug', '自定义', '插件'));

  const mkt = (n, d, st, tone, ic) => `<div class="card pad fc g6"><div class="f ai g8"><span class="avatar">${i(ic)}</span><div class="grow"><b>${n}</b><div class="tx3 s11">anthropics · ${st === '已安装' ? 'v1.0.0' : '未安装'}</div></div>${bdg(st, tone)}</div><div class="tx2" style="line-height:1.5;height:36px;overflow:hidden">${d}</div><div class="f ai g6">${st === '已安装' ? btn('管理', 'sm') : btn('安装', 'sm soft')}<span class="grow"></span><span class="tx3 s10">${i('download', 's12')} 12.4k</span></div></div>`;
  const custB = () => shell(3, `<div class="grow fc" style="padding:14px 20px;gap:12px;overflow:hidden;min-width:0">
      <div class="f ai g10"><div class="inp grow" style="height:30px">${i('search', 's12')}<span>搜索插件、技能、MCP、命令…</span></div>${btn('刷新', '', 'refresh')}</div>
      <div class="f ai g6">${cats.map(([ic, n, c], k) => `<span class="chip line ${k === 0 ? 'on' : ''}" style="height:26px">${i(ic, 's12')}${n}<span class="tx3 s10">${c}</span></span>`).join('')}<span class="grow"></span>${seg(['已安装', '市场'], 1, 'sm')}</div>
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:10px">${[['rust-analyzer-lsp', 'Rust 语言服务:跳转、补全与诊断,让 Claude 能查符号与引用。', '已安装', 'ok', 'cube'], ['gopls-lsp', 'Go 语言服务:跳转、补全与诊断。', '已安装', 'ok', 'cube'], ['statusline-pro', '状态栏显示上下文用量、token 与会话时长。', '已安装', 'ok', 'activity'], ['code-review', '多维度代码审查子 Agent,附一键修复建议。', '未安装', '', 'eye'], ['commit-helper', '按 Conventional Commits 生成提交说明与变更日志。', '未安装', '', 'commit'], ['playwright-mcp', '浏览器自动化 MCP,支持截图与表单。', '未安装', '', 'globe']].map(a => mkt(...a)).join('')}</div>
    </div>`, pageHead('plug', '自定义', '插件 · 市场'));
  S.add({
    id: 'customize', group: 'pages', title: '自定义(插件 / MCP / 技能 / 命令 / 子 Agent / 规则 / Hooks)',
    desc: '现在顶部是药丸标签 + 满宽表单,安装输入框占了首屏,也没有详情视图。',
    notes: ['方案 A 做成 master-detail:左类别(带计数)、中列表、右详情抽屉;安装变成顶部的一个按钮(弹出安装表单),不再占首屏。', '方案 B 偏「市场」:卡片网格 + 已安装 / 市场切换,适合发现新东西,管理已装项略慢。'],
    variants: [
      { label: '三栏 master-detail', tag: '推荐', rec: true, note: '管理优先,所有类别共用一套骨架,详情就地编辑。', render: custA },
      { label: '市场卡片网格', note: '发现优先,卡片带安装量与状态。', render: custB },
    ],
  });

  /* ───────── 画布 ───────── */
  const kpi = (a, b, c, tone) => `<div class="card pad grow"><div class="tx3 s11">${a}</div><div class="s20 b" style="margin:1px 0">${b}</div><div class="s11 ${tone}">${c}</div></div>`;
  const bars = (arr, c = 'var(--info)') => `<div class="f g12" style="align-items:flex-end;height:96px;padding:0 18px">${arr.map(([l, h]) => `<div class="grow fc ai g4" style="justify-content:flex-end;height:100%"><i style="width:34px;height:${h}%;background:${c};border-radius:3px 3px 0 0"></i><span class="tx3 s10">${l}</span></div>`).join('')}</div>`;
  const canvasDoc = (wide) => `<div style="max-width:${wide ? 760 : 680}px;margin:0 auto;padding:18px 20px" class="fc g12">
      <div><div class="s20 b">Atlas 第三季度质量周报</div><div class="tx2" style="margin-top:3px;line-height:1.6">本季度重点是<b>路由性能</b>与<b>指标覆盖</b>。下方数据由 Agent 汇总,可在源码视图查看 JSON。</div></div>
      <div class="f g10">${kpi('合并 PR', '42', '+8 较上季', 'okc')}${kpi('平均响应', '118 ms', '−23 ms', 'okc')}${kpi('未关闭缺陷', '7', '+2', 'warnc')}${kpi('测试覆盖率', '86%', '目标 90%', 'tx3')}</div>
      <div class="card pad"><div class="f ai g6" style="margin-bottom:8px">${i('chart', 's12 tx2')}<b>每月合并 PR</b><span class="grow"></span><span class="f ai g4 s11 tx3"><i style="width:8px;height:8px;background:var(--info);border-radius:2px"></i>功能 <i style="width:8px;height:8px;background:var(--ac);border-radius:2px;margin-left:6px"></i>修复</span></div>
        <div class="f g12" style="align-items:flex-end;height:96px;padding:0 18px">${[['7 月', 9, 6], ['8 月', 14, 5], ['9 月', 11, 9]].map(([l, a, b]) => `<div class="grow fc ai g4" style="justify-content:flex-end;height:100%"><div class="f g4" style="align-items:flex-end;height:calc(100% - 16px)"><i style="width:24px;height:${a * 6}%;background:var(--info);border-radius:3px 3px 0 0"></i><i style="width:24px;height:${b * 6}%;background:var(--ac);border-radius:3px 3px 0 0"></i></div><span class="tx3 s10">${l}</span></div>`).join('')}</div></div>
      <div class="card pad"><div class="f ai g6" style="margin-bottom:6px">${i('trend', 's12 tx2')}<b>响应时间 (ms)</b><span class="grow"></span><span class="tx3 s11">P50 / P95</span></div>
        <svg viewBox="0 0 600 90" style="width:100%;height:90px"><g stroke="var(--bd)" stroke-width="1"><path d="M0 20H600M0 50H600M0 80H600"/></g><path d="M0 22 L200 30 L400 38 L600 44" fill="none" stroke="var(--info)" stroke-width="2"/><path d="M0 62 L200 66 L400 70 L600 72" fill="none" stroke="var(--ac)" stroke-width="2"/></svg></div></div>`;
  const canvasA = () => shell(1, `<div class="fc" style="width:200px;border-right:1px solid var(--bd);flex:none"><div class="f ai jb" style="height:32px;padding:0 10px"><b class="tx2 s11">画布</b>${ib('refresh', 'sm')}</div>
      <div class="mi on" style="margin:0 6px;height:44px;flex-direction:column;align-items:flex-start;justify-content:center;gap:0;background:var(--ac-soft)"><b class="ell">Atlas 第三季度质量周报</b><span class="tx3 s10">11-15 02:47 · 21 个块</span></div>
      <div class="mi" style="margin:0 6px;height:44px;flex-direction:column;align-items:flex-start;justify-content:center;gap:0"><b class="ell">迁移计划看板</b><span class="tx3 s10">11-12 18:20 · 14 个块</span></div></div>
    <div class="grow fc" style="min-width:0"><div class="f ai g8" style="height:36px;padding:0 14px;border-bottom:1px solid var(--bd)">${i('canvas', 's14 acc')}<b>Atlas 第三季度质量周报</b><span class="bdg">v2</span><span class="bdg">21 个块</span><span class="grow"></span>${seg(['渲染', '源码'], 0, 'sm')}${ib('ext')}${ib('trash')}</div>
      <div class="grow" style="overflow:hidden">${canvasDoc(false)}</div></div>`, pageHead('canvas', '画布', ''));
  const canvasB = () => shell(1, `<div class="grow fc" style="min-width:0"><div class="f ai g8" style="height:36px;padding:0 14px;border-bottom:1px solid var(--bd)">${chip('Atlas 第三季度质量周报', 'canvas', 'line', true)}<span class="bdg">v2</span><span class="grow"></span><span class="tx3 s11">自动保存 · 刚刚</span>${seg(['渲染', '源码', '对比 v1'], 0, 'sm')}${btn('分享', 'sm', 'ext')}</div>
      <div class="f grow" style="min-height:0"><div class="grow" style="overflow:hidden">${canvasDoc(true)}</div>
        <div class="fc" style="width:230px;border-left:1px solid var(--bd);padding:10px;gap:12px;flex:none"><div><div class="grp-t">大纲</div>${['概览指标', '每月合并 PR', '响应时间', '架构图', '附录'].map((t, k) => `<div class="mi ${k === 1 ? 'on' : ''}" style="height:24px">${i('chevR', 's12 tx3')}${t}</div>`).join('')}</div>
          <div><div class="grp-t">版本</div>${[['v2', '当前 · 刚刚'], ['v1', '昨天 18:20']].map(([v, t], k) => `<div class="f ai g8" style="height:26px"><span style="width:8px;height:8px;border-radius:50%;background:${k === 0 ? 'var(--ac)' : 'var(--bd-strong)'}"></span><b>${v}</b><span class="tx3 s11">${t}</span></div>`).join('')}</div>
          <div><div class="grp-t">让 Claude 修改</div><div class="cmp"><div class="in" style="min-height:44px;padding:6px 8px">把覆盖率改成折线,加一块架构图…</div></div></div></div></div></div>`, pageHead('canvas', '画布', ''));
  S.add({
    id: 'canvas', group: 'pages', title: '画布',
    desc: 'Agent 生成的可视化文档(指标、图表、Mermaid)。现在页头与其它页不一致,也缺少版本与大纲。',
    notes: ['统一页头:返回 / 图标 / 标题 / 视图切换(渲染 | 源码)/ 全局动作;方案 B 增加大纲、版本时间线与「让 Claude 修改」输入框,把画布变成可迭代的文档。'],
    variants: [
      { label: '列表 + 文档', note: '沿用现有结构,统一页头。', render: canvasA },
      { label: '全幅文档 + 大纲 / 版本', tag: '推荐', rec: true, note: '画布切换放进顶部下拉;右侧大纲 + 版本 + 就地让 Claude 迭代。', render: canvasB },
    ],
  });

  /* ───────── 自动化 ───────── */
  const runRow = (st, tone, t, d, s, c) => `<div class="trow" style="align-items:flex-start;padding:7px 12px">${bdg(st, tone)}<div class="grow"><div class="f ai g8 tx3 s11 mono"><span>${t}</span><span>${d}</span><span>${c}</span></div><div style="margin-top:1px">${s}</div></div></div>`;
  const autoA = () => shell(0, `
    <div class="fc" style="width:230px;border-right:1px solid var(--bd);flex:none"><div class="f ai jb" style="height:32px;padding:0 10px"><b class="tx2 s11">自动化 3</b>${ib('plus', 'sm')}</div>
      ${[['每日 TODO 巡检', '每天 09:00 · 下次 11-15 09:00', 1, 1], ['每周依赖更新检查', '每周一 10:30 · 下次 11-16', 1, 0], ['每小时构建健康检查', '每 1 小时 · 已停用', 0, 0]].map(([n, s, on, sel]) => `<div class="f ai g8" style="padding:6px 10px;margin:0 6px;border-radius:7px;${sel ? 'background:var(--ac-soft)' : ''}"><div class="grow" style="min-width:0"><b class="ell" style="display:block">${n}</b><span class="tx3 s10 ell" style="display:block">${s}</span></div>${tg(on)}</div>`).join('')}</div>
    <div class="grow fc" style="min-width:0;padding:16px 24px;gap:12px;overflow:hidden"><div class="f ai g8">${i('timer', 's16 acc')}<b class="s14">每日 TODO 巡检</b>${bdg('已启用', 'ok')}<span class="grow"></span>${btn('立即运行', '', 'play')}${btn('保存', 'pri')}</div>
      <div class="f g10"><div class="fc g4 grow"><span class="tx3 s11">名称</span><div class="inp"><span class="v">每日 TODO 巡检</span></div></div><div class="fc g4" style="width:200px"><span class="tx3 s11">项目</span><span class="sel" style="height:26px;justify-content:space-between">router-demo${i('chevUD', 's12 tx3')}</span></div></div>
      <div class="fc g4"><span class="tx3 s11">提示词</span><div class="inp" style="height:64px;align-items:flex-start;padding-top:6px"><span class="v">检查今天新增的 TODO / FIXME 注释,按模块汇总成清单</span></div></div>
      <div class="f g16"><div class="fc g4"><span class="tx3 s11">运行计划</span><div class="f ai g8">${seg(['间隔', '每天', '每周'], 1, 'sm')}<span class="sel mono">09 : 00${i('chevUD', 's12 tx3')}</span><span class="tx3 s11">下次 11-15 09:00</span></div></div>
        <div class="fc g4"><span class="tx3 s11">权限(无人值守时无法询问)</span>${seg(['接受编辑', '只读不问', '绕过权限'], 0, 'sm')}</div></div>
      <div class="f ai g8" style="margin-top:4px"><b>运行记录</b><span class="grow"></span><span class="tx3 s11">近 3 次</span></div>
      <div class="card" style="overflow:hidden">${runRow('成功', 'ok', '11-14 21:47', '48 秒', '发现 6 处新增 TODO:Router 3 处、Metrics 2 处、CLI 1 处', '$0.041')}<div style="border-top:1px solid var(--bd-soft)"></div>${runRow('成功', 'ok', '11-13 21:47', '48 秒', '今天没有新增 TODO。', '$0.012')}<div style="border-top:1px solid var(--bd-soft)"></div>${runRow('失败', 'err', '11-12 21:47', '48 秒', '无法启动 claude:鉴权已过期,请重新登录。', '')}</div></div>`, pageHead('timer', '自动化', ''));
  const autoB = () => {
    const hrs = ['09', '10', '11', '12', '13', '14', '15', '16'];
    const dayN = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
    return shell(0, `<div class="grow fc" style="min-width:0;padding:14px 20px;gap:10px;overflow:hidden">
      <div class="f ai g8"><b class="s14">本周计划</b><span class="tx3">11-10 — 11-16</span>${ib('chevL', 'sm')}${ib('chevR', 'sm')}<span class="grow"></span>${seg(['周', '列表'], 0, 'sm')}${btn('新建自动化', 'pri', 'plus')}</div>
      <div class="card" style="overflow:hidden"><div style="display:grid;grid-template-columns:44px repeat(7,1fr)"><div></div>${dayN.map((d, k) => `<div class="tx2 s11 ai" style="text-align:center;padding:4px 0;border-left:1px solid var(--bd-soft);${k === 4 ? 'color:var(--ac-tx);font-weight:600' : ''}">${d}</div>`).join('')}
        ${hrs.map((h, r) => `<div class="tx3 s10" style="padding:4px 6px;text-align:right;border-top:1px solid var(--bd-soft);height:44px">${h}:00</div>${dayN.map((d, c) => { const e = (r === 0 && c !== 5 && c !== 6) ? '<div class="s10" style="margin:2px;padding:2px 4px;border-radius:4px;background:var(--ac-soft);color:var(--ac-tx);border-left:2px solid var(--ac)">TODO 巡检</div>' : (r === 1 && c === 0) ? '<div class="s10" style="margin:2px;padding:2px 4px;border-radius:4px;background:var(--info-soft);color:var(--info);border-left:2px solid var(--info)">依赖检查</div>' : ''; return `<div style="border-top:1px solid var(--bd-soft);border-left:1px solid var(--bd-soft);height:44px">${e}</div>`; }).join('')}`).join('')}</div></div>
      <div class="f g10">${[['每日 TODO 巡检', '每天 09:00', 1, 'ok', '上次成功 · $0.041'], ['每周依赖更新检查', '周一 10:30', 1, 'ok', '上次成功 · $0.012'], ['每小时构建健康检查', '每 1 小时', 0, '', '已停用']].map(([n, s, on, t, l]) => `<div class="card pad grow fc g4"><div class="f ai g6"><b class="grow ell">${n}</b>${tg(on)}</div><div class="tx2 s11">${s}</div><div class="f ai g6 s11 tx3">${dot(t)}${l}</div></div>`).join('')}</div></div>
      <div class="fc" style="width:300px;border-left:1px solid var(--bd);flex:none;padding:12px;gap:10px"><div class="f ai g6"><b>编辑 · 每日 TODO 巡检</b><span class="grow"></span>${ib('x', 'sm')}</div><div class="inp"><span class="v">检查今天新增的 TODO / FIXME,按模块汇总</span></div>
        <div class="f ai g6">${seg(['每天', '每周', '间隔'], 0, 'sm')}<span class="sel mono">09:00</span></div>${seg(['接受编辑', '只读', '绕过'], 0, 'sm')}<div class="f g6" style="margin-top:4px">${btn('保存', 'pri')}${btn('立即运行', '', 'play')}${btn('删除', 'ghost danger')}</div>
        <div class="grp-t" style="margin-top:6px">最近运行</div>${[['11-14', '成功', 'ok'], ['11-13', '成功', 'ok'], ['11-12', '失败', 'err']].map(([d, s, t]) => `<div class="f ai g8 s11" style="height:22px">${bdg(s, t)}<span class="mono tx2">${d} 21:47</span><span class="tx3 grow">48 秒</span></div>`).join('')}</div>`, pageHead('timer', '自动化', ''));
  };
  S.add({
    id: 'automation', group: 'pages', title: '自动化',
    desc: '定时 / 周期运行的 Claude 任务。现在是「列表 + 长表单」,看不出一周里什么时候会跑。',
    notes: ['方案 B 加入周视图:一眼看到本周哪些任务会在什么时间触发,冲突与空档都直观;编辑在右侧抽屉,运行记录紧跟其后。'],
    variants: [
      { label: '列表 + 表单(整理版)', note: '沿用现有结构,表单更紧凑,项目 / 权限用下拉与分段。', render: autoA },
      { label: '周历 + 编辑抽屉', tag: '推荐', rec: true, note: '时间维度可视化,新建 / 编辑不离开当前视图。', render: autoB },
    ],
  });

  /* ───────── 浏览器 ───────── */
  const article = `<div style="max-width:640px;margin:0 auto" class="fc g8"><div class="tx3 s11">阅读模式 · 已移除脚本与样式 · example.com</div><div class="s20 b">Example Domain</div><div class="md" style="color:var(--tx2)"><p>This domain is for use in illustrative examples in documents. You may use this domain in literature without prior coordination or asking for permission.</p><p><span class="acc">More information…</span></p></div></div>`;
  const browserA = () => shell(2, `<div class="grow fc" style="min-width:0"><div class="f ai g2" style="height:32px;padding:4px 8px 0;background:var(--bg-chrome);border-bottom:1px solid var(--bd);align-items:flex-end">${['Example Domain|1', 'MDN · Fetch API|0', 'Rust 文档|0'].map(r => { const [t, on] = r.split('|'); return `<div class="f ai g6" style="height:28px;padding:0 10px;border-radius:7px 7px 0 0;width:160px;${on === '1' ? 'background:var(--bg-main);box-shadow:0 0 0 1px var(--bd);border-bottom:none' : 'color:var(--tx2)'}">${i('globe', 's12 tx3')}<span class="ell grow">${t}</span>${ib('x', 'sm')}</div>`; }).join('')}<span class="ib" style="margin-bottom:2px">${i('plus')}</span></div>
      <div class="f ai g6" style="height:36px;padding:0 10px;border-bottom:1px solid var(--bd)">${ib('chevL', 'sm')}${ib('chevR', 'sm')}${ib('refresh', 'sm')}<div class="inp fill grow" style="height:26px">${i('lock', 's12 okc')}<span class="v">https://example.com/</span></div>${seg(['阅读', '原页'], 0, 'sm')}${btn('问 Claude', 'soft', 'spark')}${btn('摘要', '', 'list')}${ib('ext', '', '系统浏览器')}</div>
      <div class="grow" style="padding:26px 24px;overflow:hidden">${article}</div></div>`, pageHead('globe', '浏览器', ''));
  const browserB = () => shell(2, `<div class="grow fc" style="min-width:0"><div class="f ai g6" style="height:36px;padding:0 10px;border-bottom:1px solid var(--bd)">${ib('chevL', 'sm')}${ib('chevR', 'sm')}${ib('refresh', 'sm')}<div class="inp fill grow" style="height:26px">${i('lock', 's12 okc')}<span class="v">https://example.com/</span></div>${seg(['阅读', '原页'], 0, 'sm')}${ib('ext')}</div>
      <div class="grow" style="padding:26px 24px;overflow:hidden">${article}</div></div>
    <div class="fc" style="width:330px;border-left:1px solid var(--bd);flex:none"><div class="tabs" style="height:32px"><div class="tab on">${i('bot', 's12')}与此页对话</div><div class="tab">${i('list', 's12')}摘要</div><div class="tab">${i('link', 's12')}链接 4</div></div>
      <div class="grow fc g8" style="padding:10px"><div class="msg-user">这个页面讲什么?</div><div class="md"><p>这是 IANA 保留的示例域名页面,用于文档示例,<b>不是</b>真实服务。可在示例中直接使用,无需申请。</p></div><span class="grow"></span><div class="f g4">${chip('提取要点', '', 'line')}${chip('翻译成中文', '', 'line')}${chip('保存到画布', '', 'line')}</div>${C.cmpB({ narrow: true, hint: false })}</div></div>`, pageHead('globe', '浏览器', ''));
  S.add({
    id: 'browser', group: 'pages', title: '内置浏览器(阅读模式)',
    desc: '抓取网页正文的阅读模式浏览器。现在像一个独立的小浏览器,与 Claude 之间没有联动。',
    notes: ['方案 A:多标签 + 「问 Claude / 摘要」动作条;方案 B:右侧直接开一栏「与此页对话」,把整页当上下文。'],
    variants: [
      { label: '多标签 + 动作条', note: '轻量,保持浏览器的使用习惯。', render: browserA },
      { label: '页面 + 对话分栏', tag: '推荐', rec: true, note: '页面和 Claude 并排,提问时自动带上正文;可一键存入画布。', render: browserB },
    ],
  });

  /* ───────── 命令面板与选择器 ───────── */
  S.group('global', '全局浮层');
  const palItems = [['命令', [['plus', '新对话', '⌘N', 1], ['palette', '主题:切换…', '当前:跟随浅色', 0], ['gear', '打开设置', '⌘,', 0], ['shield', '切换权限模式', '⇧Tab', 0], ['code', '切换 IDE 模式', '⇧⌘E', 0]]], ['会话', [['bot', '完整修复一个 Bug:路由大小写匹配', '运行中', 0], ['bot', '权限请求:修改配置与执行命令', '待许可', 0]]], ['最近文件', [['file', 'router.rs', 'src/', 0]]]];
  const palListItem = ([ic, t, r, hl_]) => `<div class="mi ${hl_ ? 'hl' : ''}" style="height:28px">${i(ic, 's12 ' + (hl_ ? '' : 'tx2'))}<span class="grow ell">${t}</span><span class="${hl_ ? '' : 'tx3'} s11 mono">${r}</span></div>`;
  const behind = () => `<div class="f" style="height:100%">${C.sbA(0)}<div class="grow fc">${C.head()}<div class="grow" style="overflow:hidden"><div class="col">${C.turn('A')}</div></div><div class="col" style="padding-bottom:10px">${C.cmpA()}</div></div>${C.rp('changes')}</div>`;
  const palA = () => `${behind()}<div class="scrim"></div><div class="abs pop" style="left:340px;top:56px;width:600px;padding:0"><div class="f ai g8" style="height:42px;padding:0 14px;border-bottom:1px solid var(--bd)">${i('search', 's14 tx2')}<span class="grow tx3 s13">搜索命令、会话、项目、文件…</span>${kbd('Esc')}</div>
      <div class="f ai g4" style="padding:6px 10px 2px">${['全部', '会话', '项目', '文件', '命令'].map((t, k) => `<span class="chip ${k === 0 ? 'on' : ''}" style="height:22px">${t}</span>`).join('')}</div>
      <div style="padding:2px 6px 6px">${palItems.map(([g, its]) => `<div class="cap">${g}</div>${its.map(palListItem).join('')}`).join('')}</div>
      <div class="f ai g12 s10 tx3" style="height:26px;padding:0 12px;border-top:1px solid var(--bd)"><span>${kbd('↑↓')} 选择</span><span>${kbd('⏎')} 执行</span><span>${kbd('Tab')} 切换分段</span><span class="grow"></span><span>${kbd('@')} 主题 ${kbd('>')} 命令 ${kbd('#')} 符号</span></div></div>`;
  const palB = () => `${behind()}<div class="scrim"></div><div class="abs pop" style="left:240px;top:60px;width:800px;padding:0"><div class="f ai g8" style="height:42px;padding:0 14px;border-bottom:1px solid var(--bd)">${i('search', 's14 tx2')}<span class="grow s13 v" style="color:var(--tx)">路由</span>${kbd('Esc')}</div>
      <div class="f" style="height:360px"><div class="fc" style="width:360px;border-right:1px solid var(--bd);padding:4px 6px;overflow:hidden"><div class="cap">会话</div>${palListItem(['bot', '完整修复一个 Bug:路由大小写匹配', '运行中', 1])}${palListItem(['bot', '文件读写:梳理并整理路由层', '1 小时', 0])}<div class="cap">文件</div>${palListItem(['file', 'router.rs', 'src/', 0])}${palListItem(['file', 'routes.rs', 'src/', 0])}${palListItem(['file', 'case.rs', 'tests/', 0])}<div class="cap">符号</div>${palListItem(['cube', 'Router::match_path', 'fn', 0])}</div>
        <div class="grow fc" style="min-width:0"><div class="f ai g8" style="height:32px;padding:0 12px;border-bottom:1px solid var(--bd-soft)">${C.st.run}<b class="ell">完整修复一个 Bug:路由大小写匹配</b><span class="grow"></span>${bdg('router-demo', '')}</div>
          <div style="padding:10px 12px" class="fc g6"><div class="tx3 s11">最近一条 · 3 分钟前</div><div class="md"><p>已修复。根因是 <code>Router::match_path</code> 直接比较原始字符串…</p></div><div class="f ai g8 tx3 s11">${i('edit', 's12')}编辑 router.rs <span class="okc mono">+4</span><span class="errc mono">−1</span>${i('check', 's12 okc')}12 项测试通过</div></div>
          <span class="grow"></span><div class="f ai g6 s11 tx3" style="padding:8px 12px;border-top:1px solid var(--bd-soft)">${kbd('⏎')} 打开 ${kbd('⌘⏎')} 新窗口 ${kbd('⌥⏎')} 在 IDE 中</div></div></div></div>`;
  S.add({
    id: 'palette', group: 'global', title: '命令面板 ⌘K',
    desc: 'Agent 与 IDE 共用。现在是单列列表,有分段与分组,但看不到选中项的内容。',
    notes: ['方案 B 增加右侧预览:选会话看最近一条回复与改动,选文件看内容片段,选命令看说明,避免盲选。', '前缀快捷输入:@ 主题、> 命令、# 符号;窗口顶部 56px 处弹出,不遮挡输入框。'],
    variants: [
      { label: '单列 + 分段(整理版)', note: '轻量快速;分段 + 分组小标题 + 底部提示。', render: palA },
      { label: '列表 + 预览', tag: '推荐', rec: true, note: '搜索结果右侧给出预览,命中率更高。', render: palB },
    ],
  });

  const pickerBoard = () => `<div class="f" style="height:100%;background:var(--bg-main)">${behind().replace('class="f" style="height:100%"', 'class="f grow" style="height:100%"')}<div class="scrim"></div>
    <div class="abs pop" style="left:260px;top:56px;width:380px;padding:0"><div class="f ai g8" style="height:36px;padding:0 12px;border-bottom:1px solid var(--bd)">${i('history', 's14 tx2')}<span class="grow tx3">搜索历史会话…</span>${seg(['此项目', '全部'], 0, 'sm')}</div><div style="padding:4px 6px 6px"><div class="cap">今天</div>${C.ss('run', '完整修复一个 Bug:路由大小写匹配', '运行中', 1)}${C.ss('ask', '权限请求:修改配置与执行命令', '待许可 · 5 分钟')}<div class="cap">昨天</div>${C.ss('idle', 'Markdown 与代码高亮:发布说明', 'main')}${C.ss('idle', '文件读写:梳理并整理路由层', 'feat/router')}</div></div>
    <div class="abs pop" style="left:660px;top:56px;width:320px;padding:0"><div class="f ai g8" style="height:36px;padding:0 12px;border-bottom:1px solid var(--bd)">${i('branch', 's14 tx2')}<span class="grow v" style="color:var(--tx)">feat/</span></div><div style="padding:4px 6px 6px"><div class="cap">本地</div>${[['main', '当前', 0], ['feat/router', '2 天前', 1], ['feat/gateway', '1 周前', 0]].map(([n, t, on]) => `<div class="mi ${on ? 'hl' : ''}" style="height:26px">${i('branch', 's12')}<span class="grow">${n}</span><span class="tx3 s10">${t}</span></div>`).join('')}<div class="cap">远程</div><div class="mi" style="height:26px">${i('branch', 's12 tx3')}<span class="grow">origin/feat/auth</span><span class="tx3 s10">创建跟踪分支</span></div><div class="sep"></div><div class="mi hl" style="height:26px">${i('plus', 's12')}<span class="grow">创建并切换到 <b>feat/</b>…</span></div></div></div>
    <div class="abs pop" style="left:1000px;top:56px;width:240px;padding:4px"><div class="cap">项目</div>${[['router-demo', 1], ['web-dashboard', 0], ['docs-site', 0]].map(([n, on]) => `<div class="mi ${on ? 'on' : ''}" style="height:26px">${i('folder', 's12 tx2')}<span class="grow">${n}</span>${on ? i('check', 's12 acc') : ''}</div>`).join('')}<div class="sep"></div><div class="mi">${i('plus', 's12')}添加项目…</div><div class="mi">${i('folder', 's12')}打开最近的项目 ${i('chevR', 's12 tx3')}</div></div></div>`;
  S.add({
    id: 'pickers', group: 'global', title: '历史会话 · 分支 · 项目选择器',
    desc: '三个小型快速选择浮层,统一为同一种「搜索框 + 分组列表」样式。',
    notes: ['统一:顶部 36px 搜索行,分组小标题,选中行用强调淡底,右侧灰字放次要信息;顶部居中弹出,Esc 关闭。'],
    variants: [{ label: '快速选择浮层', tag: '推荐', rec: true, note: '', render: pickerBoard }],
  });

  /* ───────── 通知中心 · 对话框 · 更新 / 关于 ───────── */
  const toast = (c, ic, t, d, a) => `<div class="card f g8" style="padding:8px 10px;width:320px;box-shadow:var(--shadow-pop);align-items:flex-start"><span class="${c}" style="margin-top:1px">${i(ic, 's14')}</span><div class="grow"><div class="m">${t}</div><div class="tx2 s11" style="line-height:1.4">${d}</div>${a ? `<div class="f g6" style="margin-top:6px">${a}</div>` : ''}</div>${ib('x', 'sm')}</div>`;
  const dlg = (t, body, acts, w = 380) => `<div class="pop" style="width:${w}px;padding:16px"><div class="b s14" style="margin-bottom:6px">${t}</div><div class="tx2" style="line-height:1.6">${body}</div><div class="f je g6" style="margin-top:14px">${acts}</div></div>`;
  S.add({
    id: 'overlays', group: 'global', title: '通知中心 · 对话框 · 更新与关于',
    desc: '应用级的小窗体:IDE 通知(右下角 + 历史面板)、确认对话框、更新进度与关于窗口。',
    notes: ['通知分四级色(图标着色而非整块染色),动作按钮在文案下方;面板按时间倒序,可全部清除。', '对话框统一:标题 14px 加粗、正文次要色、动作右对齐,主要动作用强调色,危险动作用红字。'],
    variants: [{
      label: '浮层集合', tag: '推荐', rec: true, note: '', render: () => `<div class="f g24" style="padding:22px 28px;height:100%;gap:28px">
      <div class="fc g10" style="width:340px;flex:none"><div class="b s13">右下角通知(IDE)</div>
        ${toast('okc', 'okc', '发现新版本 0.1.23', '已自动下载,重启即可更新。', btn('重启更新', 'sm pri') + btn('稍后', 'sm ghost'))}
        ${toast('warnc', 'alert', '未安装 rust-analyzer', '代码智能暂不可用。', btn('复制安装命令', 'sm') + btn('忽略', 'sm ghost'))}
        ${toast('errc', 'errc', '切换分支失败', '存在未提交的更改会被覆盖。', btn('查看详情', 'sm'))}</div>
      <div class="fc g10" style="width:340px;flex:none"><div class="b s13">通知面板(铃铛)</div>
        <div class="pop" style="padding:0;width:340px"><div class="f ai g8" style="height:34px;padding:0 12px;border-bottom:1px solid var(--bd)"><b>通知</b>${cnt(3)}<span class="grow"></span><span class="tx3 s11">全部清除</span></div>
        ${[['okc', 'okc', '发现新版本 0.1.23', '2 分钟前'], ['warnc', 'alert', '未安装 rust-analyzer', '8 分钟前'], ['infoc', 'info', '已切换到 feat/router', '20 分钟前'], ['errc', 'errc', '提交失败:钩子未通过', '1 小时前']].map(([c, ic, t, d], k) => `<div class="f ai g8" style="height:36px;padding:0 12px;${k ? 'border-top:1px solid var(--bd-soft)' : ''}"><span class="${c}">${i(ic, 's14')}</span><span class="grow ell ${k > 1 ? 'tx2' : ''}">${t}</span><span class="tx3 s10">${d}</span></div>`).join('')}</div>
        <div class="b s13" style="margin-top:8px">更新进度(左下角胶囊)</div>
        <div class="f g8 ai"><span class="chip" style="background:var(--ac-soft);color:var(--ac-tx)">${i('download', 's12')}下载中 42%</span><span class="chip" style="background:var(--ac-soft);color:var(--ac-tx)">${i('refresh', 's12')}校验中</span><span class="chip" style="background:var(--ok-soft);color:var(--ok)">${i('refresh', 's12')}重启更新</span></div></div>
      <div class="fc g10 grow"><div class="b s13">对话框</div>
        ${dlg('切换到真实模式?', '将退出演示后端并重启 MiniCode。有 1 个终端正在运行命令,重启后会被结束。', btn('取消', 'ghost') + btn('切换并重启', 'pri'))}
        ${dlg('关闭有未保存修改的标签?', 'router.rs 有尚未保存的更改。', btn('不保存', 'danger') + btn('取消', '') + btn('保存并关闭', 'pri'))}
        <div class="b s13" style="margin-top:4px">关于窗口</div>
        <div class="pop fc ai g6" style="width:300px;padding:16px"><span class="logo" style="width:44px;height:44px;border-radius:12px">${i('mark', 's20')}</span><b class="s14">MiniCode</b><span class="tx3">版本 0.1.22 (build 82e4bfb)</span><span class="tx3 s10">构建于 11-15 02:40 · 12 分钟前</span><div class="f g6" style="margin-top:6px">${btn('检查更新…', 'sm')}${btn('复制版本信息', 'sm ghost')}</div></div></div></div>`,
    }],
  });
})();
