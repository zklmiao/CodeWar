// 全局唯一 socket 实例：App 建连，面板提交复用。
import { MatchSocket } from "./ws";

export const socket = new MatchSocket();
