// store 状态机测试：用与真实 WS 消息同构的 protojson 形态驱动（spec「测试决策」：只测外部行为）。
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import { useMatchStore } from "../src/stores/match";
import { Envelope } from "../src/api/proto/codewar/match/v1/match";

// protojson 形态：int64 字段为字符串
function envelope(seq: number, key: string, body: Record<string, unknown>) {
  return { seq, unixMs: String(Date.now()), [key]: body };
}

function panelBody() {
  return {
    attack: { id: "attack@seed:complete", statement: "攻击题", signature: "sig", starterCode: "c" },
    p1: {
      move: { id: "move@seed:p1", statement: "P1 移动题", signature: "sig", starterCode: "c" },
      defend: { id: "defend@seed:p1", statement: "P1 防御题", signature: "sig", starterCode: "c" },
    },
    p2: {
      move: { id: "move@seed:p2", statement: "P2 移动题", signature: "sig", starterCode: "c" },
      defend: { id: "defend@seed:p2", statement: "P2 防御题", signature: "sig", starterCode: "c" },
    },
  };
}

beforeEach(() => setActivePinia(createPinia()));

describe("对局状态机（事件流驱动）", () => {
  it("快照建立初始视图（位置/护盾/查看者过滤的三题面板/私有锁）", () => {
    const s = useMatchStore();
    s.applyServerMessage(
      {
        snapshot: {
          seq: 3,
          phase: "MATCH_PHASE_ROUND_ACTIVE",
          round: 1,
          deadlineUnixMs: "1767225675000",
          players: [
            { id: "p1", hp: 100, pos: { row: 3, col: 2 }, shielded: false },
            { id: "p2", hp: 97, pos: { row: 2, col: 4 }, shielded: true },
          ],
          panel: {
            attack: { id: "attack@seed", statement: "攻击题", signature: "sig", starterCode: "c" },
            move: { id: "move@seed:p1", statement: "P1 移动题", signature: "sig", starterCode: "c" },
            defend: { id: "defend@seed:p1", statement: "P1 防御题", signature: "sig", starterCode: "c" },
          },
          viewerLockUntilUnixMs: "0",
        },
      },
      "p1",
    );
    expect(s.viewerId).toBe("p1");
    expect(s.isRoundActive).toBe(true);
    expect(s.seq).toBe(3);
    expect(s.hp.p2).toBe(97);
    expect(s.panel?.attack?.id).toBe("attack@seed");
    expect(s.panel?.move?.id).toBe("move@seed:p1"); // 快照私有题按查看者过滤
    expect(s.positions.p2).toEqual({ row: 2, col: 4 });
    expect(s.shield.p2).toBe(true);
    expect(s.viewerLockUntil).toBe(0);
  });

  it("事件流驱动回合推进与掉血", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    expect(s.isRoundActive).toBe(true);
    expect(s.round).toBe(1);

    s.applyEnvelope(Envelope.fromJSON(envelope(2, "hpChanged", { player: "p2", hp: 78, delta: -22, cause: "attack" })));
    expect(s.hp.p2).toBe(78);

    s.applyEnvelope(Envelope.fromJSON(envelope(3, "matchEnded", { winner: "p1", reason: "ko" })));
    expect(s.isEnded).toBe(true);
    expect(s.result).toEqual({ winner: "p1", reason: "ko", elo: [] });
  });

  it("三槽面板装载：攻击镜像共享，私有题按 viewerId 取本方", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: { seq: 0, phase: "MATCH_PHASE_ROUND_ACTIVE", round: 1, players: [] } }, "p1");
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "panelReady", panelBody())));
    expect(s.panel?.attack?.id).toBe("attack@seed:complete");
    expect(s.panel?.move?.id).toBe("move@seed:p1");
    expect(s.panel?.defend?.id).toBe("defend@seed:p1");

    const s2 = useMatchStore();
    s2.applyServerMessage({ snapshot: { seq: 0, phase: "MATCH_PHASE_ROUND_ACTIVE", round: 1, players: [] } }, "p2");
    s2.applyEnvelope(Envelope.fromJSON(envelope(1, "panelReady", panelBody())));
    expect(s2.panel?.move?.id).toBe("move@seed:p2");
    expect(s2.panel?.defend?.id).toBe("defend@seed:p2");
  });

  it("新回合清空面板/护盾/已行动标记，保留位置", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "panelReady", panelBody())));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "move", solveMs: 0, moveTo: { row: 3, col: 0 } })));
    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "defend", solveMs: 5 })));
    expect(s.positions.p1).toEqual({ row: 3, col: 0 });
    expect(s.shield.p2).toBe(true);

    s.applyEnvelope(Envelope.fromJSON(envelope(5, "roundStarted", { round: 2, deadlineUnixMs: "999" })));
    expect(s.panel).toBeNull(); // 新回合先清面板
    expect(s.shield).toEqual({}); // 护盾不跨回合
    expect(s.viewerActed).toBe(false);
    expect(s.positions.p1).toEqual({ row: 3, col: 0 }); // 位置跨回合持续
  });

  it("行动结算：移动改位置、防御挂盾、攻击渲染闪避/护盾折减", () => {
    const s = useMatchStore();
    s.viewerId = "p1";
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "panelReady", panelBody())));

    s.applyEnvelope(Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "move", solveMs: 0, moveTo: { row: 3, col: 0 } })));
    expect(s.positions.p1).toEqual({ row: 3, col: 0 });

    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "attack", solveMs: 10, dodged: true })));
    expect(s.log.some((l) => l.text.includes("落空"))).toBe(true);

    s.applyEnvelope(Envelope.fromJSON(envelope(5, "actionResolved", { attacker: "p2", action: "attack", solveMs: 10, damage: 9, shielded: true })));
    expect(s.log.some((l) => l.text.includes("护盾折减"))).toBe(true);

    s.applyEnvelope(Envelope.fromJSON(envelope(6, "submissionJudged", { player: "p1", passed: true, category: "ok", solveMs: 0, action: "move" })));
    expect(s.viewerActed).toBe(true);
    expect(s.actedKind).toBe("move");
  });

  it("移动参数被拒：日志提示可重交", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "submissionRejected", { player: "p1", reason: "path-blocked" })));
    expect(s.log.some((l) => l.text.includes("path-blocked") && l.text.includes("重交"))).toBe(true);
  });

  it("错误提交触发查看者锁定预览", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: { seq: 0, phase: "MATCH_PHASE_ROUND_ACTIVE", round: 1, players: [] } }, "p1");
    // 锁定截止由服务端在 judged 事件中下发（与快照同字段同源），客户端不再本地推算
    const lockUntil = String(Date.now() + 5000);
    s.applyEnvelope(
      Envelope.fromJSON(
        envelope(1, "submissionJudged", {
          player: "p1",
          passed: false,
          category: "wrong-answer",
          selfDamageApplied: true,
          lockUntilUnixMs: lockUntil,
        }),
      ),
    );
    expect(s.viewerLockUntil).toBe(Number(lockUntil));
    expect(s.viewerActed).toBe(false); // 错误提交不消耗本回合行动
  });

  it("序号断裂置 gapDetected，断裂事件不应用（ADR-0002）", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "roundStarted", { round: 2, deadlineUnixMs: "999" })));
    expect(s.gapDetected).toBe(true);
    expect(s.round).toBe(1); // 断裂的那条没有生效
    expect(s.seq).toBe(1);
  });

  it("平局结果渲染（winner 空）", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "matchEnded", { winner: "", reason: "rounds-cap" })));
    expect(s.isEnded).toBe(true);
    expect(s.result).toEqual({ winner: "", reason: "rounds-cap", elo: [] });
  });

  it("终局 ELO 随 MatchEnded 下发（已结算，含平局 +0）", () => {
    const s = useMatchStore();
    s.applyEnvelope(
      Envelope.fromJSON(
        envelope(1, "matchEnded", {
          winner: "p1",
          reason: "ko",
          elo: [
            { player: "p1", userId: "u1", eloBefore: 1000, eloAfter: 1016 },
            { player: "p2", userId: "u2", eloBefore: 1000, eloAfter: 984 },
          ],
        }),
      ),
    );
    expect(s.result?.elo).toEqual([
      { player: "p1", userId: "u1", eloBefore: 1000, eloAfter: 1016 },
      { player: "p2", userId: "u2", eloBefore: 1000, eloAfter: 984 },
    ]);
  });

  it("快照 result 不带 ELO（重连快照路径归 08 号票）", () => {
    const s = useMatchStore();
    s.applyServerMessage(
      { snapshot: { seq: 0, phase: "MATCH_PHASE_ENDED", round: 10, players: [], result: { winner: "p1", reason: "ko" } } },
      "p1",
    );
    expect(s.result).toEqual({ winner: "p1", reason: "ko", elo: [] });
  });
});

// ===== 30 号票：终局统计（事件流累计）与日志 kind 分类 =====
// 事件注入与引擎发射模式同构：命中 = actionResolved{damage} + hpChanged(负 delta)；
// 闪避只有 actionResolved{dodged}（引擎验收 3：不产生 hpChanged）；
// 自损 = judged{selfDamageApplied} + hpChanged{cause:"self-damage"} 成对；
// 汲取类技能拆两条 actionResolved（伤害一条、治疗一条）。
describe("30 对局统计与日志分类", () => {
  it("跨回合事件流累计：伤害记出手方、闪避记闪避方、自损不计战斗伤害", () => {
    const s = useMatchStore();
    // R1：p1 普攻命中 22
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "actionResolved", { attacker: "p1", action: "attack", solveMs: 10, damage: 22 })));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "hpChanged", { player: "p2", hp: 78, delta: -22, cause: "attack" })));
    // R2：p2 攻击被 p1 走位闪避；p1 技能命中 25 并自疗 20；p1 提交错误自损 3
    s.applyEnvelope(Envelope.fromJSON(envelope(4, "roundStarted", { round: 2, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(5, "actionResolved", { attacker: "p2", action: "attack", solveMs: 8, dodged: true })));
    s.applyEnvelope(Envelope.fromJSON(envelope(6, "actionResolved", { attacker: "p1", action: "skill", solveMs: 12, damage: 25, shielded: true })));
    s.applyEnvelope(Envelope.fromJSON(envelope(7, "hpChanged", { player: "p2", hp: 53, delta: -25, cause: "skill" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(8, "actionResolved", { attacker: "p1", action: "skill", solveMs: 12, heal: 20 })));
    s.applyEnvelope(Envelope.fromJSON(envelope(9, "hpChanged", { player: "p1", hp: 120, delta: 20, cause: "heal" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(10, "submissionJudged", { player: "p1", passed: false, category: "wrong-answer", selfDamageApplied: true })));
    s.applyEnvelope(Envelope.fromJSON(envelope(11, "hpChanged", { player: "p1", hp: 117, delta: -3, cause: "self-damage" })));

    expect(s.stats.rounds).toBe(2);
    expect(s.stats.damageDealt).toEqual({ p1: 47, p2: 0 }); // 自损 3 不计入战斗伤害
    expect(s.stats.dodges).toEqual({ p1: 1, p2: 0 }); // p2 出手落空 → p1 闪避 +1
  });

  it("双方都行动：护盾折减按落地伤害计；平局终局统计保留", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "actionResolved", { attacker: "p1", action: "attack", solveMs: 5, damage: 22 })));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "hpChanged", { player: "p2", hp: 78, delta: -22, cause: "attack" })));
    // p2 攻击命中但 p1 本回合挂盾：落地 9（22×0.4），统计记 9 而非基础 22
    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "attack", solveMs: 9, damage: 9, shielded: true })));
    s.applyEnvelope(Envelope.fromJSON(envelope(5, "hpChanged", { player: "p1", hp: 91, delta: -9, cause: "attack" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(6, "matchEnded", { winner: "", reason: "rounds-cap" })));

    expect(s.isEnded).toBe(true);
    expect(s.stats.rounds).toBe(1);
    expect(s.stats.damageDealt).toEqual({ p1: 22, p2: 9 });
    expect(s.stats.dodges).toEqual({ p1: 0, p2: 0 });
  });

  it("边界：双方未行动即终局 → 全零统计；治疗系技能不产伤害", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    // p1 治疗系技能（满血时治疗量截 0：ActionResolved 仅 heal:0）+ 恢复事件不发生
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "actionResolved", { attacker: "p1", action: "skill", solveMs: 3, heal: 0 })));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "matchEnded", { winner: "", reason: "rounds-cap" })));

    expect(s.stats).toEqual({ rounds: 1, damageDealt: { p1: 0, p2: 0 }, dodges: { p1: 0, p2: 0 } });
  });

  it("LogEntry.kind 按事件类型打标：战斗三色归 HP 效果与落空行，提交流水归 system", () => {
    const s = useMatchStore();
    s.viewerId = "p1";
    const feed: [number, string, Record<string, unknown>][] = [
      [1, "roundStarted", { round: 1, deadlineUnixMs: "999" }],
      // 命中：actionResolved 行 + hpChanged 行都是 damage
      [2, "actionResolved", { attacker: "p1", action: "attack", solveMs: 10, damage: 22 }],
      [3, "hpChanged", { player: "p2", hp: 78, delta: -22, cause: "attack" }],
      // 闪避：攻击/技能落空行都是 dodge
      [4, "actionResolved", { attacker: "p2", action: "attack", solveMs: 8, dodged: true }],
      [5, "actionResolved", { attacker: "p2", action: "skill", solveMs: 8, dodged: true }],
      // 治疗：hpChanged 行 + 治疗系技能行动行都是 heal
      [6, "actionResolved", { attacker: "p1", action: "skill", solveMs: 6, heal: 20 }],
      [7, "hpChanged", { player: "p1", hp: 120, delta: 20, cause: "heal" }],
      // 自损：hpChanged 行是 damage（judged 的提交流水行归 system）
      [8, "hpChanged", { player: "p1", hp: 117, delta: -3, cause: "self-damage" }],
      // 技能伤害行 damage
      [9, "actionResolved", { attacker: "p2", action: "skill", solveMs: 4, damage: 25, shielded: true }],
      // 提交生命周期：通过/移动/防御/被拒/终局都归 system
      [10, "submissionJudged", { player: "p1", passed: true, category: "ok", solveMs: 100, action: "attack" }],
      [11, "actionResolved", { attacker: "p1", action: "move", solveMs: 0, moveTo: { row: 3, col: 1 } }],
      [12, "actionResolved", { attacker: "p2", action: "defend", solveMs: 7 }],
      [13, "submissionRejected", { player: "p1", reason: "path-blocked" }],
      [14, "matchEnded", { winner: "p1", reason: "ko" }],
    ];
    for (const [seq, key, body] of feed) s.applyEnvelope(Envelope.fromJSON(envelope(seq, key, body)));

    const kindOf = (sub: string) => s.log.find((l) => l.text.includes(sub))?.kind;
    expect(kindOf("受到 22 伤害")).toBe("damage");
    expect(kindOf("自损 3")).toBe("damage");
    expect(kindOf("p1 出手，造成 22 伤害")).toBe("damage");
    expect(kindOf("p2 释放技能：造成 25 伤害")).toBe("damage");
    expect(kindOf("攻击落空")).toBe("dodge");
    expect(kindOf("技能落空")).toBe("dodge");
    expect(kindOf("恢复 20 HP（剩余")).toBe("heal"); // hpChanged 行
    expect(kindOf("释放技能：恢复 20 HP")).toBe("heal"); // 行动行
    expect(kindOf("解出")).toBe("system"); // judged 通过行（题名即 action 字段原文）
    expect(kindOf("移动到")).toBe("system");
    expect(kindOf("挂起护盾")).toBe("system");
    expect(kindOf("被拒")).toBe("system");
    expect(kindOf("第 1 回合")).toBe("system");
    expect(kindOf("获胜")).toBe("system");
  });
});
