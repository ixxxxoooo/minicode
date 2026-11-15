#!/usr/bin/env python3
"""生成模型品牌 logo:同时写 mygo/internal/kit/logos_gen.go 与 docs/design/js/logos.js。

图标来自开源项目 Lobe Icons(MIT,https://github.com/lobehub/lobe-icons)的单色版本,
只用于标识模型 / 服务商的品牌,商标归各自所有者。
品牌的颜色与「模型 ID → 品牌」的匹配规则也在这里统一定义,Go 与设计稿共用同一份,避免两边不一致。

用法:python3 scripts/gen-logos.py        (需要联网;已下载过的缓存在 /tmp/minicode-logos)
"""
import json
import os
import re
import subprocess
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CACHE = "/tmp/minicode-logos"
CDN = "https://cdn.jsdelivr.net/npm/@lobehub/icons-static-svg@latest/icons/%s.svg"

# (品牌 id, 图标文件, 品牌色 / None=跟随文字色, 名称含有这些词, 名称以这些开头)
# 顺序即匹配优先级。匹配都按小写做:模型 ID 用整个 ID,服务商用名称与域名(不要传带路径的地址)。
BRANDS = [
    ("deepseek", "deepseek", "#4D6BFE", ["deepseek"], []),
    ("zhipu", "zhipu", "#3859FF", ["glm", "zhipu", "bigmodel", "智谱"], []),
    ("kimi", "kimi", "#1783FF", ["kimi", "moonshot", "月之暗面"], []),
    ("qwen", "qwen", "#615CED", ["qwen", "qwq", "dashscope", "bailian", "百炼", "通义"], []),
    ("minimax", "minimax", "#F23F5D", ["minimax", "abab"], []),
    ("doubao", "doubao", "#1E6BFF", ["doubao", "volces", "volcengine", "豆包", "火山"], []),
    ("wenxin", "wenxin", "#2932E1", ["ernie", "wenxin", "文心"], []),
    ("hunyuan", "hunyuan", "#0054E0", ["hunyuan", "混元"], []),
    ("stepfun", "stepfun", "#005AFF", ["stepfun", "阶跃"], ["step-"]),
    ("spark", "spark", "#0070F0", ["xfyun", "讯飞", "星火"], ["spark"]),
    ("baichuan", "baichuan", "#FF6933", ["baichuan", "百川"], []),
    ("yi", "yi", None, ["零一"], ["yi-"]),
    ("gemini", "gemini", "#3186FF", ["gemini", "gemma", "googleapis"], []),
    ("meta", "meta", "#0668E1", ["llama"], []),
    ("mistral", "mistral", "#FA520F", ["mistral", "mixtral", "codestral"], []),
    ("grok", "grok", None, ["grok", "x.ai"], []),
    ("openrouter", "openrouter", "#6566F1", ["openrouter"], []),
    ("siliconcloud", "siliconcloud", "#6E29F6", ["siliconflow", "siliconcloud", "硅基"], []),
    ("openai", "openai", None, ["gpt", "openai", "chatgpt"], ["o1", "o3", "o4"]),
    ("claude", "claude", "#D97757", ["claude", "opus", "sonnet", "haiku", "fable", "anthropic"], []),
]


def fetch(name):
    os.makedirs(CACHE, exist_ok=True)
    path = os.path.join(CACHE, name + ".svg")
    if not os.path.exists(path) or os.path.getsize(path) == 0:
        req = urllib.request.Request(CDN % name, headers={"User-Agent": "minicode-gen-logos"})
        with urllib.request.urlopen(req, timeout=20) as r:
            open(path, "wb").write(r.read())
    return open(path, encoding="utf-8").read()


def inner(svg):
    """取出 <svg> 里面的图形(去掉 <title>),只保留 path / circle / rect / ellipse / polygon。"""
    body = re.sub(r"</?svg[^>]*>", "", svg)
    body = re.sub(r"<title>.*?</title>", "", body, flags=re.S)
    body = re.sub(r"\s+", " ", body).strip()
    if re.search(r"<(g|defs|linearGradient|radialGradient|mask|clipPath|style|image)\b", body):
        raise SystemExit("图标含有不支持的元素(渐变 / 分组等),请换用其它图标:" + body[:80])
    return body


def main():
    bodies = {b[0]: inner(fetch(b[1])) for b in BRANDS}

    # ── Go ──
    go = ["// 由 scripts/gen-logos.py 生成,请勿手改。",
          "// 品牌图标来自 Lobe Icons(MIT)单色版,仅用于标识模型 / 服务商,商标归各自所有者。",
          "package kit", "",
          "// logoBodies 是每个品牌 24×24 画布里的填充图形(不含外层 <svg>)。",
          "var logoBodies = map[string]string{"]
    for k, v in bodies.items():
        go.append("\t%s: %s," % (json.dumps(k), json.dumps(v)))
    go += ["}", "",
           "// logoRules 是「名称 → 品牌」的匹配规则,顺序即优先级(名称先转小写)。",
           "var logoRules = []struct {",
           "\tBrand    string",
           "\tColor    string // 品牌色(#RRGGBB);空 = 跟随文字色",
           "\tContains []string",
           "\tPrefixes []string",
           "}{"]
    for b, _, col, kws, pfx in BRANDS:
        go.append("\t{%s, %s, []string{%s}, []string{%s}}," % (
            json.dumps(b), json.dumps(col or ""),
            ", ".join(json.dumps(x, ensure_ascii=False) for x in kws),
            ", ".join(json.dumps(x, ensure_ascii=False) for x in pfx)))
    go.append("}")
    open(os.path.join(ROOT, "mygo/internal/kit/logos_gen.go"), "w", encoding="utf-8").write("\n".join(go) + "\n")
    subprocess.run(["gofmt", "-w", os.path.join(ROOT, "mygo/internal/kit/logos_gen.go")], check=False)

    # ── 设计稿 ──
    rules = [{"brand": b, "color": col or "", "contains": kws, "prefixes": pfx} for b, _, col, kws, pfx in BRANDS]
    js = ["/* 由 scripts/gen-logos.py 生成,请勿手改。品牌图标来自 Lobe Icons(MIT)单色版,仅用于标识模型 / 服务商。",
          "   brandOf(名称) → 品牌 id;logo(品牌, 尺寸) → HTML;匹配规则与 Go 的 kit/logos_gen.go 同源。 */",
          "(function () {",
          "  const BODIES = " + json.dumps(bodies, ensure_ascii=False) + ";",
          "  const RULES = " + json.dumps(rules, ensure_ascii=False) + ";",
          "  const brandOf = (name) => {",
          "    const s = String(name || '').toLowerCase();",
          "    if (!s) return '';",
          "    for (const r of RULES) {",
          "      if (r.contains.some(k => s.includes(k)) || r.prefixes.some(k => s.startsWith(k))) return r.brand;",
          "    }",
          "    return '';",
          "  };",
          "  // 无法识别的模型用统一的占位图标(芯片),不乱配品牌。",
          "  const logo = (brand, size) => {",
          "    const px = size || 16;",
          "    const r = RULES.find(x => x.brand === brand);",
          "    if (!r) return `<span class=\"mlogo mlogo-gen\" style=\"width:${px}px;height:${px}px\">${U.ic('cpu', 's12')}</span>`;",
          "    const col = r.color ? `color:${r.color};` : '';",
          "    return `<span class=\"mlogo\" style=\"width:${px}px;height:${px}px;${col}\"><svg viewBox=\"0 0 24 24\" width=\"${px}\" height=\"${px}\" fill=\"currentColor\" fill-rule=\"evenodd\" aria-hidden=\"true\">${BODIES[brand]}</svg></span>`;",
          "  };",
          "  window.LOGOS = { brandOf, logo, RULES };",
          "})();", ""]
    open(os.path.join(ROOT, "docs/design/js/logos.js"), "w", encoding="utf-8").write("\n".join(js))
    print("品牌 logo:%d 个" % len(bodies))


if __name__ == "__main__":
    main()
