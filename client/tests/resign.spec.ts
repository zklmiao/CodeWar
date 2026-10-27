// 31 号票：投降认输——认输按钮可见性三态（可点/终局隐藏/教程不渲染）、
// confirm 防误触、ws 发送契约、store 终局处理与既有路径一致。
// 独立文件不扩写 match-store.spec.ts（30 号票并行在改该文件，避免同文件面）。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import RoundStatusBar from "../src/components/RoundStatusBar.vue";
import { useMatchStore } from "../src/stores/match";
import { MatchPhase } from "../src/api/proto/codewar/match/v1/match";
import { MatchSocket } from "../src/api/ws";
import { socket } from "../src/api/socket";

// ---- RoundStatusBar 挂载工具：直接注入 store 状态（组件对 mode/phase 的可见性
// 判定是组件内自治 computed，无需经 MatchView 全页挂载）----

function setRoundActive(mode: string) {
  const store = useMatchStore();
  store.phase = MatchPhase.MATCH_PHASE_ROUND_ACTIVE;
  store.matchConfig = { topic: "", cap: 3, strategy: "climb", mode, bot: "" };
  return store;
}

// ---- FakeWebSocket（与 ws-join.spec 同模式：记录发送、可控 open）----

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static OPEN = 1;
  onopen: (() => void) | null = null;
  onmessage: ((m: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  readyState = FakeWebSocket.OPEN;
  sent: string[] = [];

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
  }

  send(data: string): void {
    this.sent.push(data);
  }

  close(): void {}
}

beforeEach(() => setActivePinia(createPinia()));

afterEach(() => {
  vi.unstubAllGlobals();
  FakeWebSocket.instances = [];
});

describe("31 认输按钮可见性", () => {
  it("PvP 活跃回合：按钮在场可点", () => {
    setRoundActive("");
    const w = mount(RoundStatusBar);
    expect(w.find('[data-testid="resign-btn"]').exists()).toBe(true);
    w.unmount();
  });

  it("PvE 活跃回合：按钮在场（bot 胜收束，服务端已支持）", () => {
    setRoundActive("pve");
    const w = mount(RoundStatusBar);
    expect(w.find('[data-testid="resign-btn"]').exists()).toBe(true);
    w.unmount();
  });

  it("教程局：不渲染（教学管线不可中途认输）", () => {
    setRoundActive("tutorial");
    const w = mount(RoundStatusBar);
    expect(w.find('[data-testid="resign-btn"]').exists()).toBe(false);
    w.unmount();
  });

  it("终局后：不渲染（对局已结束无从认输）", () => {
    const store = setRoundActive("");
    store.phase = MatchPhase.MATCH_PHASE_ENDED;
    const w = mount(RoundStatusBar);
    expect(w.find('[data-testid="resign-btn"]').exists()).toBe(false);
    w.unmount();
  });
});

describe("31 confirm 防误触", () => {
  it("confirm 取消 → 不发送任何投降消息", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
    vi.stubGlobal("confirm", vi.fn(() => false));
    setRoundActive("");
    const w = mount(RoundStatusBar);
    socket.connect("p1", { onMessage: () => {}, onClose: () => {} });
    const ws = FakeWebSocket.instances[0];
    ws.onopen?.();
    expect(ws.sent).toHaveLength(1); // 仅 join

    await w.find('[data-testid="resign-btn"]').trigger("click");
    expect(ws.sent).toHaveLength(1); // 确认被拦截，无 resign 发出
    w.unmount();
  });

  it("confirm 确认 → 发送 resign", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
    vi.stubGlobal("confirm", vi.fn(() => true));
    setRoundActive("");
    const w = mount(RoundStatusBar);
    socket.connect("p1", { onMessage: () => {}, onClose: () => {} });
    const ws = FakeWebSocket.instances[0];
    ws.onopen?.();

    await w.find('[data-testid="resign-btn"]').trigger("click");
    expect(ws.sent).toHaveLength(2);
    expect(JSON.parse(ws.sent[1]).resign).toEqual({});
    w.unmount();
  });
});

describe("31 resign ws 契约", () => {
  it("resign 发送空请求体（身份由席位权威决定）；未连接时静默忽略", () => {
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
    const s = new MatchSocket();
    s.resign(); // 未连接：不抛错不发
    s.connect("p2", { onMessage: () => {}, onClose: () => {} });
    const ws = FakeWebSocket.instances[0];
    ws.onopen?.();
    expect(ws.sent).toHaveLength(1);
    s.resign();
    expect(ws.sent).toHaveLength(2);
    expect(JSON.parse(ws.sent[1]).resign).toEqual({});
  });
});

describe("31 resign 终局走既有路径", () => {
  it("matchEnded(reason=resign)：phase=ENDED、result 与既有终局同构、日志带原因", () => {
    const store = useMatchStore();
    store.viewerId = "p1";
    store.applyServerMessage(
      {
        event: {
          seq: 1,
          unixMs: "1000",
          matchEnded: { winner: "p2", reason: "resign" },
        },
      },
      "p1",
    );
    expect(store.isEnded).toBe(true);
    expect(store.result?.winner).toBe("p2");
    expect(store.result?.reason).toBe("resign");
    expect(store.log.at(-1)?.text).toContain("p2 获胜（resign）");
  });
});
