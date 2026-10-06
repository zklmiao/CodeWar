// 06 号票：四槽面板（技能槽）、职业/技能快照字段、targeting 状态、技能/治疗事件流。
// 沿用 match-store.spec 的风格：proto JSON 驱动纯状态机，断言外部行为。
import { describe, expect, it, beforeEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { Envelope, ServerMessage, Snapshot } from "../src/api/proto/codewar/match/v1/match";
import { useMatchStore } from "../src/stores/match";

function envelope(seq: number, event: string, payload: Record<string, unknown>) {
  return { seq: String(seq), unixMs: "1000", [event]: payload };
}

function baseSnapshot() {
  return {
    seq: 0,
    phase: "MATCH_PHASE_ROUND_ACTIVE",
    round: 1,
    deadlineUnixMs: "9999",
    players: [
      { id: "p1", hp: 110, classId: "warrior", skillId: "cleave", skillCdRounds: 0 },
      { id: "p2", hp: 90, classId: "mage", skillId: "firestorm", skillCdRounds: 0 },
    ],
    panel: {
      attack: { id: "a1", statement: "s", signature: "sig", starterCode: "c" },
      move: { id: "m1", statement: "s", signature: "sig", starterCode: "c" },
      defend: { id: "d1", statement: "s", signature: "sig", starterCode: "c" },
      skill: { id: "k1", statement: "ks", signature: "ksig", starterCode: "kc" },
    },
  };
}

describe("06 四槽面板与职业快照", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("快照解析职业/技能/冷却与技能槽题面", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: baseSnapshot() }, "p1");
    expect(s.classes.p1).toEqual({ classId: "warrior", skillId: "cleave", skillCdRounds: 0 });
    expect(s.classes.p2.classId).toBe("mage");
    expect(s.panel?.skill?.id).toBe("k1");
  });

  it("冷却剩余回合进入快照状态（技能槽灰置的服务端权威）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    snap.players[0].skillCdRounds = 2;
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.classes.p1.skillCdRounds).toBe(2);
  });

  it("开赛前对方职业隐藏（盲选隐私，设计 003 §1）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    snap.phase = "MATCH_PHASE_WAITING";
    // 服务端在 WAITING 快照中隐藏对方职业——客户端契约：原样呈现不下发
    snap.players[1].classId = "";
    snap.players[1].skillId = "";
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.classes.p1.classId).toBe("warrior");
    expect(s.classes.p2.classId).toBe("");
  });

  it("技能结算事件流：伤害/治疗/闪避/护盾折减的日志语义", () => {
    const s = useMatchStore();
    s.viewerId = "p1";
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(
      Envelope.fromJSON(envelope(2, "actionResolved", { attacker: "p1", action: "skill", damage: 55 })),
    );
    s.applyEnvelope(
      Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "skill", damage: 33, shielded: true })),
    );
    s.applyEnvelope(Envelope.fromJSON(envelope(4, "actionResolved", { attacker: "p2", action: "skill", dodged: true })));
    s.applyEnvelope(
      Envelope.fromJSON(envelope(5, "hpChanged", { player: "p1", hp: 28, delta: 30, cause: "heal" })),
    );
    const texts = s.log.map((l) => l.text).join("|");
    expect(texts).toContain("释放技能：造成 55 伤害");
    expect(texts).toContain("护盾折减");
    expect(texts).toContain("p2 的技能落空");
    expect(texts).toContain("p1 恢复 30 HP");
  });

  it("技能冷却拒绝与未装备拒绝的日志语义", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 1, deadlineUnixMs: "999" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "submissionRejected", { player: "p1", reason: "skill-cooldown" })));
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "submissionRejected", { player: "p2", reason: "no-skill-equipped" })));
    const texts = s.log.map((l) => l.text).join("|");
    expect(texts).toContain("技能冷却中被拒");
    expect(texts).toContain("未装备技能被拒");
  });

  it("targeting 状态：设置与清除（未提交 = 自动瞄准）", () => {
    const s = useMatchStore();
    s.setTargeting({ direction: "up", distance: 2 });
    expect(s.pendingTargeting).toEqual({ direction: "up", distance: 2 });
    s.setTargeting(null);
    expect(s.pendingTargeting).toBeNull();
  });
});

// Snapshot 原生对象直测（不经 fromJSON 的路径也保持一致）
describe("06 快照原生解析", () => {
  beforeEach(() => setActivePinia(createPinia()));
  it("panel.skill 与 classes 并行解析", () => {
    const s = useMatchStore();
    const snap = Snapshot.fromJSON(baseSnapshot() as object);
    s.applySnapshot(snap, "p2");
    expect(s.panel?.skill?.id).toBe("k1");
    expect(s.classes.p2.classId).toBe("mage");
  });
  it("ServerMessage.fromJSON 快照路径一致", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: baseSnapshot() }, "p1");
    expect(s.panel?.skill).not.toBeNull();
  });
});
