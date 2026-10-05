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

  connect(player: string, handlers: SocketHandlers): void {
    this.player = player;
    this.handlers = handlers;
    const proto = location.protocol === "https:" ? "wss" : "ws";
    this.ws = new WebSocket(`${proto}://${location.host}/ws`);
    this.ws.onopen = () => this.sendJoin();
    this.ws.onmessage = (m) => this.handlers?.onMessage(JSON.parse(m.data as string));
    this.ws.onclose = () => this.handlers?.onClose();
  }

  private sendJoin(): void {
    const msg: ClientMessage = {
      join: JoinRequest.fromJSON({ player: this.player }),
    };
    this.ws!.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  submit(code: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const msg: ClientMessage = {
      submit: SubmitRequest.fromJSON({ code }),
    };
    this.ws.send(JSON.stringify(ClientMessage.toJSON(msg)));
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }
}
