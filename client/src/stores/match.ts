// 对局状态：只消费服务端下发的快照与事件流（ADR-0002——客户端零权威）。
// 本模块是纯状态机（无 DOM/网络依赖），vitest 在此断言外部行为。
// 03 号票：三槽面板（攻击镜像 + 查看者私有移动/防御）、动态站位、护盾、闪避。
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

export interface PlayerEloView {
  player: string; // "p1" | "p2"
  userId: string;
  eloBefore: number;
  eloAfter: number;
}

export interface MatchResultView {
  winner: string; // "p1" | "p2" | ""（平局）
  reason: string; // ko | self-ko | rounds-cap
  // 双方 ELO 变化（07 号票）：已结算对局（含平局，before==after）才有内容；
  // 匿名对局为空数组，终局面板不显示 ELO 区块。
  elo: PlayerEloView[];
}

export interface PosView {
  row: number;
  col: number;
}

export interface ProblemView {
  id: string;
  statement: string;
  signature: string;
  starterCode: string;
}

// 四槽面板（06 号票）：攻击镜像 + 查看者私有行动题 + 技能槽（未装备时为 null）
export interface PanelState {
  attack: ProblemView | null;
  move: ProblemView | null;
  defend: ProblemView | null;
  skill: ProblemView | null;
}

export interface TargetingView {
  direction: string; // up | down | left | right；空 = 自动瞄准
  distance: number; // 0 = 自动瞄准
}

export interface ClassView {
  classId: string; // warrior | mage | ranger | legacy
  skillId: string; // 空 = 未装备技能位
  // 查看者自己的技能冷却剩余回合数（0 = 可用）；对手行不携带
  skillCdRounds: number;
}

// 本场对局配置（05 号票，首个 join 锁定；快照下发的服务端权威值，
// 对局内只读展示——两窗口 URL 参数冲突时以服务端锁定值为准）
export interface MatchConfigView {
  topic: string; // 题型 key；空 = 全部 6 题型
  cap: number; // 难度上限档 1..5（L1..L5）
  strategy: string; // uniform | climb
}

export interface MatchState {
  viewerId: string;
  phase: MatchPhase;
  seq: number;
  round: number;
  deadlineUnixMs: number;
  hp: Record<string, number>;
  positions: Record<string, PosView>; // 结算位置（5×5，行列 0 计）
  shield: Record<string, boolean>; // 本回合护盾
  panel: PanelState | null;
  classes: Record<string, ClassView>; // 职业/技能/冷却（06 号票快照字段）
  matchConfig: MatchConfigView | null; // 本场锁定配置（05 号票快照字段）
  pendingTargeting: TargetingView | null; // 攻击/技能槽的目标选择（未提交 = 自动瞄准）
  viewerLockUntil: number;
  viewerActed: boolean; // 本回合查看者已行动（服务端权威结论的本地缓存）
  actedKind: string | null; // 已行动的行动类型
  pendingMoveTo: PosView | null; // 移动槽的待提交目标格（网格点选）
  result: MatchResultView | null;
  log: LogEntry[];
  serverOffsetMs: number; // 服务端时钟 - 本地时钟
  gapDetected: boolean; // 事件序号断裂 → 需要重连（ADR-0002）
}

function toProblem(p: { id: string; statement: string; signature: string; starterCode: string } | null | undefined): ProblemView | null {
  if (!p) return null;
  return { id: p.id, statement: p.statement, signature: p.signature, starterCode: p.starterCode };
}

export const useMatchStore = defineStore("match", {
  state: (): MatchState => ({
    viewerId: "",
    phase: MatchPhase.MATCH_PHASE_UNSPECIFIED,
    seq: 0,
    round: 0,
    deadlineUnixMs: 0,
    hp: { p1: 100, p2: 100 },
    positions: { p1: { row: 3, col: 2 }, p2: { row: 2, col: 2 } },
    shield: {},
    panel: null,
    classes: {},
    matchConfig: null,
    pendingTargeting: null,
    viewerLockUntil: 0,
    viewerActed: false,
    actedKind: null,
    pendingMoveTo: null,
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
    // ===== 快照：接入/重连的起点（panel 已由服务端按查看者过滤私有题） =====
    applySnapshot(s: Snapshot, viewer: string) {
      this.viewerId = viewer;
      this.seq = s.seq;
      this.phase = s.phase;
      this.round = s.round;
      this.deadlineUnixMs = Number(s.deadlineUnixMs);
      this.hp = {};
      this.positions = {};
      this.shield = {};
      for (const p of s.players) {
        this.hp[p.id] = p.hp;
        this.positions[p.id] = p.pos ? { row: p.pos.row, col: p.pos.col } : { row: 0, col: 0 };
        this.shield[p.id] = p.shielded;
      }
      this.classes = {};
      for (const p of s.players) {
        this.classes[p.id] = {
          classId: p.classId ?? "",
          skillId: p.skillId ?? "",
          skillCdRounds: p.skillCdRounds ?? 0,
        };
      }
      // 05 号票：本场锁定配置（服务端权威；对局内只读展示的数据源）
      this.matchConfig = s.matchConfig
        ? { topic: s.matchConfig.topic ?? "", cap: s.matchConfig.cap ?? 0, strategy: s.matchConfig.strategy ?? "" }
        : null;
      this.panel = s.panel
        ? {
            attack: toProblem(s.panel.attack),
            move: toProblem(s.panel.move),
            defend: toProblem(s.panel.defend),
            skill: toProblem(s.panel.skill),
          }
        : null;
      this.viewerLockUntil = Number(s.viewerLockUntilUnixMs);
      this.viewerActed = false;
      this.actedKind = null;
      // 快照 result（MatchResult proto）不带 ELO：重连快照路径归 08 号票，
      // ELO 只随终局事件下发。
      this.result = s.result ? { winner: s.result.winner, reason: s.result.reason, elo: [] } : null;
    },
    // ===== 事件：逐条应用；序号断裂置 gapDetected（ADR-0002）=====
    applyEnvelope(env: Envelope) {
      if (env.seq !== this.seq + 1) {
        this.gapDetected = true;
        return;
      }
      this.seq = env.seq;
      if (Number(env.unixMs) > 0) this.serverOffsetMs = Number(env.unixMs) - Date.now();

      if (env.roundStarted) {
        this.round = env.roundStarted.round;
        this.deadlineUnixMs = Number(env.roundStarted.deadlineUnixMs);
        this.phase = MatchPhase.MATCH_PHASE_ROUND_ACTIVE;
        this.viewerActed = false;
        this.actedKind = null;
        this.pendingMoveTo = null;
        this.pendingTargeting = null;
        this.shield = {}; // 护盾不跨回合；位置跨回合持续
        this.panel = null;
        this.pushLog(`—— 第 ${this.round} 回合 ——`);
      } else if (env.panelReady) {
        this.applyPanel(env.panelReady);
      } else if (env.submissionJudged) {
        this.applyJudged(env.submissionJudged);
      } else if (env.actionResolved) {
        this.applyAction(env.actionResolved);
      } else if (env.hpChanged) {
        const p = env.hpChanged.player;
        this.hp[p] = env.hpChanged.hp;
        this.pushLog(
          env.hpChanged.cause === "self-damage"
            ? `✖ ${p} 提交错误，自损 3（剩余 ${env.hpChanged.hp}）`
            : env.hpChanged.cause === "heal"
              ? `✚ ${p} 恢复 ${env.hpChanged.delta} HP（剩余 ${env.hpChanged.hp}）`
              : `💢 ${p} 受到 ${-env.hpChanged.delta} 伤害（剩余 ${env.hpChanged.hp}）`,
        );
      } else if (env.submissionRejected) {
        const why = env.submissionRejected.reason;
        const what = why === "skill-cooldown" ? "技能冷却中" : why === "no-skill-equipped" ? "未装备技能" : "移动参数";
        this.pushLog(`⚠ ${env.submissionRejected.player} 的${what}被拒（${why}），可调整后重交`);
      } else if (env.matchEnded) {
        this.phase = MatchPhase.MATCH_PHASE_ENDED;
        this.result = {
          winner: env.matchEnded.winner,
          reason: env.matchEnded.reason,
          elo: (env.matchEnded.elo ?? []).map((e) => ({
            player: e.player,
            userId: e.userId,
            eloBefore: e.eloBefore,
            eloAfter: e.eloAfter,
          })),
        };
        this.pushLog(
          env.matchEnded.winner
            ? `🏆 ${env.matchEnded.winner} 获胜（${env.matchEnded.reason}）`
            : `🤝 平局（${env.matchEnded.reason}）`,
        );
      }
    },
    // 面板：攻击镜像共享；私有题按 viewerId 取本方的（共享事件流携带双方面板，
    // 题面是公开文本、判题用例永不下发，无泄漏）
    applyPanel(pr: NonNullable<Envelope["panelReady"]>) {
      const mine = this.viewerId === "p2" ? pr.p2 : pr.p1;
      this.panel = {
        attack: toProblem(pr.attack),
        move: toProblem(mine?.move),
        defend: toProblem(mine?.defend),
        skill: toProblem(mine?.skill),
      };
    },
    applyJudged(j: SubmissionJudged) {
      const kind = j.action || "攻击";
      if (j.passed) {
        if (j.player === this.viewerId) {
          this.viewerActed = true;
          this.actedKind = j.action;
        }
        this.pushLog(`✔ ${j.player} 解出${kind}题（${(j.solveMs / 1000).toFixed(1)}s）`);
      } else if (j.category === "locked") {
        this.pushLog(`⏳ ${j.player} 的提交在锁定期内被丢弃`);
      } else if (j.category === "judge-error") {
        this.pushLog(`⚠ 判题服务波动，${j.player} 可重新提交`);
      } else if (j.selfDamageApplied && j.player === this.viewerId) {
        // 锁定截止以服务端下发时间为准（judged 事件与快照同字段同源；服务端调参不漂移）
        const lockUntil = Number(j.lockUntilUnixMs);
        if (lockUntil > 0) this.viewerLockUntil = lockUntil;
        this.pushLog(`✖ ${this.viewerId} 提交错误，锁定并自损 3`);
      } else {
        this.pushLog(`✖ ${j.player} 提交错误（${j.category}）`);
      }
    },
    // 行动结算：移动改写位置、防御挂盾、攻击按 闪避→护盾 顺序渲染
    applyAction(a: NonNullable<Envelope["actionResolved"]>) {
      const p = a.attacker;
      if (a.action === "move" && a.moveTo) {
        this.positions[p] = { row: a.moveTo.row, col: a.moveTo.col };
        this.pushLog(`🏃 ${p} 移动到 (${a.moveTo.row}, ${a.moveTo.col})`);
      } else if (a.action === "defend") {
        this.shield[p] = true;
        this.pushLog(`🛡 ${p} 挂起护盾（本回合减伤 60%）`);
      } else if (a.action === "skill") {
        if (a.dodged) {
          this.pushLog(`💨 ${p} 的技能落空——目标不在覆盖范围`);
        } else {
          const parts: string[] = [];
          if (a.damage > 0) parts.push(`造成 ${a.damage} 伤害`);
          if (a.heal > 0) parts.push(`恢复 ${a.heal} HP`);
          const tail = a.shielded ? "（护盾折减）" : "";
          this.pushLog(`✨ ${p} 释放技能：${parts.join("、")}${tail}`);
        }
      } else if (a.action === "attack") {
        if (a.dodged) {
          this.pushLog(`💨 ${p} 的攻击落空——目标已移出相邻格`);
        } else {
          const tail = a.shielded ? "（护盾折减）" : "";
          this.pushLog(`⚔ ${p} 出手，造成 ${a.damage} 伤害${tail}`);
        }
      }
    },
    setMoveTo(pos: PosView | null) {
      this.pendingMoveTo = pos;
    },
    // 06 号票：攻击/技能槽的目标选择（未提交 = 自动瞄准，direction 空）
    setTargeting(t: TargetingView | null) {
      this.pendingTargeting = t;
    },
    pushLog(text: string) {
      this.log.push({ seq: this.seq, text });
      if (this.log.length > 50) this.log.shift();
    },
  },
});
