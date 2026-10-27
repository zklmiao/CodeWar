# CodeWar MVP 独立验收报告（2026-10-27）

- **验收对象**：main @ `6ae0d8b`（30 张 resolved 票据：01-24、26-31）
- **验收人**：独立验收工程师（agent，零代码改动；唯一写产物 = 本报告 + `gui-test-screenshots/acc-*` 截图）
- **基线文档**：`docs/specs/001-codewar-mvp.md`、`docs/design/002-004`、`docs/adr/0001-0008`、各票票尾记录
- **环境**：Windows 11 / Docker 29.8.1 / go1.26.4 + gcc 13.1.0（-race 可用）/ Python 3.14.7 / pnpm 12.4.1；`deploy/docker-compose.yml` 三服务构建启动成功（app:3001、judge:50051、postgres:25432），验收结束已 `docker compose down` 恢复

---

## 1. 总体验收结论

**有条件发布（conditional pass）。**

理由：已交付范围内**无 blocker**——层 A 自动化四套件全绿（含 -race 与真容器 pytest）、层 B 数值对照表逐项符合且多数有「代码锚点 + 精确值测试 + GUI 实测」三重证据、层 D 安全向量 4+2 项全部实跑通过、核心玩法链路（教程→解锁→配置→盲选→对局机制→认输→结算统计）在浏览器中逐项实证。存在 **1 项 major**：ADR-0008 §3 宣称的 `inject_target` 写点白名单加固在代码中不存在（文档与实现失实，安全暴露本身未变化），需人裁决「补实现」或「修订 ADR」后方可关账。其余为 3 项 minor 与 2 项观察记录。层 C 个别动态项因验收环境（IAB 内嵌浏览器客体进程周期性崩溃-恢复）无法完整走通，已按协议降级为引用自动化测试名并标注 UNVERIFIED-dynamic，自动化覆盖对这些项均充分。

---

## 2. 未交付范围声明（只声明，不评判为缺陷）

| 范围 | 票据 | 状态 |
|---|---|---|
| 用户故事 4（段位匹配 → 在线 1v1） | 08 号票 | ready-for-agent，未开工 |
| 用户故事 36 部分（编辑器禁粘贴）/ 37（节奏与粘贴指纹检测、违规审计） | 11 号票 | ready-for-agent，未开工 |
| 视觉打磨（SVG 图标替换、动效四件套 client 落地、HpBar 职业色、难度徽标） | 25 号票 | needs-info，挂起 |
| 判题遥测扩展（内存峰值/异常类型上报）与静态卫兵完整版 | 02→11 递延 | 归 11 号票 |
| 在线化前的部署加固（judge 容器非 root 化等） | ADR-0003 递延 | 归 08 号票开票时处理 |

UI 上对应的已观察现状：入口页「难度徽标位（25 号票）」占位文本可见（见 acc-08a 等截图），属 25 号票挂起项的如实呈现。

---

## 3. 已记录偏差基线清单（逐票收集；下文发现与此清单对碰后仍属新的才计入第 8 节）

| # | 来源票 | 偏差/裁决内容 | 本次对碰结果 |
|---|---|---|---|
| 1 | 01 | 伤害公式难度系数递延，01 实现 `基础×时间系数`，由 05 闭环 | 已闭环：`damageFor` 乘 `riskCoeff`（engine.go:816-834），复合数值测试在 risk_test.go |
| 2 | 03 | 移动限直线正交（L 形拒绝为 diagonal）；护盾为结算批次级（覆盖本回合全部伤害，回合结束消失）；同毫秒按提交序 tie-break | 实现与测试均按裁决落地（validateMove / resolveRound / TestShieldCoversFasterAttack 注释钉住 tie-break） |
| 3 | 05 | 风险系数改分段线性（0.85/0.925/1.0/1.10/1.20，L3=1.0 锚点）；最近档重抽推广到全部题型 + 技能双向就近回退 | risk.go 与 TestRiskCoeffTable 全表一致；TestActionQuestionsNeverPlaceholderAtCapL5 钉住重抽 |
| 4 | 06 | 剑士普攻追认为相邻正交 4 格（方向无关），003 已同步；engine 普攻路径向 `effectiveTargeting` 传 `needsDirection=true` 的无害死参数待清理 | coverageCells melee 分支符合；死参数仍在（shapes.go:96-103 调用链），属已记录基线非新发现 |
| 5 | 09 | `cmd/smoke` P1/P2 同毫秒 tie-break 间歇 rounds-cap 为既有问题；Bot 数值未经 playtest 标定 | 本次未复测 smoke（无自动化覆盖，如实记录）；bot.ParamsFor 为 spec 开放问题 3 调参面 |
| 6 | 26 | Mimosa 扫描 A 组 6 条（exec/eval、/tmp 写点）by-design 豁免，依据 ADR-0008；B 组代码侧消除 | 威胁模型本体成立（见层 D）；但 ADR-0008 中「inject_target 已落地」表述与代码不符 → **新发现 F-1** |
| 7 | 28 | 改号 27→28；fork-bomb 归因修订（子进程继承管道写端先触发「恰好一行」规则，同为惩罚路径） | 测试语义与记录一致（test_fork_bomb_limit_exceeded / 四条伪造向量测试） |
| 8 | 29 | 改号 28→29；三项裁决（过期判题丢弃、inFlight 瞬时窗口、solveSeq 落地序）；gofmt -l 既有 `internal/elo/scale_test.go` 未修 | 本次 `go vet ./...` 干净；-race 全绿含并发测试 |
| 9 | 31 | ResignRequest 请求体无字段（席位权威）；教程局以通用 `ResignLocked` 拒投；hub 连接卫生参数（pongTimeout 30s / pingPeriod 27s / writeTimeout 10s / joinTimeout 10s / maxConns=8）；conn.send 满缓冲丢弃 | 走查中动态观察到死链约 30-40s 回收（与 pongTimeout=30s 一致）；断线作废、拒绝文案如记录 |
| 10 | 10/ADR-0006 | 匿名热座 URL 直连是显式放行的实验室旁路 | 走查中匿名 P1 加入对局成功且不落库（matches=0），与记录一致 |

---

## 4. 层 A：自动化结果汇总

| 套件 | 命令 | 结果 | 备注 |
|---|---|---|---|
| 构建+静态检查 | `go build ./... && go vet ./...` | BUILD OK / VET OK | |
| 单测（含 race） | `go test -race ./... -count=1` | **全包 ok**：accounts 7.1s / bot 22.1s / classes 2.6s / elo 1.9s / hub 20.8s / match 3.9s / question 3.3s / templates 319.8s / store 3.8s / tutorial 2.9s | templates 大矩阵含模板参考解真容器冒烟 |
| DB 集成 | `CODEWAR_TEST_DSN='postgres://codewar:codewar-dev@localhost:25432/codewar_test?sslmode=disable' go test ./... -count=1` | **全包 ok**：accounts 2.5s / bot 4.9s / classes 0.6s / elo 0.4s / **hub 185.4s** / match 1.6s / question 0.7s / templates 173.1s / store 6.5s / tutorial 0.7s | store/accounts/hub 真跑未 skip（另以 `-v` 复核：18 个 `=== RUN`、0 个 SKIP）；**注意**：首轮与 -race 套件并行执行时 hub FAIL，系两套 go test 并发操作同名 `codewar_test_*` 测试库互相污染（17 号票隔离契约不覆盖跨进程并发），单跑即全绿——属验收执行注意事项，非产品缺陷，已如实记录 |
| 判题 | `python -m pytest judge/tests -q` | **44 passed**（88s，docker 实跑） | 与 28/31 号票关票记录的 44 passed 一致 |
| 前端 | `pnpm -C client test` / `pnpm -C client build` | **64/64 passed**（12 文件）/ build 通过（7.8s，仅 chunk>500kB 警告） | 与 30/31 号票记录的 64/64 一致 |
| CI 现状 | `.github/workflows/ci.yml` | 三 job（server: build+`test -race`；judge: pytest tests；client: test+build） | 与债务基线 `optimization-review-2026-10-27.md` §3 对碰：无 Postgres service、无 vue-tsc/lint 门、无缓存/并发去重——**均未恶化**（CI 现状=基线）；基线中「hub 连接无 deadline/ping/上限」一项已由 31 号票修复（hub.go 常量块 + hub_conn_hygiene_test），属改善 |

---

## 5. 层 B：数值与规则符合性对照表（报告核心）

判定口径：**规范值 | 代码锚点 | 测试锚点 | verdict**。测试锚点须为具体数值断言（见第 9 节虚假 pass 排查）；「仅锚点无测试」记 PARTIAL。

### 5.1 通用机制（spec 001 权威）

| 项 | 规范值 | 代码锚点 | 测试锚点 | GUI 实测 | verdict |
|---|---|---|---|---|---|
| 回合时长 | 75s | `ports.go:12 RoundDuration=75s` | TestRealClockRoundDoesNotAdvanceEarly | 倒计时 75s→0（acc-02 等多图）；超时滚回合实测（PvE 第 1 回合双方全错→75s 后进第 2 回合） | PASS |
| 时间系数 | 0.6~1.0 线性，越快越高 | `ports.go:17-19 CoeffMin=0.6/CoeffSpan=0.4`、`engine.go damageFor` | TestRaceTimeCoefficientAndOrder | PvE 攻击 10.2s 解题 → 系数 0.946 实测吻合 | PASS |
| 难度系数 | L1 ×0.85 → L5 ×1.20（**分段线性** 0.85/0.925/1.0/1.10/1.20，05 追认版） | `risk.go RiskCoeff`（注释含追认依据） | TestRiskCoeffTable（全表 1e-9 精确）+ TestAttackDamageScalesWithRiskTier | L5 局：26×0.843×1.2=26 实测（acc-02e）；L1 局：18×0.946×0.85=14 实测（acc-08b 日志 14 伤） | PASS |
| 防御减伤 | −60%（×0.4，仅当回合，批次级） | `engine.go:34 DefendFactor=0.4`、resolveRound 护盾集合 | TestDefendReducesDamage（22→9 精确）+ TestShieldCoversFasterAttack | 「🛡 挂起护盾（本回合减伤 60%）」+ Bot 先手攻击被折减 10（acc-06a 序 5-7）；26×0.4→10（教程 L1） | PASS |
| 错误惩罚 | 锁定 5s + 自损 3 HP | `ports.go:13-15 LockDuration=5s/SelfDamage=3` | TestWrongSubmissionPenaltyAndLock（100→97→94、5s 时钟边界、锁内不罚） | 「✖ 提交错误，锁定并自损 3」「剩余 107/104/87」（acc-06b、acc-08a）；Bot 错误分支同样自损（92=95−3） | PASS |
| 回合上限 | 10 回合按剩余 HP 裁决 | `ports.go:16 MaxRounds=10`、`engine.go:573-582` | TestRoundsCapWinner / TestRoundsCapDraw | —（路径引用测试） | PASS |
| 同时归零 | 平局 | `engine.go:567-570`（批次内统一判生死） | TestMutualKODraw | — | PASS |
| 超时判定 | 总时长耗尽按剩余 HP；先手按完成耗时序 | `HandleRoundTimeout`/resolveRound 按 solveMs 升序 + solveSeq tie-break | TestRoundTimeoutSingleSolve、TestEarlyResolution | 先手序实测：10.2s 先于 36.9s 结算（acc-08b 日志序 11/13） | PASS |
| 全对生效 | 部分通过=失败 | judge `_assemble_verdict` + 引擎仅 Passed=true 记行动 | test_partial_pass_counts_as_fail（pytest） | — | PASS |
| ELO 平局 | 双方 +0 | `elo` 包 | TestSettle | — | PASS |
| 技能题难度 | = 所选上限档 | `question/factory.go:306-313 SkillQuestion` | TestSkillQuestionCapTierAndFallback / TestSkillQuestionExactCapTierAtL5 | — | PASS |
| 镜像攻击题 | 双方同实例 | `question/factory_test.go` seed 派生 + 面板共享 attack | TestAttackMirrorSameInstance（ID+语句全等）/ TestMirrorProblemRotation（回合间轮换、私有题按玩家分） | 攻击卡「镜像 · 同题竞速」标识（各截图）；双窗题面核对因 B 窗崩溃中断（环境），由测试覆盖 | PASS |
| 移动规则 | 直线正交 1-2 格（游侠 1-3），路径不穿人、终点空格 | `engine.go validateMove`（游侠 MoveMax=3 来自职业表） | TestMoveValidationRejections | 「🏃 p1 移动到 (3, 1)/(3, 3)」（acc-02b、acc-04c） | PASS |
| 闪避 | 结算时刻位置不在覆盖格=闪避（时刻量） | resolveRound 位置判定 | TestMoveAwayDodgesAttack / TestLateMoveCannotSaveLandedAttack | 「💧 p2 的攻击落空——目标已移出相邻格」（acc-04c 序 12） | PASS |
| 方向选择 | 免费、结算时读最新、缺口自动瞄准 | `SetTargeting` + `autoAim`/`effectiveTargeting` | TestShapeCoverageMatrix / TestClassAttackShapes | PvE 填空题方向按钮 ↑↓←→（acc-08a）；autoAim 未提交时兜底 | PASS |
| 盲选 | 对方开赛前不可见 | 快照对席 HP 按 100 上报（engine.go:790-795） | TestBlindPickPrivacy | 等待期双方均显示 100 泛型（acc-03），开局揭示 90/110（A=法师/B=剑士） | PASS |
| 结算持久化 | 双方认证才落库；平局 +0 | hub `RecordMatch` 路径 | TestPvpResignEndsAndSettles / TestRealStoreResignRecordsReason / TestPveResignBotWinsNoRows | 匿名 P1 认输局 matches=0；PvE 认输局 matches=0（动态验证） | PASS |

### 5.2 职业数值（design 003 权威）

| 项 | 003 规范值 | 代码锚点（classes.go registry 字面值） | 测试锚点 | GUI 实测 | verdict |
|---|---|---|---|---|---|
| HP | 剑士 110 / 法师 90 / 游侠 95 | 同值 | TestClassHPInit | 入局血条 110/90/95（PvE 机器人=游侠 95 可见） | PASS |
| 普攻伤害 | 26 / 18 / 20 | 同值 | TestShapeCoverageMatrix 场景 + risk 场景（L3=22 为 legacy 锚点） | 18 基伤实测折算吻合（acc-08b）；教程局 26 基伤折算吻合 | PASS |
| 剑士普攻形状 | 相邻正交 4 格、方向无关（**06 追认版**） | `coverageCells` melee 分支 | TestShapeCoverageMatrix（逐格） | 网格贴脸命中（acc-02e/acc-08b 相邻位） | PASS |
| 法师普攻形状 | 直线深度 2 线扫 | `coverageCells` line 分支（depth=2） | TestShapeCoverageMatrix | 直线覆盖语义与教程 L2 关卡文案一致 | PASS |
| 游侠普攻形状 | 直线距离 1-3 单点狙 | `coverageCells` snipe（AttackRange=3） | TestShapeCoverageMatrix | PvE 机器人=游侠，距 1-3 命中实测（13-14 伤） | PASS |
| 移动距离 | 1-2 / 1-2 / 1-3 | MoveMax 2/2/3 | TestMoveValidationRejections | — | PASS |
| 巨力斩 | 伤 55 / 3×3 邻域 8 格 / CD4 | Power:55 Cooldown:4 ShapeBurst8 | TestSkillDamageAndCooldown（55 字面、90−55=35、CD 拒绝序列 R2-R4、R5 再放） | 「✨ p1 释放技能：造成 55 伤害」+ 槽位「冷却中（还剩 3 回合）」（acc-06a） | PASS |
| 求生本能 | 治 30 / CD3 | Heal:30 CD3 | TestHealOverflowCapped（溢出截断） | — | PASS |
| 烈焰风暴 | 伤 40 / 前向 6 格带 / CD3 | Power:40 CD3 ShapeBand Depth2 | TestFirestormBandEventStream（事件流逐格） | —（装备与释放语义同族已证） | PASS |
| 秘法汲取 | 伤 25 + 自疗 20 / 射程 2 / CD3 | KindDrain Power:25 Heal:20 RangeMax:2 | TestDrainHeals / TestDrainMiss | — | PASS |
| 穿云箭 | 伤 45 / 射程 4 / CD4 | Power:45 CD4 RangeMax:4 | TestPierceEndToEnd | — | PASS |
| 风行回复 | 治 28 / CD2 | Heal:28 CD2 | TestWindheal | — | PASS |
| 盲选互不可见 | 1v1 双方互不可见 | 快照规则（§5.1 行） | TestBlindPickPrivacy | acc-03/04a/04b | PASS |
| 技能字面值 | 技能不乘时间/难度系数（05 裁决 2） | damageFor 注释 + applySkillDamage 路径 | TestSkillDamageIgnoresRiskCoeff | 巨力斩 55 字面（acc-06a） | PASS |

### 5.3 出题系统（design 002）

| 项 | 规范 | 测试锚点 | verdict |
|---|---|---|---|
| seed 确定性 / 实例随机化 | 同 seed 同实例、跨 seed 不同 | TestCreateDeterministic / TestCreateDifferentSeedsDiffer / TestDeriveSeedStable | PASS |
| 攻击双形式 | complete + fill-blank（1-3 核心行挖空） | checks_test（挖空行数/核心行校验） | **PASS（GUI 实证双形态**：complete 形态见教程局，fill-blank 形态见 PvE「# ____ 填空：补全本行逻辑」acc-08a） |
| 难度区间 [L1, cap] | 均匀/爬坡两策略 | TestUniformDifficultyInRange / TestClimbMonotonicToMax / TestUniformRoughlyEven | PASS |
| 失败反馈只回类别 | 不泄漏隐藏用例信息 | test_hidden_case_info_not_leaked / test_verdict_contract_whitelist（pytest） | PASS |
| 模板矩阵覆盖 | 124 口径（12 号票裁定） | templates 包 coverage 测试（-race 全量 319.8s ok，含参考解真容器冒烟） | PASS |

---

## 6. 层 C：浏览器动态走查记录（12 步剧本 × verdict × 截图）

**环境与方法**：compose 栈 http://localhost:3001；双账号 acceptA/acceptB 经 GUI 注册（sessionStorage 按标签页隔离）；IAB 内嵌浏览器 + web-gui-tester 方法论（黑箱 GUI 操作，观察=DOM 快照×截图双验）。
**重大环境限制（如实声明）**：IAB 客体进程在长会话中周期性崩溃并恢复到旧 URL，导致 (a) 页面会话/表单丢失、(b) 对局中 WebSocket 断开。服务端按设计将断线对局作废/席位回收（约 30-40s，与 31 号票 pongTimeout=30s 一致），期间重连被拒。该限制影响连续多回合的 PvP 热座局完成度，不影响单回合内机制验证；由此产生的现象均已与 31/08 号票记录对碰。

| # | 步骤 | verdict | 证据（截图号 / 测试名） |
|---|---|---|---|
| 1 | 教程门禁：未完成账号 PvP 锁定提示 | **PASS** | acc-01（「🔒 先完成新手教程才能进入 1v1 匹配」，P1/P2 卡隐藏，PvE 保留）；DSN 测试日志亦见「新手教程需要登录」「教程第 3 关尚未解锁」拒绝 |
| 2 | 教程 L1 实操过关 + 进度持久化 | **PASS** | acc-02（完成层「✅ 第 1 关 · 读题作答 → 攻击 完成」）；刷新后入口显示「继续第 2 关」（服务端落库，users.tutorial_level=4 也经 DB 直查证实） |
| 3 | 教程完成解锁 PvP | **PASS** | acc-02e（「🎉 新手教程全部完成，1v1 匹配已解锁」）+ acc-04a 横幅「新手教程已完成 ✓ 1v1 已解锁」 |
| 4 | 配置四项随链接携带；P2 链接可用 | **PASS** | 入口读数：`/?class=mage&skill=firestorm&topic=syntax&cap=L1&strategy=climb&player=1`（题型/难度上限/抽题策略/职业装备全携带）；对局粘性条显示「题型 语法与运算 · 难度上限 L1 · 抽题爬坡」；「复制接入链接（第二窗口）」按钮存在（acc-00） |
| 5 | 职业盲选互不可见 | **PASS** | 各窗口仅携带自身 class/skill（URL）；等待期对席显示泛型 100（acc-03）；开局揭示 90/110（A 法师/B 剑士，acc-04a/04b 选择态）；TestBlindPickPrivacy |
| 6 | 热座整局 ≥4 回合全机制 | **PARTIAL**（环境受限降级） | 逐机制 GUI 已证：镜像同题语义（卡片「镜像 · 同题竞速」+ TestAttackMirrorSameInstance）；走位闪避（acc-04c「p2 的攻击落空」）；防御 −60%（acc-06a：挂盾 + 先手攻击折减 10）；技能释放与冷却灰置（acc-06a：55 伤 + 「冷却中（还剩 3 回合）」+ skill-cooldown 拒绝序列测试）；错误提交锁 5s+自损 3（acc-06b、acc-08a：110→107→104、Bot 95→92）。**连续 ≥4 回合 PvP 热座未完成**：两次开赛后均因 IAB 客体崩溃致一方连接断开、对局作废（引擎/断线语义见第 9 步）；PvE 侧完成 4 回合整局（acc-08a→08c，Bot 出手/解题/错误分支全程真实判题） |
| 7 | 结算页：胜负/原因 + 统计区块 + ELO 行 | **PARTIAL** | 胜负+原因+统计（回合数/双方总伤害/双方闪避次数）GUI 实证：acc-08c（PvE 认输：回合数 4、总伤害 P1 14·P2 27、闪避 0·0）、acc-07a（PvP 认输：回合数 1、0·0、0·0）——与事件流一致（30 号票 kind 着色亦可见：伤害红/✔绿/护盾蓝）。**ELO 变化行动态未捕获**（双方认证整局因环境未完成）→ UNVERIFIED-dynamic，引用：EndOverlay.spec「终局面板 ELO 区块」（1000→1016/984 双方渲染 + 匿名不显示）+ TestPvpResignEndsAndSettles + TestRealStoreResignRecordsReason；动态补证：匿名/PvE 局 matches=0（不落库语义正确） |
| 8 | PvE 三档难度 + 认输不落战绩 | **PARTIAL** | 普通档 2+ 回合实测：Bot 出题/提交经真实判题（错误分支 syntax→自损 3），解题节奏 36.9s/41.1s/32.6s 可感知（与教程脚本 Bot 的 0.3-0.5s 明显不同）→ acc-08a/08b；**认输生效且不落战绩 PASS**（acc-08c + DB matches=0 + TestPveResignBotWinsNoRows）。easy/hard 差异未动态逐档打（时间盒）→ 引用 09 号票 30 局/档数据（easy 均耗 58.1s 胜 0/30、hard 18.7s 胜 13/30）+ ADR-0005 ParamsFor |
| 9 | 断线重连快照恢复 | **PARTIAL** | 热座 PvP 重连被明确拒绝（拒绝文案实捕：「p1 已有玩家接入（**本票不支持重连，08 号票处理**）」「职业绑定失败：对局已开始，职业绑定被拒」）——与 08 号票未交付范围一致，属当前票设计语义而非缺陷；PvE 断线=作废重开（动态观察多次）；快照恢复机制引用 TestSnapshotMidRound + hub 重连/断连测试族（TestPveDisconnectAbortsAndRefreshRestarts 等，-race 绿） |
| 10 | 超时判定与平局路径 | **PARTIAL** | 超时路径动态观察（回合无行动→75s 超时→下一回合，PvE 第 1 回合）；75s 真实等待与平局构造在走查环境不可行 → UNVERIFIED-dynamic，引用：TestRoundTimeoutSingleSolve / TestMutualKODraw / TestRoundsCapWinner / TestSelfKO / TestLateSubmissionIgnored |
| 11 | 个人页：场次/胜率/ELO/最近对局 | **PARTIAL** | 页面渲染与数据字段实证（注册后直跳：「acceptA ELO 1000 · 共 0 场 · 胜 0 · 负 0 · 平 0 · 胜率 0.0% · 最近对局空态」）；有数据态动态未成（依赖 7/8 的结算局）。**复现观察**：经历被拒 join 重连循环的标签页，/profile 停在「加载中…」（双标签复现，同刻 curl /api/profile 32ms 正常）——系 08 号票已记录的「重连自我倍增」债务耗尽浏览器连接池所致（动态实证，见发现 F-3，非新缺陷）；API 数据层另有 store/accounts DSN 套件覆盖 |
| 12 | 风格页对照 004 | **PASS** | acc-12a（token 对照：六族 token 板、层级/字号/间距标尺）、acc-12b（三职业剪影+主色一眼可辨：剑士=赭红重甲巨剑 / 法师=紫长袍法杖 / 游侠=绿轻甲短弓；血条 25% 刻度与 ≤30% 低血 danger 语义；题卡四语义槽位）。Token 抽查 5 项：`--cw-color-class-warrior/mage/ranger`、`--cw-color-seat-p1/p2`、`--cw-color-state-danger`、`--cw-color-bg-target`、`--cw-z-overlay/modal` 与 004 §2 登记一致。注意：风格页未包含在 compose 部署物（/styleguide/ 落入 SPA 兜底路由），需本地静态服务器查看（见发现 F-4）；token 本体经 `Dockerfile.app` COPY design/ 进入 client 构建，单源契约未破 |

截图清单（`gui-test-screenshots/`，本次验收新增 21 张 `acc-*`；另有 `acc-tmp-*` 过程图 6 张与 2026-10-04 历史图 4 张未在本报告引用）：acc-00、01、02、02b、02d、02e、03、04a、04b、04c、04d、04e、06a、06b、07a、08a、08b、08c、11a、12a、12b。

---

## 7. 层 D：安全与反作弊向量结论

### 7.1 spec「测试决策」四向量（judge/tests，docker 实跑 44 passed）

| 向量 | 测试 | verdict |
|---|---|---|
| 死循环 | test_deadloop_killed_by_timeout | PASS（真实容器超时终止） |
| 部分通过 | test_partial_pass_counts_as_fail | PASS（判失败且不回传计数） |
| 资源超限 | test_memory_bomb_limit_exceeded / test_fork_bomb_limit_exceeded | PASS（limit-exceeded 归因） |
| 联网尝试 | test_network_attempt_blocked | PASS（--network=none 拒绝） |

### 7.2 28 号票（atexit 结论伪造 fail-closed）回归

- 测试四条全绿（本次 pytest 实跑）：test_atexit_forgery_keeps_penalty_path（atexit 伪造→惩罚路径）、test_import_time_stdout_forge_fails_closed（import 期伪造→fail-closed）、test_forked_child_forged_line_fails_closed（fork 子进程伪造）、test_player_print_stays_trapped_protocol_unpolluted（对照）。
- 机制锚点：worker emit 后 `os._exit(0)` 硬退 + harness「恰好一行结论行」规则——与票尾交付记录一致。**verdict：PASS（回归通过）**

### 7.3 29 号票（判题移出引擎锁）回归

- `go test -race ./internal/match/... ./internal/hub/... ./internal/bot/... -count=1` 全绿（含 engine_concurrent_test 公平性/响应性/过期丢弃/inFlight 四条并发测试）；Submit 三段式实现（锁外 received → 持锁预检+inFlight → 锁外判题重持锁落地）与票尾记录一致。**verdict：PASS（回归通过）**

### 7.4 26 号票 ADR-0008 处置符合性（executor.py 抽查）

- 一致项：容器隔离参数与 ADR-0003/0008 逐字一致（`--network=none --memory=256m --pids-limit=64 --cpus=0.5 --cap-drop=ALL --read-only --tmpfs /tmp:size=16m,exec`，executor.py:145-151）；/tmp 三个写点为字面常量 `/tmp/solution.py`、`/tmp/_worker.py`、`/tmp/_helpers.py`（:206-210），无动态路径成分，CWE-22 前提确实不成立；`guard_author_code`（长度 4000 + 禁用清单 + 必须可编译）在 :170 且注入前先验；expected 不进容器（stdin 载荷契约 + 泄漏断言测试）。
- **不一致项 → 新发现 F-1（major）**：ADR-0008 §3 称「harness 写点收敛到 `inject_target(name)`（文件名白名单 + 解析双校验，26 号票落地，测试全量覆盖）」——`grep -rn inject_target` 全仓仅命中 ADR 文档；`git log -S inject_target --all` 仅 `ec3484c`（纯文档提交）；26 号票文内零提及；executor.py 现状即三个字面 open 写点。安全暴露未变化（字面路径 + 容器边界兜底），但**验收记录与代码失实**（宣称的控制与测试不存在）。
- 11 号票未交付项（静态卫兵完整版、遥测扩展、编辑器禁粘贴、行为指纹）明确列为**未验收范围**。

---

## 8. 新发现分级清单（与第 3 节基线对碰后仍属新者；全部只记录不修复）

### F-1【major】ADR-0008 记录的 `inject_target` 加固未存在于代码

- **现象**：ADR-0008 §3「收紧（26 号票落地）：harness 写点收敛到 inject_target(name)——文件名白名单常量 + os.path.join 解析结果必须恰为 /tmp/<name>……测试全量覆盖」；代码中无此函数/白名单/对应测试（详见 7.4）。
- **锚点**：`docs/adr/0008-sandbox-threat-model.md:36` vs `judge/judgesvc/executor.py:206-210`；`git log -S inject_target --all` → 仅 `ec3484c`（docs-only）；26 号票文 0 命中。
- **影响评估**：非安全回归（写点本为字面常量，26 号票门禁豁免批准时的代码行为与现状一致；真实边界为容器隔离）。属**验收记录失实**：ADR 作为「可引用的定性锚点」包含不存在的控制描述，会误导后续审计与 08 号票部署加固的基线判断。
- **复现**：`grep -rn inject_target judge/ | wc -l` → 0（文档除外）。
- **建议**：开小票二选一（裁决归人）：(a) 落地 inject_target 白名单 + 白名单越界 fail-closed 测试；(b) 修订 ADR-0008 §3 如实记录「该收紧曾提议、未落地，写点保持字面常量」。建议票号：32（与优化评审 backlog 的 32 号票号段协调，或顺延）。

### F-2【minor】EndOverlay 认输原因文案未按观察者视角渲染

- **现象**：玩家主动认输后，其自己的结算页显示「对手主动认输（31 号票）」，与事实相反（两次动态复现：acc-08c B 认输 / acc-07a 同路径）；胜负与统计正确，对手侧显示正确。
- **锚点**：`client/src/components/EndOverlay.vue:18`（`resign: "对手主动认输（31 号票）"` 为静态映射，未按 viewer==resign 区分）。
- **建议**：按观察者是否为认输方区分文案（「你主动认输」/「对手主动认输」），resign.spec 补一条断言；可并入 30/31 号票的后续打磨或独立小票。

### F-3【minor】被拒 join 后重连自我倍增导致页面资源饥饿（08 号票已记录债务的动态实证，非新缺陷）

- **现象**：经历多次「接入被拒」的标签页，/profile 的 fetch 永不返回（双标签复现「加载中…」），同刻服务端 curl 32ms 正常——浏览器每主机 6 连接池被重连循环占满。
- **锚点**：08 号票验收清单原文已记录（「App.vue connect() 的 setTimeout 重试不清理旧定时器，被拒连接会自我倍增……重连须单飞 + 指数退避」）；本次走查为其首份动态实证。
- **处置**：计入 08 号票未交付范围（第 2 节），建议 08 实现退避时以「被拒后个人页可正常加载」作为验收观察点。

### F-4【minor】风格页未包含在 compose 部署物

- **现象**：`/styleguide/` 落入 SPA 兜底路由渲染空白页；风格页仅能按 004 注记以本地静态服务器查看。
- **影响**：不影响 004「token 单一来源」契约（design/tokens.css 经 Dockerfile.app 进入 client 构建并被引用）；仅部署环境不可直接查看，损害 discoverability。
- **建议**：compose 增加静态路由或在 README 注明查看方式（优化项）。

### O-1【观察，不计缺陷】hub 单一对局槽 + 断线作废语义的现场表现

进行中对局存在时，新 join（含不同 mode 的 PvE）会重绑进该对局并被「职业绑定失败：对局已开始」拒绝；崩溃客户端的存活 WS 使席位占位至 pongTimeout（实测约 30-40s 回收，与 31 号票参数一致）或对局自然收束。多对局并行与真重连拓扑归 08 号票。与 31 号票连接卫生设计一致，如实记录供 08 设计参考。

---

## 9. 验收质量协议执行记录

1. **覆盖合理性（37 条用户故事矩阵，37/37 无漏项）**：
   - PASS：1（注册登录，GUI+DSN）、2（个人页字段，见 6-11）、3（ELO 展示，导航栏 1000 + profile + TestSettle）、5（教程 4 关实操，A/B 双账号全通）、6（解锁门禁）、7（题型过滤，UI 6 选项 + TestAttackTopicFilterStrict）、8（难度上限）、9（难度风险回报，双 GUI 实测）、10（抽题策略）、11（四题面板+冷却灰置）、12（镜像题）、13（全对生效）、14（错误惩罚）、15（时间系数）、16（耗时序结算）、17（一回合一行动 TestOneActionPerRound）、18（空窗 TestEmptyRound）、19（普攻 22=legacy 锚点）、20（形状射程）、21（移动与闪避）、22（防御 60%）、23（技能数值，按 003 修订表）、24（冷却灰置）、25（同时归零平局）、26（超时按剩余 HP）、27（难度区间）、28（双形式）、29（四维标注）、30（参数随机化）、31（seed 确定性）、32（PvE Bot）、33（PvE 同配置 TestPveAndPvpShareQuestionConfig）、34（三职业）、35（技能装备）；
   - PARTIAL：2（最近对局有数据态动态未成，组件+DSN 覆盖）、36（判题沙箱+实例随机化已交付且实证；「编辑器禁粘贴」未交付归 11 号票）；
   - N-A（未交付）：4（段位匹配→08）、37（节奏指纹→11）。
2. **证据充分性**：每个 PASS 均标注证据类型（自动化输出 / 截图号 / 锚点+数值推演）；静态结论均升级为测试或运行证据（如 §5 各行），无法升级者降 UNVERIFIED-dynamic 并引用测试名（6-7 的 ELO 行、6-10 的平局路径）。
3. **虚假 pass 排查（随机抽 5 个关键数值断言的测试源码）**：
   - TestWrongSubmissionPenaltyAndLock：断言精确 −3（HP 97/94）、LockDuration 时钟边界、锁内零罚——非「不报错」式断言 ✓
   - TestDefendReducesDamage：断言 Damage==9（22×0.4）与 HP==91 精确值 ✓
   - TestRiskCoeffTable：五档全表 1e-9 精确 + 未配置/越界钳制 ✓
   - TestSkillDamageAndCooldown：断言 55 字面（不乘系数）、法师 90−55=35、R2-R4 逐回合 skill-cooldown 拒绝、R5 再放与 KO ✓
   - TestAttackMirrorSameInstance + TestMirrorProblemRotation：断言同回合 ID+语句全等、跨回合轮换、私有题按玩家分 ✓
   - **-count=1 重跑防缓存**：抽 2 组共 6 测试（TestWrongSubmissionPenaltyAndLock/TestMirrorProblemRotation/TestDefendReducesDamage -race 一组；TestRiskCoeffTable/TestClassHPInit/TestSkillDamageAndCooldown 一组）全部 PASS ✓
   - **UI 截图核对**：acc-06b/08a/08b/08c 等的画面内容（HP 数值、事件流文案、统计块数字）与所断言行为逐一核对一致 ✓
4. **发现处置**：分级 blocker/major/minor 如第 8 节；验收零修复；与第 3 节基线对碰后：01/03/05/06/09/26/28/29/31 记录项均按基线归类，未重复计账；inject_target（F-1）、EndOverlay 文案（F-2）、风格页部署缺失（F-4）为基线外新发现。
5. **交付核对（完整性自查）**：
   - [x] 37 条用户故事覆盖矩阵 37/37（PASS 32 / PARTIAL 3 / N-A 2）
   - [x] 数值对照表逐项无空（通用 17 项 + 职业 13 项 + 出题 5 项）
   - [x] ADR 8 篇各有合规结论：0001 技术栈（monorepo/Compose 三服务/proto 单源/gRPC+WS 形态与实态一致）、0002 事件流同步（seq 单调+快照重连+服务端权威，测试钉住）、0003 沙箱（docker.sock 外挂 compose:31、命令逐字一致、fail-closed 测试）、0004 结构契约（test_struct_contract e2e×4 真容器绿）、0005 Bot 边界（TestMatchPackageHasNoBotReferences 绿 + PvE 同源出题测试）、0006 教程（教程=PvE 模式、pvpGate 拒绝日志实捕、TestMatchPackageHasNoTutorialReferences 绿）、0007 收敛（bot 包 ApproachMove 与回归绿）、0008 威胁模型（隔离边界一致；inject_target 失实 → F-1）
   - [x] 安全向量 4+2 项有结论（四向量 + 28/29 回归 + 26 符合性）
   - [x] 走查 12 步 × verdict × 截图号（截图 21 张，编号连续引用；无法动态项均标 UNVERIFIED-dynamic 并引用测试名）
   - [x] compose 环境已恢复（`docker compose down` 完成；临时静态服务 8123 已停）；验收全程零生产代码/测试/票据改动，本报告为唯一文档写产物

---

*报告完。票据状态未做任何改动；全部发现待人裁决。*
