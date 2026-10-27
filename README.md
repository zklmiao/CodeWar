<div align="center">

# CodeWar · 代码战争

**把解题速度变成战斗力。**

一款网页端 Python 编程对抗游戏：两名玩家在 5×5 竞技场各控一个角色，每回合从四道题里选一道作答——解出攻击题开火、移动题走位、防御题减伤、技能题放大招。HP 归零者败。

<sub>A web-based competitive Python programming game — solve faster, hit harder.</sub>

![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue_3-4FC08D?logo=vuedotjs&logoColor=white)
![Python](https://img.shields.io/badge/Python-3776AB?logo=python&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker_Compose-2496ED?logo=docker&logoColor=white)
[![CI](https://github.com/OWNER/CodeWar/actions/workflows/ci.yml/badge.svg)](https://github.com/OWNER/CodeWar/actions/workflows/ci.yml)

| 入口：配置、盲选职业与技能 | 对局：棋盘 + 四张题卡 + 共享工作区 |
|:---:|:---:|
| ![入口页](docs/assets/screenshot-entry.png) | ![对局中](docs/assets/screenshot-match.png) |

</div>

## 玩法

**一回合，一个行动。** 四道题同时展示，但只能提交一道——取舍本身就是战术。

| 行动 | 题面性质 | 效果 |
|---|---|---|
| ⚔ **攻击** | 镜像题（双方同题竞速） | 完成越快伤害越高；攻击带形状与射程，对手站在覆盖格内才命中 |
| 🏃 **移动** | 私有题 + 格点选择 | 直线移动 1-2 格（游侠 3 格）；走出对方攻击形状即闪避 |
| 🛡 **防御** | 私有题 | 本回合所受伤害 ×0.4（减伤 60%） |
| ✨ **技能** | 私有题（难度 = 所选上限档） | 职业专属伤害/治疗，冷却 2-4 回合 |

- **三职业**：剑士（HP 110 / 相邻 4 格十字）· 法师（HP 90 / 直线 2 格线扫）· 游侠（HP 95 / 1-3 格点狙、移动 3 格）。
- **难度是风险回报**：难度上限 L1→L5 挂钩全局伤害系数 ×0.85→×1.20；抽题策略可选「爬坡」或「均匀随机」。
- **胜负**：HP 归零 KO；10 回合打满按剩余 HP 裁决；平局双方 ELO +0。
- **模式**：本机热座 1v1（双窗口）· 人机练习（简单/普通/困难三档 Bot）· 4 关交互式教程 · 账号、战绩与 ELO 积分。

> 在线排位 1v1 尚未实现（同步协议与认输握手已就绪）。当前版本完整支持本地热座与 PvE。

## 技术架构

```mermaid
flowchart LR
  subgraph Browser["浏览器 (Vue 3 + Pinia + CodeMirror)"]
    C["事件流渲染 · 提交上行 · 断线重连"]
  end

  subgraph Server["Go 服务（服务端权威）"]
    H["Hub：WebSocket · 席位 · 广播"]
    E["MatchEngine：回合状态机"]
    Q["QuestionFactory：模板 × seed"]
    B["PvE Bot：经 SubmissionPort"]
    A["账号 / 战绩 / ELO"]
  end

  J["Judge（Python gRPC）<br/>一次性容器沙箱"]
  P[("PostgreSQL")]

  C <-->|"提交上行 / 事件流下行<br/>快照重连"| H
  H --> E
  E --> Q
  B -->|SubmissionPort| E
  E -->|SubmissionPort| J
  H --> A --> P
```

架构以三个 seam 为骨架，实现与测试都围绕它们展开：

**MatchEngine —— 唯一集成 seam**
回合状态机（回合开始 → 面板组装 → 作答窗口 → 结算 → 回合结束），对外只发布**带序号的对局事件流**，依赖 `QuestionPort` / `SubmissionPort` / `ClockPort` 三个端口。全部战斗规则在此边界内实现；真人、脚本回放与 Bot 走同一条提交入口。

**Judge —— 纯函数 seam**
`judge(code, testCases) → {passed, category, durationMs}`。独立 Python 微服务，经 docker-outside-of-docker 为每次判题起一次性容器：禁网、限时、限资源，超时即杀；100% 用例通过才生效。支持链表/树/有状态类等结构体题契约，题作者的构造器与比较器与玩家代码过同一套静态卫兵。判题结论 fail-closed：harness 只接受恰好一行结果，玩家代码无法伪造 verdict。

**QuestionFactory —— 纯函数 seam**
`create(spec, seed) → {instance, answerKey, testCases}`。124 个模板按「题型 × 难度档 × 动作类型 × 题面形式」四维标注并参数化；同 seed 必出同实例（测试可复现），不同 seed 实例不同（背答案无用）。

### 几处值得一提的工程取舍

- **事件流同步**：客户端只渲染事件与提交动作，状态永远由服务端裁决；断线后以快照 + 尾随事件重建，序号断裂自动重连自愈。
- **判题不持引擎锁**：`solveMs` 按提交到达时刻计算，后到玩家不会因对手判题排队而被记慢——镜像竞速公平。
- **连接卫生**：WebSocket 读写 deadline + ping/pong 保活 + 并发上限；慢消费者丢事件而非阻塞全场广播，断号由客户端快照重连兜底。
- **测试纪律**：行为断言集中在 seam 边界（事件流 / Judge / QuestionFactory），错误路径优先；关键断言经过变异电池验证（写错实现必须让测试变红）。

## 快速开始

### Docker Compose（推荐，一键起全栈）

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

打开 **http://localhost:3001**。app、judge 沙箱与 Postgres 一并启动，账号、战绩与 ELO 可直接使用。

- 最快上手：入口页选职业 → **开始人机练习**（单窗口，一局 10 回合）。
- 热座对战：窗口 A 点「以 P1 接入」，窗口 B 打开同地址点「以 P2 接入」。
- 教程：注册登录后完成 4 关教程解锁 PvP 入口。

### 本地开发

```bash
# 1) 数据库（宿主 25432 → 容器 5432）
docker compose -f deploy/docker-compose.yml up -d postgres

# 2) judge 服务（需要 Docker daemon，沙箱经 docker.sock 创建）
cd judge && pip install -r requirements.txt && python -m judgesvc.server

# 3) Go 服务（:3001）
cd server
DATABASE_URL='postgres://codewar:codewar-dev@localhost:25432/codewar?sslmode=disable' \
  go run ./cmd/codewar

# 4) 客户端 dev server（:5173，已代理 /ws 与 /api 到 :3001）
pnpm -C client dev
```

开发约定（worktree 并行、端口规范、票据流程）见 [AGENTS.md](AGENTS.md)。

## 测试

```bash
# Go：全包竞态门控
cd server && go test -race ./...

# Go：真实 Postgres 集成测试（按包独立测试库，自动创建）
cd server && CODEWAR_TEST_DSN='postgres://codewar:codewar-dev@localhost:25432/codewar_test?sslmode=disable' go test ./...

# Judge：沙箱与判题契约
python -m pytest judge/tests -q

# 客户端
pnpm -C client test
pnpm -C client build
```

CI 三门并行：Go `-race`、judge pytest、client test + build（见 [.github/workflows/ci.yml](.github/workflows/ci.yml)）。

## 项目结构

```
client/                 Vue 3 + Pinia + CodeMirror；事件流驱动渲染
server/
  cmd/codewar/          主服务入口（HTTP/WS + 静态托管）
  cmd/smoke/            headless 端到端冒烟客户端
  internal/match/       对局引擎、形状判定、事件流
  internal/hub/         WebSocket 席位、广播、重连、结算
  internal/question/    出题工厂与模板库
  internal/bot/         PvE 决策层
  internal/accounts/    账号、战绩、ELO
judge/                  Python gRPC 判题服务 + 容器沙箱
proto/                  match.proto / judge.proto（Go / TS / Python 三端生成）
db/                     迁移的权威源（app 内嵌副本执行）
design/                 设计 token、占位组件与风格页
docs/
  specs/                产品规格（设计与数值的唯一真相源）
  adr/                  8 篇架构决策记录
  design/               出题系统、职业数值、占位美术规范
```

## 文档地图

- [产品规格](docs/specs/001-codewar-mvp.md)：用户故事、实现决策、测试决策、数值初稿
- [架构决策（ADR）](docs/adr/)：技术栈、同步模型、判题沙箱、结构体题契约、PvE Bot 边界、教程模式、沙箱威胁模型
- [出题系统设计](docs/design/002-question-system.md) / [职业与技能表](docs/design/003-classes-skills.md) / [占位美术规范](docs/design/004-placeholder-art-standard.md)
- [AGENTS.md](AGENTS.md)：面向 AI 协作者的工作流与仓库约定（本项目的实现过程由 spec + 票据 + ADR 驱动）

## Roadmap

- [x] 本地热座对局、三行动与形状、技能与三职业
- [x] 出题工厂（124 模板、seed 确定性）、对局配置与难度风险
- [x] 判题沙箱硬化、结构体题契约、判题结论防伪造
- [x] 账号 / 战绩 / ELO、PvE 三档 Bot、4 关交互式教程
- [x] 认输握手、连接保活与并发上限、结算统计与日志着色
- [ ] **在线排位 1v1**（匹配、服务端权威同步、断线判负/重连）
- [ ] 反作弊硬化（禁粘贴编辑器、节奏指纹、管理端）
- [ ] 对局回放与错题本、排行榜
- [ ] CI 门控补全（Postgres service、类型检查与 lint）

## 说明

- 美术资产为统一占位系统（设计 token 单源，可整包替换为正式美术）。
- `ref_resources/`（外部项目参考）不入库；本仓库未复制任何第三方代码或美术资产。
