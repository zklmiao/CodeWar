// WebSocket 封装：连接 → join → 分发服务端消息（快照/事件）。
import {
  ClientMessage,
  JoinRequest,
  SubmitRequest,
} from "../api/proto/codewar/match/v1/match";

export interface SocketHandlers {
  onMessage: (data: unknown) => void;
  onClose: () => void;
}

export interface JoinOptions {
  token?: string;
  class?: string; // 06 号票：职业盲选（本地选完再 join）
  skill?: string; // 已装备技能（须属于该职业，服务端 fail-closed 校验）
}

export class MatchSocket {
  private ws: WebSocket | null = null;
  private player: string = "";
  private handlers: SocketHandlers | null = null;
  private joinOpts: JoinOptions = {};

  connect(player: string, handlers: SocketHandlers, opts: JoinOptions = {}): void {
    this.player = player;
    this.handlers = handlers;
    this.joinOpts = opts;
    const proto = location.protocol === "https:" ? "wss" : "ws";
    this.ws = new WebSocket(`${proto}://${location.host}/ws`);
    this.ws.onopen = () => this.sendJoin();
    this.ws.onmessage = (m) => this.handlers?.onMessage(JSON.parse(m.data as string));
    this.ws.onclose = () => this.handlers?.onClose();
  }

  private sendJoin(): void {
    const msg: ClientMessage = {
      join: JoinRequest.fromJSON({
        player: this.player,
        token: this.joinOpts.token ?? "",
        class: this.joinOpts.class ?? "",
        skill: this.joinOpts.skill ?? "",
      }),
    };
    this.ws!.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  // 03/06 号票：提交带行动选择；移动题附带目标格；攻击/技能附带目标选择
  //（也可经 setTargeting 独立消息随时修改——结算读服务端最新值）
  submit(
    action: string,
    code: string,
    moveTo?: { row: number; col: number },
    targeting?: { direction: string; distance: number },
  ): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const msg: ClientMessage = {
      submit: SubmitRequest.fromJSON({
        code,
        action,
        moveTo: moveTo ? { row: moveTo.row, col: moveTo.col } : undefined,
        targeting:
          targeting && (targeting.direction || targeting.distance > 0)
            ? { direction: targeting.direction, distance: String(targeting.distance) }
            : undefined,
      }),
    };
    this.ws.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  // 06 号票：解出攻击/技能题后、结算前随时修改目标（免费操作不耗答题）
  setTargeting(t: { direction: string; distance: number }): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const msg: ClientMessage = {
      setTargeting: { targeting: { direction: t.direction, distance: String(t.distance) } },
    };
    this.ws.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }
}
