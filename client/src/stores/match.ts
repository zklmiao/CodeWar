// 对局状态：只消费服务端下发的快照与事件流（ADR-0002——客户端零权威）。
// 本模块是纯状态机（无 DOM/网络依赖），vitest 在此断言外部行为。
import { defineStore } from "pinia";
import {
  Envelope,
  MatchPhase,
  ServerMessage,
  Snapshot,
  SubmissionJudged,
} from "../api/proto/codewar/match/v1/match";

export interface LogEntry {
  seq: number;
  text: string;
}

export interface MatchResultView {
  winner: string; // "p1" | "p2" | ""（平局）
  reason: string; // ko | self-ko | rounds-cap
}

export interface MatchState {
  viewerId: string;
  phase: MatchPhase;
  seq: number;
  round: number;
  deadlineUnixMs: number;
  hp: Record<string, number>;
  problem: { id: string; statement: string; signature: string; starterCode: string } | null;
  viewerLockUntil: number;
  viewerSolved: boolean; // 本回合查看者已通过（服务端权威结论的本地缓存）
  result: MatchResultView | null;
  log: LogEntry[];
  serverOffsetMs: number; // 服务端时钟 - 本地时钟
  gapDetected: boolean; // 事件序号断裂 → 需要重连（ADR-0002）
}

export const useMatchStore = defineStore("match", {
  state: (): MatchState => ({
    viewerId: "",
    phase: MatchPhase.MATCH_PHASE_UNSPECIFIED,
    seq: 0,
    round: 0,
    deadlineUnixMs: 0,
    hp: { p1: 100, p2: 100 },
    problem: null,
    viewerLockUntil: 0,
    viewerSolved: false,
    result: null,
    log: [],
    serverOffsetMs: 0,
    gapDetected: false,
  }),
  getters: {
    isWaiting: (s) => s.phase === MatchPhase.MATCH_PHASE_WAITING,
    isRoundActive: (s) => s.phase === MatchPhase.MATCH_PHASE_ROUND_ACTIVE,
    isEnded: (s) => s.phase === MatchPhase.MATCH_PHASE_ENDED,
    // 倒计时毫秒（客户端只渲染，权威在服务端）
    remainingMs: (s) => Math.max(0, s.deadlineUnixMs - (Date.now() + s.serverOffsetMs)),
  },
  actions: {
    // ===== 服务端消息入口 =====
    applyServerMessage(data: unknown, viewer: string) {
      this.viewerId = viewer;
      const msg = ServerMessage.fromJSON(data as object);
      // ts-proto v2 把 oneof 生成为扁平可选字段（snapshot?/event?）
      if (msg.snapshot) {
        this.applySnapshot(msg.snapshot, viewer);
      } else if (msg.event) {
        this.applyEnvelope(msg.event);
      }
    },
    // ===== 快照：接入/重连的起点 =====
    applySnapshot(s: Snapshot, viewer: string) {
      this.viewerId = viewer;
      this.seq = s.seq;
      this.phase = s.phase;
      this.round = s.round;
      this.deadlineUnixMs = Number(s.deadlineUnixMs);
      this.hp = Object.fromEntries(s.players.map((p) => [p.id, p.hp]));
      this.problem = s.problem
        ? {
            id: s.problem.id,
            statement: s.problem.statement,
            signature: s.problem.signature,
            starterCode: s.problem.starterCode,
          }
        : null;
      this.viewerLockUntil = Number(s.viewerLockUntilUnixMs);
      this.viewerSolved = false;
      this.result = s.result ? { winner: s.result.winner, reason: s.result.reason } : null;
    },
    // ===== 事件：逐条应用；序号断裂置 gapDetected（ADR-0002）=====
    applyEnvelope(env: Envelope) {
      if (env.seq !== this.seq + 1) {
        this.gapDetected = true;
        return;
      }
      this.seq = env.seq;
      if (env.unixMs > 0) this.serverOffsetMs = Number(env.unixMs) - Date.now();

      if (env.roundStarted) {
        this.round = env.roundStarted.round;
        this.deadlineUnixMs = Number(env.roundStarted.deadlineUnixMs);
        this.phase = MatchPhase.MATCH_PHASE_ROUND_ACTIVE;
        this.viewerSolved = false;
        this.problem = null;
        this.pushLog(`—— 第 ${this.round} 回合 ——`);
      } else if (env.panelReady?.problem) {
        this.problem = {
          id: env.panelReady.problem.id,
          statement: env.panelReady.problem.statement,
          signature: env.panelReady.problem.signature,
          starterCode: env.panelReady.problem.starterCode,
        };
      } else if (env.submissionJudged) {
        this.applyJudged(env.submissionJudged);
      } else if (env.actionResolved) {
        this.pushLog(`⚔ ${env.actionResolved.attacker} 出手，造成 ${env.actionResolved.damage} 伤害`);
      } else if (env.hpChanged) {
        const p = env.hpChanged.player;
        this.hp[p] = env.hpChanged.hp;
        this.pushLog(
          env.hpChanged.cause === "self-damage"
            ? `✖ ${p} 提交错误，自损 3（剩余 ${env.hpChanged.hp}）`
            : `💢 ${p} 受到 ${-env.hpChanged.delta} 伤害（剩余 ${env.hpChanged.hp}）`,
        );
      } else if (env.matchEnded) {
        this.phase = MatchPhase.MATCH_PHASE_ENDED;
        this.result = { winner: env.matchEnded.winner, reason: env.matchEnded.reason };
        this.pushLog(
          env.matchEnded.winner
            ? `🏆 ${env.matchEnded.winner} 获胜（${env.matchEnded.reason}）`
            : `🤝 平局（${env.matchEnded.reason}）`,
        );
      }
    },
    applyJudged(j: SubmissionJudged) {
      if (j.passed) {
        if (j.player === this.viewerId) this.viewerSolved = true;
        this.pushLog(`✔ ${j.player} 解出本题（${(j.solveMs / 1000).toFixed(1)}s）`);
      } else if (j.category === "locked") {
        this.pushLog(`⏳ ${j.player} 的提交在锁定期内被丢弃`);
      } else if (j.category === "judge-error") {
        this.pushLog(`⚠ 判题服务波动，${j.player} 可重新提交`);
      } else if (j.selfDamageApplied && j.player === this.viewerId) {
        // 服务端锁 5 秒；本地给同步预览倒计时（权威仍在服务端）
        this.viewerLockUntil = Date.now() + this.serverOffsetMs + 5000;
        this.pushLog(`✖ ${this.viewerId} 提交错误，锁定 5 秒并自损 3`);
      } else {
        this.pushLog(`✖ ${j.player} 提交错误（${j.category}）`);
      }
    },
    pushLog(text: string) {
      this.log.push({ seq: this.seq, text });
      if (this.log.length > 50) this.log.shift();
    },
  },
});
