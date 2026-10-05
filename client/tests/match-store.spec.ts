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
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "move", solveMs: "0", moveTo: { row: 3, col: 0 } })));
    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "defend", solveMs: "5" })));
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

    s.applyEnvelope(Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "move", solveMs: "0", moveTo: { row: 3, col: 0 } })));
    expect(s.positions.p1).toEqual({ row: 3, col: 0 });

    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "attack", solveMs: "10", dodged: true })));
    expect(s.log.some((l) => l.text.includes("落空"))).toBe(true);

    s.applyEnvelope(Envelope.fromJSON(envelope(5, "actionResolved", { attacker: "p2", action: "attack", solveMs: "10", damage: 9, shielded: true })));
    expect(s.log.some((l) => l.text.includes("护盾折减"))).toBe(true);

    s.applyEnvelope(Envelope.fromJSON(envelope(6, "submissionJudged", { player: "p1", passed: true, category: "ok", solveMs: "0", action: "move" })));
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
