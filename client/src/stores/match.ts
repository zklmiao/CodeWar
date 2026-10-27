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
import { useClassesStore } from "./classes";

// 30 号票：日志按事件语义分类，EventLog 据此着色（伤害红/治疗绿/闪避黄/系统次级色）。
// 分类只看事件类型与其字段——提交生命周期类（judged/rejected/回合分隔/终局）归 system，
// HP 效果与落空行才进入战斗三色（judged 的"提交错误"文案行报的是提交流水，
// 自损的 HP 变化由成对的 hpChanged 行以 damage 呈现）。
export type LogKind = "damage" | "heal" | "dodge" | "system";

export interface LogEntry {
  seq: number;
  text: string;
  kind: LogKind;
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

// 30 号票：终局统计——由事件流逐事件累计（日志有 50 条上限，禁止从 log 文本反推）。
// damageDealt 只计攻击/技能落地伤害（actionResolved.damage，护盾折减后的实际值）；
// 提交错误自损是惩罚而非战斗伤害，不计入。dodges 记在闪避方（对方攻击/技能落空时）。
export interface MatchStatsView {
  rounds: number; // 已开始的回合数（roundStarted 事件计数）
  damageDealt: Record<string, number>; // 各方造成的战斗总伤害
  dodges: Record<string, number>; // 各方闪避次数
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
  // 09 号票：该席位是否由模拟 Bot 控制（PvE 的 p2 恒为 true）
  bot: boolean;
}

// 本场对局配置（05 号票，首个 join 锁定；快照下发的服务端权威值，
// 对局内只读展示——两窗口 URL 参数冲突时以服务端锁定值为准）
export interface MatchConfigView {
  topic: string; // 题型 key；空 = 全部 6 题型
  cap: number; // 难度上限档 1..5（L1..L5）
  strategy: string; // uniform | climb
  mode: string; // 09 号票：空 = PvP；pve = 单人练习；tutorial = 新手教程
  bot: string; // 09 号票：PvE 难度 easy | normal | hard（非 PvE 为空）
}

// 新手教程状态（10 号票）：服务端判定，客户端只渲染。
// 进度推进没有客户端入口——completed 只来自 ServerMessage.tutorial。
export interface TutorialGoalView {
  id: string;
  label: string;
  done: boolean;
}

export interface TutorialStateView {
  level: number; // 1..4
  title: string;
  mechanic: string; // 本关机制小结（完成页）
  slots: string[]; // 本关展示的行动槽
  goals: TutorialGoalView[];
  completed: boolean;
  nextLevel: number; // 0 = 无下一关
  allCompleted: boolean;
  cap: number;
  capCoeff: number;
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
  tutorial: TutorialStateView | null; // 教程状态（10 号票，服务端判定）
  pendingTargeting: TargetingView | null; // 攻击/技能槽的目标选择（未提交 = 自动瞄准）
  viewerLockUntil: number;
  viewerActed: boolean; // 本回合查看者已行动（服务端权威结论的本地缓存）
  actedKind: string | null; // 已行动的行动类型
  pendingMoveTo: PosView | null; // 移动槽的待提交目标格（网格点选）
  result: MatchResultView | null;
  log: LogEntry[];
  stats: MatchStatsView; // 30 号票：终局统计（事件流累计，重连快照不清零——seq 断号守卫防重放）
  serverOffsetMs: number; // 服务端时钟 - 本地时钟
  gapDetected: boolean; // 事件序号断裂 → 需要重连（ADR-0002）
  // 20 号票 B：技能冷却的事件推导锚点——查看者释放回合 + CD（与引擎 skillReadyFrom
  // 同语义）。仅由 judged(技能,通过,自己) 登记与 applySnapshot 重锚；对局中不再
  // 依赖快照字段（SkillCdRounds 只在接入/重连时权威）。
  skillCdUntilRound: number;
}

function toProblem(p: { id: string; statement: string; signature: string; starterCode: string } | null | undefined): ProblemView | null {
  if (!p) return null;
  return { id: p.id, statement: p.statement, signature: p.signature, starterCode: p.starterCode };
}

// 对方席位（客户端只有 p1/p2 双方，与 hp 初始态同约定）
function other(p: string): string {
  return p === "p1" ? "p2" : "p1";
}

// actionResolved 单条事件的日志 kind（引擎对"伤害+治疗"双效果拆两条事件，
// 单条至多一种战斗效果；落空优先——落空行的语义就是闪避）
function actionKind(a: { dodged?: boolean; damage?: number; heal?: number }): LogKind {
  if (a.dodged) return "dodge";
  if ((a.damage ?? 0) > 0) return "damage";
  if ((a.heal ?? 0) > 0) return "heal";
  return "system";
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
    tutorial: null,
    pendingTargeting: null,
    viewerLockUntil: 0,
    viewerActed: false,
    actedKind: null,
    pendingMoveTo: null,
    result: null,
    log: [],
    stats: { rounds: 0, damageDealt: { p1: 0, p2: 0 }, dodges: { p1: 0, p2: 0 } },
    serverOffsetMs: 0,
    gapDetected: false,
    skillCdUntilRound: 0,
  }),
  getters: {
    isWaiting: (s) => s.phase === MatchPhase.MATCH_PHASE_WAITING,
    isRoundActive: (s) => s.phase === MatchPhase.MATCH_PHASE_ROUND_ACTIVE,
    isEnded: (s) => s.phase === MatchPhase.MATCH_PHASE_ENDED,
    // 倒计时毫秒（客户端只渲染，权威在服务端）
    remainingMs: (s) => Math.max(0, s.deadlineUnixMs - (Date.now() + s.serverOffsetMs)),
    // 20 号票 B：查看者技能冷却剩余回合（0 = 可用）——由锚点与本回合派生，
    // roundStarted 更新 round 时自动逐回合递减；释放回合即显示 CD（与引擎快照
    // 语义一致：skillReadyFrom - round，释放当回合也灰置）。
    skillCdRounds: (s) => Math.max(0, s.skillCdUntilRound - s.round),
  },
  actions: {
    // ===== 服务端消息入口 =====
    applyServerMessage(data: unknown, viewer: string) {
      this.viewerId = viewer;
      const msg = ServerMessage.fromJSON(data as object);
      // ts-proto v2 把 oneof 生成为扁平可选字段（snapshot?/event?/tutorial?）
      if (msg.tutorial) {
        this.applyTutorial(msg.tutorial);
      } else if (msg.snapshot) {
        this.applySnapshot(msg.snapshot, viewer);
      } else if (msg.event) {
        this.applyEnvelope(msg.event);
      }
    },
    // 教程状态（10 号票）：目标清单 / 槽位 / 完成态都由服务端下发。
    applyTutorial(t: NonNullable<ServerMessage["tutorial"]>) {
      this.tutorial = {
        level: t.level,
        title: t.title,
        mechanic: t.mechanic,
        slots: [...(t.slots ?? [])],
        goals: (t.goals ?? []).map((g) => ({ id: g.id, label: g.label, done: g.done })),
        completed: t.completed,
        nextLevel: t.nextLevel,
        allCompleted: t.allCompleted,
        cap: t.cap,
        capCoeff: t.capCoeff,
      };
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
          bot: p.bot ?? false,
        };
      }
      // 05 号票：本场锁定配置（服务端权威；对局内只读展示的数据源）
      // 09 号票：mode/bot 同源锁定（PvE 的"机器人（难度）"标识数据源）
      this.matchConfig = s.matchConfig
        ? {
            topic: s.matchConfig.topic ?? "",
            cap: s.matchConfig.cap ?? 0,
            strategy: s.matchConfig.strategy ?? "",
            mode: s.matchConfig.mode ?? "",
            bot: s.matchConfig.bot ?? "",
          }
        : null;
      // 20 号票 B：接入/重连快照的 skill_cd_rounds 重锚冷却推导基线（权威初值）——
      // 无条件覆盖本地推导（含归零），保证双源不漂移、重连以服务端为准。
      this.skillCdUntilRound = s.round + (this.classes[viewer]?.skillCdRounds ?? 0);
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
        this.stats.rounds++; // 30 号票：回合数由 roundStarted 事件累计
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
        if (env.hpChanged.cause === "self-damage") {
          this.pushLog(`✖ ${p} 提交错误，自损 3（剩余 ${env.hpChanged.hp}）`, "damage");
        } else if (env.hpChanged.cause === "heal") {
          this.pushLog(`✚ ${p} 恢复 ${env.hpChanged.delta} HP（剩余 ${env.hpChanged.hp}）`, "heal");
        } else {
          this.pushLog(`💢 ${p} 受到 ${-env.hpChanged.delta} 伤害（剩余 ${env.hpChanged.hp}）`, "damage");
        }
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
          // 20 号票 B：冷却登记镜像引擎语义（16 号票）——judged 技能通过即登记
          // （当前回合 + CD），与结算时的命中/闪避无关，故不挂在 actionResolved 上。
          // CD 查 /api/classes 同源职业表；数据未就绪或查不到时不登记——
          // 服务端 skill-cooldown 拒绝事件仍是权威兜底（对局内快照不再下发）。
          if (j.action === "skill") {
            const me = this.classes[this.viewerId];
            const cd =
              useClassesStore()
                .byId(me?.classId ?? "")
                ?.skills.find((k) => k.id === me?.skillId)?.cooldown ?? 0;
            if (cd > 0) this.skillCdUntilRound = this.round + cd;
          }
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
    // 行动结算：移动改写位置、防御挂盾、攻击按 闪避→护盾 顺序渲染；
    // 30 号票：伤害/闪避统计在同分支累计（伤害记出手方、闪避记闪避方）
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
          this.stats.dodges[other(p)]++;
          this.pushLog(`💨 ${p} 的技能落空——目标不在覆盖范围`, "dodge");
        } else {
          const parts: string[] = [];
          if (a.damage > 0) {
            this.stats.damageDealt[p] += a.damage;
            parts.push(`造成 ${a.damage} 伤害`);
          }
          if (a.heal > 0) parts.push(`恢复 ${a.heal} HP`);
          const tail = a.shielded ? "（护盾折减）" : "";
          this.pushLog(`✨ ${p} 释放技能：${parts.join("、")}${tail}`, actionKind(a));
        }
      } else if (a.action === "attack") {
        if (a.dodged) {
          this.stats.dodges[other(p)]++;
          this.pushLog(`💨 ${p} 的攻击落空——目标已移出相邻格`, "dodge");
        } else {
          if (a.damage > 0) this.stats.damageDealt[p] += a.damage;
          const tail = a.shielded ? "（护盾折减）" : "";
          this.pushLog(`⚔ ${p} 出手，造成 ${a.damage} 伤害${tail}`, actionKind(a));
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
    pushLog(text: string, kind: LogKind = "system") {
      this.log.push({ seq: this.seq, text, kind });
      if (this.log.length > 50) this.log.shift();
    },
  },
});
