// 从设计稿的图标库(docs/design/js/icons.js + ui.js 里追加的几枚)生成 Go 的图标表。
// 用法:node scripts/gen-icons.mjs  → 写 mygo/internal/kit/icons_gen.go
import { readFileSync, writeFileSync } from 'node:fs';
import vm from 'node:vm';

const win = {};
const ctx = vm.createContext({ window: win, document: {}, ICONS: undefined, U: undefined });
vm.runInContext(readFileSync('docs/design/js/icons.js', 'utf8'), ctx);
// ui.js 在 IIFE 开头用 Object.assign(ICONS, {...}) 追加图标;只执行那一句
const ui = readFileSync('docs/design/js/ui.js', 'utf8');
const m = ui.match(/Object\.assign\(ICONS,\s*(\{[\s\S]*?\n  \})\);/);
ctx.ICONS = win.ICONS;
if (m) vm.runInContext(`Object.assign(ICONS, ${m[1]});`, ctx);
const icons = win.ICONS;
const names = Object.keys(icons).sort();
let out = '// 由 scripts/gen-icons.mjs 生成,请勿手改。图标源自设计稿 docs/design/js/icons.js(24 格线性图标,描边 1.7)。\n';
out += 'package kit\n\n// iconPaths 是每个图标 24×24 画布里的矢量内容(不含外层 <svg>)。\nvar iconPaths = map[string]string{\n';
for (const n of names) out += `\t${JSON.stringify(n)}: ${JSON.stringify(icons[n])},\n`;
out += '}\n';
writeFileSync('mygo/internal/kit/icons_gen.go', out);
console.log(`写入 ${names.length} 个图标`);
