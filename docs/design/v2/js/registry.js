/* 页面注册表:每个页面有若干「方案」,每个方案是一个返回 HTML 的函数,画在 1280×800 的 .frame 里 */
(function () {
  const S = { list: [], groups: [] };
  S.group = (id, title) => S.groups.push({ id, title });
  S.add = (s) => { s.variants = s.variants.map((v, k) => ({ key: String.fromCharCode(65 + k), ...v })); S.list.push(s); };
  S.find = (id) => S.list.find(s => s.id === id);
  window.S = S;
})();
