/* 设计稿 v3.4 · 上下文弹层
   入口:输入框右侧的「上下文」圆环(点击展开,再点 / 点外部 / Esc 收起)与 IDE 状态栏的「上下文」,两处是同一份内容(一个内容只有一个家)。
   内容:用量与阈值 → 分类明细 → 账户限额 → 更新时间与刷新 → 跳转「用量与成本」。
   状态:ok 常态 / warn 接近阈值(建议压缩)/ busy 压缩中 / load 明细或官方限额加载中 / none 第三方(限额区占位+手动点获取)/ err 查询失败;
   历史会话可先显示「来自历史记录的估算」;点刷新才向 Claude 发 get_context_usage(3 秒防抖);惰性会话有「启动并查询」。
   官方 Anthropic 登录:点开圆环时后台 get_usage(30s 内复用);判定以实际通道为准(映射未启用、非 Bedrock/Vertex、~/.claude settings 无第三方 Base URL 即官方;仅有 AWS_PROFILE 等环境变量不算第三方);第三方不自动查限额,仅点限额区或刷新按钮。
   F.ctxBody(o) 只返回内部内容(两处的外壳不同),F.ctxPop(o) 是输入框里的外壳。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib } = U;
  /** 分类:[名称, 色, token 数(k), 备注];合计 76k,与圆环 38% 对应 */
  const CATS = [['系统提示', 'var(--info)', 28, ''], ['消息', 'var(--ok)', 22, ''], ['工具定义', 'var(--warn)', 18, ''], ['MCP 工具', 'var(--c-vio)', 8, '另有 12k 按需加载,不占窗口']];
  const TOTAL = 200;
  /** 各状态的取数:used 已用 k、msg 一句话提示 */
  const ST = { ok: { used: 76 }, warn: { used: 184 }, busy: { used: 184 }, load: { used: 76 }, none: { used: 76 }, err: { used: 76 } };

  /** 用量环:≥85% 换警告色,≥95% 换错误色 */
  F.ctxRingCls = (pct) => pct >= 95 ? 'err' : pct >= 85 ? 'warn' : '';
  const row = (n, c, k, note) => `<div class="ctx-row"><span class="dot" style="background:${c}"></span><span class="grow ell">${n}${note ? `<span class="t4"> · ${note}</span>` : ''}</span><span class="num t3">${k}k</span><span class="num t4 pc">${Math.round(k / TOTAL * 100)}%</span></div>`;
  /** 分类明细:按已用量等比缩放(警告态下消息占大头) */
  const cats = (used) => {
    if (used === 76) return CATS;
    return [['系统提示', 'var(--info)', 28, ''], ['消息', 'var(--ok)', used - 28 - 18 - 8, ''], ['工具定义', 'var(--warn)', 18, ''], ['MCP 工具', 'var(--c-vio)', 8, '另有 12k 按需加载,不占窗口']];
  };
  const bar = (list, thr) => `<div class="ctxbar thr">${list.map(([, c, k]) => `<i style="width:${k / TOTAL * 100}%;background:${c}"></i>`).join('')}<b class="tick" style="left:${thr}%" title="自动压缩阈值 ${thr}%"></b></div>`;
  const lim = (n, p, sub) => `<div class="ctx-lim"><div class="f ai"><span class="grow">${n}</span><span class="num t3">${p}%</span></div><div class="meter ${p >= 85 ? 'warn' : ''}"><i style="width:${p}%"></i></div><div class="t4 fs-micro">${sub}</div></div>`;

  /** 内部内容。o.state 见文件头;o.upd 更新时间文案 */
  F.ctxBody = (o = {}) => {
    const s = o.state || 'ok', used = ST[s].used, pct = Math.round(used / TOTAL * 100), cl = F.ctxRingCls(pct);
    const head = `<div class="f ai g4"><span class="ring big ${cl}" style="--p:${pct}"></span><div class="grow"><b class="num">${used}k / ${TOTAL}k</b> <span class="${cl ? 'c-' + (cl === 'err' ? 'err' : 'warn') : 't3'}">· ${pct}%</span><div class="t3 fs-cap">Sonnet 5.5 · 自动压缩阈值 85%</div></div>${
      s === 'busy' ? btn('压缩中…', 'sm loading', '', 'disabled') : btn('压缩', s === 'warn' ? 'sm pri' : 'sm', '', 'data-act="ctx-compact"')}</div>`;
    const hint = s === 'warn' ? `<div class="ctx-hint warn">${ic('alert', 's12')}<span class="grow">已用 ${pct}%,下一轮可能触发自动压缩。现在压缩可保留更多细节。</span></div>`
      : s === 'busy' ? `<div class="ctx-hint">${ic('refresh', 's12 spin')}<span class="grow">正在压缩上下文…完成后这里会更新占用。</span></div><div class="meter indet"><i></i></div>` : '';
    const body = s === 'load' ? `<div class="fc g2 ctx-skel"><i class="skel" style="width:100%;height:6px"></i>${[70, 55, 62, 48].map(w => `<i class="skel" style="width:${w}%"></i>`).join('')}<div class="t4 fs-micro">正在读取分类明细…</div></div>`
      : `${bar(cats(used), 85)}<div class="fc g2">${cats(used).map(([n, c, k, note]) => row(n, c, k, note)).join('')}${row('空闲', 'var(--fill-3)', TOTAL - used, '')}</div>`;
    const limits = s === 'none' ? `<div class="ctx-none">${ic('info', 's12')}<span class="grow">当前使用第三方服务商,没有账户限额数据。</span></div>`
      : s === 'err' ? `<div class="ctx-hint err">${ic('alert', 's12')}<span class="grow">账户限额查询失败(网络超时),分类明细不受影响。</span>${btn('重试', 'sm ghost', 'refresh', 'data-act="ctx-refresh"')}</div>`
      : s === 'load' ? '' : `<div class="fc g3">${lim('5 小时限额', 38, '2 小时 10 分后重置')}${lim('本周限额', 21, '周五 09:00 重置')}</div>`;
    const est = o.estimated ? `<div class="ctx-hint info">${ic('info', 's12')}<span class="grow">来自历史记录的估算,点刷新向 Claude 查询精确值。</span></div>` : '';
    const foot = `<div class="f ai fs-cap t3 ctx-foot"><span class="grow ctx-upd">${s === 'load' ? '刷新中…' : o.estimated ? '估算 · 点刷新查询精确值' : '更新于 ' + (o.upd || '3 分钟前')}</span>${ib('refresh', 'sm tip-u', '刷新', 'data-act="ctx-refresh"')}${btn('用量与成本', 'sm ghost', 'arrow', 'data-act="screen" data-screen="usage"')}</div>`;
    return head + hint + est + body + limits + foot;
  };
  /** 输入框里的外壳:向上弹出,右对齐到圆环 */
  F.ctxPop = (o = {}) => `<div class="pop ctx-pop a-pop-up" data-state="${o.state || 'ok'}">${F.ctxBody(o)}</div>`;
})();
