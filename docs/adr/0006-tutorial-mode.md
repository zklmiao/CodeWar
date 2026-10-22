# ADR-0006：新手教程作为 PvE 管线的一种模式

**状态**：已接受（10 号票，2026-10-22）。
**关联**：Spec 用户故事 5-6、10；ADR-0002（事件流同步）、ADR-0005（PvE Bot 边界）

## 背景

新手教程需要做到：每关只教一个机制、必须实际操作成功才过关、完成前锁定 1v1
匹配。若为教程另起一套对局实现，会出现三套规则（PvP/PvE/教程）漂移，且
"教程里学到的操作"无法证明与正式对局同源。

## 决策

**教程是 PvE 管线的一种模式（`mode=tutorial&level=N`），复用同一
`match.Engine`、同一 `Submit`/事件流/Snapshot 接入面。**

1. **模式路由归 hub**：首个 join 锁定 `mode`/`level`（与 05 的 topic/cap/strategy、
   09 的 pve/bot 同一锁定语义）。教程人类固定 p1，p2 是脚本 Bot；单连接即开赛，
   断线即作废（与 PvE 相同）。
2. **固定题包实现 `match.QuestionPort`**：`server/internal/tutorial` 内每关
   1-2 道手工题，含服务端内部 `AnswerKey`。教程不走 `question.Factory` 的随机
   出题——教学数值与题面必须可复现。引擎对题包来源零感知。
3. **脚本化 Bot 复用 `bot.Controller` 骨架**：新增 `bot.Script`（按回合的确定性
   行动/耗时/固定目标），仍经同一条 `act → engine.Submit` 路径提交。教程不引入
   第二套 Bot 实现，也不走难度分布采样。
4. **目标判定器与 Bot 同层**：`tutorial.Checker` 观察 hub 广播的事件流（与
   `bot.Controller.OnEvent` 同一 observer 通道），判定只发生在服务端。
5. **进度只由服务端写入**：目标达成 → hub 调 `Store.AdvanceTutorial`（migration
   000002：`users.tutorial_level` / `tutorial_completed_at`）→ 落库成功后才下发
   完成态并作废本场。客户端没有 advance/上报端点，只经 `GET /api/tutorial` 读
   进度、经 `ServerMessage.tutorial` 渲染目标清单。
6. **PvP 门禁**：已登录且未通关的用户在 hub 的 PvP join 校验处被拒绝
   （`Hub.pvpGate`，08 的在线匹配入口复用同一位置）。匿名热座 URL 直连是
   实验室旁路，显式放行并在代码注释标注。
7. **引擎零改动**：教程只经 `Submit`、事件流、`BindClass`、`Snapshot` 交互；
   `server/internal/tutorial` 的结构测试 grep 钉住 match 包无 tutorial 引用。

## 后果

- **(+)** 教程与 PvP/PvE 共用回合状态机、判题、伤害/闪避/护盾/冷却规则；
  教程里通过的操作即正式对局的操作。
- **(+)** 进度无法由客户端伪造：没有写入端点，判定与落库都在服务端。
- **(+)** 目标清单是纯渲染：服务端事件 → 判定器 → `TutorialState`，客户端不
  复制判定逻辑。
- **(−)** 教程题包是静态内容，新增关卡需要改代码而非配置（教学关数量少，
  可接受；通用题库模板归 12 号票）。
- **(−)** 完成态必须等 DB 写成功才下发，DB 故障时玩家会停在"目标已点亮但
  未完成"；进度不丢，下一次事件会重试落库。

## 备选方案（被否原因）

- **独立 TutorialEngine**：规则双写、无法证明与正式对局同源。否决。
- **客户端本地判定过关 + 上报**：可自报造假，违背服务端权威。否决。
- **教程复用随机出题工厂**：教学数值/题面不可复现，且会引入未讲授的题型。否决。
