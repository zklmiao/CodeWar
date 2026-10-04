// 终局面板 ELO 区块组件测试：有 elo 显示双方 before→after，匿名（空）不显示。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import EndOverlay from "../src/components/EndOverlay.vue";
import { useMatchStore } from "../src/stores/match";
import { MatchPhase } from "../src/api/proto/codewar/match/v1/match";

beforeEach(() => setActivePinia(createPinia()));

function endMatch(winner: string, elo: { player: string; userId: string; eloBefore: number; eloAfter: number }[]) {
  const s = useMatchStore();
  s.viewerId = "p1";
  s.phase = MatchPhase.MATCH_PHASE_ENDED;
  s.result = { winner, reason: "ko", elo };
  return s;
}

describe("终局面板 ELO 区块", () => {
  it("已结算显示双方 before→after 与增量", () => {
    endMatch("p1", [
      { player: "p1", userId: "u1", eloBefore: 1000, eloAfter: 1016 },
      { player: "p2", userId: "u2", eloBefore: 1000, eloAfter: 984 },
    ]);
    const w = mount(EndOverlay);
    const text = w.text();
    expect(text).toContain("1000 → 1016");
    expect(text).toContain("+16");
    expect(text).toContain("984");
  });

  it("匿名对局不显示 ELO 区块", () => {
    endMatch("p1", []);
    const w = mount(EndOverlay);
    expect(w.find(".elo").exists()).toBe(false);
    // 终局本身仍展示
    expect(w.text()).toContain("你赢了");
  });

  it("平局 +0 照样显示（与匿名可区分）", () => {
    endMatch("", [
      { player: "p1", userId: "u1", eloBefore: 1000, eloAfter: 1000 },
      { player: "p2", userId: "u2", eloBefore: 1000, eloAfter: 1000 },
    ]);
    const w = mount(EndOverlay);
    expect(w.find(".elo").exists()).toBe(true);
    expect(w.text()).toContain("平局");
  });
});
