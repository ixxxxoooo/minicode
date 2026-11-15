#!/bin/bash
# 用 DeepSeek(Anthropic 兼容接口)作为 claude 的后端来跑真实场景:
#   scripts/with-deepseek.sh <命令…>
# 例:scripts/with-deepseek.sh claude -p "只回复 ok"
#     scripts/with-deepseek.sh scripts/dev.sh --real
#     MINI_REAL_CLAUDE=1 scripts/with-deepseek.sh go test ./internal/claude -run Real   (在 mygo/ 下)
# 密钥只在运行时从 ~/.minicode/provider-keys.json(0600,不在仓库里)读取,注入到子进程环境变量;
# 本脚本不含、也不输出任何密钥。模型可用 DEEPSEEK_MODEL 覆盖(默认 deepseek-chat;可选 deepseek-flash / deepseek-v4-pro,二者带思考)。
set -e
[ $# -gt 0 ] || { echo "用法:$0 <命令…>" >&2; exit 2; }
keyfile="$HOME/.minicode/provider-keys.json"
[ -f "$keyfile" ] || { echo "找不到 $keyfile,先在 设置 → 服务商 里给 DeepSeek 填密钥" >&2; exit 1; }
token=$(python3 -c 'import json,os,sys; print(json.load(open(os.path.expanduser("~/.minicode/provider-keys.json"))).get("deepseek",""))')
[ -n "$token" ] || { echo "provider-keys.json 里没有 deepseek 的密钥" >&2; exit 1; }
model="${DEEPSEEK_MODEL:-deepseek-chat}"
export ANTHROPIC_BASE_URL="https://api.deepseek.com/anthropic"
export ANTHROPIC_AUTH_TOKEN="$token"
export ANTHROPIC_MODEL="$model"
export ANTHROPIC_DEFAULT_OPUS_MODEL="$model"
export ANTHROPIC_DEFAULT_SONNET_MODEL="$model"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="$model"
export ANTHROPIC_SMALL_FAST_MODEL="$model"
export CLAUDE_CODE_SUBAGENT_MODEL="$model"
export API_TIMEOUT_MS="${API_TIMEOUT_MS:-600000}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
unset ANTHROPIC_API_KEY
exec "$@"
