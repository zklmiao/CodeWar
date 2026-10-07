// 05 号票：对局配置——快照 match_config 进入客户端状态（对局内只读展示的
// 数据源，服务端锁定值而非 URL 参数）；沿用 match-store.spec 的风格：
// proto JSON 驱动纯状态机，断言外部行为。
import { describe, expect, it, beforeEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useMatchStore } from "../src/stores/match";

function baseSnapshot() {
  return {
    seq: 0,
    phase: "MATCH_PHASE_ROUND_ACTIVE",
    round: 1,
    deadlineUnixMs: "9999",
    matchConfig: { topic: "hash", cap: 5, strategy: "climb" },
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
    expect(s.matchConfig).toEqual({ topic: "hash", cap: 5, strategy: "climb" });
  });

  it("题型全部 = topic 空（默认锚点全部题型 / L3 / climb）", () => {
    const s = useMatchStore();
    const snap = baseSnapshot();
    snap.matchConfig = { topic: "", cap: 3, strategy: "climb" };
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.matchConfig).toEqual({ topic: "", cap: 3, strategy: "climb" });
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
    snap.matchConfig = { topic: "hash", cap: 5, strategy: "uniform" };
    s.applyServerMessage({ snapshot: snap }, "p1");
    expect(s.matchConfig?.strategy).toBe("uniform");
  });
});
