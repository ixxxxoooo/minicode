/* 设计稿 v3 · 用量与成本整页
   从「设置」里搬出来:它是数据看板,不是偏好。入口:账号菜单、状态栏「上下文」弹层、命令面板。
   范围切换(7 天 / 30 天 / 本月)会真的重算图表与汇总;柱子逐根自下而上长出。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, cnt, seg } = U;
  const RANGES = [['7 天', 7], ['30 天', 30], ['本月', 15]];
  /** 确定性的每日数据:越靠近今天越高,周末偏低 */
  const day = (k) => { const w = (k % 7); return Math.max(.08, .22 + .42 * Math.abs(Math.sin(k * 1.37 + 1)) + (w === 5 || w === 6 ? -.14 : 0)); };
  const series = (n) => Array.from({ length: n }, (_, i) => { const k = 30 - n + i, d = new Date(2026, 10, 15 - (n - 1 - i)); return { d: `${d.getMonth() + 1} 月 ${d.getDate()} 日`, v: +(day(k) * 1.2).toFixed(2) }; });
  const money = (x) => '$' + x.toFixed(2);
  const MODELS = [['Sonnet 5.5', .72, 'info'], ['Opus 5.5', .21, 'c-vio'], ['Haiku 5.5', .07, 'ok']];
  const PROJ = [['router-demo', 5.2, 28], ['web-dashboard', 3.1, 17], ['docs-site', 1.6, 9], ['scenario-lab', .9, 10]];
  const SESS = [['完整修复一个 Bug:路由大小写匹配', 'router-demo', 1.42, 'Sonnet 5.5'], ['2000 轮长会话:滚动与加载', 'scenario-lab', .86, 'Opus 5.5'], ['权限请求:修改配置与执行命令', 'router-demo', .55, 'Sonnet 5.5'], ['子 Agent:代码审查与调研', 'web-dashboard', .48, 'Sonnet 5.5']];

  const stat = (v, l, sub = '') => `<div class="stat-c"><b class="num">${v}</b><span class="t3 fs-cap">${l}</span>${sub ? `<span class="fs-micro ${sub.startsWith('↑') ? 'c-warn' : 'c-ok'}">${sub}</span>` : ''}</div>`;
  const chart = (data) => { const mx = Math.max(...data.map(x => x.v));
    const mean = total / data.length;
    return `<div class="bars" style="--n:${data.length}">${data.map((x, k) => {
      const cmp = Math.abs(x.v - mean) < .02 ? '与均值持平' : (x.v > mean ? `高于均值 ${Math.round((x.v - mean) / mean * 100)}%` : `低于均值 ${Math.round((mean - x.v) / mean * 100)}%`);
      return `<i class="tip-u" style="--k:${k};height:${Math.max(4, x.v / mx * 100)}%" data-tip="${x.d} · ${money(x.v)} · ${cmp}"></i>`;
    }).join('')}</div>
      <div class="f jb t4 fs-micro" style="padding:4px 2px 0"><span>${data[0].d}</span><span>${data[Math.floor(data.length / 2)].d}</span><span>${data[data.length - 1].d}</span></div>`; };
  const bar = (n, p, v, c = '') => `<div class="f ai g4" style="height:28px"><span style="width:96px" class="ell">${n}</span><div class="meter grow" style="height:6px"><i style="width:${p * 100}%;${c ? `background:var(--${c.startsWith('c-') ? c : c})` : ''}"></i></div><span class="num t3" style="width:58px;text-align:right">${v}</span></div>`;

  W.pages.usage = (st) => { const r = st.useR ?? 1, [rl, n] = RANGES[r], data = series(n), total = data.reduce((s, x) => s + x.v, 0), prev = total * (r === 0 ? 1.18 : .91);
    const dlt = ((total - prev) / prev * 100);
    return `<div class="pg-h"><div class="grow"><div class="fs-head b">用量与成本</div><div class="t3">本机各会话的 token 与费用估算(取自会话记录,第三方服务商没有费用时只显示 token),以及账户限额。</div></div>${seg(RANGES.map(x => x[0]), r, '', 'use-range')}${btn('导出 CSV', '', 'download', 'data-act="toast" data-msg="已导出 usage.csv"')}${ib('x', '', '关闭 Esc', 'data-act="page" data-page="work"')}</div>
      <div class="pg-split"><div class="pg-main scroll" id="use-main"><div class="f g3 stagger">${stat(money(total), rl + '费用', (dlt >= 0 ? '↑ ' : '↓ ') + Math.abs(dlt).toFixed(0) + '% 较上一段')}${stat((total * .31).toFixed(1) + 'M', '输入 token')}${stat(Math.round(total * 17) + 'k', '输出 token')}${stat(Math.round(total * 5.2), '会话', '缓存命中 90%')}</div>
        <div class="card" style="padding:12px 14px;margin-top:10px"><div class="f ai g3" style="margin-bottom:8px"><b>每日费用</b><span class="t3 fs-cap">悬停看每天的数值</span><span class="grow"></span><span class="f ai g2 fs-cap t3"><span class="dot ac"></span>费用</span></div>${chart(data)}</div>
        <div class="grid2" style="margin-top:10px"><div class="card" style="padding:10px 14px"><b>按模型</b><div style="margin-top:4px">${MODELS.map(([m, p, c]) => bar(m, p, money(total * p), c === 'info' ? 'info' : c === 'ok' ? 'ok' : 'c-vio')).join('')}</div></div>
          <div class="card" style="padding:10px 14px"><b>按项目</b><div style="margin-top:4px">${PROJ.map(([p, v, s]) => `<div class="f ai g4" style="height:28px"><span class="grow ell">${p}</span><span class="t3 num fs-cap">${s} 个会话</span><span class="num" style="width:54px;text-align:right">${money(v * total / 10.8)}</span></div>`).join('')}</div></div></div>
        <div class="card" style="padding:10px 6px;margin-top:10px"><div style="padding:0 8px 4px"><b>最贵的会话</b></div>${SESS.map(([t, p, v, m]) => `<div class="row" style="height:30px" data-act="toast" data-msg="已在 Agent 形态打开该会话"><span class="grow ell">${t}</span><span class="t3 fs-cap">${p}</span>${tag(m)}<span class="num" style="width:50px;text-align:right">${money(v)}</span></div>`).join('')}</div></div>
      <aside class="drawer"><div class="f ai g3"><b class="grow">账户限额</b><span class="t4 fs-cap" id="lim-at">更新于 3 分钟前</span>${ib('refresh', 'sm', '刷新', 'data-act="lim-refresh"')}</div>
      <div class="note" style="margin:-4px 0 8px">${ic('info', 's12')}<span>限额查询中 / 失败 / 无会话 · 专屏见 flows 页矩阵</span></div>
        <div class="fc g4" id="lim-box">${[['5 小时窗口', 38, '2 小时 10 分后重置'], ['每周', 12, '周四重置'], ['Opus 每周', 6, '周四重置']].map(([n, p, r]) => `<div class="fc g2"><div class="f ai"><span class="grow m">${n}</span><span class="num t3">${p}%</span></div><div class="meter" style="height:6px"><i style="width:${p}%;${p > 80 ? 'background:var(--warn)' : ''}"></i></div><span class="t4 fs-micro">${r}</span></div>`).join('')}</div>
        <div class="sep" style="margin:0"></div><b>建议</b>
        <div class="fc g3">${[['okc', 'c-ok', '缓存命中率 90%', '长会话没有浪费输入 token'], ['alert', 'c-warn', 'Opus 占费用 21%', '简单任务切到 Sonnet 能省约 $0.8'], ['info', 'c-info', '2000 轮长会话较贵', '用 /compact 压缩后再继续']].map(([i, c, t, d]) => `<div class="f g3">${ic(i, 's12 ' + c)}<div class="grow"><div class="m">${t}</div><div class="t3 fs-cap" style="line-height:1.45">${d}</div></div></div>`).join('')}</div>
        <span class="grow"></span><div class="f ai g3 t3 fs-cap">${ic('info', 's12')}<span>限额只在打开此页或点「刷新」时查询,不轮询。</span></div></aside></div>`; };
  W.usage = { RANGES };
})();
