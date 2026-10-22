// 09 号票：join 契约的客户端侧回归——mode/bot 必须真的进入 WebSocket join
// 消息。只测 store/链接时，删掉 ws.ts 里的 mode/bot 仍然全绿，PvE 会被
// 服务端当成 PvP（或默认 normal），所以这里直接断言发出的 JSON。
// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { MatchSocket } from "../src/api/ws";

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

afterEach(() => {
  vi.unstubAllGlobals();
  FakeWebSocket.instances = [];
});

describe("09 join 消息", () => {
  it("PvE 的 mode/bot 与其余配置一起进入 join", () => {
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
    const s = new MatchSocket();
    s.connect(
      "p1",
      { onMessage: () => {}, onClose: () => {} },
      {
        mode: "pve",
        bot: "hard",
        topic: "hash",
        cap: "L5",
        strategy: "uniform",
        class: "ranger",
        skill: "pierce",
        token: "tok-1",
      },
    );
    const ws = FakeWebSocket.instances[0];
    ws.onopen?.();
    expect(ws.sent).toHaveLength(1);
    const msg = JSON.parse(ws.sent[0]);
    expect(msg.join).toMatchObject({
      player: "p1",
      token: "tok-1",
      class: "ranger",
      skill: "pierce",
      topic: "hash",
      cap: "L5",
      strategy: "uniform",
      mode: "pve",
      bot: "hard",
    });
  });

  it("PvP 缺省仍发空 mode/bot（服务端按 PvP 处理）", () => {
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
    const s = new MatchSocket();
    s.connect("p2", { onMessage: () => {}, onClose: () => {} }, {});
    const ws = FakeWebSocket.instances[0];
    ws.onopen?.();
    const msg = JSON.parse(ws.sent[0]);
    // proto3 默认值在 toJSON 中省略：PvP 缺省不携带 mode/bot，服务端按空处理。
    expect(msg.join.mode).toBeUndefined();
    expect(msg.join.bot).toBeUndefined();
  });
});
