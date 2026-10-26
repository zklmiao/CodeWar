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

// ===== 30 号票：统计区块（回合数/双方伤害/双方闪避；匿名局同样显示） =====
describe("终局面板统计区块", () => {
  it("显示事件流累计的回合数、双方总伤害与闪避次数", () => {
    const s = endMatch("p1", [
      { player: "p1", userId: "u1", eloBefore: 1000, eloAfter: 1016 },
    ]);
    s.stats = { rounds: 3, damageDealt: { p1: 47, p2: 9 }, dodges: { p1: 1, p2: 2 } };
    const w = mount(EndOverlay);
    const rows = w.findAll('[data-testid="end-stats"] .statrow');
    expect(rows).toHaveLength(3);
    expect(rows[0].text()).toContain("回合数");
    expect(rows[0].text()).toContain("3");
    expect(rows[1].text()).toContain("总伤害");
    expect(rows[1].text()).toContain("P1 47 · P2 9");
    expect(rows[2].text()).toContain("闪避");
    expect(rows[2].text()).toContain("P1 1 · P2 2");
  });

  it("匿名局（elo 空）统计照样显示，ELO 区块仍隐藏", () => {
    const s = endMatch("p1", []);
    s.stats = { rounds: 10, damageDealt: { p1: 0, p2: 66 }, dodges: { p1: 0, p2: 0 } };
    const w = mount(EndOverlay);
    expect(w.find(".elo").exists()).toBe(false);
    const rows = w.findAll('[data-testid="end-stats"] .statrow');
    expect(rows).toHaveLength(3);
    expect(rows[1].text()).toContain("P1 0 · P2 66");
  });
});
