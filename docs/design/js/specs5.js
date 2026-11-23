/* 设计稿 v3.7 · 会话流:待发送区(排队 / 引导)、多题问答卡与回答卡片、回到最新按钮
   依据:Cursor(运行中发消息 = 排队,⌘⏎ 立即引导,队列在输入框上方)、Claude Code(运行中输入先进队列)、
   ChatGPT / Claude.ai / Cursor(滚离底部时,输入框正上方居中出现圆形向下箭头)、Claude Code 的 AskUserQuestion(多题、可跳过)。
   每页都带「实现状态」列,与 Go 代码保持一致。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, tag, kbd } = U;
  const { head, sec } = P.h;
  const F = window.F;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, keys, body, why) => `<div><div class="cap2"><b>${t}</b>${keys ? kbd(...keys) : ''}</div>${body}<div class="why">${why}</div></div>`;
  const snip = (body, w = 560) => `<div class="snip" style="height:auto;width:${w}px;padding:12px 14px">${body}</div>`;

  /** 模拟对话栏:上面是滚动中的会话流,下面是输入框;按钮绝对定位在流区底部居中 */
  const chatMock = (btnHtml, over = '') => `<div class="snip" style="height:300px;width:560px;position:relative;display:flex;flex-direction:column;padding:0;overflow:hidden">
    <div style="flex:1;min-height:0;position:relative;padding:14px 18px;overflow:hidden;opacity:.92"><div class="fc g3">${F.user('看一下路由大小写问题')}<div class="md"><p>问题出在路由表直接用原始路径做键……</p></div>${F.step('read', 'router.rs', 'src/', '86 行')}${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span>')}</div>${btnHtml}</div>
    <div style="padding:4px 14px 12px;flex:none">${over}${F.composer({ running: true })}</div></div>`;

  P['flow-queue'] = () => `<div class="ds" style="max-width:1240px">${head('会话流 · 排队引导 / 问答卡 / 回到最新', '三件事:① 回合进行中发的消息一律先落在输入框上方的「待发送区」,分「排队」「引导」两种;② 提问(AskUserQuestion)有多个问题时可以来回切换,回答完在会话流里以卡片显示「问题 + 回答」;③ 滚离底部时,「回到最新」是输入框正上方居中的圆形向下箭头,不再放右下角。')}
    ${sec('待发送区:排队与引导', '对标 Cursor / Claude Code。输入框上方这一块只放「还没真正进入对话」的消息:排队 = 等本轮结束再按顺序发;引导 = 立刻写给 Claude,下一次工具调用之间被吸收,吸收之前先显示在这里(不会同时出现在会话流里),被接收后才变成会话流里的用户消息。', grid(
      cell('排队 + 引导 同时存在', '', snip(F.tdock() + F.queue() + F.composer({ running: true, focus: true })), '引导的行靠前、带强调色标签与「等 Claude 接收」;排队的行带序号。悬停排队行出现:改为引导 / 停止并发送 / 删除;点文字退回输入框编辑。'),
      cell('回合被中断:队列暂停', '', snip(F.queue([{ k: 'queue', t: '顺便把 README 里的路由示例也更新一下' }], { paused: true }) + F.composer({ focus: true })), '中断或出错时不自动发送;「继续」恢复自动发送,「撤回」清空。引导消息已发给 Claude,不在暂停之列。'),
      cell('带图片 / 窄档对话栏', '', `<div class="snip" style="height:auto;width:360px;padding:12px 14px">${F.queue([{ k: 'queue', t: '按这张截图改一下间距', img: 2 }, { k: 'steer', t: '', img: 1 }]).replace('<span class="f ai g2">' , '<span class="f ai g2 dock-wide">')}</div>`, '附带图片的消息:文字照常显示,行尾多一个「N 张图」小标;只有图片没有文字时显示「(图片)」。对话栏 < 420 宽时头部隐藏 ⏎ / ⌘⏎ 的提示,行内不变。'),
      cell('进入方式', ['⏎', '⌘⏎'], `<div class="snip" style="height:auto;width:560px;padding:6px 14px">${tbl([['操作', 150], ['结果']], [
        ['⏎(默认)', '排队:进待发送区,序号递增;本轮正常结束后自动发队首,逐条发送'],
        ['⌘⏎', '引导:立刻发给 Claude(priority: next),待发送区里以「引导」行显示,直到被接收'],
        ['设置 → 对话 → 回合进行中再发消息', '把 ⏎ 与 ⌘⏎ 的含义对调;输入框占位文字随之变化'],
        ['空闲时', '⏎ 与 ⌘⏎ 都是直接发送,不经过待发送区'],
        ['引导被接收', '工具调用一结束 / 回合结束 / 流里出现新的步骤即视为接收:该行淡出,会话流里出现这条用户消息'],
        ['出口', '排队行:退回编辑 / 改为引导 / 停止并发送 / 删除 / 暂停后撤回;引导行发出后不可撤回(已写进 Claude 的输入)']])}</div>`, '输入框占位在回合进行中写明「⏎ 排队,⌘⏎ 引导」,不靠用户记快捷键。'))
    )}
    ${sec('下一步建议:回答之后,点一下就发', '每轮正常结束后,在这一轮回答的末尾(回合脚注「复制 / 重试 / 分叉 · 模型 · 用时」之后)排出 3 行竖排的「下一步」建议,整行可点,点哪行就把哪句当作你的消息直接发送,不用再手动输入。它是回答的延续、会话流的一部分,随时间线一起滚动,<b>不贴在输入框上方</b>(原横排胶囊句子长了会折成参差的两行,又像输入框的附属物,已废弃)。第一条是最推荐的,只把行首图标染成强调色,不加描边 / 底色。建议随回答一起给出,不另发请求:系统提示词里约定 Claude 在整个回合最后一条回复的末尾带一个 <next_steps>…</next_steps> 块(3 行,口吻是用户对 Claude 说的话,每条 ≤ 24 字、能直接执行,如「运行测试确认没问题」「把这次改动提交」),界面在流式正文里把它截走、不进会话流,回合结束时才显示成胶囊。', grid(
      cell('回合结束后出现', '', snip(F.tfoot() + F.nextSteps() + F.composer({ focus: false })), '建议在回答流到末尾时就已收齐,回合一结束立刻出现,没有等待、没有骨架态。Claude 这一轮没带(只跑了工具 / 简单问答 / 它省略了)就不显示,不会补发请求。块不会在会话流里露出来,历史会话回放同样被剥掉。'),
      cell('悬停一条', '', snip(F.tfoot() + F.nextSteps().replace('class="ns "', 'class="ns hov"') + F.composer({ focus: false })), '悬停一行:整行浅底色、行首图标变强调色,右侧出现小铅笔「放进输入框编辑」(把这句填进输入框、光标在末尾,不发送,适合想改几个字再发)与「发送」字样;提示里给完整文字(过长的行单行省略)。点行本身 = 立即发送。行间 1px 细分隔线,分隔线与悬停底色占满正文列,行首图标缩进 8px。'),
      cell('什么时候出现 / 消失', '', `<div class="snip" style="height:auto;width:560px;padding:6px 14px">${tbl([['情况', 170], ['行为']], [
        ['出现', '回合正常结束(end_turn)、本轮 Claude 有文字回复、输入框是空的、没有待处理的权限 / 提问卡、没有排队消息'],
        ['不出现', '回合被中断 / 出错 / 拒绝、设置里关了、对话栏被权限卡或提问卡占用、排队里还有消息(下一轮马上开始)'],
        ['重启 / 重新打开会话', '会话文件里最后一轮回答本来就带着 <next_steps> 块,回放结束后把它还原成建议(只还原最后一轮;之后又有新提问、设置关着、或最后一轮没带块就不显示),所以重启后上次给的下一步还在。回放期间不弹通知、不发请求。'],
        ['消失', '发送任意消息 · 新回合开始 · 切换会话 · 点标题行右侧 ✕(本轮不再显示)。在输入框里打字<b>不</b>收起(它在时间线里,不挡输入,收起会让列表跳动)。只保留最后一轮的建议,上一轮的随新回合开始直接移除,不留历史残影'],
        ['没有「换一批」', '建议是回答的一部分,不单独生成;想要别的方向就直接在输入框里说。标题行只有 ✕,与「下一步」同在一行最右。'],
        ['发送方式', '等同于在输入框里敲完这句再回车:走同一条发送路径(斜杠命令、附件规则不变);排队 / 引导只在回合进行中才有,而建议只在空闲时出现,所以总是直接发送']])}</div>`, '没做:⌥1 / ⌥2 / ⌥3 快捷键直接发送第 N 条(待用户确认后再加)。'),
      cell('窄档对话栏', '', `<div class="snip" style="height:auto;width:360px;padding:12px 14px">${F.nextSteps(['运行测试确认没问题', '把这次改动提交', '再检查一下边界情况'])}${F.composer({ focus: false })}</div>`, '竖排不需要为窄档单独适配:每条仍是一行,过长省略,悬停提示给全文;窄档悬停时只留铅笔,「发送」字样隐去。'),
      cell('设置', '', `<div class="snip" style="height:auto;width:560px;padding:6px 14px">${tbl([['设置', 190], ['说明']], [
        ['设置 → 对话 → 下一步建议', '开关,默认开。新会话生效:说明随系统提示词在启动 claude 时传入;每轮只多几十个输出 token,没有额外请求。关掉后新会话不再要求 Claude 给'],
        ['演示模式', '演示后端不会带建议块,用写好的 3 条,便于复现界面;不是真实能力']])}</div>`, ''))
    )}
    ${sec('问答卡:多个问题可来回切换', '问题多于一个时,卡片顶部多一排「步骤条」:点任一步直接跳到那一题(已答的带对勾);底部「上一题 / 下一题」,最后一题的主按钮变「提交」;← → 也能切换。切换不丢已选答案;提交前每一题都能改。单选必须先选才能进入下一题,多选可以一个都不选,「跳过」把当前题记为跳过。选了「其它」要填说明。', grid(
      cell('第 1 题(单选)', ['←', '→'], snip(F.ask({ cur: 0 })), '步骤条:已答 = 对勾,当前 = 强调描边。只有一题时没有步骤条与「上一题」。'),
      cell('第 2 题(多选 + 其它)', '', snip(F.ask({ cur: 1 })), '最后一题:「下一题」换成「提交」。可点上面的步骤条、「上一题」或 ← → 试一下切换。'),
      cell('只有一个问题', '', snip(F.ask({ single: true })), '没有步骤条 / 计数 / 上一题;主按钮直接是「提交」。'))
    )}
    ${sec('回答之后:会话流里的卡片', '提交(或跳过)后,输入框恢复,会话流里留下一张「回答卡片」:标题「已回答 N 个问题」,每个问题一行,回答是强调淡底的小胶囊;多选 = 多个胶囊,自由输入 = 方角胶囊,跳过 = 灰色「已跳过」。不再是一行灰字。卡片与计划卡同一层级(raised 底、阴影 1、圆角 md),点不展开,内容总是全部可见。', grid(
      cell('全部回答', '', snip(F.askDone()), '问题序号淡色,回答在问题下一行缩进 16px,与步骤行文字列对齐。'),
      cell('含跳过与自由输入', '', snip(F.askDone([['末尾斜杠要怎么处理?', ['忽略末尾斜杠'], ''], ['要给哪些路由补测试?', [], 'skip'], ['还有别的要求吗?', ['别动 handlers 里的错误类型'], 'free']])), '标题右侧显示跳过的数量;自由输入的回答用方角胶囊,与选项回答区分。'),
      cell('提问卡 → 回答卡的过渡', '', snip(`<div class="t2" style="line-height:1.7">提交后提问卡原位淡出(120 ms),输入框淡入并恢复草稿;同时会话流里<b>就地</b>出现回答卡片(插在提问发出的位置,不追加到末尾),吸底时随新内容滚入视野。</div>`), '回答卡片属于「回合」的一部分,回合折叠时和步骤行一起折进「已处理」。'),
      cell('运行中等待回答', '', snip(`<div class="note-line">${ic('info', 's12')}<span class="shimmer">等待你回答 2 个问题…</span></div>`), '提问未回答时,会话流只有一行流光提示;卡片在输入框位置,避免两处都有按钮。'))
    )}
    ${sec('左栏资源管理器:视口内落下(会话流无切换入场)', '换分段回到「文件 / 搜索 / 源代码管理」时,列表行仍可用 <b>navDrop</b> 自上而下落下,但<b>只对视口内行排队</b>:视口上方瞬时就位、下方不等待。文件树用虚拟列表只构建视口附近行,滚动位置首帧恢复(<code>treeList.ScrollIntoView</code>)。<b>会话流</b>不再播切换入场 stagger(仍记住滚动);发送 / 状态行等原有动效保留。', grid(
      cell('停在列表中间', '', `<div class="snip" style="height:220px;width:560px;position:relative;padding:10px 14px;overflow:hidden">
        <div style="position:absolute;inset:0;background:linear-gradient(180deg,var(--bg) 0 20%,transparent 20% 55%,var(--bg) 55% 100%);opacity:.5;pointer-events:none"></div>
        <div class="t3 fs-cap" style="margin-bottom:8px">灰色 = 已滚过,瞬时可见 · 彩色 = 视口内 stagger · 下方未构建的行滚动到时直接出现</div>
        ${F.step('read', 'router.rs', 'src/', '86 行')}<div style="opacity:.35">${F.step('grep', '"match_path"', '', '5 处')}</div>
        ${F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span>')}<div class="t3 fs-cap" style="margin-top:60px;opacity:.5">…下方 200 行不等待上方动画</div></div>`, '示意:不是从第 1 行开始排队,而是从当前可见的第一行起算序号。'),
      cell('规则摘要', '', `<div class="snip" style="height:auto;width:560px;padding:6px 14px">${tbl([['项', 130], ['规则']], [
        ['触发', '打开已有会话且条目载入完成(含接回后台会话);空会话 / 新会话不播'],
        ['滚动', '离开会话时记住滚动偏移,切回来恢复'],
        ['视口内', '从第一个可见行起 stagger,间隔 30ms,第 12 行起 8ms,总延迟 ≤ 400ms'],
        ['视口外', '上方 / 下方(含虚拟列表未构建的行)不透明、无延迟'],
        ['首帧', '列表尚未量高时,有偏移则估算视口;否则整帧不播'],
        ['重复', '同一会话本轮进程内只播一次切换入场'],
        ['不减', '新用户消息 / 状态行 / 工具流式到达仍用原有动效']])}</div>`, '实现: flow_rise.go + flow_rows.go;flowRiseViewport / flowRiseDelay 单测。'))
    )}
    ${sec('回到最新:圆形向下箭头,输入框正上方居中', '对标 ChatGPT / Claude.ai / Cursor:滚离底部超过 32px 才出现,位置在会话流区的底边、水平居中,正好在输入框「正中往上一点」,滚到上面一眼就能点到;不再放右下角(会和滚动条、右侧内容抢位)。点击平滑滚到底并恢复自动跟随;已在底部时淡出。', grid(
      cell('滚离底部', '', chatMock(F.toLatest(false)), '30px 圆,raised 底 + 1px 描边 + 阴影 2,图标 12px 向下箭头;悬停底色变深并上移 1px。'),
      cell('离开期间有新内容', '', chatMock(F.toLatest(true)), '回合还在输出时,按钮右上角出现 7px 强调色圆点,提示「下面还在变」;点击后圆点消失。'),
      cell('规则', '', `<div class="snip" style="height:auto;width:560px;padding:6px 14px">${tbl([['项', 130], ['规则']], [
        ['出现', '距底部 > 32px(与「停止自动跟随」同一阈值);会话为空时不显示'],
        ['位置', '会话流区底边上 10px、水平居中(与输入框同一中轴),不遮挡输入框与待发送区'],
        ['点击', '平滑滚到最新并重新贴底跟随;键盘:聚焦后 ⏎ / 空格,屏幕阅读器读作「回到最新」'],
        ['子 Agent / 任务输出标签', '同一个圆形按钮,居中放在各自滚动区底部'],
        ['减弱动态', '出现 / 消失不做位移,直接淡入淡出']])}</div>`, '任何滚动区都只有这一种「回到最新」形态。'))
    )}
    ${sec('实现状态', '与 Go 代码逐项对应;有 flow_test / dock_test 等单测,未经开发版 computer use 实测的仍标「未测试」。', tbl([['项', 220], ['说明', 0], ['状态', 70]], [
      ['下一步建议:回答末尾 3 行可一键发送的竖排列表', '随回答一起给出(不另发请求):core/nextsteps.go 的 NextStepsPrompt 经 settings_wire.go FillClaudeConfig 追加到 --append-system-prompt(设置 chat.nextSteps 开着时);claude/translate.go 在流式正文里用 NextStepsFilter 截走 <next_steps> 块(开标签被切在两段增量之间也处理)、完整消息 / 历史回放用 StripNextSteps 剥掉,发 core.EvNextSteps;app/nextsteps.go 先存 pending,回合正常结束才显示,flow_rows.go buildFlowRows 在末尾追加一行 "nsteps"(回合脚注之后,随虚拟列表滚动),nextsteps.go nextStepsRow 画竖排列表;点击发送(submitPrompt)/ 铅笔填入 / ✕ / 新回合 / 切会话即收起,打字不收起;历史回放结束时 nextStepsOnReplayEnd 把最后一轮带的块还原显示(重启后仍在);演示模式用写好的 3 条;-ed nextsteps[:none]。未做:⌥1/2/3 快捷键。已编译;未跑 vet / test、未用 computer use 实测(含 Claude 是否真的按约定带块、第三方服务商是否照做)', st('part')],
      ['待发送区:排队 + 引导共用', 'dock.go queueCard + composer_queue.steerSend/syncSteers/visibleItems;引导行「等 Claude 接收」。未测试', st('ok')],
      ['问答卡 Esc 跳过', 'dock_ask.go:KeyEscape 调 skip()。未测试', st('ok')],
      ['回答卡 · 回合折叠', 'flow_turn_fold.turnFoldedItem 含已答 QuestionRequest;turnMustStayOpen 挡未完成提问。未测试', st('ok')],
      ['子 Agent 主流调用卡', 'flow_sub.go 不内联 subagentTimeline;任务预览+待确认徽标+最新一步。未测试', st('part')],
      ['子 Agent 工作区会话流', 'ws_views_sub+flow_sub_stream 复用 stepGroup/assistant。未测试', st('part')],
      ['排队行动作', 'dock.go 悬停:改引导/停止并发送/删除;qStopAndSend。未测试', st('ok')],
      ['排队消息带图片', 'dock splitDraftImages+queueImgBadge;submitPrompt 带 Images。未测试', st('ok')],
      ['引导接收判定', 'queue.syncSteers:新非用户条目+steerDwell / 回合结束 / steerMaxWait 兜底(无后端确认信号)。未测试', st('ok')],
      ['问答卡 · 多题切换', 'dock_ask 步骤条+上/下题+←→;dockBadge 审批序号。未测试', st('ok')],
      ['回答卡片', 'dock_ask.askEcho + askEchoMenu。未测试', st('ok')],
      ['回到最新按钮', 'flow_scroll.toLatestBtn + flowTailMark;距底阈值 32px,贴 dock 上缘 10px。未测试', st('ok')],
      ['会话流切换入场', '已关闭 triggerFlowRise/flowRiseRow;仍恢复 flowScroll。flowRiseViewport 复用于侧栏', st('ok')],
      ['资源管理器视口落下 + 虚拟树', 'ide_tree_flat + ui.List + treeNavDrop;side_drop navDropViewport。未 computer use', st('part')],
      ['面包屑下拉文件图标', 'file_menu_icon.go mItemFile 与 fileIc 同源徽标;fileIconID 单测。未 computer use', st('part')],
      ['大纲时间线 sash 在标题上缘', 'ide_tree.go resizeHandle 在 sec-h 上;ide_side_sash_test。未 computer use', st('part')],
      ['交互卡入场与呼吸描边', 'kit.PopUpIn/DockGlow + dock_pending.dockCardAt;处理完 composer PopIn 120ms。未测试', st('ok')],
      ['引导行附带图片', 'steerPend.imgN + queueImgBadge。未测试', st('ok')]]))}
  </div>`;

  /* ───────── 交互:切题、选项、回到最新 ───────── */
  document.addEventListener('click', (e) => {
    const t = e.target.closest('[data-act="qa-step"], [data-act="qa-go"], [data-act="qa-opt"], [data-act="to-latest"]');
    if (!t) return;
    const card = t.closest('.qa');
    if (t.dataset.act === 'qa-step' && card) card.dataset.cur = t.dataset.i;
    else if (t.dataset.act === 'qa-go' && card) {
      const n = Math.max(0, Math.min(1, +card.dataset.cur + +t.dataset.d)); card.dataset.cur = n;
    } else if (t.dataset.act === 'qa-opt') {
      const multi = !!t.closest('.qp') && t.closest('.qp').textContent.includes('可多选');
      if (!multi) t.parentElement.querySelectorAll('.opt').forEach(o => o.classList.remove('on'));
      t.classList.toggle('on', multi ? !t.classList.contains('on') : true);
    } else if (t.dataset.act === 'to-latest') t.classList.remove('fresh');
    if (card) card.querySelectorAll('.qa-step').forEach(s => s.classList.toggle('cur', s.dataset.i === card.dataset.cur));
  });
})();
