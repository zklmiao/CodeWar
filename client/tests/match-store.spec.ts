// store 状态机测试：用与真实 WS 消息同构的 protojson 形态驱动（spec「测试决策」：只测外部行为）。
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import { useMatchStore } from "../src/stores/match";
import { Envelope } from "../src/api/proto/codewar/match/v1/match";

// protojson 形态：int64 字段为字符串
function envelope(seq: number, key: string, body: Record<string, unknown>) {
  return { seq, unixMs: String(Date.now()), [key]: body };
}

beforeEach(() => setActivePinia(createPinia()));

describe("对局状态机（事件流驱动）", () => {
  it("快照建立初始视图（含查看者私有锁）", () => {
    const s = useMatchStore();
    s.applyServerMessage(
      {
        snapshot: {
          seq: 3,
          phase: "MATCH_PHASE_ROUND_ACTIVE",
          round: 1,
          deadlineUnixMs: "1767225675000",
          players: [
            { id: "p1", hp: 100 },
            { id: "p2", hp: 97 },
          ],
          problem: {
            id: "palindrome-L2",
            statement: "判断回文",
            signature: "def is_palindrome(s: str) -> bool:",
            starterCode: "def is_palindrome(s: str) -> bool:\n",
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
    expect(s.problem?.id).toBe("palindrome-L2");
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
    expect(s.result).toEqual({ winner: "p1", reason: "ko" });
  });

  it("面板事件装载题目，回合切换时清空", () => {
    const s = useMatchStore();
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 2, deadlineUnixMs: "999" })));
    expect(s.problem).toBeNull(); // 新回合先清面板
    s.applyEnvelope(
      Envelope.fromJSON(envelope(2, "panelReady", { problem: { id: "two-sum-L3", statement: "s", signature: "sig", starterCode: "c" } })),
    );
    expect(s.problem?.id).toBe("two-sum-L3");
  });

  it("错误提交触发查看者锁定预览", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: { seq: 0, phase: "MATCH_PHASE_ROUND_ACTIVE", round: 1, players: [] } }, "p1");
    s.applyEnvelope(
      Envelope.fromJSON(envelope(1, "submissionJudged", { player: "p1", passed: false, category: "wrong-answer", selfDamageApplied: true })),
    );
    expect(s.viewerLockUntil).toBeGreaterThan(Date.now());
    expect(s.viewerSolved).toBe(false);
  });

  it("查看者解出后本回合标记已解", () => {
    const s = useMatchStore();
    s.viewerId = "p1";
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "submissionJudged", { player: "p1", passed: true, category: "ok", solveMs: "12300" })));
    expect(s.viewerSolved).toBe(true);
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
    expect(s.result).toEqual({ winner: "", reason: "rounds-cap" });
  });
});
