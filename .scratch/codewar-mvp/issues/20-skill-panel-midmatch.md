# 20: 技能槽对局中失效——PanelReady 事件漏填 skill 字段 + 冷却显示不更新

**What to build:** 修复技能槽在对局进行中的两个缺陷，还玩家"每 CD 回合一次、每次解新题"的设计语义：

**A（主缺陷，服务端小改）**：`PanelReady` 事件的 `PrivatePanel` 漏填 `skill` 字段——`privatePanelToPb`（engine.go:761）只填 move/defend，proto 里定义了的 skill 字段（06 号票）从未被填。后果：客户端每回合用事件刷新面板时技能题被清空（match.ts:304 读 `mine?.skill` 得 undefined），**技能题从第 2 回合起消失**（与冷却无关），冷却结束后也不回来；技能槽按钮仍可点（可用性只看是否装备），提交空代码必判错。

**B（伴生缺口，客户端）**：`SkillCdRounds` 只在接入/重连快照里更新，对局中技能槽的灰置倒计时永不刷新——冷却期槽看起来可用，提交才被服务端 `skill-cooldown` 拒绝。修法：客户端从事件流本地推导（judged 技能通过 → 冷却至 当前回合+CD，CD 查 /api/classes 职业表；roundStarted 递减），快照值保留为接入/重连时的权威初值。

## 根因证据（2026-10-24 PvE 手动走查 + 代码定位）

```txt
proto:   message PrivatePanel { move, defend, skill }   ← 字段存在
服务端:  privatePanelToPb(move, defend)                  ← skill 永远为空（engine.go:761）
客户端:  applyPanel → skill: toProblem(mine?.skill)      ← 每回合清空（match.ts:304）
第 1 回合技能题来自接入快照（快照路径正确，engine.go:721）→ 玩家能释放一次 → 之后消失
```

**测试为何没抓住**：引擎测试只断言了快照路径的技能题，未断言 PanelReady **事件**的字段；客户端 vitest 用自构造事件（带技能题）——两边各自绿，proto 契约在 seam 交叉处断了无人看（与 12 号合并红态同族的缺口形态）。

**Blocked by:** None（与 19 号票 bot 包、08 号票 hub/匹配零文件冲突，可三线并行）

**Status:** resolved

- [x] A：PanelReady 事件对已装备技能的玩家携带**当回合**技能题（每回合新实例、ID 随回合变化）；未装备玩家不携带——引擎事件流断言，且必须含"红→绿"证据（修复前该断言必红：skill 恒为空）
- [x] A：真实链路验证技能题在对局中持续存在且冷却结束后可再次释放（引擎级多回合推进断言为最低要求；起栈手动/探针验证作为汇报中的人工指引）
- [x] B：对局中技能槽灰置与倒计时随事件流刷新（释放→灰置 CD 回合→逐回合递减→恢复可点），vitest store/组件测试；重连场景快照值仍为权威（双源不漂移）
- [x] 既有语义零回归：engine/vitest 套件全绿；16 号票两条高强度回归不退（`go test -race ./internal/bot/... -run TestSkillCooldown -count=100` 与 `./internal/hub/... -run 'TestPve' -count=200`——引擎面板组装路径被动过）
- [x] 快照路径不回归：接入/重连快照的技能题与 SkillCdRounds 语义不变（既有测试继续绿）
