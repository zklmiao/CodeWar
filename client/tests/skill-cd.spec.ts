// 20 号票 B：对局中技能冷却的事件流推导（纯客户端状态机 + 组件灰置态）。
// 契约：judged(技能,通过,自己) → 登记 cdUntilRound = round + CD（CD 查 /api/classes
// 同源职业表，与引擎 skillReadyFrom 同语义）；skillCdRounds getter 派生
// max(0, cdUntil - round)，roundStarted 更新回合时逐回合递减至 0 恢复；
// 接入/重连快照的 skill_cd_rounds 无条件重锚推导基线（权威初值，双源不漂移）。
// 负例：未通过不登记；actionResolved（含闪避）不动已登记冷却；对方释放不影响自己。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { describe, expect, it, beforeEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { Envelope } from "../src/api/proto/codewar/match/v1/match";
import { useMatchStore } from "../src/stores/match";
import { useClassesStore } from "../src/stores/classes";
import QuestionPanel from "../src/components/QuestionPanel.vue";

// 剑士+巨力斩（CD=4，设计 003 §4）——与引擎测试同源数值
const WARRIOR = {
  id: "warrior",
  name: "剑士",
  hp: 110,
  attackDamage: 26,
  attackShape: "melee",
  attackDepth: 0,
  attackRange: 1,
  moveMax: 2,
  color: "#b4552d",
  skills: [
    { id: "cleave", name: "巨力斩", kind: "damage", power: 55, heal: 0, cooldown: 4, shape: "burst8", depth: 0, rangeMax: 1 },
  ],
};

const PROBLEM = { id: "k1", statement: "s", signature: "sig", starterCode: "c" };

function envelope(seq: number, event: string, payload: Record<string, unknown>) {
  return { seq: String(seq), unixMs: "1000", [event]: payload };
}

function snapshotWithCd(cd: number, round = 1) {
  return {
    seq: 0,
    phase: "MATCH_PHASE_ROUND_ACTIVE",
    round,
    deadlineUnixMs: "9999",
    players: [
      { id: "p1", hp: 110, classId: "warrior", skillId: "cleave", skillCdRounds: cd },
      { id: "p2", hp: 90, classId: "mage", skillId: "firestorm", skillCdRounds: 0 },
    ],
    panel: {
      attack: PROBLEM,
      move: PROBLEM,
      defend: PROBLEM,
      skill: PROBLEM,
    },
  };
}

function seedMatch(s: ReturnType<typeof useMatchStore>, cd = 0) {
  s.applyServerMessage({ snapshot: snapshotWithCd(cd) }, "p1");
}

function judgedSkill(seq: number, player: string, passed: boolean) {
  return Envelope.fromJSON(
    envelope(seq, "submissionJudged", { player, passed, category: passed ? "ok" : "wrong-answer", action: "skill" }),
  );
}

describe("20 对局中技能冷却推导（store 状态机）", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    useClassesStore().classes = [WARRIOR];
  });

  it("释放（judged 技能通过）→ 灰置 CD 回合 → roundStarted 逐回合递减 → 恢复", () => {
    const s = useMatchStore();
    seedMatch(s);
    expect(s.skillCdRounds).toBe(0);
    // R1 释放：登记 cdUntil = 1+4 = 5，释放当回合即灰置（与引擎快照语义一致）
    s.applyEnvelope(judgedSkill(1, "p1", true));
    expect(s.skillCdRounds).toBe(4);
    // R2~R4 逐回合递减；R5（cdUntil）恢复可用
    for (let round = 2; round <= 5; round++) {
      s.applyEnvelope(Envelope.fromJSON(envelope(round, "roundStarted", { round, deadlineUnixMs: "999" })));
      expect(s.skillCdRounds).toBe(5 - round);
    }
  });

  it("重连快照为权威初值：无条件重锚推导基线（含覆盖本地遗留推导归零）", () => {
    const s = useMatchStore();
    seedMatch(s);
    s.applyEnvelope(judgedSkill(1, "p1", true));
    expect(s.skillCdRounds).toBe(4);
    // 断线重连：服务端快照（round 2、剩 3 回合）重锚 cdUntil = 2+3 = 5，事件续推
    s.applyServerMessage({ snapshot: snapshotWithCd(3, 2) }, "p1");
    expect(s.skillCdRounds).toBe(3);
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 3, deadlineUnixMs: "999" })));
    expect(s.skillCdRounds).toBe(2);
    // 服务端权威"已可用"（skill_cd_rounds=0）覆盖本地遗留推导：冷却归零且不再复发
    s.applyServerMessage({ snapshot: snapshotWithCd(0, 4) }, "p1");
    expect(s.skillCdRounds).toBe(0);
    s.applyEnvelope(Envelope.fromJSON(envelope(1, "roundStarted", { round: 5, deadlineUnixMs: "999" })));
    expect(s.skillCdRounds).toBe(0);
  });

  it("负例：judged 未通过不登记冷却", () => {
    const s = useMatchStore();
    seedMatch(s);
    s.applyEnvelope(judgedSkill(1, "p1", false));
    expect(s.skillCdRounds).toBe(0);
  });

  it("负例：actionResolved（含闪避）不影响已登记冷却（登记以 judged 为准，16 号票语义）", () => {
    const s = useMatchStore();
    seedMatch(s);
    s.applyEnvelope(judgedSkill(1, "p1", true));
    expect(s.skillCdRounds).toBe(4);
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "actionResolved", { attacker: "p1", action: "skill", dodged: true })));
    expect(s.skillCdRounds).toBe(4);
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "actionResolved", { attacker: "p1", action: "skill", damage: 55 })));
    expect(s.skillCdRounds).toBe(4);
  });

  it("负例：对方释放不影响我的冷却", () => {
    const s = useMatchStore();
    seedMatch(s);
    s.applyEnvelope(judgedSkill(1, "p2", true));
    expect(s.skillCdRounds).toBe(0);
  });
});

describe("20 技能槽灰置随事件推导刷新（组件挂载）", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    useClassesStore().classes = [WARRIOR];
  });

  async function mountPanel() {
    // CodeEditor 换桩：灰置态断言不依赖编辑器内部（避开 codemirror 的 jsdom 依赖）
    return mount(QuestionPanel, { global: { stubs: { CodeEditor: true } } });
  }

  it("就绪可点 → 释放后灰置倒计时 → roundStarted 递减 → 冷却结束恢复可点", async () => {
    const s = useMatchStore();
    seedMatch(s);
    const wrapper = await mountPanel();
    const slot = wrapper.find(".slot.skill");
    expect(slot.classes()).not.toContain("cooldown");
    expect(wrapper.find(".submit.skl").attributes("disabled")).toBeUndefined();

    // R1 释放 → 灰置 4 回合
    s.applyEnvelope(judgedSkill(1, "p1", true));
    await nextTick();
    expect(slot.classes()).toContain("cooldown");
    expect(slot.text()).toContain("冷却中（还剩 4 回合）");
    expect(wrapper.find(".submit.skl").attributes("disabled")).toBeDefined();

    // R2 roundStarted → 剩 3（面板虽为 null，槽头倒计时照常刷新）
    s.applyEnvelope(Envelope.fromJSON(envelope(2, "roundStarted", { round: 2, deadlineUnixMs: "999" })));
    await nextTick();
    expect(slot.text()).toContain("还剩 3 回合");
    expect(wrapper.find(".submit.skl").attributes("disabled")).toBeDefined();

    // 冷却结束（round ≥ cdUntil）→ 灰置解除、恢复可点
    s.applyEnvelope(Envelope.fromJSON(envelope(3, "roundStarted", { round: 5, deadlineUnixMs: "999" })));
    await nextTick();
    expect(slot.classes()).not.toContain("cooldown");
    expect(slot.text()).toContain("就绪");
    expect(wrapper.find(".submit.skl").attributes("disabled")).toBeUndefined();
    wrapper.unmount();
  });
});
