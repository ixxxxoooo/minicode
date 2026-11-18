/* 设计稿 v4.2 · 流光主题(液态玻璃):设计原则、分层、令牌、材质样本、降低透明度、性能预算、Go 映射
   依据:macOS 26 Liquid Glass(NSGlassEffectView)的做法 + MyGo 官方 glass 插件的能力(Regular / Clear 两种材质、Tint、Interactive、Clarity;着色器画折射)。
   性能一栏是读 MyGo 源码(ui/material.go、internal/scene/damage.go)得出的推断,Go 版落地后要用 diag.SpanSlow 实测再改写;页面最下方有「实现状态」。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, tag, kbd, seg, tog, inp, THEMES } = U;
  const { head, sec } = P.h;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k ? 'class="t2"' : ''}>${k ? c : `<b>${c}</b>`}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const frame = (theme) => `<div class="gl-wide"><div class="fit"><div class="frame" data-scheme="b" data-theme="${theme}">${W.win({ scheme: 'b', mode: 'agent', screen: 'work', theme })}</div></div></div>`;

  /* ───────── 样本台:壁纸 + 一组控件;theme 决定浅 / 夜,extra 加 no-glass 演示降级 ───────── */
  const specimens = (theme, extra = '') => `<div class="gl-tile ${extra}" data-theme="${theme}" data-scheme="b">
    <div class="gl-lb">按钮 · 输入 · 开关</div>
    <div class="gl-row">${btn('默认', '', 'file')}${btn('主要', 'pri', 'play')}${btn('强调柔底', 'soft', 'spark')}${btn('幽灵', 'ghost')}${ib('gear', '', '设置')}${tog(1)}${tog(0)}</div>
    <div class="gl-row">${seg(['会话', '文件', '搜索'], 0, '', 'noop')}${inp('搜索会话、文件与命令', { icon: 'search', style: 'width:220px' })}${tag('已实现', 'ok')}${tag('部分', 'warn')}${kbd('⌘', 'K')}</div>
    <div class="gl-lb">选中行 · 卡片</div>
    <div class="gl-row" style="align-items:stretch"><div class="fc g1" style="width:190px"><div class="row">${ic('chat', 's12')}<span class="ell">路由大小写匹配</span></div><div class="row on">${ic('chat', 's12')}<span class="ell">流式输出卡顿排查</span></div><div class="row">${ic('chat', 's12')}<span class="ell">升级依赖</span></div></div>
      <div class="card" style="flex:1;min-width:200px;padding:10px 12px"><b>读取 3 个文件</b><div class="t3 fs-cap" style="margin-top:2px">src/router.rs · tests/router_test.rs</div></div></div>
    <div class="gl-lb">浮层 · 弹出菜单</div>
    <div class="gl-row" style="align-items:flex-start"><div class="pop" style="width:200px"><div class="mi on">${ic('file', 's12')}新建会话<span class="grow"></span>${kbd('⌘', 'N')}</div><div class="mi">${ic('folder', 's12')}打开项目…</div><div class="mi">${ic('gear', 's12')}设置</div></div>
      <div class="pop" style="flex:1;min-width:220px;padding:10px 12px"><div class="f ai g3"><b class="grow">发现新版本 0.3.9</b>${tag('42%', 'ac')}</div><div class="meter" style="margin:7px 0"><i style="width:42%"></i></div><div class="t3 fs-cap">正在后台下载,完成后提示重启。</div></div></div>
    <div class="cmp" style="margin-top:2px"><div class="cmp-in">给 Claude 发消息…</div><div class="cmp-bar"><span class="cfg">${ic('plus', 's12')}</span><span class="grow"></span>${ib('arrow', 'send', '发送')}</div></div></div>`;

  /** 三层结构示意:壁纸 → 内容层 → 控件层,逐层叠上去 */
  const stack = (theme) => `<div class="gl-pair3">
    ${[['① 壁纸层', '静态渐变,整窗只画一次。玻璃的「颜色」都来自它。', ''],
      ['② 内容层', '导航、对话、工作区是大块玻璃:更实(厚),文字与代码画在它上面,背后只有壁纸。', '<div class="pane chat" style="position:absolute;inset:18px 18px 18px 18px;border-radius:var(--r-lg);box-shadow:var(--glass-sh)"><div style="padding:14px"><b>会话内容</b><div class="t3 fs-cap" style="margin-top:3px">文字在玻璃之上,不参与模糊</div></div></div>'],
      ['③ 控件层', '输入框、按钮、弹层是小块、更透的玻璃,浮在内容之上;滚动内容会从它们下面穿过。', '<div class="pane chat" style="position:absolute;inset:18px;border-radius:var(--r-lg);box-shadow:var(--glass-sh)"><div style="padding:14px"><b>会话内容</b></div></div><div class="cmp" style="position:absolute;left:34px;right:34px;bottom:30px"><div class="cmp-in" style="min-height:0;padding:7px 12px 0">给 Claude 发消息…</div><div class="cmp-bar"><span class="grow"></span>' + ib('arrow', 'send', '发送') + '</div></div>']]
      .map(([t, d, inner]) => `<div><div class="gl-stk" data-theme="${theme}" data-scheme="b">${inner}</div><div style="padding:8px 2px 0"><b>${t}</b><div class="t3 fs-cap" style="line-height:1.55;margin-top:2px">${d}</div></div></div>`).join('')}</div>`;

  /** 令牌表的色块:壁纸上叠一层玻璃色,看得出透明度 */
  const sw = (v) => `<span class="gl-sw" data-theme="glow"><i style="background:var(${v})"></i></span><span class="gl-sw" data-theme="glow-night"><i style="background:var(${v})"></i></span>`;
  const tokRow = (v, name, use) => [`<code>${v}</code>`, name, sw(v), use];

  P['glow-spec'] = (st) => `<div class="ds" style="max-width:1240px">${head('流光 · 液态玻璃主题', '第七、八套主题,同一份令牌 + 一层独立的材质样式(css/glass.css),不改动其余六套。配色取 Claude 品牌色:浅色「流光」是象牙白壁纸 + 陶土橙强调色,「流光·夜」是暖石板灰壁纸 + 柔和的陶土橙;壁纸不用任何彩色;推荐配方案 B(浮岛),方案 A / C 也成立。点顶栏最右两枚圆点即可整稿切换。')}
    ${sec('整体 · 浅色「流光」', '壁纸是象牙白底,只在明度上做一点暖沙 / 牛皮纸色的深浅变化,外加一道斜向的高光带(给折射一条明暗边缘),整体淡雅、没有别的色相;面板是浮在壁纸上的玻璃岛,标题栏与状态栏直接坐在壁纸上,只有控件是玻璃。', frame('glow'))}
    ${sec('整体 · 夜「流光·夜」', '同一套结构。壁纸换成暖石板灰底,只带一点极淡的陶土橙微光;玻璃变成暖褐色半透明,边缘高光变弱,避免发白。', frame('glow-night'))}
    ${sec('设计原则 · 三层', '玻璃是「导航与控件」的材质,不是内容的材质。这条决定了好不好看,也决定了性能。', `${stack('glow')}<div style="height:16px"></div>${stack('glow-night')}`)}
    ${sec('材质令牌', '所有玻璃都只读这一组变量;组件规则里没有写死的透明度。色块画在壁纸上,能看出透明度。', tbl([['令牌', 230], ['名称', 90], ['浅 / 夜', 110], ['用途']], [
      tokRow('--glass-1', '常规', '侧栏、工作区面板(导航与工作区更透)'), tokRow('--glass-2', '厚', '对话 / 整页面板、浮层、输入框(文字多,更实)'),
      tokRow('--glass-3', '薄', '按钮、输入框、分段槽、标题栏搜索胶囊'), tokRow('--glass-card', '卡片', '面板里的卡片、起手卡'), tokRow('--glass-sel', '选中', '选中行、分段滑块、悬停按钮'),
      ['<code>--glass-blur / --glass-sat</code>', '模糊 · 饱和', '26 · 1.7 / 30 · 1.6', '面板;小控件固定 12–16px。弹层再加 6px'],
      ['<code>--glass-rim / -lo / --glass-hi</code>', '边缘高光', '1px 渐变描边', '左上亮、右下暗;顶部再加 1px 内高光。夜色下降到 10%–34%'],
      ['<code>--glass-sheen</code>', '顶部光泽', '26% 高度的白色渐变', '让玻璃有「厚度」,不是平的半透明'],
      ['<code>--glass-sh / --glass-sh-s</code>', '悬浮阴影', '大 / 小两档', '只在方案 B 的面板上出现;A / C 面板相接,只留细线'],
      ['<code>--ac-glow</code>', '强调光晕', '强调色 70% / 55%', '主要按钮与发送键下方的一圈光,是页面里唯一的「发光」'],
      ['<code>--wall</code>', '壁纸', '5 团径向色晕 + 斜向高光带 + 底色', '低饱和、静态;不做漂移动画(见性能)']]))}
    ${sec('材质样本', '同一组控件分别落在浅色与夜色壁纸上。注意:主要按钮是着色玻璃(强调色几乎不透明,只透出一点);强调色用 Claude 的陶土橙(浅 #BD5A36,由品牌色 #D97757 压深一档让白字够对比;夜 #E8967A),整套保持淡雅。', `<div class="gl-pair">${specimens('glow')}${specimens('glow-night')}</div>`)}
    ${sec('交互状态', '沿用现有时长与曲线,只加了玻璃自己的反馈。', tbl([['状态', 90], ['表现'], ['时长', 110]], [
      ['悬停', '薄玻璃升为「选中」玻璃(更亮);主要按钮亮度 +6%', '120 ms'], ['按下', '整体缩到 0.96;Go 版交互玻璃按 AppKit 的做法左右各胀约 1.1 DIP', '80 ms'],
      ['焦点', '强调色 3px 外环 + 强调色内描边,玻璃本身不变', '120 ms'], ['选中', '换成「选中」玻璃 + 高光边,不再用强调色底,避免和主要按钮抢色', '180 ms'],
      ['禁用', '整体 45% 不透明度,高光与光晕一并去掉', '—'], ['出现 / 退场', '弹层从触发点轻微放大淡入;模糊在动画结束后才完全到位,避免第一帧发白', '180 / 120 ms']]))}
    ${sec('降低透明度(系统辅助功能)', '系统开了「降低透明度」或「增强对比度」,整套玻璃令牌换成不透明兜底:同一组变量,组件规则一行不改。左:正常;右:降级后。', `<div class="gl-pair">${specimens('glow')}${specimens('glow', 'no-glass')}</div>`)}
    ${sec('性能预算(按源码推断,待实测)', 'MyGo 的玻璃在 Metal 着色器里画:抓取背后画面 → 高斯模糊 → 折射着色。成本跟「玻璃面积」成正比,而且玻璃背后只要有东西在变,它就要重算(damage.go 会把玻璃背后的采样区并进重绘范围)。', tbl([['区域', 150], ['材质', 90], ['背后会变吗', 150], ['成本', 70], ['策略']], [
      ['壁纸', '静态渐变', '不会', '低', '不做漂移动画;Go 版画成一张缓存位图'],
      ['侧栏 / 对话 / 工作区', '常规 · 厚', '不会:文字画在玻璃之上,背后只有壁纸', '中', '面板互不重叠、背后只放壁纸 → 流式输出和终端刷新不会触发它们重新模糊'],
      ['输入框(浮在会话流上)', '厚', '会:会话流滚到它下面', '中', '面积小(约 700×110),可接受;滚动时不降级'],
      ['标题栏 / 按钮 / 分段', '薄', '不会', '低', '模糊半径固定 12–16px'],
      ['命令面板 / 弹层 / 提示条', '厚', '很少', '低', '短暂出现;玻璃不叠玻璃超过两层(面板 + 面板内弹层)'],
      ['代码块 / 终端 / 差异', '不透明', '—', '无', '内容层不用玻璃,可读性与帧率都不吃亏']]) + '<p class="t3" style="margin-top:8px">预算的数字不在设计稿里编:Go 版落地后,在「流式输出 + 终端刷新」同时进行的场景下用 diag.SpanSlow 测帧耗时,再决定能铺多大面积、要不要给低配机器一个「简化玻璃」开关。</p>')}
    ${sec('Go 实现映射', '不用 WebView,用 MyGo 官方 glass 插件(github.com/egoist/mygo/plugins/glass),着色器版而非 Native:Native 采样的是窗口背后的桌面,看不到应用自己画的壁纸。', tbl([['设计稿', 200], ['MyGo'], ['说明']], [
      ['面板 · 厚 / 常规', '<code>.Material(glass.Glass{Style: glass.Regular, Clarity: 0 … 0.3})</code>', '大面板自带更强的磨砂;导航与工作区用较大的 Clarity,更通透'],
      ['按钮 / 控件 · 薄', '<code>glass.Glass{Interactive: true}</code>', '按下微胀、悬停亮边,与设计稿交互一致'],
      ['主要按钮', '<code>glass.Glass{Tint: accent}</code> 或实色渐变', '先试 Tint;不够压得住就用实色渐变 + 高光'],
      ['壁纸', '<code>ui.Canvas</code> 画一次缓存,或 <code>shader</code> 径向渐变', '玻璃采样的就是它'],
      ['降低透明度', '<code>p.Preferences().ReduceTransparency</code>', '插件自己会降级;令牌层同时换成不透明色板'],
      ['色板', '<code>theme.Palettes</code> 增加 glow / glow-night', 'Palette 只有 13 个颜色;玻璃的透明度由插件决定,所以设计稿的 --glass-* 数值只作视觉参照,落地时对着实测截图微调']]))}
    <p class="t3">CSS 的 backdrop-filter 做不出边缘透镜折射,所以设计稿里的玻璃比 Go 版少一层「弯曲」;其余(磨砂、饱和、高光边、顶部光泽、悬浮阴影)是同一套。</p></div>`;
})();
