// 30 号票：QuestionPanel 100ms 心跳的资源生命周期回归。
// 心跳的外部行为 = 驱动锁定倒计时（挂载期间推进 now，锁定到期后恢复可提交）；
// 卸载必须清理定时器（对照 RoundStatusBar 同款写法）——漏清理即内存泄漏回归，
// 以 unmount 触发 clearInterval 为等价证据（fake timers 下句柄已被替换为 mock）。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import QuestionPanel from "../src/components/QuestionPanel.vue";
import { useMatchStore } from "../src/stores/match";

const PROBLEM = { id: "a1", statement: "攻击题", signature: "sig", starterCode: "c" };

beforeEach(() => setActivePinia(createPinia()));
afterEach(() => vi.useRealTimers());

function seedRoundActive() {
  const s = useMatchStore();
  s.applyServerMessage(
    {
      snapshot: {
        seq: 0,
        phase: "MATCH_PHASE_ROUND_ACTIVE",
        round: 1,
        deadlineUnixMs: "9999",
        players: [
          { id: "p1", hp: 100 },
          { id: "p2", hp: 100 },
        ],
        panel: { attack: PROBLEM, move: PROBLEM, defend: PROBLEM },
      },
    },
    "p1",
  );
  return s;
}

describe("QuestionPanel 心跳生命周期", () => {
  it("挂载期间心跳驱动锁定到期恢复提交，卸载时清理定时器", async () => {
    vi.useFakeTimers();
    const s = seedRoundActive();
    s.viewerLockUntil = Date.now() + 500; // 错误提交后的锁定预览（服务端下发值）
    const w = mount(QuestionPanel, { global: { stubs: { CodeEditor: true } } });
    const submit = () => w.find('[data-testid="submit-attack"]');
    expect(submit().attributes("disabled")).toBeDefined(); // 锁定期内禁提交

    // 心跳推进 600ms（6 个 tick）走过锁定期 → 恢复可提交
    await vi.advanceTimersByTimeAsync(600);
    await nextTick();
    expect(submit().attributes("disabled")).toBeUndefined();

    // 卸载：心跳句柄被清理（漏写 onBeforeUnmount 清理 → 此处红）
    const clearSpy = vi.spyOn(globalThis, "clearInterval");
    w.unmount();
    expect(clearSpy).toHaveBeenCalled();
  });
});
