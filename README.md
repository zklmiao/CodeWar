<div align="center">

# CodeWar · 代码战争

**把解题速度变成战斗力。**

网页端 Python 编程对抗游戏：两名玩家在 5×5 竞技场各控一名角色，每回合四道题只能解一道——攻击、走位、格挡、放大招。解题越快伤害越高，HP 先归零者败。

<sub>A turn-based competitive Python programming game — solve faster, hit harder.</sub>

[玩法](#玩法) · [对局实况](#对局实况) · [技术架构](#技术架构) · [快速开始](#快速开始) · [测试](#测试) · [文档](#文档地图) · [Roadmap](#roadmap)

![对局实况：左栏棋盘与战报，右栏四张题卡与共享工作区](docs/assets/screenshot-match.png)

![Go 1.26](https://img.shields.io/badge/Go_1.26-00ADD8?logo=go&logoColor=white) ![Vue 3.5](https://img.shields.io/badge/Vue_3.5-4FC08D?logo=vuedotjs&logoColor=white) ![TypeScript 5](https://img.shields.io/badge/TypeScript_5-3178C6?logo=typescript&logoColor=white) ![Python 3.12](https://img.shields.io/badge/Python_3.12-3776AB?logo=python&logoColor=white) ![PostgreSQL 16](https://img.shields.io/badge/PostgreSQL_16-4169E1?logo=postgresql&logoColor=white) ![Docker Compose](https://img.shields.io/badge/Docker_Compose-2496ED?logo=docker&logoColor=white)

</div>

## 玩法

### 一回合，一个行动

四道题同时展示，但一回合只能提交一道——取舍本身就是战术。

| 行动 | 题面性质 | 结算效果 |
|---|---|---|
| ⚔ **攻击** | 镜像题（双方同题竞速） | 完成越快伤害越高；对手不在攻击形状覆盖格内即闪避 |
| 🏃 **移动** | 私有题 + 棋盘点选 | 直线移动 1-2 格（游侠 3 格）；走出对方攻击形状即闪避 |
| 🛡 **防御** | 私有题 | 本回合所受伤害 ×0.4（减伤 60%） |
| ✨ **技能** | 私有题（难度 = 所选上限档） | 职业专属伤害 / 治疗，冷却 2-4 回合；冷却中灰置 |

```mermaid
flowchart LR
  S["回合开始<br/>四题同屏 · 75 秒"] --> A{"只选一道作答"}
  A --> AT["⚔ 攻击<br/>镜像竞速"]
  A --> MV["🏃 移动<br/>走位闪避"]
  A --> DF["🛡 防御<br/>减伤 60%"]
  A --> SK["✨ 技能<br/>职业大招"]
  AT --> R["按解题耗时排序结算<br/>事件流广播"]
  MV --> R
  DF --> R
  SK --> R
  R --> S
```

### 三职业

| 职业 | HP | 普攻 | 普攻形状 | 移动 | 技能（伤害 / 治疗 · 冷却） |
|---|---|---|---|---|---|
| **剑士** | 110 | 26 | 相邻 4 格十字（无方向） | 1-2 | 巨力斩 55 · CD4 / 求生本能 30 · CD3 |
| **法师** | 90 | 18 | 直线 2 格线扫 | 1-2 | 烈焰风暴 40 · CD3 / 秘法汲取 25 + 自疗 20 · CD3 |
| **游侠** | 95 | 20 | 1-3 格点狙 | **1-3** | 穿云箭 45 · CD4 / 风行回复 28 · CD2 |

### 伤害与风险

```txt
伤害 = 基础伤害 × 时间系数(0.6~1.0) × 难度系数(L1 ×0.85 → L5 ×1.20)
             × 形状命中判定 × 防御减伤(×0.4)
```

- **难度是风险回报**：开局自选题型、难度上限与抽题策略（爬坡 / 均匀）；上限越高，全局伤害与结算奖励系数越高。
- **错误有代价**：提交错误锁定 5 秒并自损 3 HP；一道题全部用例通过才生效。
- **胜负**：HP 归零 KO；10 回合打满按剩余 HP 裁决；同时归零平局，ELO 双方 +0。

### 模式

- **本机热座 1v1** —— 两个浏览器窗口各控一方；双方账号登录时结算 ELO。
- **人机练习** —— 简单 / 普通 / 困难三档 Bot，走与真人相同的 `SubmissionPort`；不记战绩。
- **4 关交互式教程** —— 读题作答 → 走位闪避 → 防御与技能冷却 → 难度收益；完成解锁 PvP 入口，进度随账号保存。
- **账号与战绩** —— 注册登录、历史对局、胜率与 ELO 跨会话保留。

> 在线排位 1v1 尚未实现：事件流同步协议（ADR-0002）与认输握手已就绪，匹配与在线同步归 08 号票。

## 对局实况

| 入口：题型 / 难度 / 抽题策略 + 职业盲选 | 教程：每关只教一个机制，实操过关 |
|:---:|:---:|
| ![入口页](docs/assets/screenshot-entry.png) | ![教程关](docs/assets/screenshot-tutorial.png) |

## 技术架构

```mermaid
flowchart LR
  subgraph Browser["浏览器 · Vue 3 + Pinia + CodeMirror"]
    C["事件流渲染<br/>提交上行 / 快照重连"]
  end

  subgraph Server["Go 服务 · 服务端权威"]
    H["Hub<br/>WebSocket · 席位 · 广播"]
    E["MatchEngine<br/>回合状态机 + 事件流"]
    Q["QuestionFactory<br/>模板 × seed"]
    B["PvE Bot<br/>决策层"]
    U["账号 · 战绩 · ELO"]
  end

  J["Judge<br/>Python gRPC · 一次性容器沙箱"]
  P[("PostgreSQL 16")]

  C <-->|"提交上行 / 事件流下行"| H
  H --> E
  E --> Q
  B -->|"SubmissionPort"| E
  E -->|"SubmissionPort"| J
  H --> U
  U --> P
```

架构以三个 seam 为骨架，实现与测试都围绕它们展开：

| seam | 边界契约 |
|---|---|
| **MatchEngine**（唯一集成 seam） | 回合状态机（回合开始 → 面板组装 → 作答窗口 → 结算 → 回合结束），对外只发布**带序号的对局事件流**，依赖 `QuestionPort` / `SubmissionPort` / `ClockPort`。全部战斗规则在此边界内实现；真人、PvE Bot、脚本回放走同一条提交入口，引擎对 Bot 零特判。 |
| **Judge**（纯函数 seam） | `judge(code, testCases) → {passed, category, durationMs}`。独立 Python 微服务，每次判题起一次性容器：禁网、限时、限资源，100% 用例通过才生效；支持链表 / 树 / 有状态类等结构体题契约，题作者的构造器与比较器与玩家代码过同一套静态卫兵。 |
| **QuestionFactory**（纯函数 seam） | `create(spec, seed) → {instance, answerKey, testCases}`。124 个模板按「题型 × 难度档 × 动作类型 × 题面形式」四维标注并参数化；同 seed 必出同实例（测试可复现），不同 seed 实例不同（背答案无用）。 |

### 几处值得一提的工程取舍

| 主题 | 做法 |
|---|---|
| 事件流同步 | 客户端只渲染事件、只上行提交，状态永远由服务端裁决；断线以快照 + 尾随事件重建，序号断裂自动重连自愈。 |
| 判题不持引擎锁 | `solveMs` 按提交到达时刻计算，后到玩家不会因对手判题排队而被记慢——镜像竞速公平。 |
| 连接卫生 | 读写 deadline + ping/pong 保活 + 并发上限；慢消费者丢事件而非阻塞全场广播，断号由客户端快照重连兜底。 |
| verdict 防伪造 | 判题 harness 只接受恰好一行结论，玩家代码无法用 `atexit` 伪造协议行；异常一律 fail-closed。 |
| 测试纪律 | 行为断言集中在三个 seam 的外部边界（事件流 / Judge / QuestionFactory），错误路径优先；关键断言经过变异电池验证——写错实现必须让测试变红。 |

## 快速开始

### Docker Compose（推荐，一键起全栈）

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

打开 **http://localhost:3001** —— app、judge 沙箱与 PostgreSQL 一并启动，账号、战绩与 ELO 可直接使用。首次构建会拉取 `python:3.12-slim` 沙箱镜像，需要本机 Docker daemon 运行（判题容器经 `docker.sock` 创建）。

上手路径：

- **单窗口最快**：入口页选职业 → 开始人机练习（一局 10 回合）。
- **热座对战**：窗口 A 点「以 P1 接入」，窗口 B 打开同地址点「以 P2 接入」。
- **教程与 ELO**：注册登录 → 完成 4 关教程 → PvP 解锁；两个账号登录对局即结算 ELO。

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

```txt
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
  assets/               README 截图
```

## 文档地图

- [产品规格](docs/specs/001-codewar-mvp.md)：37 条用户故事、实现决策、测试决策、数值初稿
- [架构决策（ADR）](docs/adr/)：技术栈、同步模型、判题沙箱、结构体题契约、PvE Bot 边界、教程模式、沙箱威胁模型
- [出题系统设计](docs/design/002-question-system.md) / [职业与技能表](docs/design/003-classes-skills.md) / [占位美术规范](docs/design/004-placeholder-art-standard.md)
- [票据与优化清单](.scratch/codewar-mvp/)：逐票交付记录、验收证据与后续 backlog
- [AGENTS.md](AGENTS.md)：面向 AI 协作者的工作流与仓库约定（本项目由 spec + 票据 + ADR 驱动）

## Roadmap

**已交付**

- [x] 本地热座对局 · 三行动与攻击形状 · 技能与三职业
- [x] 出题工厂（124 模板 · seed 确定性）· 对局配置与难度风险
- [x] 判题沙箱硬化 · 结构体题契约 · 判题结论防伪造 · 判题锁外并发
- [x] 账号 / 战绩 / ELO · PvE 三档 Bot · 4 关交互式教程
- [x] 认输握手 · 连接保活与并发上限 · 结算统计与日志着色

**下一步**

- [ ] **在线排位 1v1**：段位就近匹配、服务端权威同步、断线判负 / 重连（08）
- [ ] 反作弊硬化：禁粘贴编辑器、节奏与粘贴指纹、管理端（11）
- [ ] 视觉打磨：SVG 图标、动效落地、职业色血条、难度徽标（25）
- [ ] 对局回放与错题本、排行榜、CI 门控补全

## 许可

本项目采用 **源码可见（source-available）** 模式，不是 OSI 开源项目：

| 范围 | 许可 |
|---|---|
| 代码（`client/` `server/` `judge/` `proto/` `db/` `deploy/` `scripts/`，含题库模板） | [PolyForm Noncommercial 1.0.0](LICENSE) —— 非商用自由使用、修改与分发；含专利授权；须保留 [NOTICE](NOTICE) 署名 |
| 内容（`docs/` `design/` `.scratch/` 与项目文档） | [CC BY-NC-SA 4.0](LICENSE-CONTENT) —— 署名、非商用、相同方式共享 |

- **商用需授权**：任何商业用途（商业产品集成、付费服务、商业分发）须获得作者书面许可，联系 <hysytess@gmail.com>。
- **署名**：保留 `NOTICE` 中的 `Required Notice: Copyright (c) 2026 zklmiao` 行；内容转载须注明「CodeWar · 代码战争 / zklmiao」及许可链接。
- **贡献**：欢迎 PR，提交即表示同意按相同许可并入；商用授权仅覆盖作者本人创作部分，含外部贡献的部分需另行取得相应贡献者同意。

## 说明

- 美术资产为统一占位系统（设计 token 单源，可整包替换为正式美术）。
- `ref_resources/`（外部项目参考）不入库；本仓库未复制任何第三方代码或美术资产。
- README 截图取自真实对局与教程界面（1366×768 与 1440×900）。
