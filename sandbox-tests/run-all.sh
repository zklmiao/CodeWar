#!/usr/bin/env bash
# 一键跑全部沙箱验收脚本并断言预期结果（Git Bash）。
# 用法：bash run-all.sh   —— 任一项 FAIL 则退出码非零。
set -u
cd "$(dirname "$0")"

pass=0
fail=0

check() {
  local name="$1" expect="$2" out="$3"
  if echo "$out" | grep -q "$expect"; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name  (期望输出含 [$expect]，实际: $out)"
    fail=$((fail + 1))
  fi
}

out=$(bash 01-ok.sh)
check "01-ok" "'category': 'ok'" "$out"

out=$(bash 02-deadloop-timeout.sh)
check "02-deadloop" "'category': 'timeout'" "$out"

out=$(bash 03-network-blocked.sh)
if echo "$out" | grep -q "'category': 'ok'"; then
  echo "FAIL  03-network (联网竟判 ok——断网失效)"
  fail=$((fail + 1))
else
  echo "PASS  03-network"
  pass=$((pass + 1))
fi

out=$(bash 04-memory-limit.sh)
check "04-memory-limit" "'category': 'limit-exceeded'" "$out"

out=$(bash 05-partial-pass-no-leak.sh)
if echo "$out" | grep -q "'category': 'wrong-answer'" && echo "$out" | grep -q "'detail': ''"; then
  echo "PASS  05-partial-pass"
  pass=$((pass + 1))
else
  echo "FAIL  05-partial-pass  (实际: $out)"
  fail=$((fail + 1))
fi

out=$(bash 06-syntax-detail.sh)
check "06-syntax" "'category': 'syntax'" "$out"

out=$(bash 07-service-alive.sh)
if [ "$(echo "$out" | grep -c "'category': 'timeout'")" -ge 2 ] && echo "$out" | grep -q "'category': 'ok'"; then
  echo "PASS  07-service-alive"
  pass=$((pass + 1))
else
  echo "FAIL  07-service-alive  (实际: $out)"
  fail=$((fail + 1))
fi

echo "----------------------------------------"
echo "汇总: $pass PASS / $fail FAIL"
[ "$fail" -eq 0 ]
