// 09 号票：对局内“机器人（难度）”标识的渲染回归。只测 store 时，
// 删掉 HpBar 的 playerLabel 分支仍然全绿；这里挂载组件断言可见文案。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import HpBar from "../src/components/HpBar.vue";
import { useMatchStore } from "../src/stores/match";

beforeEach(() => setActivePinia(createPinia()));

describe("09 机器人标识", () => {
  it("PvE 的 p2 显示机器人（困难），p1 不显示", () => {
    const s = useMatchStore();
    s.hp = { p1: 100, p2: 110 };
    s.classes = {
      p1: { classId: "ranger", skillId: "pierce", skillCdRounds: 0, bot: false },
      p2: { classId: "warrior", skillId: "cleave", skillCdRounds: 0, bot: true },
    };
    s.matchConfig = { topic: "", cap: 3, strategy: "climb", mode: "pve", bot: "hard" };
    const w = mount(HpBar);
    expect(w.text()).toContain("P2 · 机器人（困难）");
    expect(w.text()).not.toContain("P1 · 机器人");
    w.unmount();
  });

  it("PvP 两行都只显示 P1/P2", () => {
    const s = useMatchStore();
    s.hp = { p1: 100, p2: 100 };
    s.classes = {
      p1: { classId: "ranger", skillId: "", skillCdRounds: 0, bot: false },
      p2: { classId: "mage", skillId: "", skillCdRounds: 0, bot: false },
    };
    s.matchConfig = { topic: "", cap: 3, strategy: "climb", mode: "", bot: "" };
    const w = mount(HpBar);
    expect(w.text()).not.toContain("机器人");
    w.unmount();
  });
});
