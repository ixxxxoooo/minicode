/* 设计稿 v3 · 模型服务商:全局状态、设置页模板、运行配置弹层
   原型演示:URL 含 example.com → 获取失败(401);含 legacy → 不支持模型列表;其余成功。
   @author ixxxxoooo */
(function () {
  const { ic, btn, ib, chip, tag, seg, inp, field, esc, chk, sel } = U;

  const OFFICIAL_MODELS = [
    ['fable', 'Fable 5.2', '最强'],
    ['opus', 'Opus 5.5', ''],
    ['sonnet', 'Sonnet 5.5', ''],
    ['haiku', 'Haiku 5.5', '最快'],
  ];

  const PRESETS = [
    { name: 'DeepSeek', url: 'https://api.deepseek.com/anthropic', auth: 0, modelsUrl: '', models: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-v3'], abbr: 'DS' },
    { name: '智谱 GLM', url: 'https://open.bigmodel.cn/api/anthropic', auth: 0, modelsUrl: '', models: ['glm-4.6', 'glm-4.5-air', 'glm-4-flash'], abbr: 'GLM' },
    { name: 'Kimi', url: 'https://api.moonshot.cn/anthropic', auth: 0, modelsUrl: '', models: ['kimi-k2-turbo-preview', 'moonshot-v1-128k'], abbr: 'KM' },
    { name: 'MiniMax', url: 'https://api.minimaxi.com/anthropic', auth: 0, modelsUrl: '', models: ['MiniMax-M2', 'abab6.5s-chat'], abbr: 'MM' },
    { name: '阿里百炼 Qwen', url: 'https://dashscope.aliyuncs.com/apps/anthropic', auth: 0, modelsUrl: '', models: ['qwen3-coder-plus', 'qwen-max'], abbr: 'QW' },
    { name: 'Anthropic API', url: 'https://api.anthropic.com', auth: 1, modelsUrl: '', models: ['claude-opus-5-5', 'claude-sonnet-5-5', 'claude-haiku-5-5'], abbr: 'API' },
  ];

  const SLOTS = [
    ['default', '默认', 'ANTHROPIC_MODEL', 1],
    ['opus', 'Opus', 'ANTHROPIC_DEFAULT_OPUS_MODEL', 0],
    ['sonnet', 'Sonnet', 'ANTHROPIC_DEFAULT_SONNET_MODEL', 0],
    ['haiku', 'Haiku(后台小任务)', 'ANTHROPIC_DEFAULT_HAIKU_MODEL', 0],
    ['subagent', '子 Agent', 'CLAUDE_CODE_SUBAGENT_MODEL', 0],
  ];

  const defaultState = () => ({
    active: 'official',
    officialModel: 'sonnet',
    picks: { official: 'sonnet', deepseek: 'deepseek-chat' },
    editId: '',
    editNew: 0,
    mappingProv: 'deepseek',
    slots: { default: 'deepseek-chat', opus: 'deepseek-reasoner', sonnet: 'deepseek-chat', haiku: 'deepseek-chat', subagent: '' },
    list: [
      { id: 'official', name: 'Claude 官方登录', url: '', fixed: 1, status: 'ok', hasKey: 1, models: [], selected: [], manual: [], fetchedAt: '', auth: 1, abbr: '' },
      { id: 'deepseek', name: 'DeepSeek', url: 'https://api.deepseek.com/anthropic', status: 'ok', hasKey: 1, models: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-v3'], selected: ['deepseek-chat', 'deepseek-reasoner'], manual: [], fetchedAt: '刚刚获取', auth: 0, modelsUrl: '' },
      { id: 'glm-empty', name: '智谱 GLM', url: 'https://open.bigmodel.cn/api/anthropic', status: 'warn', hasKey: 1, models: [], selected: [], manual: [], fetchedAt: '', auth: 0, modelsUrl: '' },
      { id: 'legacy', name: 'Legacy 网关', url: 'https://api.legacy.local/anthropic', status: 'warn', hasKey: 1, models: ['custom-model-a'], selected: ['custom-model-a'], manual: ['custom-model-a'], fetchedAt: '', auth: 0, modelsUrl: '' },
      { id: 'gateway', name: '自建网关', url: 'https://llm.example.com/v1', status: '', hasKey: 0, models: [], selected: [], manual: [], fetchedAt: '', auth: 0, modelsUrl: '' },
    ],
  });

  const idSlug = (n) => n.toLowerCase().replace(/[^\w\u4e00-\u9fa5]+/g, '-').replace(/^-|-$/g, '') || 'prov';

  /** APP 还没建好时(overlays.js 等模块在加载期就会渲染一次弹层)用的临时状态,APP 就绪后改用 st.prov */
  let bootState = null;
  const ensure = () => {
    const st = window.APP && window.APP.st;
    if (!st) return bootState || (bootState = defaultState());
    const d = defaultState();
    if (!st.prov) st.prov = d;
    else {
      st.prov = Object.assign({}, d, st.prov);
      if (!st.prov.list || !st.prov.list.length) st.prov.list = d.list;
    }
    return st.prov;
  };

  const prov = (id) => ensure()?.list.find(p => p.id === id);
  const isOfficial = (id) => id === 'official';
  // 品牌 logo:服务商按名称 + 域名识别(不看带路径的地址),模型按模型 ID,认不出的模型沿用服务商的品牌;都认不出显示占位芯片
  const hostOf = (u) => { try { return new URL(u).host; } catch (e) { return ''; } };
  const provBrand = (p) => p.fixed ? 'claude' : LOGOS.brandOf(p.name) || LOGOS.brandOf(hostOf(p.url));
  const modelBrand = (m, pb) => LOGOS.brandOf(m) || pb;
  const abbr = (p) => p.abbr || (p.name.length <= 4 ? p.name : p.name.slice(0, 2).toUpperCase());

  const statusTag = (p) => {
    if (p.fixed) return tag('已连通', 'ok');
    if (p.status === 'testing') return tag('测试中', 'ac');
    if (p.status === 'ok') return tag('已连通', 'ok');
    if (p.status === 'err') return tag('失败', 'err');
    if (p.status === 'warn') return tag('未验证', 'warn');
    return '';
  };

  const provMeta = (p) => {
    const n = p.selected.length;
    const key = p.fixed ? '官方账号' : (p.hasKey ? '密钥已保存' : '未填写');
    return `${n ? n + ' 个模型 · ' : ''}${key}`;
  };

  const provRow = (p) => {
    const on = ensure().active === p.id;
    const acts = p.fixed
      ? `<span class="prov-st">${on ? tag('当前', 'ac') : btn('切换', 'sm', '', 'data-act="prov-use"')}</span>`
      : `<span class="prov-st">${on ? tag('当前', 'ac') : btn('切换', 'sm', '', 'data-act="prov-use"')}</span>${btn('测试', 'sm ghost', 'activity', 'data-act="prov-test"')}${ib('edit', 'sm', '编辑', 'data-act="prov-edit" data-id="${p.id}"')}${ib('trash', 'sm', '删除', 'data-act="del-row"')}`;
    return `<div class="set-row" data-prov data-id="${p.id}" data-rm ${p.fixed ? 'data-fixed="1"' : ''}><span class="si-box">${LOGOS.logo(provBrand(p), 16)}</span><div class="grow min-w-0"><div class="m f ai g3">${esc(p.name)}<span class="prov-res">${statusTag(p)}</span></div><div class="d mono ell">${esc(p.url || '通过本机 claude 登录')}</div><div class="t3 fs-cap">${provMeta(p)}</div></div>${acts}</div>`;
  };

  const modelInventory = (p, draft) => {
    const ed = draft || p;
    const all = [...new Set([...(ed.models || []), ...(ed.manual || [])])];
    const picked = ed.selected || [];
    const fetched = ed.fetchedAt;
    const load = ed._load;
    const err = ed._fetchErr;
    const unsup = ed._unsupported;
    const filter = ed._filter || '';
    const q = filter.trim().toLowerCase();
    const rows = all.filter(m => !q || m.toLowerCase().includes(q));
    const canFetch = !!(ed.hasKey || ed._keyDraft);
    const pickN = picked.filter(m => all.includes(m)).length;
    const head = `<div class="prov-inv-h f ai g3"><b>模型清单</b><span class="t3 fs-cap prov-inv-cnt">已选 ${pickN} / 共 ${all.length}</span><span class="grow"></span>${btn('获取模型列表', `sm ghost${load ? ' loading' : ''}${canFetch ? '' : ' is-disabled'}`, 'refresh', `data-act="prov-fetch"${canFetch ? '' : ' data-tip="先填写 API Key"'}`)}</div>
      <div class="t3 fs-cap prov-inv-hint">只有勾选的模型会出现在「模型映射」下拉与会话里的模型选择器。</div>
      ${fetched ? `<div class="t4 fs-cap prov-inv-time">上次获取 · ${esc(fetched)}</div>` : ''}
      ${err ? `<div class="prov-banner err"><span class="grow">${esc(err)}</span>${btn('重试', 'sm', '', 'data-act="prov-fetch"')}${btn('改为手动添加', 'sm ghost', '', 'data-act="prov-manual-focus"')}</div>` : ''}
      ${unsup ? `<div class="prov-banner warn"><span class="grow">该服务商没有模型列表接口,请手动添加,或填写上方的模型列表地址。</span>${btn('手动添加', 'sm', '', 'data-act="prov-manual-focus"')}</div>` : ''}`;

    const toolbar = all.length ? `<div class="prov-inv-bar f g2 ai">${inp('过滤模型 ID', { cls: 'fill', style: 'flex:1', id: 'prov-mq', v: filter, attr: 'data-in="prov-mq"' })}${btn('全选', 'sm ghost', '', 'data-act="prov-all"')}${btn('清空', 'sm ghost', '', 'data-act="prov-clear"')}</div>` : '';

    let body;
    if (load) body = `<div class="prov-inv-list fc g2">${[1, 2, 3, 4].map(() => '<div class="prov-skel"><i class="skel" style="width:18px;height:18px;border-radius:4px"></i><i class="skel grow"></i></div>').join('')}</div>`;
    else if (!all.length) body = `<div class="prov-inv-empty">${ic('list', 's20 t4')}<b>还没有模型</b><div class="f g2">${btn('获取模型列表', 'sm pri', 'refresh', `data-act="prov-fetch"${canFetch ? '' : ' disabled'}`)}${btn('手动添加', 'sm ghost', 'plus', 'data-act="prov-manual-focus"')}</div></div>`;
    else body = `<div class="prov-inv-list scroll">${rows.map(m => {
      const on = picked.includes(m);
      const man = (ed.manual || []).includes(m);
      return `<div class="prov-mod ${on ? 'on' : ''}" data-act="prov-toggle" data-model="${esc(m)}" tabindex="0">${chk(on ? 'done' : '')}${LOGOS.logo(modelBrand(m, provBrand(ed)), 14)}<span class="mono grow ell">${esc(m)}</span>${man ? `${tag('手动', 'info')}${ib('x', 'sm', '移除', `data-act="prov-rm-model" data-model="${esc(m)}"`)}` : ''}</div>`;
    }).join('')}${rows.length ? '' : `<div class="t3 fs-cap" style="padding:8px 6px">没有匹配的模型</div>`}</div>`;

    const foot = `<div class="prov-inv-add f g2">${inp('模型 ID', { id: 'prov-add-m', style: 'flex:1', attr: 'data-in="prov-add-m"' })}${btn('添加', 'sm pri', 'plus', 'data-act="prov-add-model"')}</div>`;
    return `<div class="prov-inv card" style="padding:10px 12px;gap:8px">${head}${toolbar}${body}${foot}</div>`;
  };

  const editPanel = (st) => {
    const P = ensure();
    const id = P.editId;
    const draft = P._draft;
    if (!id && !P.editNew) return '';
    const p = draft || (id ? prov(id) : { name: '', url: '', hasKey: 0, auth: 0, modelsUrl: '', models: [], selected: [], manual: [] });
    const title = P.editNew ? '新服务商' : `编辑 · ${p.name}`;
    const showDel = id && !p.fixed;
    const auth = p.auth ?? 0;
    return `<div class="fold open" id="prov-edit"><div class="fold-in">${G(title,
      R('名称', '', field(p.name || '', 190, { id: 'prov-name', ph: '例如 DeepSeek' })),
      R('Base URL', '需以 http:// 或 https:// 开头,兼容 Anthropic 协议', field(p.url || '', 280, { mono: 1, id: 'prov-url', ph: 'https://…' })),
      R('API Key', '单独存放在 0600 权限的文件;界面不回显', `<span class="f g2 ai">${field('', 200, { mono: 1, id: 'prov-key', ph: p.hasKey && id ? '已保存,留空保持原值' : 'sk-…', attr: 'type="password"' })}${ib('eye', 'sm', '显示 / 隐藏', 'data-act="prov-key-eye"')}</span>`),
      R('鉴权方式', 'Bearer → Authorization 头;x-api-key → Anthropic 官方样式', `<div class="fc g1">${seg(['Bearer', 'x-api-key'], auth, '', 'prov-auth')}<span class="t3 fs-cap">Bearer 对应 Authorization: Bearer;x-api-key 对应 x-api-key 请求头。</span></div>`),
      R('模型列表地址', '可选;留空则自动尝试 {Base URL}/v1/models', field(p.modelsUrl || '', 280, { mono: 1, id: 'prov-models-url', ph: '留空自动尝试 {Base URL}/v1/models' }), { restart: 0 }),
      `<div class="t3 fs-cap" style="margin:-4px 0 6px">有些服务商的模型列表不在 Anthropic 路径下,例如 DeepSeek 为 https://api.deepseek.com/models</div>`,
      modelInventory(p, draft),
      `<div class="set-row" style="justify-content:space-between;flex-wrap:wrap;gap:8px">${showDel ? btn('删除', 'danger', 'trash', 'data-act="prov-delete"') : '<span></span>'}<span class="f g2">${btn('取消', 'ghost', '', 'data-act="prov-cancel"')}${btn('保存', '', '', 'data-act="prov-save"')}${btn('保存并测试', 'pri', '', 'data-act="prov-save" data-test="1"')}</span></div>`)}</div></div>`;
  };

  const G = (title, ...rows) => `<div class="set-grp"><div class="gt">${title}</div><div class="card">${rows.join('')}</div></div>`;
  const R = (t, d, ctl, o = {}) => `<div class="set-row" data-k="${esc(t)}"><div class="grow"><div class="m">${t}</div>${d ? `<div class="d">${d}</div>` : ''}</div>${ctl}</div>`;

  const presetPop = () => `<div class="pop prov-preset a-pop-up" style="padding:4px;min-width:260px">${PRESETS.map(pr => `<div class="mi" data-act="prov-preset" data-name="${esc(pr.name)}"><span class="grow"><b>${esc(pr.name)}</b><span class="t3 fs-cap d block mono">${esc(pr.url)}</span></span></div>`).join('')}<div class="sep"></div><div class="mi" data-act="prov-preset" data-custom="1"><span class="grow">自定义服务商…</span></div></div>`;

  const mappingBlock = () => {
    const P = ensure();
    const mp = P.mappingProv;
    const official = isOfficial(mp);
    const items = P.list.map(p => {
      let dis = 0, reason = '';
      if (!p.fixed && !p.hasKey) { dis = 1; reason = '未填写 API Key'; }
      else if (!p.fixed && !p.selected.length) { dis = 1; reason = '模型清单为空'; }
      return { label: p.name, value: p.id, disabled: dis, hint: reason, logo: 1, brand: provBrand(p) };
    });
    const cur = prov(mp);
    let mapTag = tag('映射生效中', 'ok');
    if (official) mapTag = tag('使用官方登录', 'info');
    else if (!cur?.hasKey) mapTag = tag('未生效 · 未填写 API Key', 'warn');
    else if (!cur?.selected.length) mapTag = tag('未生效 · 模型清单为空', 'warn');

    const slotRows = SLOTS.map(([key, label, env, req]) => {
      const v = P.slots[key] || '';
      const disp = v || (req ? (cur?.selected[0] || '—') : '不设置(沿用默认)');
      const models = (cur && !official ? cur.selected : []).map(m => ({ label: m, value: m, logo: 1, brand: modelBrand(m, provBrand(cur)) }));
      if (!req) models.unshift({ label: '不设置(沿用默认)', value: '' });
      const selItems = models.length ? models : [{ label: '—', value: '', disabled: 1, hint: '请先勾选模型' }];
      return R(label, official ? '' : `<span class="t3 fs-cap" data-tip="${env}">${env}</span>`, sel(selItems, Math.max(0, selItems.findIndex(x => x.value === (official ? '' : v))), { mono: 1, w: 220, disabled: official, attr: `data-sel="map-slot" data-slot="${key}"${official ? ' data-official="1"' : ''}` }), {});
    });

    return `${G('模型映射',
      R('使用的服务商', '新会话与重新连接时,以环境变量注入映射', `<span class="f g2 ai wrap">${sel(items, Math.max(0, items.findIndex(x => x.value === mp)), { w: 200, attr: 'data-sel="map-prov"' })}${mapTag}</span>`),
      `<div class="prov-map-slots ${official ? 'dim' : ''}">${official ? '<div class="t3 fs-cap" style="padding:4px 2px 8px">使用 Claude 官方登录,不注入映射。</div>' : ''}${slotRows.join('')}</div>`,
      `<div class="t3 fs-cap" style="padding:0 2px 4px">映射在启动 claude 时以环境变量注入,不改 ~/.claude;对之后新建或重新连接的会话生效。</div>`)}`;
  };

  const providersSection = () => {
    const P = ensure();
    const rows = P.list.map(provRow).join('');
    const add = `<div class="f g3 ai rel">${btn('添加服务商', '', 'plus', 'data-act="prov-add-menu"')}<span class="t3 fs-cap grow">第三方需兼容 Anthropic 协议;可在下方映射 Opus / Sonnet / Haiku 等槽位。</span></div>`;
    return `${G('服务商', rows)}${add}${editPanel()}`;
  };

  const cfgPopHtml = (o = {}) => {
    const P = ensure();
    const active = P.active;
    const perm = [['shield', '逐项确认', 1], ['edit', '自动接受编辑'], ['list', '计划模式'], ['bolt', '跳过权限确认']];
    const groups = [];
    const offPick = P.picks.official || P.officialModel || 'sonnet';
    const offHtml = OFFICIAL_MODELS.map(([id, t, d]) => {
      const on = active === 'official' && offPick === id;
      return `<div class="mi ${on ? 'on' : ''}" data-act="cfg-model" data-prov="official" data-model="${id}">${LOGOS.logo('claude', 14)}<span class="grow">${t}</span><span class="t3 fs-cap">${d}</span>${on ? ic('check', 's12') : ''}</div>`;
    }).join('');
    groups.push(`<div class="cap cfg-grp-h" data-act="cfg-fold" data-prov="official">${ic('chevD', 's12 fc-car open')}${LOGOS.logo('claude', 13)}<span class="grow">Claude 官方</span><span class="t3 fs-cap">4 个模型</span></div><div class="cfg-grp open" data-grp="official">${offHtml}</div>`);

    P.list.filter(p => !p.fixed).forEach(p => {
      const disabled = !p.hasKey || !p.selected.length;
      const reason = !p.hasKey ? '未填写 API Key' : '模型清单为空';
      const folded = active === p.id ? 'open' : '';
      const chev = active === p.id ? 'open' : '';
      const models = p.selected.map(m => {
        const on = active === p.id && (P.picks[p.id] || '') === m;
        return `<div class="mi ${on ? 'on' : ''} ${disabled ? 'dim' : ''}" data-act="cfg-model" data-prov="${p.id}" data-model="${m}">${LOGOS.logo(modelBrand(m, provBrand(p)), 14)}<span class="grow mono">${esc(m)}</span>${on ? ic('check', 's12') : ''}</div>`;
      }).join('');
      groups.push(`<div class="cap cfg-grp-h f ai g2 ${disabled ? 'dim' : ''}" data-act="cfg-fold" data-prov="${p.id}">${ic('chevD', `s12 fc-car ${chev}`)}${LOGOS.logo(provBrand(p), 13)}<span class="grow">${esc(p.name)}</span>${disabled ? `<span class="t3 fs-cap">${reason}</span>${btn('去设置', 'sm ghost', '', 'data-act="settab" data-tab="models" data-goto="settings"')}` : `<span class="t3 fs-cap">${p.selected.length} 个模型</span>`}</div><div class="cfg-grp ${folded}" data-grp="${p.id}">${disabled ? `<div class="t3 fs-cap" style="padding:6px 10px">填写密钥并勾选模型后可选。</div>` : models}</div>`);
    });

    return `<div class="pop cfg-pop${o.shell ? ' sb-pop' : ''} a-pop-up" style="padding:4px;gap:0;max-width:320px"><div class="cap">权限模式</div>
      ${perm.map(([n, t, on]) => `<div class="mi ${on ? 'on' : ''}">${ic(n, 's12')}<span class="grow">${t}</span>${on ? ic('check', 's12') : ''}</div>`).join('')}
      <div class="sep"></div><div class="cap">模型</div><div class="cfg-models scroll">${groups.join('')}</div>
      <div class="sep"></div><div class="mi" data-act="settab" data-tab="models" data-goto="settings">${ic('sliders', 's12')}<span class="grow">管理服务商…</span></div>
      ${o.shell ? '' : `<div class="f ai jb" style="padding:4px 8px 6px"><span class="t3 fs-cap m">思考强度</span>${seg(['关', '低', '中', '高'], 2)}</div>`}</div>`;
  };

  const modelLabel = () => {
    const P = ensure();
    if (P.active === 'official') {
      const id = P.picks.official || 'sonnet';
      return (OFFICIAL_MODELS.find(x => x[0] === id) || OFFICIAL_MODELS[2])[1];
    }
    const p = prov(P.active);
    const m = P.picks[P.active] || p?.selected[0] || '—';
    return { short: `${abbr(p)} · ${m}`, full: `${p?.name} · ${m}`, abbr: abbr(p), model: m, brand: modelBrand(m, provBrand(p)) };
  };

  const cfgChipHtml = () => {
    const lab = modelLabel();
    if (typeof lab === 'string') return lab;
    return `<span data-tip="${esc(lab.full)}">${LOGOS.logo(lab.brand, 13)}</span><span class="mono ell">${esc(lab.model)}</span>`;
  };

  const fetchModels = (p, url) => {
    if (/example\.com/i.test(url || p.url)) return { err: 'HTTP 401: invalid api key' };
    if (/legacy/i.test(url || p.url)) return { unsupported: 1 };
    const preset = PRESETS.find(x => x.name === p.name);
    const base = preset?.models || ['model-a', 'model-b', 'model-c'];
    const extra = Array.from({ length: 8 + (p.name.length % 12) }, (_, i) => `${(preset?.models[0] || 'm').split('-')[0] || 'm'}-${i + 1}`);
    return { models: [...new Set([...base, ...extra])].slice(0, 12 + (p.name.length % 17)) };
  };

  window.PV = {
    PRESETS, SLOTS, OFFICIAL_MODELS, ensure, prov, provRow, providersSection, mappingBlock, presetPop, editPanel, modelInventory,
    cfgPopHtml, cfgChipHtml, modelLabel, fetchModels, idSlug, abbr, isOfficial, provBrand, modelBrand,
  };

  if (window.F) F.cfgPop = () => PV.cfgPopHtml();
})();