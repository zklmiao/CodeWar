# mattpocock/skills 使用指南

> 基于对仓库 https://github.com/mattpocock/skills （v1.2.3）README 与全部 27 篇 skill 文档的完整阅读整理。

## 〇、这是什么，怎么开动

Matt Pocock（Total TypeScript 作者）日常在用的一套 AI 编码工程 skill。定位是"真实工程，不是 vibe coding"：**小、可组合、可魔改**。它刻意不做接管全流程的框架，每个 skill 只解决一个失败模式（需求对不齐、话说不清、代码跑不通、烂泥球架构），流程由你自己拼。

**心智模型：skill 分两类。**

| 类型 | 触发方式 | 代表 |
|---|---|---|
| 编排类（user-invoked） | 只能你手动敲斜杠命令，agent 不会自动调用 | grill-with-docs、to-spec、to-tickets、implement、wayfinder、triage、retro… |
| 纪律/参考类（model-invoked） | 你可调用，agent 判断任务匹配时也会自动触发 | tdd、diagnosing-bugs、grilling、prototype、research、code-review、codebase-design、domain-modeling、pr |

**全局地图**——六个场景全部从这张图分叉：

```txt
主流程:  grill-with-docs → to-spec → to-tickets → implement → code-review → retro
                              ↑ 合流点
支线入口: triage（别人报来的 issue）   wayfinder（大到装不进一个会话）
          improve-codebase-architecture（代码库体检，定期跑）
独立工具: prototype / research / handoff / to-questionnaire / wait-what / wizard
词汇层:   codebase-design（深模块、seam 等架构词汇）+ domain-modeling（GLOSSARY.md、ADR）
```

**动手前两步**：

1. 本机 ZCode 已装好 v1.2.3，直接用；其他环境可 `claude plugins install mattpocock-skills` 或 `npx skills@latest add mattpocock/skills`（后者可改源码）。
2. **每个仓库跑一次 `/setup-matt-pocock-skills`**：问三件事——issue 放哪（GitHub / GitLab / 本地 `.scratch/` markdown / 其他如 Jira）、triage 标签体系、文档放哪（GLOSSARY.md + docs/adr/），结果写成 `docs/agents/*.md`。注意：它只写映射文件，tracker 里的实际 label 要你自己去建（这是文档明确承认的局限）。

不知道用哪个就问 `/ask-matt`（路由 skill）。它有两个已知 bug：可能谎称某 skill 不存在（问它 "open the SKILL.md"），可能错误描述某 skill 的用途（让它先读文档再答）。

---

## 场景 1：已有工程重构

**不知道哪里值得重构** → `/improve-codebase-architecture`：

- 只读不改。扫描整个代码库找 "deepening opportunities"（哪里该挖成深模块），产出交互式 HTML 报告，每条带 Strong / Worth exploring / Speculative 强度徽章。
- 最好的问法不是"帮我找问题"，而是告诉它你接下来要做什么："I need to add multi-tenant support next month — how can we make that change easy?" 代码库健康要服务于你要做的变更。
- ⚠️ 一次会话只挑一个候选改造。报告 HTML 引用 CDN（Tailwind/Mermaid），需联网打开。只想要报告就说 "don't grill me, just show the report"。
- 无共享词汇的老仓库：先跑 `/grill-with-docs` 说 "help me document my repo" 建立 GLOSSARY.md，体检效果会好很多。

**已经知道要重构什么** → `/grill-with-docs` 拷问方案（产出的是决策不是 diff）→ `/to-spec` → `/to-tickets` → `/implement`。

**大范围重构**（几十个文件、不可能一次改完）：在 `/to-tickets` 里说明，它会排成 expand–contract 序列——先加新路径、逐个迁移调用方、最后删旧的。

平时讨论架构时 agent 会自动使用 `/codebase-design` 的词汇（deletion test、seam、"one adapter = hypothetical seam"）。重构正确性由 `/code-review` 的 refactoring 轴（Fowler 坏味道）兜底。

## 场景 2：bug 修复

**别人报来的 bug** → 先 `/triage`：验证是否成立、本地复现并把复现命令写进 brief、写成 agent-ready brief、打 ready-for-agent 标签。只用于"不是你创建的" issue。

**确认要修** → `/diagnosing-bugs`（model-invoked；说 "diagnose/debug this" 或描述故障就自动触发）。六阶段两道硬门：

```txt
复现 → 最小化 →【门：没有稳定复现命令，禁止进入下一阶段】
     → 3~5 个假设按概率排序，由"你"挑先验证哪个
     → 埋点（[DEBUG-xxxx] 标记）→ 修复 + 回归测试 → 清理埋点
```

- 修复必须走 `/tdd` 的 test-driven debugging：先在正确的 seam 上写失败测试，亲眼看它以"对的原因"失败（test the test），再改实现。
- 提交信息要写出"哪个假设最终是对的"。
- 如果代码库没有任何 seam 能锁住这个 bug，它会转手给 improve-codebase-architecture——先修架构再修 bug。
- 坑：问简单问题时容易过度触发，说 "just answer this, don't diagnose"；贴复现输出前自己脱敏（skill 不处理 secrets）；埋点→修复之间没有确认门，需要就明说。

修得很痛苦 → `/retro` 复盘。

## 场景 3：现有工程新增需求

**小需求**（一个会话装得下）：`/grill-with-docs`（拷问清楚，定下来的写进 GLOSSARY.md 和 ADR）→ 直接 `/implement`，小东西跳过 spec。

**多会话需求**：

```txt
/grill-with-docs → /to-spec → /to-tickets → /implement（逐票）→ /code-review → /retro
                                          └→ 或 /implement-spec（前沿并行 + git worktrees）
```

- `/to-spec`：先定 seams 再写 prose（先决定代码怎么切，再写需求），读 GLOSSARY/ADR，发布到 tracker。
- `/to-tickets`：tracer-bullet 垂直切片，每张票做完都能跑起来看效果；带 blocking edges 和 quiz 步骤——quiz 时它故意切得过碎你要反驳，检查标准是"这张票我能 demo 什么"。需要前置重构就先排一张 prefactoring 票。⚠️ to-spec 和 to-tickets 之间不要清会话。
- `/implement`：一票一会话，票间 `/clear`；提交到当前分支，**不关票**（关票是你的事）。
- `/code-review`：双轴（Standards 按仓库标准 + Spec 对照需求）；⚠️ 先 commit 再跑（它 diff 固定点…HEAD，看不见未提交改动）；最好开新会话；有 subagent 重入 bug，agent 数异常就让它重发。

大到装不下 → `/wayfinder`（终点/雾区/前沿三段式决策地图，每张票现场选 grilling / prototype / research / task 执行，最终在 to-spec 合流）。

## 场景 4：测试场景和用例探索与创建

先说实话：这个仓库**没有**独立的"测试设计/用例管理" skill，明确拒绝 BDD/Gherkin 的 given-when-then 模板。测试方法论全部封装在 `/tdd`（model-invoked）里，由 `/code-review` 兜底。分四步用：

**① 探索"该测什么"**

- 需求期：`/grill-with-docs` 拷问行为边界时边缘情况会自然浮出，让它们落进 spec 和 ADR。
- 编码期：问 agent（tdd 自动生效），它的标准答案很明确：
  - 测**行为**不测实现，测试名描述行为（describe/it）；
  - happy path / edge case / error path 都要，**优先 error path**——happy path 会被真实使用覆盖，error path 才暴露盲区；
  - 每个测试只锁一个行为；integration test 优先于 implementation test；
  - go one level deeper：在最能锁定行为的最低层测；
  - mock 只 mock 自己拥有的 seam；能用真东西验证就不 mock（能连真数据库就别 mock 数据库）；不 mock 第三方库内部——包一层自己的 adapter 再 mock。

**② 新代码创建测试** → `/tdd`：绿地功能走 tracer bullet（先定类型 → 数据结构 → 第一个行为测试），然后一次只加一个测试，红→绿→verify（test the test：临时破坏实现，确认测试以正确原因失败）。已有功能的行为变更由外向内，从用户可见行为开始。

**③ 存量代码补测试** → characterization 模式：**不要**对遗留代码 test-driven。先 `git log` 找最近变动的文件、跑现有测试套件当快照；时间允许且行为稳定才写 characterization tests（记录现状，不判断对错）。测出意外行为先问用户是不是 bug，别顺手"修正"。

**④ 兜底** → `/code-review` 的 TDD 段。硬规则不可违反：never skip failing tests、never weaken assertions、keep CI green；软规则与现实冲突时有 safety valve——停下来问你。

**适配判断**（tdd 文档明确列出）：适合 CLI 工具、有真实业务逻辑的服务；**不适合** CRUD 壳、薄封装、以视觉为主的工作——这些场景 tdd 会主动停下问你，别硬上。

补充：想检验"测试计划全不全"，把计划丢给 `/grill-me` 拷问；某个诡异行为值不值得测拿不准，用 `/prototype` 做个一次性实验。

## 场景 5：项目性能优化

- **有明确前后对比的回归**（"这接口以前 200ms 现在 2s"）→ `/diagnosing-bugs`，它有专门的性能分支：**先测 baseline，再二分定位，先测量后修复**，流程与 bug 六阶段一致。
- **"帮我找瓶颈"式主动巡检** → 仓库明确说**没有**这个 skill（issue #431 已关闭，作者认为不属其职责）。最接近的替代是 `/improve-codebase-architecture`：它找的是复杂度集中在哪些活动代码区，和热点常有重叠，但那是结构体检不是性能剖析——别指望火焰图。
- **修完之后** → `/retro`，把"以后怎么提前发现这类回归"落成环境改进（如在 CI 加 benchmark）。
- 需要调研外部事实（某库的性能特性）→ `/research` 后台跑，产出带引用的 markdown。

## 场景 6：创意探索和澄清

按"想法在哪、卡在哪"分流：

- **没仓库/非代码想法** → `/grill-me`：无状态、多轮、一次一个问题，直击论证最薄弱点。要主动回答不要被动应付；**用嘴说不清的问题别硬聊**。
- **有仓库** → `/grill-with-docs`（拷问 + 落 GLOSSARY/ADR 文档）。
- **聊不出结果的问题**（UI 长什么样、状态机手感、API 顺不顺手）→ `/prototype`：一次只回答一个问题；逻辑问题做单 HTML 文件，UI 问题做 3 个激进变体对比；用完就扔（park 在 `prototype/<name>` 分支），别当 app demo。
- **卡在"只有别人知道答案"** → `/to-questionnaire`：先采访你收件人是谁，生成对方自己就能填完的问卷。
- **自己讲着讲着丢了** → `/wait-what`；想法大到装不进一个会话 → `/wayfinder`；外部事实 → `/research`；想学概念 → `/teach`；要交接给下一个 agent → `/handoff`（⚠️ 写到临时目录，记得拷走）。

---

## 附：通用坑速查

1. code-review 与 Claude Code 内置 `/code-review` 同名，用带插件前缀的形式调用。
2. implement-spec 会试图并行跑完所有票——它自己文档承认经常失败，单票用 implement 更稳。
3. grill-with-docs 依赖装载（grilling + domain-modeling）偶尔失败，问一句 "which skills are loaded?"。
4. GLOSSARY.md 会越写越臃肿，定期让 grill-with-docs 精简。
5. research 有嵌套 bug：后台任务可能自己再开后台任务，注意盯着。
6. grilling 默认一次一个问题，嫌慢可在 CLAUDE.md 写明允许多问。
7. `/improve-codebase-architecture` 需要仓库有文档才能发挥好，配合 `/grill-with-docs` 用。

**起步建议**：新仓库先跑 `/setup-matt-pocock-skills`；第一次体验主流程，拿一个中等需求走一遍 `grill-with-docs → to-spec → to-tickets → implement → code-review → retro`，走完你就知道每一步的问答节奏值多少了。
