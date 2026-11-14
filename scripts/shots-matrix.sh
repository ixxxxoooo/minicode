#!/bin/sh
# 批量截图矩阵:场景 × 变体(浅色 / 深色 / 窄窗 / 精简 / 详细),输出 /tmp/mini-<id>-<变体>.png
# 用法:scripts/shots-matrix.sh [场景id...]   不给场景时跑后台任务与全场景那一组
# WAIT 控制每张等待秒数(默认 7);后台任务场景是真实节奏,想看到终态可调大。
cd "$(dirname "$0")/.." || exit 1
cargo build -p mini-app || exit 1
ids="$*"
[ -z "$ids" ] && ids="bg-shell bg-monitor bg-agent bg-parallel bg-nested bg-resume bg-notes tools-status notices-system showcase-all"
for id in $ids; do
  SUFFIX=-light MINI_WINDOW_SIZE=1280,860 MINI_THEME=light scripts/shots.sh "$id"
  SUFFIX=-dark MINI_WINDOW_SIZE=1280,860 MINI_THEME=dark scripts/shots.sh "$id"
  SUFFIX=-narrow MINI_WINDOW_SIZE=700,860 scripts/shots.sh "$id"
  SUFFIX=-compact MINI_WINDOW_SIZE=1280,860 MINI_DENSITY=compact scripts/shots.sh "$id"
  SUFFIX=-detailed MINI_WINDOW_SIZE=1280,860 MINI_DENSITY=detailed scripts/shots.sh "$id"
done
ls /tmp/mini-*-light.png /tmp/mini-*-dark.png /tmp/mini-*-narrow.png /tmp/mini-*-compact.png /tmp/mini-*-detailed.png 2>/dev/null | wc -l
