// 30 号票：EventLog 按 kind 着色的渲染回归。kind→class 是组件的外部契约
// （伤害红/治疗绿/闪避黄/系统默认次级色）；只断言 class 挂载，不窥探 CSS 内部。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import EventLog from "../src/components/EventLog.vue";
import { useMatchStore } from "../src/stores/match";
import type { LogEntry } from "../src/stores/match";

beforeEach(() => setActivePinia(createPinia()));

function seed(entries: LogEntry[]) {
  useMatchStore().log = entries;
}

describe("EventLog kind 着色", () => {
  it("damage/heal/dodge 行挂对应 kind class，system 行不挂任何战斗色 class", () => {
    seed([
      { seq: 1, text: "—— 第 1 回合 ——", kind: "system" },
      { seq: 2, text: "⚔ p1 出手，造成 22 伤害", kind: "damage" },
      { seq: 3, text: "✚ p1 恢复 20 HP（剩余 120）", kind: "heal" },
      { seq: 4, text: "💨 p2 的攻击落空——目标已移出相邻格", kind: "dodge" },
    ]);
    const w = mount(EventLog);
    // 渲染为倒序（新行在上）：seq 4 / 3 / 2 / 1
    const lines = w.findAll(".line");
    expect(lines).toHaveLength(4);
    expect(lines[0].classes()).toContain("kind-dodge");
    expect(lines[1].classes()).toContain("kind-heal");
    expect(lines[2].classes()).toContain("kind-damage");
    const systemRow = lines[3];
    expect(systemRow.text()).toContain("第 1 回合");
    for (const combat of ["kind-damage", "kind-heal", "kind-dodge"]) {
      expect(systemRow.classes()).not.toContain(combat);
    }
  });

  it("文案与 seq 原样渲染（着色不改动内容）", () => {
    seed([{ seq: 9, text: "💢 p2 受到 22 伤害（剩余 78）", kind: "damage" }]);
    const w = mount(EventLog);
    const line = w.find(".line");
    expect(line.text()).toContain("9");
    expect(line.text()).toContain("p2 受到 22 伤害（剩余 78）");
  });
});
