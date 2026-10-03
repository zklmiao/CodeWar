#!/usr/bin/env bash
# 生成 proto 契约的三端代码（ADR-0001：proto 单一真相源，生成产物入库）
# 前置：protoc、protoc-gen-go、protoc-gen-go-grpc、pip grpcio-tools、client 侧 pnpm install（ts-proto）
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p server/gen judge/gen client/src/api/proto

# --- Go：原生生成 ---
protoc -I proto \
  --go_out=server --go_opt=module=codewar/server \
  --go-grpc_out=server --go-grpc_opt=module=codewar/server \
  proto/codewar/judge/v1/judge.proto proto/codewar/match/v1/match.proto

# --- Python：judge 侧（grpcio-tools 自带 protoc 与 grpc_python 插件） ---
python -m grpc_tools.protoc -I proto \
  --python_out=judge/gen --pyi_out=judge/gen --grpc_python_out=judge/gen \
  proto/codewar/judge/v1/judge.proto
# 生成产物按包路径落盘，补齐 __init__ 链
for d in judge/gen judge/gen/codewar judge/gen/codewar/judge judge/gen/codewar/judge/v1; do
  mkdir -p "$d"; touch "$d/__init__.py"
done

# --- TypeScript：client 侧（ts-proto，JSON 方法纯代码生成，无运行时依赖） ---
BIN=client/node_modules/.bin
TS_PLUGIN="$(pwd)/$BIN/protoc-gen-ts_proto"
if [ -e "$TS_PLUGIN.cmd" ]; then
  TS_PLUGIN="$TS_PLUGIN.cmd"           # Windows：protoc 只认可执行体，用 cmd 垫片
elif [ ! -e "$TS_PLUGIN" ]; then
  cat > "$TS_PLUGIN" <<'EOF'           # Unix 兜底：ts-proto 包内 node 入口
#!/usr/bin/env node
require("ts-proto/build/protoc-gen-ts_proto").main();
EOF
  chmod +x "$TS_PLUGIN"
fi
protoc -I proto \
  --plugin=protoc-gen-ts_proto="$TS_PLUGIN" \
  --ts_proto_out=client/src/api/proto \
  --ts_proto_opt=esModuleInterop=true,outputJsonMethods=true,env=browser,forceLong=string,snakeToCamel=true \
  proto/codewar/match/v1/match.proto

echo "proto codegen done:"
git status --short server/gen judge/gen client/src/api/proto 2>/dev/null || true
