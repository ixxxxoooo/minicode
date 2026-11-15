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
    ${sec('实现状态', '与 Go 代码逐项对应;「部分」= 已写完、仅 go build 通过、未测试。', tbl([['项', 220], ['说明', 0], ['状态', 70]], [
      ['待发送区:排队 + 引导共用', '排队(⏎)与引导(⌘⏎)都先进输入框上方的待发送区;引导行带「等 Claude 接收」,被接收后淡出并进会话流;占位文字写明快捷键。未测试', st('part')],
      ['排队行动作', '点文字退回编辑 / 改为引导 / 停止并发送 / 删除 / 暂停后继续·撤回(原有),新增「改为引导」。未测试', st('part')],
      ['排队消息带图片', '排队项保留草稿里的图片语法;卡上缩略图 chip;退回输入框还原;自动/引导/停止并发送经 submitPrompt 带 Images。演示 b4-queue-img / shot -state queue-imgs。未测试', st('part')],
      ['引导接收判定', '启发式:发出后会话流出现新的非用户条目、或回合结束即视为接收(后端没有显式确认信号)', st('part')],
      ['问答卡 · 多题切换', '步骤条 + 上一题 / 下一题 + ← → 键;切换保留已选;提交 / 跳过沿用原逻辑。未测试', st('part')],
      ['回答卡片', '会话流里的卡片(问题 + 回答胶囊,多选 / 跳过 / 自由输入三种样式),替换原先的一行灰字。未测试', st('part')],
      ['回到最新按钮', '输入框正上方居中的圆形向下箭头,替换右下角的「↓ 回到底部」文字按钮;子 Agent 过程流同形态。「有新内容」小圆点已做。未测试', st('part')]]))}
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
