# app：构建客户端 → 构建服务端（静态产物直接托管在 Go 进程）
FROM node:22-alpine AS client
WORKDIR /src
RUN corepack enable
COPY pnpm-workspace.yaml package.json ./
COPY client/package.json client/
RUN pnpm install --frozen-lockfile=false
COPY client/ client/
RUN pnpm --dir client build

FROM golang:1.26-alpine AS server
WORKDIR /src
COPY server/ server/
RUN cd server && go build -o /out/codewar ./cmd/codewar

FROM gcr.io/distroless/static-debian12
COPY --from=server /out/codewar /srv/codewar
COPY --from=client /src/client/dist /srv/dist
EXPOSE 3001
ENTRYPOINT ["/srv/codewar", "-addr", ":3001", "-judge", "judge:50051", "-static", "/srv/dist"]
