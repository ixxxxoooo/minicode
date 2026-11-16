/* 设计稿 v3 · 规范补全:交互状态矩阵、动效(时长 / 模式 / 过渡编排 / 减弱动态)、补充浮层(账号菜单、重启条、快捷键录制、提示条家族…)
   在 specs.js / overlays.js 之后加载:覆盖 P.motion 与 P.demos,并把 P.ovExtra() 接到 P.overlays 末尾。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, kbd, tog, seg, chk, inp } = U;
  const { head, sec, spec } = P.h;
  const dash = '<span class="t4">—</span>';
  const cell = (h) => `<div><div class="st-cell">${h}</div></div>`;

  /* ───────── 交互状态矩阵 ───────── */
  const STATES = ['默认', '悬停', '按下', '聚焦', '选中', '禁用', '加载', '错误'];
  const ROWS = [
    ['按钮', [btn('保存'), btn('保存', 'is-hover'), btn('保存', 'is-active'), btn('保存', 'is-focus'), dash, btn('保存', 'is-disabled'), btn('保存', 'loading'), btn('删除', 'danger')]],
    ['主按钮', [btn('应用', 'pri'), btn('应用', 'pri is-hover'), btn('应用', 'pri is-active'), btn('应用', 'pri is-focus'), dash, btn('应用', 'pri is-disabled'), btn('应用', 'pri loading'), dash]],
    ['图标按钮', [ib('copy', ''), ib('copy', 'is-hover'), ib('copy', 'is-active'), ib('copy', 'is-focus'), ib('copy', 'on'), ib('copy', 'is-disabled'), ib('refresh', ''), ib('alert', 'c-err')]],
    ['胶囊', [chip('Sonnet', '', ''), chip('Sonnet', '', 'is-hover'), chip('Sonnet', '', 'is-active'), chip('Sonnet', '', 'is-focus'), chip('Sonnet', '', 'on'), chip('Sonnet', '', 'is-disabled'), dash, dash]],
    ['开关', [`<span class="tog"></span>`, dash, dash, `<span class="tog is-focus"></span>`, `<span class="tog on"></span>`, `<span class="tog on is-disabled"></span>`, dash, dash]],
    ['输入框', [`<span class="inp" style="width:100%">${ic('search', 's12')}<span class="t4">搜索</span></span>`, `<span class="inp is-hover" style="width:100%">${ic('search', 's12')}<span class="t4">搜索</span></span>`, dash, `<span class="inp is-focus" style="width:100%">${ic('search', 's12')}<span>router</span></span>`, dash, `<span class="inp is-disabled" style="width:100%">${ic('search', 's12')}<span class="t4">搜索</span></span>`, dash, `<span class="inp err" style="width:100%"><span>/Users/x</span></span>`]],
    ['列表行', [`<div class="row">${ic('file', 's12')} router.rs</div>`, `<div class="row is-hover">${ic('file', 's12')} router.rs</div>`, dash, `<div class="row is-focus">${ic('file', 's12')} router.rs</div>`, `<div class="row on">${ic('file', 's12')} router.rs</div>`, `<div class="row" style="opacity:.42">${ic('file', 's12')} router.rs</div>`, dash, dash]],
    ['标签', [`<span class="tab">改动</span>`, `<span class="tab is-hover">改动</span>`, dash, `<span class="tab is-focus">改动</span>`, `<span class="tab on">改动</span>`, dash, dash, dash]],
  ];
  const matrix = `<div class="spec"><div class="st-grid"><div class="h"></div>${STATES.map(s => `<div class="h">${s}</div>`).join('')}${ROWS.map(([n, cs]) => `<div class="r">${n}</div>${cs.map(cell).join('')}`).join('')}</div></div>`;

  const RULE = [
    ['悬停', '背景叠 <span class="mono">--fill-1</span>(文字升到 tx-1);卡片再上浮 1px 并加深阴影。touch 设备不设悬停。', '120ms'],
    ['按下', '缩到 96%,松开回弹(<span class="mono">--ease-spring</span>);开关与分段只动滑块,不缩放整体。', '80ms'],
    ['聚焦', '只在键盘聚焦(:focus-visible)出现 3px 聚焦环 + 强调色内描边;鼠标点击不出环。输入框聚焦始终显示。', '120ms'],
    ['选中', '强调色淡底(<span class="mono">--ac-soft</span>)+ 强调色文字;列表行选中不改高度,只改底色。', '120ms'],
    ['禁用', '整体 42% 透明、不响应指针;悬停给 tooltip 说明原因(例:「没有未保存的更改」)。不使用「灰色但可点」。', '—'],
    ['加载', '按钮保持原宽,图标换成转圈、文字不变;列表用骨架;超过 400ms 才显示,免得闪一下。', '800ms 旋转'],
    ['错误', '描边与淡光环换成 err 色,下方一行 12px 说明;提交时再加 320ms 抖动,焦点回到出错字段。', '320ms'],
  ];
  const rules = `<table class="ia-tbl"><thead><tr><th style="width:80px">状态</th><th>规则</th><th style="width:90px">时长</th></tr></thead><tbody>${RULE.map(([a, b, c]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td><td class="t3 mono">${c}</td></tr>`).join('')}</tbody></table>`;

  const FB = [
    ['设置项被改动', '行左侧出现强调色圆点;悬停显示「还原」;顶部「未保存」不存在 —— 设置即时生效,没有保存按钮'],
    ['需要重启的项', '底部固定条「N 项更改需要重启后生效 · 稍后 / 立即重启」;点「稍后」只对当前这批收起,再改一项会重新出现'],
    ['删除一行(规则 / 变量 / 条目)', '先折叠到 0 高度再移除,同时底部提示「已删除 · 撤销」,6 秒内可撤销,撤销后原位展开'],
    ['不可逆的删除', '弹确认框(红色主按钮);确认框默认焦点在「取消」'],
    ['校验失败', '字段描边变红 + 下方文案 + 抖动;不弹提示条,不清空输入'],
    ['异步动作(测试连接 / 检查更新)', '按钮进入加载 → 结果以行内状态点 + 文案呈现;失败可「重试」,不用提示条打断'],
    ['复制', '按钮文字变「已复制」并带对勾,1.5 秒后还原;不弹提示条'],
    ['后台完成的事(下载 / 安装)', '右下角提示条,带进度;完成变成带动作按钮的提示(重启 / 打开)'],
    ['定位到某处(搜索跳转、新建)', '目标行闪一次强调色底(1.4 秒),并滚动到可视区居中']];
  const fb = `<table class="ia-tbl"><thead><tr><th style="width:210px">发生了什么</th><th>怎么反馈</th></tr></thead><tbody>${FB.map(([a, b]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td></tr>`).join('')}</tbody></table>`;

  P.states = () => `<div class="ds">${head('交互状态', '每个控件都要有 8 个状态的答案:默认 / 悬停 / 按下 / 聚焦 / 选中 / 禁用 / 加载 / 错误。下表是强制显示的静态样子(真实界面走伪类),「—」表示该控件没有这个状态。')}
    ${sec('状态矩阵', '全部只引用语义令牌,所以浅色 / 深色 / 三套方案下自动成立。切换上方的方案与主题检查。', matrix)}
    ${sec('每个状态的规则', '一条规则对所有控件生效。', rules)}
    ${sec('反馈的选择', '同一类事件永远用同一种反馈 —— 用户不用学习。原则:能在原地说清楚的,不弹提示条;提示条只给「你已经看不到原地」的事。', fb)}
    ${sec('真实控件', '下面的控件是真的,可以悬停、点按、用 Tab 聚焦。', `<div class="spec"><div class="f g5 ai wrap">
      ${btn('主要', 'pri')}${btn('次要')}${btn('文字', 'ghost')}${btn('危险', 'danger')}${ib('gear', '')}${chip('Opus')}${tog(1)}${seg(['左右', '统一'], 0)}
      <label class="inp" style="width:180px">${ic('search', 's12')}<input class="cmp-input" placeholder="聚焦我"></label></div></div>`)}</div>`;

  /* ───────── 动效 ───────── */
  const track = (name, val, d, e) => `<div class="mrow"><span class="m">${name}</span><span class="t3 mono">${val}</span><div class="track" style="--d:${d};--e:${e}"><i></i></div></div>`;
  const loop = (inner, extra = '') => `<div class="demo-loop" ${extra}>${inner}</div>`;
  const NEW = [
    ['弹层退出 pop-out', '--dur-fast · --ease-in · 比进入快,不拖泥带水', () => loop(`<div class="pop" style="width:150px;animation:demo-pop 2.8s var(--ease-out) infinite"><div class="mi on">Sonnet 5.5</div><div class="mi">Opus 5.5</div></div>`), 1],
    ['提示条进出', '自右侧滑入 --dur-slow,3.8 秒后向右滑出', () => loop(`<div class="pop f g4 ai" style="padding:8px 11px;animation:demo-toast 3s var(--ease-out) infinite">${ic('okc', 'c-ok')}<b>已复制</b><span class="t3">版本信息</span></div>`), 1],
    ['删除折叠 collapse', '固定高度 → 折叠到 0 + 淡出;--dur-slow · --ease-io', () => loop(`<div style="width:200px"><div class="row" style="white-space:nowrap;overflow:hidden;animation:demo-rm 3s var(--ease-io) infinite">${ic('file', 's12')}Bash(npm run *)</div><div class="row">${ic('file', 's12')}Read(./.env)</div></div>`), 1],
    ['校验抖动 shake', '320ms · 描边变红 + 一行说明;点击重放', () => `<div class="a-shake" style="width:100%"><span class="inp err" style="width:100%">${ic('folder', 's12')}<span>/不存在的路径</span></span><div class="err-msg" style="margin-top:5px">${ic('alert', 's12')}路径不存在</div></div>`],
    ['定位闪烁 flash', '1.4 秒强调色底淡出;搜索跳转 / 新建项后', () => `<div class="a-flash row" style="width:100%">${ic('sliders', 's12')}<span class="grow">字号</span>${seg(['小', '中', '大'], 1)}</div>`],
    ['不确定进度 indeterminate', '1.2 秒循环,只在「不知道要多久」时用', () => loop(`<div class="meter indet" style="width:180px"><i></i></div>`), 1],
    ['抽屉进出 drawer', '--dur-slow · 自右侧位移 22px + 淡入', () => loop(`<div class="pop" style="width:140px;padding:10px;animation:demo-drawer 3s var(--ease-out) infinite"><b>编辑规则</b><div class="t3 fs-cap">右侧抽屉</div></div>`), 1],
    ['模态 modal', '遮罩淡入 + 卡片 pop;退出同时淡出', () => loop(`<div style="position:absolute;inset:0;background:var(--scrim);animation:demo-modal-scrim 3s infinite"></div><div class="dlg" style="width:170px;padding:10px 12px;position:relative;animation:demo-modal-card 3s var(--ease-out) infinite"><b>删除这个项目?</b><div class="t3 fs-cap">取消 · <b class="c-err">删除</b></div></div>`), 1],
    ['骨架 → 内容', '内容到达时骨架淡出、内容淡入,位置与高度不变', () => loop(`<div style="position:relative;width:180px;height:44px"><div class="fc g3" style="position:absolute;inset:0;animation:demo-skel 3.2s infinite"><i class="skel" style="height:12px;border-radius:4px"></i><i class="skel" style="height:12px;width:60%;border-radius:4px"></i></div><div class="fc" style="position:absolute;inset:0;animation:demo-real 3.2s infinite"><b>router.rs</b><span class="t3 fs-cap">src/ · 86 行</span></div></div>`), 1],
    ['拖起 lift', '拖动时上浮 3px + 阴影加深,落点处出现插入线', () => loop(`<div class="card" style="padding:7px 12px;animation:lift 2.4s var(--ease-io) infinite">${ic('timer', 's12')} 每日构建</div>`), 1],
    ['成功 ok-pop', '--ease-spring · 对勾从 40% 弹到 118% 再回 100%', () => `<div class="f ai g4"><span class="a-ok c-ok">${ic('okc', 's20')}</span><b>已更新到 0.2.4</b></div>`],
    ['柱状生长 grow', '图表进入时每根柱自下而上生长,逐根延迟 24ms', () => `<div class="bars" style="height:64px;width:100%;--n:10">${[40, 62, 35, 80, 55, 92, 48, 70, 30, 66].map((h, i) => `<i style="height:${h}%;animation:grow-y var(--dur-slow) var(--ease-out) ${i * 24}ms both"></i>`).join('')}</div>`],
  ];
  P.demos = P.demos.concat(NEW);

  const CH = [
    ['Agent ⇄ IDE', '传送带式整体平移:IDE → Agent 向左(对话栏移到中间,编辑区被挤出左缘、工作区自右缘移入);Agent → IDE 反过来向右', '<span class="mono">--dur-morph · --ease-morph</span>'],
    ['进入设置 / 整页功能', '整页内容整体淡入;设置里分组依次上升,间隔 30ms;返回时反向淡出', '<span class="mono">--dur-base</span>'],
    ['设置分类切换', '正文分组 stagger 上升;搜索结果即时替换,不做动画(边打字边变不该晃)', '<span class="mono">--dur-base</span>'],
    ['抽屉内容切换', '换选中项时内容就地淡入(a-swap),抽屉本身不再滑动', '<span class="mono">--dur-base</span>'],
    ['弹层 / 菜单', '进入 pop(自锚点缩放 96%);退出 pop-out,时长是进入的 2/3', '<span class="mono">--dur-base / --dur-fast</span>'],
    ['模态', '遮罩淡入 + 卡片 pop;Esc / 取消同时淡出;焦点圈在对话框里,关闭后还给触发者', '<span class="mono">--dur-base</span>'],
    ['提示条', '自右下角上滑进入,最多叠 3 条,超出时最早的先退;悬停暂停计时;错误不自动消失', '<span class="mono">--dur-slow</span>'],
    ['删除一项', '折叠 + 淡出 + 撤销提示;撤销时原位展开', '<span class="mono">--dur-slow</span>'],
    ['异步结果', '按钮加载 → 结果行内展示;成功的勾用 ok-pop', '<span class="mono">--dur-slow</span>'],
    ['图表', '进入时柱 / 折线生长一次,切范围时数据平滑过渡,不重播入场', '<span class="mono">--dur-slow</span>'],
  ];
  const chor = `<table class="ia-tbl"><thead><tr><th style="width:170px">场景</th><th>编排</th><th style="width:210px">令牌</th></tr></thead><tbody>${CH.map(([a, b, c]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td><td>${c}</td></tr>`).join('')}</tbody></table>`;

  const DO = ['动效只回答三件事:它从哪来、正在进行、已经完成', '进入用 ease-out,离开用 ease-in,位移用 ease-io', '离开比进入快(约 2/3)', '同一时刻最多一个「主」动画,其余是配角', '长列表只给首屏几行做进入,其余直接出现', '所有时长走令牌,关闭动效时一处归零'];
  const DONT = ['不用动效填充空白或「显得高级」', '不让用户等动画结束才能操作(除形态切换的 470ms)', '不循环播放装饰性动画;循环只给「正在进行」', '不在输入、打字、搜索结果上做动画', '不为同一件事叠两种过渡', '不写裸毫秒数或自造曲线'];
  const dd = (title, list, c) => `<div class="spec"><div class="lb"><span>${title}</span></div><div class="fc g3 t2" style="line-height:1.55">${list.map(x => `<div class="f g3"><span class="${c}">${ic(c === 'c-ok' ? 'check' : 'x', 's12')}</span><span>${x}</span></div>`).join('')}</div></div>`;

  P.motion = () => `<div class="ds">${head('动效', '动效只做三件事:说明「它从哪来」、提示「正在进行」、确认「已经完成」。时长 5 档、曲线 5 条,组件里不出现裸数字;一键关闭或跟随系统「减弱动态效果」。')}
    ${sec('时长与曲线', `所有过渡都是「一个时长令牌 + 一条曲线令牌」。${btn('重放', 'sm soft', 'play', 'data-act="tracks" style="margin-left:6px"')}`,
      `<div class="grid2"><div class="spec tracks">${[['--dur-instant', '80ms', '按压 / 勾选'], ['--dur-fast', '120ms', '悬停 / 收起 / 退出'], ['--dur-base', '180ms', '弹层 / 行进入 / 滑块'], ['--dur-slow', '260ms', '折叠 / 抽屉 / 提示条'], ['--dur-morph', '440ms', 'Agent ⇄ IDE']].map(([n, v, u]) => track(`${n}<span class="t3 fs-cap" style="font-weight:400;margin-left:6px">${u}</span>`, v, `var(${n})`, 'var(--ease-out)')).join('')}</div>
      <div class="spec tracks">${[['--ease-out', '进入'], ['--ease-in', '离开'], ['--ease-io', '位移'], ['--ease-spring', '滑块 / 勾选'], ['--ease-morph', '布局变形']].map(([n, u]) => track(`${n}<span class="t3 fs-cap" style="font-weight:400;margin-left:6px">${u}</span>`, '900ms', '900ms', `var(${n})`)).join('')}</div></div>`)}
    ${sec('动效模式', `共 ${P.demos.length} 个。组件只引用模式名(a-rise / a-pop / a-shake…),不直接写关键帧。带重放图标的点一下重播;循环的是为了看清楚离场。`, `<div class="grid3">${P.demos.map((_, k) => P.demo(k)).join('')}</div>`)}
    ${sec('沙漏等待', '全站进行中图标(步骤行 / 子代理 / 任务坞 / 状态栏)默认沙漏;按钮 loading 仍用小圆环。周期 2.4s:上腔 → 下腔细流,末 0.5s ease-in-out 翻转;多枚共享全局相位。', `<div class="grid2">
      ${spec('对比', ['.hg-wait', '.spin'], `<div class="f g5 ai">${ic('hourglass', 's16 hg-wait')}${ic('hourglass', 's16 hg-wait')}${ic('hourglass', 's16 hg-wait')}<span class="vsep"></span><span class="spin"></span><span class="t3">旧圆环仅保留给按钮 loading</span></div>`)}
      ${spec('设置', ['appearance.waitAnim'], `<div class="t2">设置 → 外观 → 等待指示样式:沙漏(默认) / 圆环 / 点阵 / 脉冲。Go:kit.Spin 画沙漏;kit.SpinRing 仅供按钮;WaitSpinStyle 迁移旧偏好 0→圆环。</div>`)}</div>`)}
    ${sec('流光扫光', '对标 Vercel AI Shimmer / Claude Code 状态行:约 100° 斜向高光带(宽 ≈35–40%),Tx3 底 + Tx1 峰,1.6s 匀速 + 0.5s 停;同一屏共享相位;仅唯一进行中文字;MyGo 无 background-clip:text → Go 用 RichText 分段混色(≤48 字,约 30fps)。', `<div class="grid2">
      ${spec('新 · 扫光', ['.shimmer'], `<div class="f g4 ai wrap"><span class="m shimmer">正在生成提交说明…</span><span class="m shimmer">思考</span></div>`)}
      ${spec('旧 · 整段呼吸(降级)', ['.shimmer-legacy'], `<div class="m shimmer-legacy">正在生成提交说明…</div><p class="t3 fs-cap">appearance.shimmer = 呼吸 / 无 时回落</p>`)}</div>`)}
    ${sec('思考跑马灯', '折叠且流式:右侧 NoWrap 预览(约 200 字尾部,换行压成 ·),左缘 32px 渐隐,文本向左缓动追赶。', `<div class="spec">${typeof F !== 'undefined' ? F.thinkMarquee({ stream: true }) : ''}</div>`)}
    ${sec('过渡编排', '多个元素同时变化时,谁先谁后、各自多久。', chor)}
    ${sec('减弱动态与关闭动效', '两个来源,效果一致:设置 → 外观 → 动效 = 关闭,或系统「减弱动态效果」。', `<div class="grid2">
      ${spec('会变成什么', ['body.no-motion', 'prefers-reduced-motion'], '<div class="t2" style="line-height:1.7">· 所有时长令牌归 1ms:状态直接跳到终态,没有位移与缩放<br>· 循环动画停止:流光变成静态文字色、呼吸描边变成实线、不确定进度条变成静态条<br>· 形态切换仍然成立,只是瞬时完成;<span class="mono">.morphing</span> 锁定随之缩到 1 帧<br>· 状态的「含义」不靠动画传达:运行中始终有状态点与文字</div>')}
      ${spec('怎么验证', ['QA'], '<div class="t2" style="line-height:1.7">在「外观」里把动效切到「关闭」:Agent ⇄ IDE、设置切换、提示条、弹层、折叠都应当瞬时完成,且功能不受影响。设计稿的 <b>设置 → 外观 → 动效</b> 是真的开关,可在本页立刻试。</div>')}</div>`)}
    ${sec('准则', '写新动效前先对照。', `<div class="grid2">${dd('该做', DO, 'c-ok')}${dd('不该做', DONT, 'c-err')}</div>`)}</div>`;

  /* ───────── 补充浮层 ───────── */
  const acctStatic = (() => { try { return W.acctMenu().replace('a-pop-up', 'a-pop-up static'); } catch (e) { return ''; } })();
  const searchSample = `<div class="pop" style="width:100%;padding:6px"><div class="f g3 ai" style="padding:4px 6px 8px">${ic('search', 's12')}<b>代理</b><span class="grow"></span><span class="t3 fs-cap">3 项</span></div>
    <div class="row on"><span class="grow">HTTP 代理</span><span class="t3 fs-cap">Claude Code · 运行环境</span></div>
    <div class="row"><span class="grow">不走代理的地址</span><span class="t3 fs-cap">运行环境</span></div>
    <div class="row"><span class="grow">浏览器代理</span><span class="t3 fs-cap">工作台 · 浏览器</span></div>
    <div class="t3 fs-cap" style="padding:6px 6px 2px">没有想要的?试试 <b>命令面板 ⌘K</b></div></div>`;
  const rbar = `<div class="rbar" data-static style="position:static;animation:none"><span class="grow">${ic('refresh', 's12')} <b>2</b> 项更改需要重启后生效</span>${btn('稍后', 'sm ghost')}${btn('立即重启', 'sm pri')}</div>`;
  const recSample = `<div class="fc g3"><div class="f g3 ai"><span class="grow">发送消息</span><span class="keycap rec">按下新的组合键…</span>${btn('取消', 'sm ghost')}</div>
    <div class="f g3 ai"><span class="grow">换行</span><span class="keycap">${kbd('⇧', '↩')}</span></div>
    <div class="kb-warn" style="display:flex">${ic('alert', 's12')}<span class="grow"><b>⌘↩</b> 已被「立即引导」使用</span>${btn('替换', 'sm soft')}${btn('保留原样', 'sm ghost')}</div></div>`;
  const provSample = `<div class="fc g3">${[['idle', '', '测试连接', ''], ['run', 'loading', '测试中', ''], ['ok', '', '连接正常 · 312ms', 'c-ok'], ['err', '', '401 · 密钥无效', 'c-err']].map(([k, c, t, col]) => `<div class="f g3 ai"><span class="t3" style="width:52px">${{ idle: '空闲', run: '测试中', ok: '成功', err: '失败' }[k]}</span>${btn(k === 'run' ? '测试中' : '测试连接', 'sm ' + c)}<span class="${col} fs-cap">${k === 'idle' || k === 'run' ? '' : ic(k === 'ok' ? 'okc' : 'alert', 's12') + ' ' + t}</span></div>`).join('')}</div>`;
  const toastRow = (lvl, i, t, sub, extra = '') => `<div class="pop toast" style="width:100%"><div class="nt"><span class="t-ic">${ic(i, 's16 ' + lvl)}</span><div class="grow" style="min-width:0"><div class="m">${t}</div><div class="t3 fs-cap">${sub}</div></div>${extra}${ib('x', 'sm', '关闭')}</div></div>`;
  const toasts = `<div class="fc g3">${toastRow('c-info', 'info', '发现新版本 0.2.4', '正在后台下载', '')}${toastRow('c-ok', 'okc', '已复制版本信息', '')}${toastRow('c-warn', 'alert', '没有安装 rust-analyzer', '语言服务不可用', btn('复制安装命令', 'sm ghost'))}${toastRow('c-err', 'alert', '连接失败', '401 · 密钥无效 · 不会自动消失', btn('重试', 'sm ghost'))}
    <div class="pop toast" style="width:100%"><div class="nt"><span class="t-ic">${ic('download', 's16 c-info')}</span><div class="grow"><div class="m">正在下载 0.2.4</div><div class="t3 fs-cap">34.4 MB / 82.0 MB</div></div>${ib('x', 'sm', '关闭')}</div><div class="meter"><i style="width:42%"></i></div></div>
    ${toastRow('c-ok', 'trash', '已删除 Bash(npm run *)', '6 秒内可撤销', btn('撤销', 'sm ghost'))}</div>`;
  const newDlg = `<div class="dlg wide" style="width:100%"><div class="f g3"><span class="dlg-i">${ic('plus', 's16')}</span><div class="grow"><b>新建 Skill</b><div class="t3" style="margin-top:3px">会写入 <span class="mono">~/.claude/skills/&lt;名称&gt;/SKILL.md</span></div></div></div>
    <div class="fgrp"><span class="lbl">名称</span><span class="inp is-focus" style="width:100%"><span>release-notes</span></span></div>
    <div class="fgrp"><span class="lbl">范围</span>${seg(['用户', '项目'], 0)}</div>
    <div class="acts">${btn('取消', 'ghost')}${btn('创建', 'pri')}</div></div>`;
  const delDlg = `<div class="dlg" style="width:100%"><div class="f g3"><span class="dlg-i err">${ic('trash', 's16')}</span><div class="grow"><b>删除自动化「每日构建」?</b><div class="t3" style="margin-top:3px;line-height:1.55">已有的 12 条运行记录会一并删除,无法恢复。</div></div></div><div class="acts">${btn('取消', 'ghost')}${btn('删除', 'danger')}</div></div>`;

  const extra = () => sec('补充浮层与反馈', '设置、自定义与账号相关的交互,逐个给出静态样子;动起来的版本在对应页面里可以点。', `<div class="grid2">
    ${spec('账号菜单', ['acct', '从触发点向上展开'], `<div class="fc g3" style="position:relative;display:flex;flex-direction:column;align-items:center">${acctStatic}${W.acctMenu({ loading: true })}${W.acctMenu({ empty: true, noCost: true, noLim: true })}</div>`)}
    ${spec('设置搜索结果', ['⌘F 聚焦', '键盘 ↑↓ ↩'], searchSample)}
    ${spec('需要重启条', ['rbar', '设置页底部固定'], rbar)}
    ${spec('快捷键录制与冲突', ['keycap.rec', '冲突就地提示'], recSample)}
    ${spec('连通性测试的四态', ['服务商 / 网络'], provSample)}
    ${spec('提示条家族', ['info / ok / warn / err / 进度 / 撤销'], toasts)}
    ${spec('新建项对话框', ['dlg.wide', 'Enter 提交 · Esc 取消'], newDlg)}
    ${spec('不可逆删除确认', ['dlg', '默认焦点在取消'], delDlg)}</div>`);
  P.ovExtra = extra;
  const base = P.overlays;
  P.overlays = (...a) => { const h = base(...a); const i = h.lastIndexOf('</div>'); return i < 0 ? h + extra() : h.slice(0, i) + extra() + h.slice(i); };
})();
