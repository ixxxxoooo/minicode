/* 设计稿 v3.3 · 规范页「图标与品牌」
   应用图标的构造与六种样式、四种外观 × 六套主题、尺寸阶梯、Dock / ⌘Tab / 访达 / 通知 / 安装盘等系统场景,
   菜单栏托盘的状态矩阵 / 动画 / 菜单,文件类型图标,应用内用法,设置里的更换方式与实现状态。
   页面里所有图标都跟随「设置 → 外观 → 应用图标 / 菜单栏图标」的选择(点这里的样式卡同样会改)。
   @author ixxxxoooo */
(function () {
  const { ic, btn, tag, kbd, THEMES } = U;
  const B = U.brand, { app, tray } = B;
  const { head, sec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const stt = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const cap = (t, sub = '') => `<div class="bg-cap">${t}${sub ? `<span class="t3">${sub}</span>` : ''}</div>`;
  const why = (t) => `<div class="bg-why">${t}</div>`;
  const cards = (n, ...c) => `<div class="bg-cards c${n}">${c.join('')}</div>`;
  const code = (t) => `<span class="kcode">${t}</span>`;

  /* ───────── 构造图 ───────── */
  const construct = () => { const z = 440, L = (x, y, t, a = 'middle') => `<text class="dim-t" x="${x}" y="${y}" text-anchor="${a}">${t}</text>`;
    const D = (d) => `<path d="${d}" fill="none" stroke="var(--ac)" stroke-width="3" stroke-linecap="round"/>`;
    return `<div style="position:relative;width:${z}px;height:${z}px">${app({ s: 'weave', m: 'light', t: 'paper', z, flat: 1 })}
      <svg viewBox="0 0 1024 1024" width="${z}" height="${z}" style="position:absolute;inset:0" aria-hidden="true">
        <use href="#mc-sq" fill="none" stroke="var(--ac)" stroke-width="2.5" stroke-dasharray="14 10" opacity=".75"/>
        <rect x="192" y="192" width="640" height="640" fill="none" stroke="var(--info)" stroke-width="2.5" stroke-dasharray="6 8" opacity=".65"/>
        <path d="M512 100V924M100 512H924" stroke="var(--tx-4)" stroke-width="2" stroke-dasharray="4 8"/>
        <rect x="236" y="236" width="552" height="552" fill="none" stroke="var(--ac)" stroke-width="2.5" opacity=".85"/>
        ${D('M318 232v-12M430 232v-12M318 226H430')}${L(374, 204, '112')}${D('M430 232v-12M456 232v-12M430 226h26')}${L(443, 180, '26')}${D('M456 232v-12M568 232v-12M456 226H568')}${L(512, 204, '112')}
        ${D('M800 236h12M800 318h12M806 236v82')}${L(826, 284, '82', 'start')}
        ${D('M100 960v-12M924 960v-12M100 954H924')}${L(512, 940, '824 · 连续圆角', 'middle')}
        ${L(512, 884, '安全区 640 · 图形区 552', 'middle')}</svg></div>`; };

  /* ───────── 文件类型图标:纸页 + 小图形 + 扩展名 ───────── */
  const docSvg = (label, o = {}) => { const id = B.uidNext(), r = B.resolve({ m: 'light', t: o.t }), st = o.s || B.cur('icoStyle');
    return `<svg class="bi-svg" width="${o.z || 96}" height="${o.z || 96}" viewBox="0 0 1024 1024" aria-hidden="true"><defs>
      <linearGradient id="${id}p" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ffffff"/><stop offset="1" stop-color="#eeece6"/></linearGradient>
      <linearGradient id="${id}t" x1="250" y1="250" x2="774" y2="774" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="${r.th[0]}"/><stop offset="1" stop-color="${r.th[1]}"/></linearGradient></defs>
      <g filter="url(#mc-sh-l)"><path d="M252 80H664L812 228V904Q812 944 772 944H252Q212 944 212 904V120Q212 80 252 80Z" fill="url(#${id}p)"/></g>
      <path d="M252 80H664L812 228V904Q812 944 772 944H252Q212 944 212 904V120Q212 80 252 80Z" fill="none" stroke="rgba(0,0,0,.14)" stroke-width="3"/>
      <path d="M664 80V188Q664 228 704 228H812Z" fill="#dedbd2" stroke="rgba(0,0,0,.14)" stroke-width="3" stroke-linejoin="round"/>
      <g transform="translate(512 500) scale(.5) translate(-512 -512)">${B.ART[st === 'weave' ? 'weave' : st]({ ink: r.ink, th: `url(#${id}t)` })}</g>
      <rect x="282" y="770" width="460" height="104" rx="52" fill="url(#${id}t)"/><text x="512" y="842" text-anchor="middle" font-family="-apple-system,SF Pro Text,sans-serif" font-weight="700" font-size="62" fill="#fff">${label}</text></svg>`; };

  /* ───────── 菜单栏 / 菜单的仿真 ───────── */
  const WIFI = '<svg class="sys" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M2.5 9a14 14 0 0 1 19 0M5.5 12.5a9.5 9.5 0 0 1 13 0M8.6 16a5 5 0 0 1 6.8 0"/><circle cx="12" cy="19.2" r=".9" fill="currentColor"/></svg>';
  const BAT = '<svg class="sys" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><rect x="2" y="7" width="17" height="10" rx="2.6"/><rect x="4" y="9" width="9" height="6" rx="1.2" fill="currentColor" stroke="none"/><path d="M21.2 10.6v2.8"/></svg>';
  const SRCH = '<svg class="sys" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="6"/><path d="m20 20-4.2-4.2"/></svg>';
  const mbar = (d, state, o = {}) => `<div class="mbar ${d ? 'd' : ''}"><span class="mi">${SRCH}</span><span class="mi">${WIFI}</span><span class="mi">${BAT}</span><span class="mi ${o.sel ? 'sel' : ''}">${tray({ state, z: 18, col: o.col })}${o.n ? `<span>${o.n}</span>` : ''}</span><span class="clk">周日 21:33</span></div>`;
  const menu = (d) => `<div class="nsmenu ${d ? 'd' : ''}"><div class="hd">MiniCode · 1 个待确认 · 2 个运行中</div>
    ${[['router-demo', [['wait', '修复路径匹配的大小写问题']]], ['web-dashboard', [['run', '迁移到 axum 0.8'], ['', '补充单元测试']]], ['docs-site', [['run', '更新安装说明'], ['unread', '整理变更日志(刚完成)']]]].map(([p, rows]) => `<hr><div class="gh">${p}</div>${rows.map(([k, t]) => `<div class="mn"><span class="sd ${k}">${k === 'run' ? '<b></b>' : '<i></i>'}</span><span class="ell">${t}</span></div>`).join('')}`).join('')}
    <hr><div class="mn hl">新对话<small>⌘N</small></div><div class="mn">打开 MiniCode</div><div class="mn">暂停通知<small>▸</small></div><hr><div class="mn">设置…<small>⌘,</small></div><div class="mn">检查更新…</div><div class="mn">重启 MiniCode</div><div class="mn">退出 MiniCode<small>⌘Q</small></div></div>`;

  /* ───────── Dock 状态卡 ───────── */
  const dockCard = (caption, wall, inner) => `<div><div class="wall wall-${wall} wall-c" style="height:122px"><div class="mk-dock ${wall === 'd' ? 'd' : ''}"><span class="dk sm"><i class="gen-ic" style="--h:${wall === 'd' ? 250 : 205}"></i></span>${inner}<span class="dk sm"><i class="gen-ic" style="--h:${wall === 'd' ? 165 : 335}"></i></span></div></div><div class="bg-why" style="text-align:center;margin-top:5px">${caption}</div></div>`;

  /* ───────── 四种外观 × 六套主题 ───────── */
  const matrix = () => `<div class="bm-tbl"><div class="bm-h">主题 · 色板</div>${['浅色', '深色', '透明 Clear', '着色 Tinted'].map(t => `<div class="bm-h">${t}</div>`).join('')}
    ${THEMES.map(t => { const p = B.PAL[t.id], wl = t.dark ? 'd' : 'l';
      return `<div class="bm-r"><b>${t.name}</b><span class="t3 fs-cap">${t.sub} · 自动 → ${t.dark ? '深色' : '浅色'}</span><span class="sws"><i style="background:${p.ac[0]}"></i><i style="background:${p.ac[1]}"></i><i style="background:${p.L[1]}"></i><i style="background:${p.D[1]}"></i></span></div>`
        + ['light', 'dark', 'clear', 'tint'].map(m => `<div class="bm-c ${m === 'clear' ? 'wall wall-' + wl : ''}">${app({ t: t.id, m, z: 92 })}</div>`).join(''); }).join('')}</div>`;

  /* ───────── 页面 ───────── */
  P['brand-spec'] = () => `<div class="ds" style="max-width:1240px">${head('图标与品牌 · 应用图标、菜单栏托盘与各场景', '一个母题(织机)、六种样式、四种外观、六套主题,应用图标与菜单栏图标同源。设置 → 外观里可以自己换;这一页的所有图标都跟随那里的选择(下面的样式卡点一下就换),先选一个再往下看。')}

    ${sec('设计语言 · 织机', '经线(墨色)是会话、终端、改动三条主线;纬线(主题色)是 Claude 的思路,从它们之间穿过;压穿交替,像人审批 ⇄ Agent 执行的节奏。线宽即令牌,所有衍生图标按同一比例缩放。',
      `<div class="bg-hero"><div class="bg-stage" style="min-height:480px">${construct()}</div>
        <div class="fc g4">${tbl([['令牌', 96], ['数值(1024 画布)'], ['用在哪']], [
          ['BAND', '112 · 带宽', '线的粗细;提示符、花括号的笔画、文档页上的缩略图都沿用'],
          ['GAP', '26 · 带间缝', '线与线的呼吸;≤ 28px 的小尺寸改用 2×2「经纬」,缝加到 34'],
          ['CUT', '14 · 压穿切口', '上层线从下层线上「压过去」时,下层线两侧各让出 14,不靠背景色挖洞,所以透明 / 着色外观同样成立'],
          ['TAIL', '82 · 线头', '线头伸出织纹区,一眼看出是「线」而不是格子'],
          ['图形', '824 · 连续圆角', 'macOS Big Sur 起的图标网格:824 居中,四周 100 留给投影;圆角用超椭圆(n≈5)而不是圆弧,和系统图标的曲率一致'],
          ['安全区', '640', '主图形不越过它;织纹图形区 552,留出视觉重心的余量'],
          ['光影', '投影 dy12 σ16 · 顶部高光', '浅色 20% / 深色 50% 黑;顶部 1px 内描边(浅 95%、深 16% 白)。≤ 36px 去掉投影与高光']])}
          <div class="t3 fs-cap" style="line-height:1.7">与现有资源的关系:<span class="kcode">assets/icon/gen.py</span> 里已有的浅 / 深两版就是「织纹 · 浅色 / 深色」,这里只是把它参数化(切口改为分段,不再依赖背景色),并补上其余样式、透明与着色外观。</div></div></div>`)}

    ${sec('应用图标样式 · 六选一', '点一张卡就是换图标(与 设置 → 外观 → 应用图标 同一个选择)。六种共用同一套线宽、配色与光影,只在图形上不同。', `<div class="ico-pick" role="radiogroup" style="grid-template-columns:repeat(3,minmax(0,1fr));gap:10px">${B.STYLES.map(s => `<div class="ico-card big ${s.id === B.cur('icoStyle') ? 'on' : ''}" data-act="ico-style" data-v="${s.id}" tabindex="0" role="radio" aria-checked="${s.id === B.cur('icoStyle')}">${app({ s: s.id, z: 132 })}<b style="font-size:13px">${s.name} <span style="font-weight:400">${s.en}</span></b><p>${s.d}</p></div>`).join('')}</div>`)}

    ${sec('外观 × 主题 · 24 种组合', '外观对应 macOS 26 的四种图标外观(默认 / 深色 / 透明 / 着色);「自动」按系统深浅在浅色与深色之间选。配色默认跟随 MiniCode 当前主题,也可以固定为某一套(设置 → 配色)。透明 Clear 放在壁纸上才看得出;着色 Tinted 是单色相的深底浅图形,和系统「着色」图标一致。', matrix())}
    ${cards(2, `<div>${cap('分层与导出', 'Icon Composer')}${tbl([['图层', 90], ['内容']], [['背景', '渐变底(浅 / 深各一)+ 顶部高光;透明外观里它变成玻璃'], ['经线', '墨色带,组成「织纹 / 经纬 / 棋盘 / 行 / 花括号」的主体'], ['纬线', '主题渐变带,着色外观里它与经线同色相、不同明度']])}${why('三层分别导出,交给 Icon Composer 合成 .icon,系统会自己生成默认 / 深色 / 透明 / 着色四种外观;本稿里的四列是它们应有的样子,用于对照验收。')}</div>`,
      `<div>${cap('配色规则')}${tbl([['外观', 90], ['背景', 130], ['线']], [['浅色', '近白 → 主题浅灰', '墨 + 主题渐变(亮 → 深)'], ['深色', '主题暖 / 冷黑', '浅墨 + 更亮一档的主题渐变'], ['透明', '白 40% → 12% 玻璃', '白 + 主题色淡化 30–55%'], ['着色', '主题色压暗 38% → 74%', '浅色相 + 主题色']])}${why('深色不是系统滤镜反相,而是单独调过的一套:主题渐变整体提亮,保证在 Dock 的深色壁纸上仍有 4.5:1 以上的对比。')}</div>`)}

    ${sec('尺寸阶梯与光学尺寸', '同一枚图标在 16–256 px 不能只是等比缩放:≤ 36px 去掉投影与高光(它们在小尺寸只会发灰),≤ 28px 的织纹自动换成 2×2「经纬」(带宽 ≥ 2px,否则 3×3 会糊成一团)。', `<div class="sz-row">${[[256, '256 · 完整'], [128, '128'], [64, '64 · Dock 常见'], [36, '36'], [28, '28 · 自动换「经纬」'], [20, '20'], [16, '16 · 菜单 / 列表']].map(([z, t]) => `<figure><span style="display:block;margin:${-Math.round(z * .1)}px">${app({ z })}</span><figcaption>${t}</figcaption></figure>`).join('')}</div>`)}

    ${sec('Dock · 状态', 'Dock 里的图标就是应用图标本体;状态只靠系统三件套表达,我们不另画:运行点、角标、进度。运行中的小黑点是系统画的,不进图标。', `<div class="dk-grid">
      ${dockCard('默认(未运行)', 'l', `<span class="dk">${app({ z: 56 })}</span>`)}
      ${dockCard('运行中:系统画的小圆点', 'l', `<span class="dk run">${app({ z: 56 })}</span>`)}
      ${dockCard('待确认:角标 = 待处理数,并弹跳一次', 'l', `<span class="dk run bounce">${app({ z: 56 })}<b class="dk-badge">2</b></span>`)}
      ${dockCard('下载更新:自绘进度条', 'l', `<span class="dk run">${app({ z: 56 })}<span class="dk-prog"><i style="width:62%"></i></span></span>`)}
      ${dockCard('深色壁纸 · 深色外观', 'd', `<span class="dk run">${app({ z: 56, m: 'dark' })}</span>`)}</div>
      ${why('角标内容可选:关 / 待处理数(权限 + 提问,默认)/ 运行中的会话数;弹跳可选 不弹 / 一次(默认)/ 直到处理(只对权限请求建议用)。设置里同一处调整。')}`)}

    ${sec('系统各处', '图标会出现在 Dock 之外的很多地方,大小从 16 到 128 不等。下面是同一个选择在各处的样子:⌘Tab 取图标本体、Spotlight 与访达用小尺寸、启动台用 76。', cards(2,
      `<div>${cap('⌘Tab 应用切换器')}<div class="wall wall-d wall-c" style="height:150px"><div class="cmdtab"><div class="ct"><i class="gen-ic" style="--h:205"></i><span>访达</span></div><div class="ct on">${app({ z: 72 })}<span>MiniCode</span></div><div class="ct"><i class="gen-ic" style="--h:150"></i><span>终端</span></div><div class="ct"><i class="gen-ic" style="--h:330"></i><span>备忘录</span></div></div></div></div>`,
      `<div>${cap('启动台')}<div class="wall wall-l wall-c" style="height:150px"><div class="lpad"><div class="lp"><i class="gen-ic" style="--h:205"></i>访达</div><div class="lp">${app({ z: 76 })}MiniCode</div><div class="lp"><i class="gen-ic" style="--h:25"></i>日历</div><div class="lp"><i class="gen-ic" style="--h:280"></i>播客</div></div></div></div>`,
      `<div>${cap('Spotlight')}<div class="wall wall-l wall-c" style="height:176px"><div class="spot"><div class="q"><span>mini</span><span class="caret-b"></span></div><div class="r on">${app({ z: 44 })}<span>MiniCode</span><small>应用程序</small></div><div class="r"><i class="gen-ic" style="--h:150;width:30px;height:30px"></i><span>MiniTool</span><small>应用程序</small></div></div></div></div>`,
      `<div>${cap('访达 · 侧栏 / 列表 / 图标')}<div class="wall wall-l wall-c" style="height:176px"><div class="fndr"><div class="sb"><h6>个人收藏</h6><div class="it on">${app({ z: 20, flat: 1 })}应用程序</div><div class="it">下载</div></div><div class="mn"><div class="rw sm">${app({ z: 20, flat: 1 })}MiniCode.app<small>0.2.0</small></div><div class="rw on">${app({ z: 50 })}MiniCode<small>应用程序</small></div></div></div></div></div>`))}

    ${sec('窗口与安装盘', '关于窗口是应用自绘的(图标 96,随主题走);安装盘背景是固定的浅色画面,图标取浅色外观、不随主题变,避免用户在不同机器上看到不同的安装界面。', cards(2,
      `<div>${cap('关于 MiniCode')}<div class="wall wall-c" style="height:272px;background:var(--bg-sunken)"><div class="about-w"><span class="tl"><i></i><i></i><i></i></span>${app({ z: 116 })}<b class="fs-head" style="margin-top:2px">MiniCode</b><span class="t3 fs-cap">版本 0.2.0(构建 4f2a91c)<br>构建于 3 天前</span><span style="display:flex;gap:6px;margin-top:6px">${btn('检查更新', 'sm', 'refresh')}${btn('复制版本信息', 'sm ghost', 'copy')}</span></div></div></div>`,
      `<div>${cap('MiniCode-0.2.0.dmg')}<div class="wall wall-l wall-c" style="height:272px"><div class="dmg" style="width:100%"><div class="tb"><span class="tl"><i></i><i></i><i></i></span><b>MiniCode</b></div><div class="stage"><div class="fi">${app({ z: 118, m: 'light', t: 'paper' })}MiniCode</div><svg class="arr" viewBox="0 0 64 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h54M48 4l10 8-10 8"/></svg><div class="fi"><span class="fold-ic"></span>应用程序</div></div><div class="hint">将 MiniCode 拖到「应用程序」文件夹</div></div></div></div>`))}

    ${sec('通知横幅', '通知图标一律是应用图标(系统取包内的图标,不是运行时换上的)。标题放项目名,正文一句话,不堆符号;需要你拍板的通知带「允许 / 拒绝」动作,并标成「时效性」,可穿透专注模式。', cards(3,
      `<div>${cap('回合完成')}<div class="wall wall-l wall-c" style="height:122px"><div class="notif"><div class="ni">${app({ z: 48, flat: 1 })}</div><div class="nb"><b>router-demo</b><p>已完成:修复路径匹配的大小写问题,3 个文件有改动。</p></div><span class="nt">现在</span></div></div></div>`,
      `<div>${cap('需要确认', '时效性 · 带动作')}<div class="wall wall-d wall-c" style="height:122px"><div class="notif d"><div class="ni">${app({ z: 48, flat: 1, m: 'dark' })}</div><div class="nb"><b>router-demo · 需要确认</b><p>允许运行 <span style="font-family:var(--font-mono);font-size:12px">cargo test</span>?</p><div class="ts">${ic('alert', 's12')}时效性</div><div class="na"><span>允许一次</span><span>拒绝</span></div></div><span class="nt">现在</span></div></div></div>`,
      `<div>${cap('出错')}<div class="wall wall-l wall-c" style="height:122px"><div class="notif"><div class="ni">${app({ z: 48, flat: 1 })}</div><div class="nb"><b>web-dashboard · 会话中断</b><p>claude 进程退出,自动重启失败。点按查看日志。</p></div><span class="nt">2 分钟前</span></div></div></div>`))}

    ${sec('菜单栏托盘 · 设计规则', '菜单栏图标是模板图像:只有黑色与透明,系统按菜单栏明暗自动着色、选中态自动反白,所以它不能靠颜色传达信息——每个状态都有独立的形状。', cards(3,
      `<div class="usage-card">${cap('模板图像规范')}${tbl([['项', 90], ['规则']], [['画布', '18 × 18 pt(36 × 36 @2x),图形 15–16 pt,四周留 1.5 pt'], ['笔画', '≈ 1.6 pt(@2x 3 px 起),与 SF Symbols 的中等字重对齐'], ['颜色', '只有黑 + alpha;<span class="kcode">isTemplate = true</span>'], ['垂直', '图形光学居中,与相邻系统图标同一基线']])}</div>`,
      `<div class="usage-card">${cap('徽标规范')}${tbl([['徽标', 90], ['形状与位置']], [['待确认', '右上角 12.8 px 实心圆 + 镂空「!」'], ['已完成', '右上角 9 px 实心圆,无符号'], ['出错', '圆 + 镂空「×」'], ['更新', '圆 + 镂空向下箭头'], ['暂停 / 演示', '半透明主体 + 暂停符;演示为空心圆环']])}${why('徽标周围先镂空 2.3 px 再画,保证和主体之间有缝;符号用镂空而不是描线,小尺寸不糊。')}</div>`,
      `<div class="usage-card">${cap('彩色徽标(可选)')}<div class="f g3 ai" style="margin:2px 0 4px">${['waiting', 'done', 'error', 'update'].map(s => `<span class="mb-chip w">${tray({ state: s, z: 22, col: 1 })}</span>`).join('')}</div>${why('开启后徽标用系统语义色(橙 / 绿 / 红 / 蓝),主体仍是模板图,因此整体不再是纯模板——我们在 NSAppearance 变化时重绘一次。徽标形状始终不变,色弱用户不依赖颜色也能分辨;「减弱透明度 / 增强对比度」时自动回退成纯模板。')}</div>`))}

    ${sec('菜单栏图标样式 · 五选一', '与应用图标是同一个图形的单色版(「经纬」只作应用图标的小尺寸替身,菜单栏里不单列)。点卡片即换,每张同时给出浅色与深色菜单栏上的样子。', `<div class="usage-card" style="padding:12px">${B.pickTray()}</div>`)}

    ${sec('托盘状态矩阵', '八个状态、四种背景:浅色菜单栏、深色菜单栏、被点开时的高亮底、以及彩色徽标版(壁纸上)。优先级:待确认 > 出错 > 已完成 > 运行中 > 有更新 > 暂停 > 空闲(演示是叠加标记,与其余并存时只留一个徽标,保留演示环)。',
      tbl([['状态', 78], ['浅色', 70], ['深色', 70], ['点开', 70], ['彩色', 70], ['何时出现']], B.TRAY_STATES.map(([id, n, d]) => [n,
        `<span class="mb-chip l">${tray({ state: id, z: 20, col: 0 })}</span>`, `<span class="mb-chip d">${tray({ state: id, z: 20, col: 0 })}</span>`, `<span class="mb-chip sel">${tray({ state: id, z: 20, col: 0 })}</span>`, `<span class="mb-chip w">${tray({ state: id, z: 20, col: 1 })}</span>`, d])))}

    ${sec('运行中的动画', '「织线流动」:三根纬线依次变暗再亮,周期 1.5 秒、相位差 0.17 秒,像梭子来回穿线;「呼吸」整体 2.2 秒明暗。动画由定时换帧实现(≤ 8 帧 / 秒),系统「减弱动态效果」开启时停在静态图。', cards(2,
      `<div class="usage-card">${cap('织线流动 · 关键帧')}<div class="f g4 ai">${[0, 1, 2].map(k => `<div class="fc ai g2"><span class="mb-chip l" style="min-width:56px;height:34px">${tray({ state: 'running', z: 26, frame: k })}</span><span class="t3 fs-cap" style="white-space:nowrap">第 ${k + 1} 帧</span></div>`).join('')}<span class="grow"></span><div class="fc ai g2"><span class="mb-chip l" style="min-width:56px;height:34px">${tray({ state: 'running', z: 26 })}</span><span class="t3 fs-cap" style="white-space:nowrap">实时(当前设置)</span></div></div></div>`,
      `<div class="usage-card">${cap('其它样式的动效对象')}${tbl([['样式', 80], ['动起来的是']], [['织纹', '三根纬线依次明暗'], ['提示符', '光标闪烁'], ['棋盘', '浅色方块逐个明暗'], ['行', '被高亮的片段依次明暗'], ['花括号', '中间的线结呼吸']])}</div>`))}

    ${sec('点开之后 · 菜单', '菜单头一行是总览,接着会话按项目分组(灰色组名 + 每行只有「小标记 + 标题」,最多 6 个,标题截到约 24 字,菜单整体不超过 236px 宽),下面才是固定命令(含「重启 MiniCode」:有运行中的会话或未保存文件时先确认,不改真实 / 演示记忆)。待确认的会话排最前。行首标记:运行中 = 四点、待确认 = 橙点、未读 / 刚完成 = 蓝点,看过的会话留空占位让文字左缘对齐。', cards(2,
      `<div>${cap('浅色菜单栏')}<div class="wall wall-l" style="padding:12px 12px 16px;display:flex;flex-direction:column;align-items:flex-end;gap:6px">${mbar(0, 'waiting', { sel: 1, col: 1, n: '1' })}${menu(0)}</div></div>`,
      `<div>${cap('深色菜单栏')}<div class="wall wall-d" style="padding:12px 12px 16px;display:flex;flex-direction:column;align-items:flex-end;gap:6px">${mbar(1, 'running', { sel: 1 })}${menu(1)}</div></div>`))}
    ${cards(2, `<div class="usage-card">${cap('图标右侧的数字(可选)')}<div class="f g3 ai"><span class="mb-chip l">${tray({ state: 'waiting', z: 20, col: 0 })}<span>1</span></span><span class="mb-chip d">${tray({ state: 'running', z: 20 })}<span>3</span></span><span class="mb-chip l">${tray({ state: 'idle', z: 20 })}</span></div>${why('数字 = 待处理数(权限 + 提问),0 时隐藏;只在「待确认」或「运行中」时出现。关闭后菜单栏只剩图标,最省位置。')}</div>`,
      `<div class="usage-card">${cap('点击行为')}${tbl([['操作', 100], ['默认'], ['可选']], [['单击', '弹出菜单', '直接打开主窗口'], ['⌥ 单击', '打开主窗口', '弹出菜单'], ['右键', '弹出菜单', '—'], ['⌘ 拖出', '移除图标', '(系统行为,不拦截)']])}</div>`)}

    ${sec('文件类型图标', '为 MiniCode 自己的文件准备三枚文档图标:纸页 + 小图形 + 扩展名。图形取当前样式、固定用浅色(文档图标不随主题变)。dev.sh 已在 Info.plist 注册三种类型,双击 / 拖到 Dock 图标 / <span class="kcode">open -a</span> 都会交给应用。', `<div class="doc-row">${[['.mcworkspace', '多根工作区', 'MCWS'], ['.mccanvas', '画布文档', 'CANVAS'], ['.mcsettings', '设置备份(不含密钥)', 'CONF']].map(([ext, n, lab]) => `<figure>${docSvg(lab, { z: 108 })}<figcaption><b>${n}</b><br><code>${ext}</code></figcaption></figure>`).join('')}<div class="grow" style="min-width:300px">${tbl([['做法', 96], ['说明']], [['声明', 'Info.plist 里 <span class="kcode">UTExportedTypeDeclarations</span> + <span class="kcode">CFBundleDocumentTypes</span>,图标放进包内 Resources'], ['双击', '工作区 → 打开并加入侧栏(并切进该工作区);画布 → 导入为一张新画布并打开画布页;设置备份 → 弹导入确认,确认后才合并'],
        ['格式', '三种都是 JSON:工作区 = 一条工作区记录(名称 / 文件夹 / CLAUDE.md 开关);画布 = 与 <span class="kcode">~/.minicode/canvases/</span> 里单张画布同格式(含版本历史);设置备份 = 偏好键值,<b>导出时剔除密钥类键</b>'],
        ['出口', '画布列表行「···」→ 导出为 .mccanvas;设置 → 高级 → 导出 = 写到「下载」并在访达里显示;工作区文件可由工作区「···」导出'],
        ['失败与冲突', '文件不存在 / 不是 JSON / 太大(画布 > 8 MB)→ toast 写明原因,不动现有数据;同名画布不覆盖,另存为新画布;工作区里不存在的文件夹被跳过并提示,全都不存在则拒绝;同一组文件夹已有工作区 → 直接切过去;应用还没就绪(冷启动双击)→ 排队,窗口出来后再处理'], ['小尺寸', '≤ 32px 去掉扩展名文字,只留纸页与图形']])}</div></div>`)}

    ${sec('应用内的品牌用法', '应用内不再用一个通用的「&lt;/&gt;」符号当标志,统一用应用图标本身;小尺寸自动换「经纬」。下面几处都用设计令牌,跟随主题。', cards(3,
      `<div class="usage-card">${cap('标题栏 / 侧栏小标', '18 px')}<div class="f ai g3" style="height:34px;padding:0 12px;border-radius:var(--r-md);background:var(--bg-chrome);box-shadow:inset 0 0 0 1px var(--bd-1)"><span class="tl"><i></i><i></i><i></i></span>${B.logo(18)}<b>router-demo</b><span class="t4 fs-cap">main</span></div>${why('图标本体 18 px、无投影无高光;和标题文字同一基线,不加背景块。')}</div>`,
      `<div class="usage-card" style="align-items:center;text-align:center">${cap('欢迎页', '64 px · 光晕取强调色')}<span class="halo" style="margin:14px 0 10px">${B.logo(64)}</span><b class="fs-head">今天想做点什么?</b><span class="t3 fs-cap">只在没有任何内容的新会话里出现一次</span></div>`,
      `<div class="usage-card">${cap('加载 / 空状态', '同源动画')}<div class="brand-load">${tray({ s: 'weave', state: 'running', z: 22, anim: 'flow' })}<span class="t2">正在启动 claude…</span></div><div class="fc ai g2" style="margin-top:8px;color:var(--tx-4)">${tray({ s: 'weave', z: 44 })}<span class="t3 fs-cap">还没有会话</span></div>${why('织线动画就是菜单栏「运行中」的那一套,用来替代通用转圈;空状态用 25% 的单色图形,不抢内容。')}</div>`))}
    ${cards(2, `<div class="usage-card">${cap('锁定版式 · 横 / 竖')}<div class="f g6 ai" style="gap:36px;flex-wrap:wrap"><span class="lockup">${app({ z: 56 })}MiniCode</span><span class="lockup v">${app({ z: 76 })}MiniCode</span></div>${why('字标用系统 SF Pro Semibold、字距 −2.5%,不另造字体。图标与字标间距 = 图标本体宽的 1/8;安全留白 = 图标本体宽的 1/4。')}</div>`,
      `<div class="usage-card">${cap('不要这样用')}${tbl([['错误', 150], ['原因']], [['给图标再加一圈描边 / 背景块', '系统已有图形遮罩;重复会变成「图标里套图标」'], ['把线头裁掉', '线头是识别点,去掉就只剩一个格子'], ['拉伸或改比例', '连续圆角与线宽比例会失真'], ['在图标上叠文字', '名字交给标签;状态交给角标'], ['用彩色图标做菜单栏主体', '模板图才能适配深浅与选中态']])}</div>`)}

    ${sec('设置里怎么换', '入口在 设置 → 外观 → 应用图标 / 菜单栏图标(一项配置只有一个家;通用里的「菜单栏图标」只放一个跳转)。选了立刻生效,不需要重启。', `<div class="f g3" style="margin-bottom:10px">${btn('打开 设置 → 外观', 'pri', 'arrow', 'data-act="settab" data-tab="appearance" data-goto="settings"')}<span class="t3 fs-cap" style="align-self:center">里面的预览(浅 / 深壁纸上的 Dock + 16 / 24 / 40 px)会随选项实时变化</span></div>
      ${tbl([['选项', 170], ['影响哪里', 190], ['macOS 实现', 260], ['要注意']], [
        ['图标样式 / 外观 / 配色', 'Dock、⌘Tab、应用内的所有标志', '<span class="kcode">NSApp.applicationIconImage = …</span>,启动时按偏好套用一次,之后即时生效', '退出后 Dock 里「保留」的那一格会退回包内默认图标;通知中心、访达、启动台仍显示包内图标'],
        ['同时更换访达里的图标', '访达、启动台、关于本机', '<span class="kcode">NSWorkspace.setIcon(_:forFile:)</span> 给 MiniCode.app 写自定义图标', '应用更新(替换 .app)后需重新套用;默认关闭;只改本机'],
        ['Dock 角标 / 弹跳 / 进度', 'Dock 图标', '<span class="kcode">dockTile.badgeLabel</span> · <span class="kcode">requestUserAttention</span> · 自绘 <span class="kcode">dockTile.contentView</span>', '应用在前台时系统不弹跳;进度只在下载更新等「长于 2 秒」的操作时出现'],
        ['菜单栏:显示 / 样式 / 状态', '菜单栏图标', '<span class="kcode">NSStatusItem.button.image</span>,<span class="kcode">isTemplate = true</span>', '开关与「⌘ 拖出移除」同步到偏好;样式即时换图'],
        ['菜单栏:动画', '运行中状态', '定时换帧(≤ 8 fps),不用 CAAnimation(模板图要换 image)', '「减弱动态效果」或省电模式下停在静态图'],
        ['菜单栏:彩色徽标', '待确认 / 完成 / 出错 / 更新', '合成非模板图,<span class="kcode">NSApp.effectiveAppearance</span> 变化时重绘', '「减弱透明度 / 增强对比度」时回退为纯模板']])}`)}

    ${sec('无障碍与系统设置', '', tbl([['场景', 190], ['行为'], ['状态']], [
      ['减弱动态效果', '托盘:trayAnimStatic→静止帧;Dock:减弱动态时 maybeDockBounce 直接跳过(角标仍 syncDockBadge,不会「只弹角标」替代);应用内加载仍 kit.Spin,未接静态织纹', stt('part')],
      ['减弱透明度 / 增强对比度', 'dockIconSuffix 透明/着色在 a11y 降低透明度或增强对比度时退回实色;trayWantsColor 同期关彩色徽标;未测试', stt('ok')],
      ['VoiceOver', '托盘 ToolTip + postA11yAnnouncement(mac_a11y_darwin)已有;NSStatusItem 的 accessibilityLabel 仍待 MyGo 暴露', stt('part')],
      ['色弱', '所有状态靠形状区分(! / × / ↓ / 圆点 / 圆环 / 暂停符),颜色只是第二通道', stt('ok')],
      ['系统深浅切换', 'appIconMode=自动 时 syncBrand→systemIsDark→setDockIconFromPrefs;retheme 也会 syncBrand;未测试', stt('ok')]]))}

    ${sec('资源与导出', '设计稿里的图标是 SVG 现场生成的;落到产品里要导出成系统认的格式。仓库里 assets/icon/ 已有第一版流水线,按下表扩展。', tbl([['产物', 190], ['规格', 210], ['来源'], ['状态']], [
      ['MiniCode.icns / MiniCode-dark.icns', '16–512 @1x/@2x(含 1024)', '<span class="kcode">gen.py</span> → <span class="kcode">render.py</span> → <span class="kcode">iconutil</span>(现有)', stt('ok')],
      ['app-light.png / app-dark.png', '1024,Dock 运行时随主题切换用', '同上(现有)', stt('ok')],
      ['其余 5 种样式 × 4 种外观', '1024 PNG + .icns;透明 / 着色 app-style*-clear|tint', '<span class="kcode">gen.py</span> + <span class="kcode">build_all.sh</span> 已生成', stt('ok')],
      ['tray-*.png', '36×36 模板八态 + running 三帧', 'build_all.sh 已生成', stt('ok')],
      ['文档图标 .icns', '16–512', 'doc-mcworkspace / mccanvas / mcsettings', stt('ok')],
      ['MiniCode.icon(Icon Composer)', '背景 / 经线 / 纬线三层', 'assets/icon/layers/*.svg;未打包 .icon(需 Icon Composer)', stt('part')]]))}

    ${sec('实现状态', '如实标出 Go 版(feat/mygo 分支)现在做到哪一步,避免设计稿与代码对不上。', tbl([['能力', 240], ['现状'], ['状态']], [
      ['应用图标:织纹 浅 / 深 + .icns', 'brandassets 内嵌 + dockIconData 兜底磁盘/.icns;默认 MiniCode.icns / MiniCode-dark.icns', stt('ok')],
      ['运行时 Dock 图标随主题 / 系统深浅', 'retheme→syncBrand;appIconMode 自动=0 跟系统', stt('ok')],
      ['图标样式 / 外观 / 配色偏好', 'app-style{0-5}[-dark|-clear|-tint].icns', stt('ok')],
      ['Dock 角标 / 弹跳 / 进度', 'syncDockBadge(SetBadge)·maybeDockBounce(仅 EvPromoteToPermission)·dockSetProgress(下载/偏好开时回合 0.35);问答待确认不弹跳;进度偏好默认关', stt('part')],
      ['菜单栏图标(<span class="kcode">general.tray</span> 开关)', '五样式 × 八态 PNG 内嵌进二进制(不依赖工作目录)+ running 三帧 + 彩色徽标(随菜单栏深浅重绘);设置里有真实预览与八态条', stt('ok')],
      ['托盘状态 / 动画 / 菜单 / 数字', 'brandSummarize 优先级与八态内嵌素材(样式 0–4)已接;出错/已完成 notice_sys+前台清除;MINI_TRAY 八种图标已 Computer Use 截图核对;菜单/暂停/数字/单击偏好已接;缺口:非织纹样式无设计稿专属运行动效(仍用 running 三帧/静态图);应用内无品牌加载动画;未测菜单项图标库不渲染(用 emoji 状态点)', stt('part')],
      ['应用菜单栏(MiniCode / 文件 / 编辑 / 视图 / 窗口 / 帮助)', '标准项走系统角色(关于 / 服务 / 隐藏 / 退出 / 编辑 / 窗口 / 全屏);自定义项不挂快捷键,快捷键仍由应用内按用户设置分发;新建窗口 ⇧⌘N', stt('ok')],
      ['文件类型图标与关联', 'dev.sh Info.plist UTType + 文档 .icns;OnOpenFile 接 open_external.go:.mcworkspace 导入并进入 / .mccanvas 导入成新画布 / .mcsettings 弹确认后合并(剔除密钥键);冷启动先排队;出口:画布行 ··· / 工作区右键 / 设置 → 高级导出;go test 覆盖;已用 Computer Use + `open -a` 在开发版实测:画布双击导入、设置备份确认 → 导入(密钥键被忽略)、工作区导入并进入(缺失文件夹跳过并提示);Finder 双击图标与冷启动排队未实测', stt('part')],
      ['应用内小标统一用应用图标', '欢迎 64px / 关于 96px brandLogo、IDE 空态 brandEmptyMark(托盘样式·25% 单色)已接;标题栏/侧栏 18px 小标未接(titlebar 仍无 brandLogo);未测试', stt('part')]]))}
    </div>`;
})();
