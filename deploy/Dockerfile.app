# app：构建客户端 → 构建服务端（静态产物直接托管在 Go 进程）
FROM node:22-alpine AS client
WORKDIR /src
RUN corepack enable
COPY pnpm-workspace.yaml package.json ./
COPY client/package.json client/
RUN pnpm install --ignore-scripts
COPY client/ client/
RUN pnpm --dir client build

FROM golang:1.26-alpine AS server
WORKDIR /src
# 容器内默认 proxy.golang.org 在当前网络不可达
ENV GOPROXY=https://goproxy.cn,direct
COPY server/ server/
RUN cd server && CGO_ENABLED=0 go build -o /out/codewar ./cmd/codewar

# scratch：无基础镜像拉取问题（gcr.io 在当前网络不可达）；纯静态 Go 二进制
FROM scratch
COPY --from=server /out/codewar /srv/codewar
COPY --from=client /src/client/dist /srv/dist
EXPOSE 3001
ENTRYPOINT ["/srv/codewar", "-addr", ":3001", "-judge", "judge:50051", "-static", "/srv/dist"]
