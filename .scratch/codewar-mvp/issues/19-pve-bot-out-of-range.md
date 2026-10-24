# 19: PvE Bot 超射程自我放逐（随机走位后永久 miss）

**What to build:** 修复 PvE Bot 的走位决策缺陷：Bot 按行动权重随机"移动"时可能走出自身攻击射程，此后每回合攻击被引擎自动瞄准的距离钳制（`min(射程, 距离)`）打在错误格子上——`dodged=true` 永久落空，而 Bot 的移动权重从不主动接近，对局空转到回合上限。修复在 **bot 包决策层**：与对手距离超出自身攻击射程时，移动决策优先"接近"（目标格使距离落入射程内），而非随机方向。引擎的瞄准钳制是玩家侧正确行为（超距攻击落空是 spec 的走位博弈语义），**不改引擎**。

## 根因证据（2026-10-24 全栈里程碑验证，hard Bot 局）

```txt
06:26:18 结算 p2 move           ← Bot 随机移动（权重 5%）走远
06:27:33 结算 p2 attack dodged=true
06:28:48 结算 p2 attack dodged=true   ← 此后每回合永久 miss
06:31:18 结算 p2 attack dodged=true
终局: winner=p2 reason=rounds-cap（75:9）——Bot 打不死玩家，靠回合上限判胜
```

Bot（游侠，普攻射程 3 / 穿云箭射程 4）移动后与玩家距离 > 射程：自动瞄准钳到射程边界的格子 ≠ 玩家实际位置 → 永久闪避。16 号票的冷却同步修复无关（该局无技能冷却失同步）。

**Blocked by:** None（可立即开工；与 08 号票无文件冲突面——bot 包 vs hub/匹配）

**Status:** resolved

- [x] 超距时 Bot 移动决策优先接近：与对手距离 > 自身攻击射程时，移动目标选择"向对手方向、落入射程内"的合法格（仍遵守移动距离上限与路径规则）；射程内时维持现行权重决策
- [x] 回归测试复现原缺陷场景（种子/脚本构造"Bot 移动出射程"的局面）：修复前永久 miss、修复后恢复命中（事件流断言 dodged 由 true 转为命中）
- [x] 既有 bot 包测试全绿，含 16 号票高强度回归不退：`go test -race ./internal/bot/... -run TestSkillCooldown -count=100` 与 `go test -race ./internal/hub/... -run 'TestPve' -count=200`
- [x] 引擎零改动（修复只落在 bot 包 decide/aim 层；grep 断言 engine 包无本票改动）
