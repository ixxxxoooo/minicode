/* 设计稿 v4.1 · 会话流:工具分组与详细程度(设置 → 对话 → 工具调用详细程度)、思考行宽度分配
   依据(业界做法,2026-11 检索核对;细节以各产品当时版本为准):
   - Claude Code:连续的读取 / 搜索折叠成 CollapsedReadSearch「Read N files (+ M more)」一行,Ctrl+O 展开逐条。
   - Cursor:连续的读取 / 搜索收成「Explored N files, M searches」,编辑是带 +/− 的文件行,命令是单独的卡。
   - 通用组件库(如 Threadplane ChatToolCalls):按「相邻」成组,只有一个调用时不套外壳,≥2 个才出可折叠条,默认折叠。
   - hmanlab:相邻的只读调用并成一块,「展开其中任意一条会把它从组里拆出来」——本稿不采用(展开不拆组,保持行位置稳定)。
   - CC Switch:一个回合的长串动作折成一行「步骤 · 44 · 27 条命令 · 2 个文件改动 · 1 失败 · 27m45s」,失败标红,点开看参数 / 输出 / 退出码。
   每页都带「实现状态」列,与 Go 代码保持一致。
   @author ixxxxoooo */
(function () {
  const { ic, ib, tag } = U;
  const { head, sec } = P.h;
  const F = window.F;
  const tbl = (cols, rows) => `<table class="ia-tbl"><thead><tr>${cols.map(([t, w]) => `<th ${w ? `style="width:${w}px"` : ''}>${t}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map((c, k) => `<td ${k === 0 ? '' : 'class="t2"'}>${k === 0 ? `<b>${c}</b>` : c}</td>`).join('')}</tr>`).join('')}</tbody></table>`;
  const st = (s) => ({ ok: tag('已实现', 'ok'), part: tag('部分', 'warn'), todo: tag('待做', '') }[s]);
  const grid = (...cells) => `<div class="gal">${cells.join('')}</div>`;
  const cell = (t, body, why) => `<div><div class="cap2"><b>${t}</b></div>${body}<div class="why">${why}</div></div>`;
  const snip = (body, w = 520) => `<div class="snip" style="height:auto;width:${w}px;padding:12px 14px"><div class="fc g1">${body}</div></div>`;
  const nest = (...rows) => `<div class="nest">${rows.join('')}</div>`;

  /** 同一个回合里的几个步骤 */
  const S = {
    think: () => F.step('think', '定位大小写敏感的来源', '', '4 秒'),
    read1: (o) => F.step('read', 'router.rs', 'src/', '86 行', o),
    read2: (o) => F.step('read', 'router_test.rs', 'tests/', '40 行', o),
    grep: (o) => F.step('grep', '"match_path"', 'src/', '5 处', o),
    edit1: (o) => F.step('edit', 'router.rs', 'src/', '<span class="c-ok">+4</span> <span class="c-err">−1</span>', o),
    edit2: (o) => F.step('edit', 'router_test.rs', 'tests/', '<span class="c-ok">+12</span>', o),
    run: (o) => F.step('bash', 'cargo test router', '', '2.4 秒', o),
    merged: (txt, o) => F.step('group', txt, '', '', Object.assign({ verb: '' }, o || {})),
  };
  const SUM_ALL = '读取了 2 个文件、搜索了 1 次、编辑了 2 个文件、运行了 1 条命令';
  const EXPLORE = F.step('group', '2 个文件 · 1 次搜索', '', '', { verb: '探索了' });
  const exploreOpen = F.step('group', '2 个文件 · 1 次搜索', '', '', { verb: '探索了', open: 1, body: nest(S.read1(), S.read2(), S.grep()) });
  const exploreRows = nest(S.read1(), S.read2(), S.grep());
  const text = '<div class="md" style="padding:4px 0 0 22px"><p>问题出在路由表直接用原始路径做键……</p></div>';

  P['flow-detail'] = () => `<div class="ds" style="max-width:1240px">${head('会话流 · 工具分组与详细程度', '设置 → 对话 → 「工具调用详细程度」只回答一个问题:<b>一串连续的工具调用,在会话流里占几行、默认展开多少</b>。本页给出成组规则、三档的完整表现(含进行中 / 失败 / 窄档)、默认展开矩阵,以及思考行的宽度分配。它与「回合折叠 · 已处理」(chat.fold)是两层:回合折叠管「回合结束后整段收起」,本页管「回合进行中与展开后每一串怎么排」。')}
    ${sec('什么时候成一组', '一律按「相邻」判断,不跨过任何别的条目。成组靠规则、不靠模型:同一回合里,相邻的工具调用凑够 2 个才成组;只有 1 个就是普通步骤行,不套外壳。', tbl([['规则', 150], ['内容']], [
      ['相邻', '中间只要出现下列任一条目,这一串就在那里断开,后面重新计数:助手正文 · 用户消息 · 思考行 · 子 Agent 卡 · 计划 / 待办 / 提问卡 · 权限请求(待处理)· 压缩上下文卡 · 错误提示卡。思考行永远单独成行(它是模型在想,不是工具),所以「思考 → 读 → 读 → 思考 → 读」是两串而不是一串。'],
      ['凑够 2 个', '精简档:任意种类的工具调用,≥2 个相邻就合并;标准 / 详细档:只合并「探索类」(读取 / 搜索 / 查找),编辑与命令永远各占一行。'],
      ['组的身份', '组 id = 组内第一个调用的 id;之后追加的调用并入同一组,所以「用户手动展开过」的状态不会因为又来了新调用而丢。'],
      ['组的展开状态', '默认折叠(详细档的探索组默认展开);点行展开,再点收起;状态记在会话内,切走再回来保留。展开后每一条都是普通步骤行(各自再点才展开详情),左侧一条细引导线。展开其中一条不会把它从组里拆出来。'],
      ['顺序与措辞', '摘要按固定顺序列出:读取 → 搜索 → 编辑 → 运行 → 抓取 → 其它,用「、」连接,量词固定(「N 个文件」「N 次」「N 条命令」「N 个页面」「N 次工具」);探索组沿用「探索了 N 个文件 · M 次搜索」。'],
      ['被拒绝 / 待确认', '权限请求待处理时它是一张卡(断开成组);用户拒绝后该调用带删除线、图标是锁,仍留在原来的组里,计入「N 项失败」。']]))}
    ${sec('三档对照:同一个回合', '同一段过程:思考 → 读 2 个文件 + 搜 1 次 → 改 2 个文件 → 跑测试。精简只留一行;标准保持「读 / 搜」合并、其余一行一条且默认收起;详细把差异与命令输出直接铺在行下。设置页的预览用的是同一份数据。', grid(
      cell('精简', snip(S.think() + S.merged(SUM_ALL) + text), '一整串 = 一行;图标是分组图标,没有动词(「读取了…」本身就是动词);点开看逐条。右侧不放用时 / 行数。'),
      cell('标准(默认)', snip(S.think() + EXPLORE + S.edit1() + S.edit2() + S.run() + text), '只有读 / 搜合并;编辑与命令一行一条。点行才展开差异 / 输出(失败的命令例外,见下表)。'),
      cell('详细', snip(S.think() + exploreOpen + S.edit1({ open: 1, body: F.diff(false) }) + S.run({ open: 1, body: F.term(false) }) + text), '探索组默认展开,每一条逐行列出;编辑直接显示差异,命令直接显示输出(超长按「长输出限高」处理)。'))
    )}
    ${sec('精简档的行为', '精简档把「过程」压成一行,所以它必须把三件事说清楚:现在在干什么、有没有出错、能不能点开看。', grid(
      cell('进行中', snip(S.think() + F.step('group', 'cargo test router', '', '已 4 步', { verb: '运行', state: 'run' }) ), '组里还有调用在跑:行进入运行态(沙漏 + 流光),文字换成「最新一步」的动词 + 对象,右侧写「已 N 步」;结束后淡入 120 ms 换成摘要句。点开能看到已完成的逐条与正在跑的那条。'),
      cell('有失败', snip(F.step('group', '读取了 2 个文件、运行了 2 条命令', '', '<span class="c-err">1 项失败</span>', { verb: '', state: 'err' })), '失败不拆组、不自动展开:图标换成红色错误图标,右侧红字「N 项失败」。点开后失败的那一条自带红色图标与退出码;失败的步骤不自动展开,点才看详情。'),
      cell('展开后', snip(F.step('group', SUM_ALL, '', '', { verb: '', open: 1, body: nest(S.read1(), S.read2(), S.grep(), S.edit1(), S.edit2(), S.run()) })), '逐条是普通步骤行(各自点才展开详情),引导线与探索组一致;展开 / 收起沿用步骤行的淡入下落,高度不插值。'),
      cell('窄档(对话栏 < 420)', snip(F.step('group', '读 2 · 搜 1 · 改 2 · 跑 1', '', '', { verb: '' }), 320), '窄档用短句:动词只留一个字、去掉「了」与量词,仍按固定顺序;摘要放不下时末尾省略号,不换行。右侧附注一律让位(沿用步骤行窄档规则)。'))
    )}
    ${sec('默认展开矩阵', '表里的「展开」指默认状态;任何时候点行都能手动切换,手动状态优先于默认。', tbl([['条目', 130], ['精简', 140], ['标准', 200], ['详细']], [
      ['读取 / 搜索 / 查找', '并入摘要行', '≥2 个合并成探索组,默认收起;1 个是普通行,默认收起', '探索组默认展开;单个读取行展开显示入参与结果'],
      ['编辑 / 写入', '并入摘要行', '一行一条,默认收起(+/− 常驻在行右侧);<b>被拒绝</b>时展开原因(<b>失败</b>不自动展开,只在行上标红)', '一行一条,差异默认展开(>8 行限高并出「展开全部」)'],
      ['命令(Bash)', '并入摘要行', '一行一条,默认收起,右侧写用时;<b>运行中</b>展开实时输出(结束后不自动收起,避免跳动);<b>非 0 退出码</b>展开', '输出默认展开(超长限高)'],
      ['抓取 / MCP / 浏览器', '并入摘要行', '一行一条,默认收起', '入参与结果默认展开'],
      ['思考', '单独一行(不并入)', '单独一行,默认收起;流式折叠时出跑马灯', '同标准;「思考内容默认展开」另有开关'],
      ['子 Agent / 计划 / 提问 / 压缩 / 权限', '各自成卡,与档位无关', '同左', '同左']]))}
    ${sec('与「回合折叠」的关系', '回合结束且开了「回合结束后折叠过程」时,整个回合的步骤收成一行「已处理 1 分 13 秒 · 3 步 · 2 个文件」(见「浮层与面板」页)。展开它之后,里面每一串按当前档位再排一次——精简档里是「摘要行」,标准档里是探索组 + 逐行。最后一段助手正文、计划卡、回答卡、压缩卡、待处理的权限卡不被折进去。回合进行中不折叠。', grid(
      cell('回合结束 · 折叠', snip(F.step('group', '1 分 13 秒', '3 步 · 2 个文件', '', { verb: '已处理' }) + text), '外层一行;内层按档位(此处不展示)。'),
      cell('折叠 + 展开 · 精简', snip(F.step('group', '1 分 13 秒', '3 步 · 2 个文件', '', { verb: '已处理', open: 1, body: nest(S.think(), S.merged(SUM_ALL)) }) + text), '「已处理」展开后与合并工具同构:同一条引导线(正对上一行图标中线,1px)、子项同缩进;里面的摘要 / 探索组展开后在子项图标中线再画自己的线。思考正文的竖线也是这一条线(位置、粗细与合并工具一致,不再是 2px 贴正文)。'))
    )}
    ${sec('思考行的宽度分配', '一行思考 = 图标 · 「思考」· <b>标题</b> · 用时。<b>流式折叠时</b>标题位置换成上下翻预览(不并排放两份文字);结束后是静态标题。窄档不显示预览,始终是标题。标题 / 预览都只按宽度省略,不按字数截断。', tbl([['优先级', 90], ['规则']], [
      ['1 标题', '取思考正文的第一句(按换行 / 句末标点切,文件名里的点不切),<b>不按字数截断</b>,放不下时行尾省略号;正文还没有句号 / 换行(流式刚开始)就取已有的全部文字;一个字都没有时写「思考中」。标题是普通步骤行的对象文字:单行、可被挤压(最后才省略),颜色 Tx3。'],
      ['2 用时', '右侧固定,等宽数字;流式时「5.8s」,结束后「4 秒」。永不换行、永不被挤掉。'],
      ['3 跑马灯', '只在「流式 + 折叠 + 有正文」时出现,顶替标题的位置,占「思考」与用时之间的<b>全部宽度</b>(flex 1);窄档(&lt; 420)不显示、回到静态标题。框内上下边缘各 20% 渐隐(滚出 / 滚入的句子在边缘淡出淡入,稳态那一行不受影响)。只显示思考里<b>最新的一句</b>(按换行 / 句末标点切,最多 160 字,超出省略号),靠左紧挨标题;不再横向平移、不再渐隐。'],
      ['4 空白', '行内不允许出现「标题后面一大片空白而预览挤在前面」:跑马灯框是弹性的,有预览就撑满,没有预览(未流式 / 展开 / 窄档)就不占位,用时靠 Spacer 贴右。'],
      ['动效', '<b>上下翻</b>:新一句开始时,旧句向上滚出一行、新句从下面翻进来(260 ms 缓出,旧句淡出 / 新句淡入);同一句只是又长了几个字则原地接着写、不翻页。窗口失焦或「减弱动态」时不翻页、直接显示最新一句,不开持续帧;流式结束时预览淡出 160 ms,同时用时淡入。']]))}
    ${sec('思考行 · 各种宽度', '同一条思考在三种对话栏宽度下:宽档标题 + 跑马灯 + 用时;中档标题略省略、跑马灯变短;窄档(&lt; 420)不显示跑马灯,只留标题与用时。下面第二行是英文长句(模型用英文思考时很常见)与「还没有标题」的情形。', grid(
      cell('宽档 · 712', snip(F.thinkMarquee({}), 520), '标题 + 上下翻预览(靠左、单行省略)+ 用时。'),
      cell('中档 · 520', snip(F.thinkMarquee({ preview: '…最小改动是抽一个 normalize · 顺带处理末尾斜杠' }), 400), '跑马灯随剩余宽度缩短,最小 96 px。'),
      cell('窄档 · 352', snip(F.thinkMarquee({ narrow: 1 }), 300), '窄档隐藏跑马灯,标题单行省略,用时贴右。'),
      cell('英文长句', snip(F.thinkMarquee({ title: 'The user wants me to check whether the app…', preview: 'rather than another quick claim. Best response: do a quick factual check of the app’s router before answering' }), 520), '标题 28 字截断;跑马灯仍占满剩余宽度、文字尾部贴右,不会「全挤在前半截」。'),
      cell('还没有标题', snip(F.thinkMarquee({ title: '思考中', preview: 'Hmm' }), 520), '流式刚开始只有几个字:标题写「思考中」,预览文本没溢出时不画渐隐、靠右。'),
      cell('结束 / 展开', snip(F.thinkMarquee({ stream: false }) + F.thinkMarquee({ stream: false, open: 1 }), 520), '结束后跑马灯收起,用时变「4 秒」;展开显示全文(无预览)。'))
    )}
    ${sec('业界做法对照', '本稿取各家共同点,不照搬某一家:成组靠相邻、单个不套壳、默认折叠、失败可见;不取「展开即拆组」。', tbl([['产品', 140], ['做法'], ['本稿取舍']], [
      ['Claude Code', '连续读 / 搜 → 「Read N files (+ M more)」一行;Ctrl+O 看逐条', '取:相邻合并、摘要句。不取:仅限读 / 搜(精简档放宽到所有种类)'],
      ['Cursor', '「Explored N files, M searches」;编辑是文件行 +/−;命令独立卡', '取:标准档形态'],
      ['CC Switch(会话查看器)', '整回合一行「步骤 · 44 · 27 条命令 · 2 个文件改动 · 1 失败」,失败标红', '取:精简档摘要 + 失败计数红字'],
      ['通用 ChatToolCalls', '按相邻同名成组,单个不套壳,默认折叠', '取:≥2 才成组、单个不套壳'],
      ['hmanlab TUI', '只读调用并成一块,展开任一条会把它拆出组', '不取:拆组会让行位置跳动']]))}
    ${sec('实现状态', '未测试:Go 侧仅 go build + cmd/shot(-detail 0|1|2)截图;后加的几行只有单测文件(flow_group_perm_test.go,未运行),未用 computer use 实测。', `<table class="ia-tbl"><thead><tr><th style="width:240px">项</th><th>说明</th><th style="width:70px">状态</th></tr></thead><tbody>
      ${[
        ['精简档合并摘要', 'flow_merge.go 相邻 ≥2 合并;进行中 / 失败 / 窄档短句已接线', 'part'],
        ['精简档进行中 / 失败态', '最新一步 + 「已 N 步」;红 errc + 「N 项失败」;结束后 120ms 摘要淡入未接', 'part'],
        ['窄档短句', 'mergeSummaryNarrow:读 2 · 搜 1 · 改 2 · 跑 1', 'part'],
        ['标准档默认折叠(失败 / 运行中例外)', 'present.go applyToolDefaultOpen', 'part'],
        ['详细档探索组默认展开', 'exploreGroup ToolDetailLevel==2 默认展开', 'part'],
        ['断组条目(思考 / 子 Agent / 卡片)', 'stepGroup 已按条目类型断开;核对压缩卡 / 错误卡', 'part'],
        ['思考行宽度分配', '标题常驻 + 跑马灯 flex(max 52%、<96px 隐藏);cmd/shot -ed thinking 截图,未 computer use', 'part'],
        ['已处理的权限请求并入分组', 'mergeable / isExplore 用 doneToolOf 解开权限请求里的工具调用,「需要确认的编辑 / 命令」不再把相邻一串截断;被拒绝的留在组里、计入「N 项失败」(仅编译,未测试)', 'part'],
        ['组内每一条可展开', 'groupChild:探索组与精简摘要组的逐条都是普通步骤行(点才展开入参 / 结果),被拒绝 / 进行中的自动展开(失败的不自动展开),展开不拆组', 'part'],
        ['回合折叠展开后按档位再成组', 'turnFoldGroup 经 stepSeq 再排一次(精简 = 摘要行,标准 = 探索组 + 逐行),已答提问卡把串断开', 'part'],
        ['命令运行中展开、结束后不自动收起', 'applyToolDefaultOpen 把运行中自动展开记进 folds;用户运行期间自己收起的不覆盖', 'part'],
        ['差异限高', 'diffCard:详细档 > 8 行、其余 > 12 行内部滚动 + 「展开全部 · 共 N 行 / 收起」,展开上限 480px', 'part'],
        ['工具令牌补全', '后台 Bash(后台运行 + 已启动)/ Monitor(监听)/ 后台输出 / 结束后台任务 / 联网搜索(N 条结果)/ Glob(N 个文件)各有动词与图标', 'part'],
        ['失败 / 拒绝 / 中断的右侧结果', '失败的命令写「退出码 N」(红),被拒绝「已拒绝」,被中断「已中断」', 'part'],
        ['精简摘要行结束时 120 ms 淡入换句', 'mergeGroup 记 runWas / fadeAt,stepRow 按 fadeAt 画 120 ms 透明度淡入(仅编译,未测试)', 'part'],
        ['「最新一步」翻滚(.ticker)', 'tickerRow:换步时新行自下滑入 + 淡入、旧行向上顶出 + 淡出,200 ms 缓出;探索组、子 Agent 调用卡、并行组共用,减弱动态直接换(仅编译,未测试)', 'part']
      ].map(([a, b, c]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td><td>${st(c)}</td></tr>`).join('')}
    </tbody></table>`)}
    ${sec('正文里的路径与代码块', '参照 BaoCode:正文里出现的文件 / 文件夹可以直接点开;行内代码与代码块的样子一起收拾得精致些。', grid(
      cell('可点路径 + 行内代码', `<div class="md" style="padding:10px 14px;line-height:1.9"><p>入口在 <span class="pathref">${ic('folder', 's12')}<code>src/router</code></span>,处理函数见 <span class="pathref">${ic('code', 's12')}<code>src/router/match.rs:42</code></span>,配置是 <span class="pathref">${ic('gear', 's12')}<code>Cargo.toml</code></span>。不存在的 <code>src/ghost.rs</code> 和命令 <code>cargo test</code> 只是普通行内代码。</p></div>`, '只有本机真实存在的路径才成为链接并带图标(路径里可以有空格、中文和各种标点,也认 ~/ 开头,如「~/Downloads/MiniPost 界面重设计 — 组件 · 标题栏.html」)(图标按类型取彩色的 Material Icon Theme,和路径文字一起装在一枚整体的圆角胶囊里(配色跟主题强调色走,与旁边的行内代码同一族:强调色淡底 + 细边 + 强调色文字;图标与文字同一条水平中线),没有接缝,如 index.html 是网页图标、gen_icon.py 是 Python 图标、SHA256SUMS 是钥匙)(后台探测一次、缓存;探测完下一帧才变成链接)。悬停:手型光标 + 芯片外一圈强调色淡底(不画下划线),停一会儿出提示「点击打开」。点文件 = 图片开灯箱;网页(.html / .htm)用系统默认浏览器打开;PDF / Office / 压缩包 / 音视频等非文本文件交给系统默认程序打开;文本文件在工作区打开(带 :行:列 就跳到行);点文件夹 = 切到左栏「文件」并展开选中(项目外的在访达里显示)。右键:在工作区打开 / 在访达中显示 / 复制路径。'),
      cell('代码块:带语言', `<div class="md" style="padding:6px 0"><pre><div class="ph"><span class="grow">rust</span>${ib('copy', 'sm', '复制')}</div><code>fn main() {\n    println!("你好");\n}</code></pre></div>`, '标题栏 26px:语言名(tx-3)+ 复制钮,复制钮悬停代码块时才出现;圆角 r-lg,内边距 8 / 12,字体用「设置 → 编辑器 → 字体」的代码字体链。'),
      cell('读取图片(步骤行 + 展开的图)', `<div class="snip" style="height:auto;width:520px;padding:12px 14px"><div class="fc g1">${F.step('read', 'prototype.png', '/tmp/…/', '', { open: 1, body: '<div style="width:260px;height:160px;border-radius:var(--r-md);box-shadow:inset 0 0 0 1px var(--bd-1);background:var(--fill-1);display:flex;align-items:center;justify-content:center;color:var(--tx-4)">' + ic('image', 's16') + '</div>' })}</div></div>`, 'Claude 读取的文件是图片(png / jpg / gif / webp …)时——显示的是工具结果里带回来的那张图(翻译器已落盘到临时目录),所以原文件被清理(如 /tmp 下的截图)也能看;没有带回图时才找原文件——步骤行带展开箭头、默认展开(精简档收起),行下面直接显示这张图:按原比例缩进 480×320 的框里,圆角 + 1px 细边,悬停边框加深 + 提示「点击查看」,点图片打开全屏灯箱;步骤行上不再写路径附注(图就在下面),悬停动作与右键菜单的「打开」改为「查看图片」(打开的是步骤里显示的那张图,原文件被清理也能开,不再提示「文件已不存在」);折叠的分组行(「读取了 N 个文件…」「已处理 …」)右边按张显示缩略图——里面读到几张图就显示几枚(高 20,宽随比例,最多 8 枚,多的写「+N」),点任一枚打开灯箱并可左右翻看全部;解码中显示带图片图标的占位,解码失败显示一行文字。文件已不存在时回到原来的「读取行(点整行 = 工作区打开 / 提示已不存在)」。参照 BaoCode「已读取 xxx.png ⌄ + 图」。'),
      cell('代码块:纯文本', `<div class="md" style="padding:6px 0"><pre><span class="cp">${ib('copy', 'sm', '复制')}</span><code>目录结构\n├─ src\n└─ Cargo.toml</code></pre></div>`, '没写语言(或 text)的块不画标题栏,复制钮悬停时浮在右上角。')
    ))}
    ${sec('正文路径与代码块 · 实现状态', '', `<table class="ia-tbl"><tbody>${[
      ['行内代码配色不变 + 跟随代码字体链', '仍是强调色字 + 强调色 11% 底(中性配色的尝试已按要求撤回);Markdown 主题 CodeFont 跟随设置里的代码字体;未测试', 'part'],
      ['会话里的路径可点', 'path_ref.go:后台探测存在性并缓存;存在的路径在渲染前被改写成 [`路径`](<路径>) 链接(框架自带的 CodeLink 画不了图标),LinkIcon 在代码前画带底色的彩色类型图标(matico 包,取自 Material Icon Theme,MIT,与 BaoCode 同一份),点击打开 / 文件树定位,右键加「在访达中显示」。悬停效果与「无下划线」靠 MyGo 补丁(scripts/mygo-overlay.sh,补丁是 mygo/_patches/*.patch 里的统一 diff,与上游对不上时脚本会报警、package 直接失败;dev / package / compare 脚本自动带上,直接 go build 则没有补丁、悬停仍是框架自带的下划线)。胶囊由补丁让 LinkIcon 对象整枚自画(底色 + 边 + 图标 + 路径文字),链接自己的文字不再另画;所以胶囊里的路径文字不能单独选中复制(右键「复制路径」可以),过长的路径从中间省略(≤380px);没打补丁时退回「图标 + 链接文字」。已知缺口:只在助手正文里生效(用户消息、工具输出里的路径不链接);相对路径只按项目根解析;未测试', 'part'],
      ['读取图片的步骤直接显示图 + 折叠行缩略图', 'imagesOf / stepThumbs:探索组、精简合并行、回合折叠行折叠时,右侧按张显示缩略图,点开灯箱;悬停动作 / 右键「查看图片」不再走原路径;present.go 读图时 hasBody / imgRead,toolBody 走 readImageCard(flow_read_image.go):localImage 后台解码,480×320 内按比例缩放,点击开灯箱;分组里的读图行(groupChild)也默认展开;未测试', 'part'],
      ['代码块标题栏 / 悬停复制', '带语言 = 标题栏 + 悬停才出现的复制钮,纯文本块无标题栏、复制钮浮在右上;圆角 r-lg、内边距 8 / 12;未测试', 'part']
    ].map(([a, b, c]) => `<tr><td><b>${a}</b></td><td class="t2">${b}</td><td>${st(c)}</td></tr>`).join('')}
    </tbody></table>`)}
  </div>`;
})();
