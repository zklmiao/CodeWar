# 27: 技能形状判定测试补齐 + 形状语义单源化

**What to build:** 06 号票关票复核（2026-10-26 双轴 code review）确认六个技能的形状判定实现与设计 003 §4 逐格一致，但留下三类工程债：1) 四处技能路径缺直接断言；2) 「形状 → 需方向/需距离」的语义映射在 engine 普攻、engine 技能、client 瞄准、bot 决策四处各写一份；3) 普攻结算仍向 `effectiveTargeting` 传对 melee 无效的 `needsDirection=true`。本票只做测试补齐与语义收敛，**不改任何数值/形状/平衡**（设计 003 §4 是唯一权威，改动须另立设计裁决）。

**Blocked by:** None (can start immediately；06 已 resolved，本票是其复核遗留)

**Status:** resolved

**现状证据（2026-10-26 复核，HEAD 行号）：**

测试缺口（`server/internal/match/engine_class_test.go`）：

- 烈焰风暴（band）仅断言 up 方向 6 格（`TestShapeCoverageMatrix` :49-50）；left/right 的横向旋转分支（shapes.go 中 `(dc,dr)` 旋转）无覆盖——最易错的几何分支。
- 穿云箭（游侠 snipe，射程 4）只有矩阵单元用例（:41-45），无端到端技能流（提交技能题 → 命中 → 伤害 45 → CD 4 登记）。
- 秘法汲取落空分支无断言（`TestDrainHeals` :261 仅覆盖命中：−25/+20）；设计 003 §1 要求未命中时伤害与自疗都不发生。
- 风行回复（游侠治疗 28）完全无直接断言；剑士求生本能的溢出封顶已由 `TestHealOverflowCapped` :230 覆盖，可作模板。

语义映射散落四处：

- `server/internal/match/engine.go:428-429` 普攻：`needsDist := cls.AttackShape == classes.ShapeSnipe`，方向恒 `true`（melee 忽略 → 死参数）。
- `server/internal/match/engine.go:456-457` 技能：`needsDir := sk.Shape != ShapeBurst8 && sk.Shape != ShapeSelf`；`needsDist := sk.Shape == ShapeSnipe`。
- `client/src/components/QuestionPanel.vue:92-106` `aimShape()`：`c.attackShape !== "melee"`、`k.shape !== "burst8" && k.shape !== "self"`。
- `server/internal/bot/aim.go:33` `PlanTargeting` 的 shape 分支。

新增形状须同步改四处；任一处漂移即「客户端可选的方向/距离与服务端判定不一致」或「bot 与引擎失准」。

伴生重复（同题材，纳入本票）：

- `dirDelta`（shapes.go:11）与 `delta`（bot/aim.go:89）逐行相同。
- 点狙距离钳 `[1, rangeMax]` 三处重复（shapes.go 的 coverageCells 与 effectiveTargeting、bot/aim.go:37-43）。
- bot `aimAxis`（aim.go:13）复刻引擎 `autoAim` 的主轴裁决。

- [x] 烈焰风暴四向均有断言；left/right 的横向旋转覆盖格逐格正确（6 格）
- [x] 穿云箭端到端：命中伤害 45、CD 4 登记；距离 1/4 边界与超距落空均有断言
- [x] 秘法汲取落空：Dodged 事件、无伤害 HpChanged、无自疗
- [x] 风行回复：治疗 28、满血不溢出、冷却 2 生效（不依赖瞄准/命中路径）
- [x] 形状语义单一来源：`classes` 包暴露需方向/需距离（及 depth/range）判定，engine 普攻+技能、client、bot 全部消费同一来源；四处散落 if/switch 消除（客户端经 `/api/classes` 或现职业载荷获取，不引入 proto 变更）
- [x] 重复辅助单源化：`dirDelta`/`delta` 合并、点狙距离钳单一实现、bot 主轴裁决复用引擎实现（或票内记录不能复用的理由）
- [x] `needsDirection` 死参数消除：普攻路径按形状语义传参，melee 不再走无效分支
- [x] 行为零变化：`go test -race ./...`（有 `CODEWAR_TEST_DSN` 时并行含 DB 包）、`pnpm -C client test`、`pnpm -C client build` 全绿；设计 003 §4 数值/形状零改动
- [x] 测试走事件流 seam（spec「测试决策」）；几何逐格断言集中在既有 `TestShapeCoverageMatrix` 用例表内扩展（shapes.go 头注豁免延续，不另建直测函数），命中/落空/伤害/冷却等行为断言一律走事件流

> **交付记录（2026-10-26，勾票）**：`ticket/27-skill-shape-tests-and-semantics` 分支提交 `b0a6e1d`（四处测试缺口）、`89bdb52`（语义单源 + 重复收敛）、`5733ae8`（code-review 加固：满血治疗 HP 不变量与场景走位落点钉住，变异电池 M1/M3/M4/M5/M7/M8 + band 旋转全部必红后还原）。回归：`go vet ./...`、`go test -race ./...`、`CODEWAR_TEST_DSN` 并行 DB 包、`pnpm -C client test`（47/47）、`pnpm -C client build` 全绿。双轴 code review（Standards/Spec）无硬违规、无漏做；勾票依据见会话验收报告。

> **来源记录**：06 号票追认记录（2026-10-26）的「后续」项；同日双轴 code review 的 Standards 轴 4 项坏味道与 Spec 轴 4 处测试缺口在本票收口。

> **关票记录（2026-10-26，人授权"合并关票"）**：分支 `ticket/27-skill-shape-tests-and-semantics`（b0a6e1d / 89bdb52 / 5733ae8）以 merge 431ac15 合入 main，合并后回归全绿（`go vet` + `go test -race ./...` 全包 ok——templates 180s 大矩阵含内、`pnpm -C client test` 47/47、`pnpm -C client build` 通过）。Status 由 ready-for-agent 翻转；worktree 与分支随关票清理。
