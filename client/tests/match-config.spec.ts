// 05 号票：对局配置——快照 match_config 进入客户端状态（对局内只读展示的
// 数据源，服务端锁定值而非 URL 参数）；沿用 match-store.spec 的风格：
// proto JSON 驱动纯状态机，断言外部行为。
import { describe, expect, it, beforeEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useMatchStore } from "../src/stores/match";

interface SnapshotFixture {
  seq: number;
  phase: string;
  round: number;
  deadlineUnixMs: string;
  matchConfig: { topic: string; cap: number; strategy: string; mode: string; bot: string };
  players: Array<{ id: string; hp: number; bot?: boolean }>;
  panel: {
    attack: { id: string; statement: string; signature: string; starterCode: string };
    move: { id: string; statement: string; signature: string; starterCode: string };
    defend: { id: string; statement: string; signature: string; starterCode: string };
  };
}

function baseSnapshot(): SnapshotFixture {
  return {
    seq: 0,
    phase: "MATCH_PHASE_ROUND_ACTIVE",
    round: 1,
    deadlineUnixMs: "9999",
    matchConfig: { topic: "hash", cap: 5, strategy: "climb", mode: "", bot: "" },
    players: [
      { id: "p1", hp: 100 },
      { id: "p2", hp: 100 },
    ],
    panel: {
      attack: { id: "a1", statement: "s", signature: "sig", starterCode: "c" },
      move: { id: "m1", statement: "s", signature: "sig", starterCode: "c" },
      defend: { id: "d1", statement: "s", signature: "sig", starterCode: "c" },
    },
  };
}

describe("05 对局配置快照", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("快照携带题型过滤/难度上限/抽题策略", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: baseSnapshot() }, "p1");
    expect(s.matchConfig).toEqual({ topic: "hash", cap: 5, strategy: "climb", mode: "", bot: "" });
  });

  it("题型全部 = topic 空（默认锚点全部题型 / L3 / climb）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    snap.matchConfig = { topic: "", cap: 3, strategy: "climb", mode: "", bot: "" };
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.matchConfig).toEqual({ topic: "", cap: 3, strategy: "climb", mode: "", bot: "" });
  });

  it("旧快照无配置字段时不崩且为 null（01~04 链路兼容）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    delete (snap as Partial<typeof snap>).matchConfig;
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.matchConfig).toBeNull();
    expect(s.panel?.attack?.id).toBe("a1");
  });

  it("重连快照刷新配置状态（同一锁定值幂等）", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: baseSnapshot() }, "p1");
    const snap = baseSnapshot();
    snap.matchConfig = { topic: "hash", cap: 5, strategy: "uniform", mode: "", bot: "" };
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.matchConfig?.strategy).toBe("uniform");
  });
});

// 09 号票：PvE 快照——PlayerView.bot 标记 + 服务端锁定的模式/难度进入状态。
describe("09 PvE 人机练习", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("p2 的 bot 标记与锁定难度进入客户端状态（HpBar 显示数据源）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    snap.matchConfig = { topic: "", cap: 3, strategy: "climb", mode: "pve", bot: "hard" };
    snap.players = [
      { id: "p1", hp: 110 },
      { id: "p2", hp: 95, bot: true },
    ];
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.classes.p1.bot).toBe(false);
    expect(s.classes.p2.bot).toBe(true);
    expect(s.matchConfig).toEqual({ topic: "", cap: 3, strategy: "climb", mode: "pve", bot: "hard" });
  });

  it("PvP/旧快照无 bot 字段 → false（向后兼容）", () => {
    const s = useMatchStore();
    s.applyServerMessage({ snapshot: baseSnapshot() }, "p1");
    expect(s.classes.p1.bot).toBe(false);
    expect(s.classes.p2.bot).toBe(false);
  });

  it("bot 难度展示名：未知值回退普通", async () => {
    const { botLabel } = await import("../src/topics");
    expect(botLabel("easy")).toBe("简单");
    expect(botLabel("hard")).toBe("困难");
    expect(botLabel("")).toBe("普通");
    expect(botLabel("bogus")).toBe("普通");
  });
});
