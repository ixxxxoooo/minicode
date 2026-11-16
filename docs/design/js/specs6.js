/* 设计稿 v3.8 · 会话流:响应耗时与等待(发送后的等待行、脚注分段条、瀑布弹卡、慢因提示)
   依据(均已核对):
   - claude 真实协议(2.1.325 实测):发出提示后 ~50 ms 内先给 system init + system status(requesting),
     之后上游静默到 stream_event message_start(这一段就是「等待模型」);message_start.usage 带本次调用的
     input / cache_read / cache_creation,所以「有没有缓存」在第一个字到达的瞬间就知道;每个工具之后会再发一次请求,
     所以一个回合有 N 次「等待模型」;最后一个 message_stop 到 result 之间还可能有「收尾」(实测一次等了 53 s)。
   - Claude Code 自己:发送后立即显示 spinner + 已用时 + token 计数,不等第一个字。
   - Chrome DevTools Network「Timing」:把一个请求拆成 Queueing / 连接 / Waiting for server response(TTFB) / Content Download,
     并用瀑布条按比例画出来 —— 「分段 + 比例条」是业界排查延迟的通行形态。
   - OpenTelemetry / Langfuse 的 LLM trace:每次模型调用一个 span,TTFT 单独成指标,缓存读写 token 挂在 span 上。
   - Anthropic Prompt Caching:每次响应的 usage 给出 cache_read_input_tokens / cache_creation_input_tokens;
     默认 5 分钟 TTL(命中即续期),可选 1 小时;读 0.1×、写 1.25×(5 分钟)/ 2×(1 小时)。
   - Claude Code 多文件夹:--add-dir 默认只授权访问,不加载该文件夹的 CLAUDE.md;设环境变量
     CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 才加载(官方 memory 文档)。所以「工作区读取 CLAUDE.md」是每次请求都要多付的上下文。
   每页都带「实现状态」列,与 Go 代码保持一致。
   @author ixxxxoooo */
(function () {
  const { ic, tag } = U;
  const { head, sec } = P.h;
  const F = window.F;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, body, why) => `<div><div class="cap2"><b>${t}</b></div>${body}<div class="why">${why}</div></div>`;
  const snip = (body, w = 440) => `<div class="snip" style="height:auto;width:${w}px;padding:12px 14px">${body}</div>`;

  /** 阶段:key → [名称, 色类] */
  const K = { conn: ['连接 Claude', 'k-conn'], prep: ['本地准备', 'k-prep'], wait: ['等待模型', 'k-wait'], think: ['思考', 'k-think'], write: ['生成', 'k-write'], tool: ['工具', 'k-tool'], ask: ['等你确认', 'k-ask'], fin: ['收尾', 'k-fin'] };

  /** 动画字符帧(Claude Code 同款;主分支 appearance.rs 的 SPINNERS 同源) */
  const GLYPHS = { cc: ['·', '✢', '✳', '✶', '✻', '✽', '✻', '✶', '✳', '✢'], poetic: ['◜', '◠', '◝', '◞', '◡', '◟'], geek: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'], min: ['●', '◉', '○', '◉'] };
  const glyph = (set = 'cc', i = 4, cls = '') => `<span class="rt-gl ${cls}" data-set="${set}" data-i="${i}">${GLYPHS[set][i % GLYPHS[set].length]}</span>`;

  /** 状态行:整个回合里始终在场。state = conn / wait / think / write / tool / fin / ask / slow;el = 已用时 */
  const STAGE_WORD = { conn: '连接 Claude', wait: '等待模型', think: '思考', write: '生成', tool: '运行 Bash', fin: '收尾', slow: '等待模型' };
  const statusRow = (state, el, o = {}) => {
    if (state === 'ask') return `<div class="rt-st ask"><span class="gc">${ic('pause', 's12')}</span><span>等待你的批准</span></div>`;
    const verb = o.verb || '沉思中…';
    const meta = `${el} · <span class="rt-sw">${STAGE_WORD[state]}</span> · Esc 中断`;
    return `<div class="rt-st ${state === 'slow' ? 'slow' : ''}" data-act="rt-steps"><span class="gc">${glyph(o.set || 'cc', o.i ?? 4)}</span><span class="shimmer vb">${verb}</span><span class="rt-meta">(${meta})</span></div>${o.hint ? `<div class="rt-hint">${ic('alert', 's12')}<div>${o.hint}</div></div>` : ''}${o.steps ? waitSteps(o.steps) : ''}`;
  };
  const waitSteps = (rows) => `<div class="rt-steps">${rows.map(([ok, name, n]) => `<div class="r ${ok === 'now' ? 'now' : ''}"><span class="si" style="width:16px;display:grid;place-items:center">${ok === 'now' ? glyph('cc', 4) : ic('check', 's12 c-ok')}</span>${name}<span class="n">${n}</span></div>`).join('')}</div>`;

  /** 脚注里的迷你分段条:segs = [[key, 占比]] */
  const bar = (segs) => `<span class="rt-bar">${segs.map(([k, w]) => `<i class="${K[k][1]}" style="flex:${w}"></i>`).join('')}</span>`;
  const foot = (text, segs, on) => `<div class="tfoot">${U.ib('copy', 'sm', '复制')}${U.ib('refresh', 'sm', '重试')}${U.ib('fork', 'sm', '从这里分叉')}<span class="meta"><span>Sonnet 5.5</span>·<span class="rt-foot ${on ? 'on' : ''}" data-act="rt-pop">${text}${bar(segs)}</span>·<span>↑ 26.2k ↓ 36</span>·<span>缓存 98%</span></div>`;

  /** 瀑布弹卡:rows = [key, 标签, 起点%, 宽度%, 耗时, 备注, slow] */
  const pop = (title, total, rows, o = {}) => `<div class="rt-pop"><div class="h"><b>${title}</b>${o.tag || ''}<span class="t">${total}</span></div>
    ${rows.map(([k, lb, s, w, ms, sub, slow]) => `<div class="row ${slow ? 'slow' : ''}"><span class="lb">${lb}</span><span class="trk"><i class="${K[k][1]}" style="left:${s}%;width:${w}%"></i></span><span class="ms">${ms}</span></div>${sub ? `<div class="rtsub">${sub}</div>` : ''}`).join('')}
    <div class="sep"></div>${o.cache || ''}${o.tip ? `<div class="tip">${o.tip}</div>` : ''}</div>`;

  const cacheBlock = `<div class="kv"><span>缓存</span><span><b style="color:var(--ok)">命中 98%</b> · 读 25.7k · 写 17 · 未缓存 4</span></div>
    <div class="kv"><span>上下文</span><span>26.2k tokens · 工作区 5 个文件夹</span></div>`;

  P['resp-time'] = () => `<div class="ds" style="max-width:1240px">${head('会话流 · 响应耗时与等待', '解决「点发送后十几秒没反应、不知道慢在哪」。三件事:① 发送的同一帧起,会话流尾有一行始终在场的「状态行」(Claude Code 式动画字符 + 动词 + 已用时 · 当前阶段),直到回合结束,中间任何阶段都不留白,慢到一定程度写出可能的原因;② 助手回复脚注的「用时」带一条迷你分段条,点开是 Chrome DevTools Timing 式的瀑布弹卡,逐阶段、逐次模型调用列出耗时;③ 每次模型调用「有没有命中缓存」在第一个字到达时就知道,逐次记录并显示在瀑布弹卡里。')}
    ${sec('一个回合的阶段(哪一段算谁的)', '全部来自 claude 的真实事件,不靠猜。每段都能回答「这时在等谁」:自己、模型、工具,还是你。', `<div class="snip" style="height:auto;width:1040px;padding:6px 14px">${tbl([['阶段', 120], ['起止(事件)', 300], ['在等谁', 90], ['说明', 0]], [
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-conn"></i></span> 连接 Claude`, '发送 → 进程握手完成(仅冷启动才有)', '本机', '预热过的进程没有这一段;冷启动实测 0.2~0.6 s'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-prep"></i></span> 本地准备`, '写入提示 → 收到 system status: requesting', '本机', '通常 < 0.1 s;小于 100 ms 不单独画'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-wait"></i></span> 等待模型`, 'requesting → message_start(第一个字节)', '上游', '即 TTFT。<b>每次模型调用一段</b>:工具之后会再来一次,所以长回合有多段;上下文越大、缓存没命中越慢'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-think"></i></span> 思考`, '第一个 thinking 块开始 → 结束', '模型', '只有开了思考才有'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-write"></i></span> 生成`, '第一个 text / tool_use 块 → message_stop', '模型', '输出正文与组织工具调用'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-tool"></i></span> 工具`, 'message_stop(stop_reason = tool_use)→ 第一个 tool_result', '本机', '命令 / 读写文件 / MCP;和步骤行里的耗时是同一段'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-ask"></i></span> 等你确认`, '权限 / 提问 / 计划请求到达 → 你作答', '你', '<b>不算慢</b>:从「等待」里扣掉,单独成段,避免把你的思考时间算成模型慢'],
      [`<span class="rt-bar" style="width:18px;vertical-align:middle"><i class="k-fin"></i></span> 收尾`, '最后一个 message_stop → result', '上游 / 后台', '正常 < 0.5 s;实测过一次 53 s(上游迟迟不关流),超过 3 s 要在弹卡里点名']])}</div>`)}
    ${sec('状态行:整个回合里始终在场,永不留白', '对标 Claude Code 与主分支(Rust 版)的「处理中」行:发送的同一帧就出现,直到回合结束才消失,中间不管模型在想、在写、在跑工具,流尾始终有一行会动的状态。形态:动画字符(· ✢ ✳ ✶ ✻ ✽,每 150 ms 换一帧,强调色)+ 动词(「沉思中…」,一个回合内固定不换,文字流光)+ 灰色括号 (已用时 · 当前阶段 · Esc 中断)。动词与字符跟「等待动画」风格走(设置 → 外观,6 种,默认 Claude Code)。阶段词在换阶段时淡入(120 ms),不整行重绘。', grid(
      cell('发送瞬间(冷启动)', snip(`<div class="t3" style="margin-bottom:8px">${F.user('看一下路由大小写问题')}</div>${statusRow('conn', '0.4s')}`), '点发送的<b>同一帧</b>:用户消息已在流里,状态行已在流尾——不等后端回「回合开始」,也不等进程启动(冷启动时后端先发用户消息再启动进程)。'),
      cell('等待模型(正常)', snip(statusRow('wait', '2.1s')), '动词固定为一个(本回合「沉思中…」),阶段词写「等待模型」。已用时 1 秒内不显示数字,之后每 100 ms 刷新,tabular-nums 防抖。'),
      cell('思考 / 生成 / 工具', snip(`${F.note('brain', '思考 · 3 秒')}${statusRow('think', '5.8s', { i: 7 })}`), '状态行不因为别处有内容就消失:思考块、正文、工具行各自有自己的动态,状态行仍在最下面陪着;阶段词随之换成「思考 / 生成 / 运行 Bash」。'),
      cell('正文流式输出中', snip(`<div class="md"><p>问题出在路由表直接用原始路径做键,大小写不同的请求会……</p></div>${statusRow('write', '9.4s', { i: 2 })}`), '正文在状态行<b>上方</b>长高,状态行始终贴在最后一行下面、高度固定 26px,所以内容再长也不会出现「文字写完了、下面空一截」。'),
      cell('等待较久(> 8 s)', snip(statusRow('slow', '11.3s', { i: 6, hint: '比平时慢。这次请求带着约 <b>52k tokens</b> 的上下文;工作区有 <b>4 个附加文件夹</b>并读取了它们的 CLAUDE.md,这部分每次都计入。 <a data-act="rt-tip">[编辑工作区…]</a>' })), '8 秒起整行转琥珀,下面多一张提示卡(规则见下表);点状态行展开阶段清单。'),
      cell('展开:阶段清单', snip(statusRow('wait', '5.8s', { steps: [['ok', '连接 Claude', '已就绪'], ['ok', '本地准备', '0.1s'], ['now', '等待模型 · 第 1 次调用', '5.7s']] })), '点状态行展开 / 收起(再点收起)。已完成的阶段打勾带耗时,当前阶段用同一个动画字符。'),
      cell('收尾', snip(statusRow('fin', '12.4s', { i: 3, hint: '最后一段回复已经出完,Claude 还没返回「完成」。常见于上游迟迟不关流或有后台任务在收尾;可以直接发新消息,不用等。' })), '最后一个字之后立刻显示「收尾」(不再等 3 秒才出现),超过 3 秒转琥珀并给提示;输入框始终可用。'),
      cell('等你确认', snip(statusRow('ask', '')), '权限 / 提问 / 计划卡在输入框位置,状态行换成静止的「等待你的批准」(暂停图标、琥珀、不动),不再假装在转;作答后立刻恢复动画。'),
      cell('回合结束:收进脚注', snip(`<div class="md"><p>已修复,测试通过。</p></div><div style="opacity:.55;transform:translateY(0)">${statusRow('fin', '17.9s')}</div>${foot('用时 17.9 秒', [['wait', 5.7], ['write', .5], ['tool', .8], ['wait', 3.9], ['write', .6]])}`), '结束时状态行淡出(160 ms)并把高度收到 0(CollapseHide),同一时刻脚注淡入;下面的内容不跳、不留 26px 空洞。'))
    )}
    ${sec('不留白保证:每个时刻都有会动的东西', '把一个回合逐阶段摊开,列出「流里能看到什么在动」。任何一行的「状态行」列都不能是空——这就是本节的验收标准;「额外在动」列是它之外的动态,有就更好,没有也不能空。', `<div class="snip" style="height:auto;width:1040px;padding:6px 14px">${tbl([['时刻', 190], ['状态行写什么', 220], ['额外在动', 260], ['过渡']], [
      ['点发送的同一帧', '连接 Claude / 本地准备', '用户消息已入流', '用户消息与状态行同帧淡入下落(120 ms)'],
      ['请求已发出,等第一个字节', '等待模型 + 已用时', '(无)', '阶段词淡入;8 s 转琥珀'],
      ['第一个字节到,思考块开始', '思考', '思考行转圈 +「思考中」', '思考行自上而下淡入,状态行不动'],
      ['正文流式输出', '生成', '正文逐字增长', '正文在状态行上方长高'],
      ['tool_use 参数在流、工具还没开始', '生成', '工具行出现(运行态)', '工具行淡入'],
      ['工具运行中', '运行 <工具名>', '工具行转圈、命令输出实时尾读', '—'],
      ['工具结束 → 下一次请求之间', '等待模型(第 2 次调用)', '工具行已完成态', '阶段词由「运行」淡入成「等待模型」,不空档'],
      ['权限 / 提问 / 计划卡待处理', '等待你的批准(静止)', '输入框位置的卡片', '作答后立刻恢复动画'],
      ['最后一个字之后、result 之前', '收尾', '(无)', '> 3 s 转琥珀 + 提示'],
      ['回合结束', '(淡出 160 ms)', '脚注淡入', '高度收折,无跳动'],
      ['出错 / 中断', '(淡出)', '错误提示卡 / 「已中断」行', '同回合结束']])}</div>`)}
    ${sec('等待动画风格(设置 → 外观 → 等待动画)', '直接沿用主分支的 6 种风格,字符与动词表一致;选中后状态行与展开的阶段清单用这种字符动画和动词(步骤行里各工具的小转圈保持原样,不随风格变)。默认 Claude Code。', `<div class="snip" style="height:auto;width:1040px;padding:6px 14px">${tbl([['风格', 110], ['动画字符', 200], ['动词(每个回合固定挑一个)']], [
      ['经典', glyph('cc', 5) + ' &nbsp;· ✢ ✳ ✶ ✻ ✽ ✻ ✶ ✳ ✢', '思考中 · 琢磨中 · 推敲中 · 构思中 · 斟酌中 · 酝酿中 · 梳理中 · 盘算中'],
      ['趣味', glyph('cc', 3) + ' &nbsp;同上', '摸鱼中 · 炼丹中 · 薅头发中 · 掐指一算中 · 憋大招中 · 搬砖中 · 喂猫中 · 冲咖啡中'],
      ['Claude Code(默认)', glyph('cc', 4) + ' &nbsp;同上', '沉思中 · 踱步中 · 冥想中 · 酿造中 · 蹦跶中 · 琢磨透中 · 神游中 · 凝思中'],
      ['诗意', glyph('poetic', 2) + ' &nbsp;◜ ◠ ◝ ◞ ◡ ◟', '凝神中 · 遐思中 · 斟字酌句中 · 静待花开中 · 拾掇思绪中 · 望月沉吟中 · 研墨中 · 听风中'],
      ['极客', glyph('geek', 4) + ' &nbsp;⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏', '编译宇宙中 · 梳理依赖中 · 调优中 · 遍历语法树中 · 回收内存中 · 预热缓存中 · 握手中 · 解析意图中'],
      ['极简', glyph('min', 1) + ' &nbsp;● ◉ ○ ◉', '工作中']])}<div class="t3 fs-cap" style="padding:8px 4px;line-height:1.7">动词按「会话 id + 回合开始时刻」稳定挑选,同一回合不会乱跳;有进行中的待办时,动词换成待办的进行时(如「补充大小写测试…」),对标 Claude Code 的状态行。动画限频 10 帧 / 秒(换帧间隔 150 ms),已用时 100 ms 刷新一次——不为一个转圈让整个会话流每秒重排 60 次。减弱动态 / 动效关闭时:字符固定为第一帧、不流光,已用时照常更新。</div></div>`)}
    ${sec('慢因规则(只写能从事件里判断出来的)', '提示只给「最可能的一条」,按优先级匹配;文案写事实 + 下一步,不猜测、不责怪。只用等待那一刻就能确定的事实(上下文大小、工作区设置),<b>不</b>在等待中猜「缓存没命中」「思考强度高」——这两件事要等第一个字节到达后才知道,放在回合结束后的瀑布弹卡里。整组提示可在设置里关掉。', `<div class="snip" style="height:auto;width:1040px;padding:6px 14px">${tbl([['条件(按优先级)', 300], ['文案', 0], ['下一步']], [
      ['等待模型 > 8 s 且上下文 > 20k 且工作区读取了 ≥ 1 个附加文件夹的 CLAUDE.md', '比平时慢。这次请求带着约 N k tokens 的上下文;工作区有 N 个附加文件夹并读取了它们的 CLAUDE.md,这部分每次都计入。', '按钮「编辑工作区…」:关掉「读取 CLAUDE.md」(Claude Code 默认也不读)'],
      ['等待模型 > 8 s 且上下文 > 40k', '比平时慢。这次请求带着约 N k tokens 的上下文,模型要先读完才会出第一个字;长会话可以压缩上下文。', '按钮「压缩上下文」'],
      ['等待模型 > 8 s(其余)', '<服务商>响应较慢(已等 N 秒)。这段时间在等模型的第一个字,不是本地卡住;结束后点回复下方的「用时」可看各阶段与缓存。', '无'],
      ['收尾 > 3 s', '最后一段回复已经出完,Claude 还没返回「完成」。常见于上游迟迟不关流或有后台任务在收尾;可以直接发新消息,不用等。', '无'],
      ['连接 > 3 s', 'Claude 进程启动较慢(已等 N 秒)。设置 → 对话 → 连接预热选「打开会话时」可以提前启动。', '无']])}</div>`)}
    ${sec('脚注:迷你分段条 + 瀑布弹卡', '助手回复脚注里的「用时 18.4 秒」后面加一条 56×4 的迷你分段条,按各阶段占比上色(颜色 = 上面的阶段表),一眼看出时间花在哪。悬停 / 点击弹出瀑布卡(再点或 Esc 收起),对标 Chrome DevTools Timing:每个阶段一行,条的起点与宽度按它在整个回合里的位置与占比画,右侧是耗时;「等待模型」行下方写本次调用的缓存与上下文。', grid(
      cell('脚注(收起)', snip(foot('用时 17.9 秒', [['wait', 5.7], ['write', .5], ['tool', .8], ['wait', 3.9], ['write', .6], ['fin', 6.4]]), 480), '分段条只用 6 色,段数多时合并相邻同色段;小于总长 2% 的段保底 2px。窄档(< 420)条隐藏,脚注只留模型 + 用时,弹卡照常可点。'),
      cell('脚注(悬停,展开)', snip(foot('用时 17.9 秒', [['wait', 5.7], ['write', .5], ['tool', .8], ['wait', 3.9], ['write', .6], ['fin', 6.4]], true) + pop('响应耗时', '17.9 s', [
        ['wait', '等待模型', 0, 32, '5.7 s', '第 1 次调用 · 缓存写入 26.2k(未命中)· 上下文 26.2k', true],
        ['write', '生成', 32, 3, '0.5 s'],
        ['tool', '工具 Bash', 35, 5, '0.8 s'],
        ['wait', '等待模型', 40, 22, '3.9 s', '第 2 次调用 · 缓存命中 98%'],
        ['write', '生成', 62, 3, '0.6 s'],
        ['fin', '收尾', 65, 35, '6.4 s', '最后一段回复后 6.4 s 才结束', true]], { tag: tag('首字偏慢', 'warn'), cache: cacheBlock, tip: '本回合 <b>5.7 s</b> 花在第 1 次等待模型,且没有命中缓存。工作区读取了 4 个附加文件夹的 CLAUDE.md。' }), 480), '弹卡 340px 宽,出现在脚注上方(空间不够翻到下方);慢段(≥ 8 s 的等待、≥ 3 s 的收尾)耗时转琥珀并加粗;最多画 10 行,更多折成「另 N 段 · 共 x s」。'),
      cell('没有分段数据(历史回放)', snip(`<div class="tfoot">${U.ib('copy', 'sm', '复制')}${U.ib('refresh', 'sm', '重试')}${U.ib('fork', 'sm', '从这里分叉')}<span class="meta"><span>Sonnet 5.5</span>·<span>用时 1 分 13 秒</span>·<span>↑ 44.1k ↓ 624</span></span></div>`, 480), '重开会话回放出来的回合只有记录时间差,没有逐段数据:脚注沿用旧样式,没有分段条,悬停只列 token / 缓存。不伪造分段。'),
      cell('冷启动回合', snip(pop('响应耗时', '9.8 s', [
        ['conn', '连接 Claude', 0, 6, '0.6 s'], ['prep', '本地准备', 6, 1, '0.1 s'],
        ['wait', '等待模型', 7, 60, '5.9 s', '第 1 次调用 · 缓存写入 26.2k(未命中)', true], ['write', '生成', 67, 33, '3.2 s']], { cache: cacheBlock }), 380), '只有冷启动的回合才有「连接 Claude」行;预热命中的回合直接从「本地准备」起。'),
      cell('含等你确认', snip(pop('响应耗时', '41 s(模型 6.3 s)', [
        ['wait', '等待模型', 0, 6, '2.1 s'], ['write', '生成', 6, 3, '0.9 s'],
        ['ask', '等你确认', 9, 70, '28 s', '权限请求 · Bash(rm -rf dist)· 不计入模型耗时'], ['tool', '工具 Bash', 79, 6, '2.4 s'], ['wait', '等待模型', 85, 10, '4.2 s'], ['write', '生成', 95, 5, '3.4 s']], { cache: cacheBlock }), 380), '标题里的总耗时旁用括号写「模型 x s」(扣掉等你的时间),脚注的「用时」仍是墙钟总时间。'))
    )}
    ${sec('缓存:每次调用都知道有没有命中', '依据 Anthropic usage 字段:message_start.usage 里 cache_read_input_tokens(命中)/ cache_creation_input_tokens(本次写入)/ input_tokens(未缓存)。claude 在第一个字节到达时就给出,所以不必等回合结束。一个回合里每次模型调用各一份。', grid(
      cell('命中', snip(`<div class="note-line">${ic('check', 's12 c-ok')}缓存命中 98% · 读 25.7k · 写 17 · 未缓存 4</div>`, 440), '读 ≥ 总输入 80% 记「命中」(绿);20%~80% 记「部分命中」(灰);写入占主导记「首次写入」(琥珀,说明这是第一次带这段上下文,慢是正常的);读 = 0 且上下文 > 20k 记「未命中」(琥珀)。'),
      cell('规则', `<div class="snip" style="height:auto;width:440px;padding:6px 14px">${tbl([['项', 100], ['规则']], [
        ['位置', '瀑布弹卡「等待模型」行的备注(逐次调用)· 弹卡底部「缓存」行(整回合合计)· 脚注原有「缓存 N%」(保持不变)。等待行里不放:第一个字节一到它就让位了'],
        ['过期提示', 'Anthropic 默认缓存 5 分钟 TTL(命中续期),可选 1 小时。不在界面里硬写 TTL:只以「上一次调用的 usage」判断,读 = 0 就如实说没命中'],
        ['第三方代理', '代理 / 模型不返回缓存字段(三项全 0)时,不显示缓存 chip、不报「未命中」,避免误导'],
        ['多文件夹', '等待行「上下文 N」chip 的悬停与弹卡「上下文」行写「含 N 个附加文件夹的 CLAUDE.md」(工作区开了读取规则才写)']])}</div>`, '依据:Anthropic Prompt Caching 文档。'))
    )}
    ${sec('业界做法对照', '都是只拿能核对的事实;没核对过的产品细节不写进来。', `<div class="snip" style="height:auto;width:1040px;padding:6px 14px">${tbl([['来源', 190], ['做法', 0], ['我们怎么用']], [
      ['Claude Code CLI', '发送即显示 spinner + 已用时 + token 数,不等第一个字;/cost 看缓存读写', '等待行同形态:旋转 + 阶段 + 已用时;缓存 chip'],
      ['Chrome DevTools · Network › Timing', '一个请求拆 Queueing / 连接 / Waiting for server response(TTFB)/ Content Download,瀑布条按比例画', '脚注弹卡的瀑布;TTFB 对应「等待模型」'],
      ['OpenTelemetry / Langfuse(LLM trace)', '每次模型调用一个 span,TTFT 单独成指标,缓存 token 挂在 span 上', '「第 N 次调用」一段一行,缓存写在备注'],
      ['Anthropic Prompt Caching', 'usage 返回 cache_read / cache_creation;默认 5 分钟 TTL,可选 1 小时;读 0.1×', '以每次调用的 usage 判断命中,不自行猜 TTL'],
      ['Claude Code 多文件夹(--add-dir)', '只授权访问;CLAUDE.md 默认不加载,需环境变量才加载', '工作区「读取 CLAUDE.md」默认关(已是);慢响应提示里直达「编辑工作区」']])}</div>`)}
    ${sec('实现状态', '与 Go 代码逐项对应;仅 go build 通过,未经开发版 computer use 实测的一律「未测试」。演示后端触发词:发「慢」= 冷启动 + 11 秒等待 + 收尾;「多次」= 两次模型调用 + 工具 + 缓存命中;其余普通提问 = 1.4 秒短等待。', tbl([['项', 240], ['说明', 0], ['状态', 70]], [
      ['后端分阶段计时', 'claude/stages.go stageClock(连接 / 本地准备 / 等待模型 / 思考 / 生成 / 工具 / 等你 / 收尾 + 每次调用的缓存,子 Agent 事件不参与);backend.go 接入;日志「回合完成」带「阶段 / 调用次数 / 缓存」;stages_test.go 已写、未运行', st('part')],
      ['实时阶段事件', 'core.EvTurnPhase → State.Live / LiveDone;演示后端同样发出(demo/stages_demo.go)。未测试', st('part')],
      ['状态行(始终在场)', 'app/flow_wait.go:发送同一帧出现(界面先把回合置为运行中,后端再确认)、整个回合不消失、阶段词淡入、8 秒转琥珀、点击展开阶段清单、结束时淡出并收折;后端先发用户消息再启动进程。未测试', st('part')],
      ['Esc 中断', 'app.go escInterrupts:仅 Agent 形态主界面、回合进行中、无浮层 / 编辑态 / 斜杠菜单时 Esc = 停止(状态行括号里的「Esc 中断」只在可用时才写);IDE 形态不启用(编辑器里 Esc 有别的用途)。未测试', st('part')],
      ['发送瞬间乐观运行态', 'app.go send:非引导的 CmdPrompt 在空闲时先本地置为运行中(状态行同帧出现),后端 EvTurnStarted 幂等;后端先发用户消息再启动进程(冷启动不再让用户消息晚几百毫秒才出现)。未测试', st('part')],
      ['等待动画风格(6 种)', 'app/spinner_style.go:字符帧与动词表移植自主分支;设置 → 外观 → 等待动画(appearance.spinner,默认 Claude Code)。未测试', st('part')],
      ['慢因提示卡', 'app/flow_wait_hint.go 按上表;设置 → 对话 → 显示慢响应提示(chat.slowHint);动作按钮「编辑工作区…」「压缩上下文」。未测试', st('part')],
      ['脚注迷你分段条 + 瀑布弹卡', 'app/turn_timing_pop.go:点击「用时」开弹卡,窄档隐藏条;最多 10 行;含「等你确认」时标题写「模型 x s」;底部缓存 / 上下文 / 一句结论。设计稿里的「悬停即开」Go 版改为点击(避免鼠标经过误触)。未测试', st('part')],
      ['缓存判定', 'core.CallInfo.Verdict / Text(命中 / 部分命中 / 首次写入 / 未命中;第三方不返回缓存字段时不判);弹卡里逐次调用显示。等待行不放缓存 chip(见上)', st('part')],
      ['等你确认单独成段', '权限 / 提问 / 计划请求到达 → 作答(含取消)。未测试', st('part')],
      ['历史回放分段', '回放没有逐段数据,脚注沿用旧样式(设计如此,不伪造)', st('ok')],
      ['复制诊断 / 动作按钮「复制诊断」', '设计稿慢因表里提过,Go 未做', st('todo')]]))}
  </div>`;

  /* ───── 动画字符:每 150 ms 换一帧(减弱动态时固定第一帧) ───── */
  setInterval(() => {
    if (document.body.classList.contains('no-motion')) return;
    document.querySelectorAll('.rt-gl').forEach((g) => { const set = GLYPHS[g.dataset.set] || GLYPHS.cc; const i = (+g.dataset.i + 1) % set.length; g.dataset.i = i; g.textContent = set[i]; });
  }, 150);

  /* ───── 交互:展开阶段清单、脚注弹卡 ───── */
  document.addEventListener('click', (e) => {
    const w = e.target.closest('[data-act="rt-steps"]');
    if (w) { const n = w.nextElementSibling; if (n && n.classList.contains('rt-steps')) n.hidden = !n.hidden; }
    const f = e.target.closest('[data-act="rt-pop"]');
    if (f) { f.classList.toggle('on'); const p = f.closest('.snip').querySelector('.rt-pop'); if (p) p.hidden = !p.hidden; }
  });
})();
