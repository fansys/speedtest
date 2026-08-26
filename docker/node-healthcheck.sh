#!/bin/sh
# 节点 agent 的 HEALTHCHECK 脚本：/healthz 要求 X-Node-Key，所以这里要先拿到 key。
# 优先用 NODE_KEY 环境变量（手动模式）；否则从自动注册写回的 NODE_INI 里读 node_key。
set -eu

NODE_PORT="${NODE_PORT:-8081}"
NODE_INI="${NODE_INI:-/data/node.ini}"

key="${NODE_KEY:-}"
if [ -z "$key" ] && [ -f "$NODE_INI" ]; then
    key=$(sed -n 's/^[[:space:]]*node_key[[:space:]]*=[[:space:]]*//p' "$NODE_INI" | tail -n1)
fi

if [ -z "$key" ]; then
    echo "node-healthcheck: 没有可用的 node_key（NODE_KEY 为空且 $NODE_INI 里没有）" >&2
    exit 1
fi

exec wget -q -O /dev/null --header "X-Node-Key: ${key}" "http://127.0.0.1:${NODE_PORT}/healthz"
