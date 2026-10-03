# ADR-0001：MVP 技术栈选型

**状态**：已接受（2026-10-03）
**关联**：Spec 001；票据 01（脚手架遵循本 ADR）、02（judge 沙箱机制 ADR 在此架构内）、08（在线对战）

## 背景

约束条件：服务端权威架构（Spec 实现决策）；judge 必须以 OS 级隔离承载不可信 Python 代码；1v1 实时同步；MatchEngine 事件流架构与 seed 确定性（设计 002）；单人开发 + AI 代理协作；开发机为 Windows。

## 决策

**整体形态：Go 主服务 + 独立 Python 判题微服务 + Vue 3 前端，Protobuf 为三方契约单一真相源。**

| 层 | 选型 | 要点 |
|---|---|---|
| 主服务 | **Go**（chi 路由、slog 日志、gorilla/websocket、go-playground/validator、env 配置） | 判题器是"编排者"不是"执行者"——安全边界在沙箱执行器，主服务语言不承担隔离职责 |
| 数据库 | **PostgreSQL 16 + sqlc + golang-migrate** | SQL 即真相；jsonb（实例快照/事件日志）、数组（标签/参与者）、部分索引（违规审计）直接可用 |
| 判题服务 | **Python 微服务**，与主服务间 **同步 gRPC + 每用例 deadline** | 判题在回合内必须同步返回；沙箱机制（容器运行时 / 用户命名空间）由 02 号票 ADR 在此服务边界内选定 |
| 异步批量 | 主服务内 worker goroutine 直调 judge 异步接口 | 模板批量冒烟（黄金样板生产线的校验环节）用；不引入 Redis/NATS 消息中间件 |
| 契约 | **Protobuf 单一真相源**（`proto/` 目录） | judge 链路 = gRPC（proto 编码）；客户端链路 = WebSocket 上跑 JSON（ts-proto 生成 TS 类型 + fromJson，浏览器可调试） |
| 前端 | **Vue 3 + Vite + Pinia + CodeMirror 6**（TypeScript） | 5×5 网格与题卡面板用 DOM/CSS 渲染，回合制无游戏循环，不引入 Phaser 类引擎；CodeMirror 6 适配 ≤18 行小代码段 |
| 部署 | **Docker Compose 三服务**：`app`(Go, distroless 镜像) + `judge`(Python) + `postgres` | Go 单静态二进制；本地开发需 Docker Desktop（WSL2 后端） |
| 仓库结构 | monorepo：`client/`（pnpm）、`server/`（Go module）、`judge/`（Python）、`proto/`、`db/migrations/`、`deploy/` | pnpm workspace 覆盖前端与 TS 代码生成工具链 |

## 后果

- **(+)** 判题隔离与业务服务天然分界，judge 可独立重启/扩容，且升级路径清晰（02 号票在既有 gRPC 边界内换沙箱机制不动主服务）。
- **(+)** Go 二进制部署与内存占用极低；Compose 一键起全环境。
- **(+)** sqlc 产物显式可审查，schema 与查询均为文本工件，AI 协作与人工审都是低摩擦。
- **(−)** 放弃了 TS 单体的两端共享类型，改用 proto 代码生成补偿——契约变更必须走 `proto/`，双端重新生成（这是纪律代价，防漂移收益更大）。
- **(−)** 服务端双语言（Go 业务 + Python 判题），开发者需同时维护两套工程；边界清晰是补偿。

## 备选方案（被否原因）

- **TS/Node 单体**：判题仍需外部沙箱执行器，安全收益相同但失去用户偏好的 Go 运维与部署优势。
- **Python 全栈（FastAPI）**：判题原生，但前后端双语言且共享契约更弱，判题与业务耦合在同一进程内反而模糊安全边界。
- **MySQL 8**：运维熟悉度可成立，但 sqlc 体验降级、无数组类型、JSON 审计查询绕路（讨论记录：本项目 jsonb/数组/部分索引为真实需求）。
- **GORM / ent**：隐式行为与概念负担；sqlc 的显式性对 AI 生成代码的人工审查更有利。
- **OpenAPI 单源 / 手写双端契约**：judge 必须走 gRPC（proto 不可省），OpenAPI 会造成双 IDL；手写漂移风险最高。
