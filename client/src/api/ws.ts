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

export class MatchSocket {
  private ws: WebSocket | null = null;
  private player: string = "";
  private handlers: SocketHandlers | null = null;

  connect(player: string, handlers: SocketHandlers, token?: string): void {
    this.player = player;
    this.handlers = handlers;
    const proto = location.protocol === "https:" ? "wss" : "ws";
    this.ws = new WebSocket(`${proto}://${location.host}/ws`);
    this.ws.onopen = () => this.sendJoin(token ?? "");
    this.ws.onmessage = (m) => this.handlers?.onMessage(JSON.parse(m.data as string));
    this.ws.onclose = () => this.handlers?.onClose();
  }

  private sendJoin(token: string): void {
    const msg: ClientMessage = {
      join: JoinRequest.fromJSON({ player: this.player, token }),
    };
    this.ws!.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  // 03 号票：提交带行动选择；移动题附带目标格（服务端校验参数合法性）
  submit(action: string, code: string, moveTo?: { row: number; col: number }): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const msg: ClientMessage = {
      submit: SubmitRequest.fromJSON({
        code,
        action,
        moveTo: moveTo ? { row: moveTo.row, col: moveTo.col } : undefined,
      }),
    };
    this.ws.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }
}
