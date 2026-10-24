<script setup lang="ts">
// 四槽题面板（06 号票）：攻击（镜像竞速）/ 移动（私有 + 网格点选）/ 防御（私有）
// / 技能（私有 + 冷却灰置 + 目标选择）。各槽独立编辑器——一回合仅一题生效。
import { computed, ref, watch } from "vue";
import { useMatchStore } from "../stores/match";
import { useClassesStore } from "../stores/classes";
import { strategyLabel, topicLabel } from "../topics";
import { socket } from "../api/socket";
import CodeEditor from "./CodeEditor.vue";

const store = useMatchStore();
const classesStore = useClassesStore();

// 10 号票：教程只展示本关相关行动槽（未传 = 四槽全开，PvP/PvE 行为不变）。
const props = defineProps<{ slots?: string[] }>();
function show(action: string): boolean {
  return !props.slots || props.slots.includes(action);
}

// 各槽独立编辑器内容：新题就绪时以 starter code 初始化
const codes = ref({ attack: "", move: "", defend: "", skill: "" });
watch(
  () => store.panel,
  (p) => {
    codes.value = {
      attack: p?.attack?.starterCode ?? "",
      move: p?.move?.starterCode ?? "",
      defend: p?.defend?.starterCode ?? "",
      skill: p?.skill?.starterCode ?? "",
    };
  },
);

// ===== targeting 控件（裁决 3：解出后至结算前可随时改，免费不耗答题） =====
const DIRS = [
  { v: "up", a: "↑" },
  { v: "down", a: "↓" },
  { v: "left", a: "←" },
  { v: "right", a: "→" },
];
// 查看者职业与已装备技能（/api/classes 同源数据 + 快照 skill_id）
const myClass = computed(() => classesStore.byId(store.classes[store.viewerId]?.classId ?? ""));
const mySkill = computed(() => myClass.value?.skills.find((k) => k.id === store.classes[store.viewerId]?.skillId));
const cur = computed(() => store.pendingTargeting ?? { direction: "", distance: 0 });

// 某行动是否需要方向/距离（形状语义来自职业表；melee/burst8/self 无方向）
function aimShape(action: string): { dir: boolean; dist: boolean; rangeMax: number } {
  if (action === "attack") {
    const c = myClass.value;
    if (!c) return { dir: false, dist: false, rangeMax: 0 };
    return {
      dir: c.attackShape !== "melee",
      dist: c.attackShape === "snipe",
      rangeMax: c.attackRange,
    };
  }
  const k = mySkill.value;
  if (!k) return { dir: false, dist: false, rangeMax: 0 };
  return {
    dir: k.shape !== "burst8" && k.shape !== "self",
    dist: k.shape === "snipe",
    rangeMax: k.rangeMax,
  };
}

function aim(action: string, direction: string, distance: number) {
  const t = { direction, distance };
  store.setTargeting(t);
  socket.setTargeting(t);
  void action;
}
function aimDir(action: string, d: string) {
  const a = aimShape(action);
  aim(action, cur.value.direction === d ? "" : d, a.dist ? Math.max(1, cur.value.distance) : 0);
}
function aimDist(action: string, delta: number) {
  const a = aimShape(action);
  const next = Math.min(a.rangeMax, Math.max(0, (cur.value.distance || 1) + delta));
  aim(action, cur.value.direction || "", next);
}

const now = ref(Date.now());
setInterval(() => (now.value = Date.now()), 100);

const lockRemaining = computed(() =>
  Math.max(0, store.viewerLockUntil - (now.value + store.serverOffsetMs)),
);
const canSubmit = computed(
  () => store.isRoundActive && !store.viewerActed && lockRemaining.value <= 0,
);
const remainingSec = computed(() => Math.ceil(store.remainingMs / 1000));
const actedLabel = computed(() => {
  const map: Record<string, string> = { move: "移动", defend: "防御", skill: "技能" };
  return map[store.actedKind ?? ""] ?? "攻击";
});
// 查看者技能冷却剩余回合（20 号票：对局中由事件流本地推导——judged 技能通过登记、
// roundStarted 更新回合时递减；接入/重连快照值经 store 重锚为初值。组件只读派生值，
// 不再直读快照字段 classes[..].skillCdRounds——那在对局中不更新，正是本票缺陷 B）
const skillCd = computed(() => store.skillCdRounds);
const hasSkillSlot = computed(() => !!store.classes[store.viewerId]?.skillId);

function submitAttack() {
  if (!canSubmit.value || !store.panel?.attack) return;
  socket.submit("attack", codes.value.attack, undefined, store.pendingTargeting ?? undefined);
}
function submitMove() {
  if (!canSubmit.value || !store.panel?.move || !store.pendingMoveTo) return;
  socket.submit("move", codes.value.move, store.pendingMoveTo);
}
function submitDefend() {
  if (!canSubmit.value || !store.panel?.defend) return;
  socket.submit("defend", codes.value.defend);
}
// 技能槽：冷却中灰置（服务端权威 skill_cd_rounds，本地只渲染）；未装备不渲染整槽
function submitSkill() {
  if (!canSubmit.value || !store.panel?.skill || skillCd.value > 0) return;
  socket.submit("skill", codes.value.skill, undefined, store.pendingTargeting ?? undefined);
}
</script>

<template>
  <div class="panel">
    <div v-if="!store.isRoundActive" class="placeholder">等待对局开始…</div>
    <template v-else>
      <!-- 05 号票：本场锁定的对局配置（快照下发，只读——两窗口参数冲突时以服务端为准） -->
      <div v-if="store.matchConfig" class="match-cfg" data-testid="match-config-view">
        题型 {{ topicLabel(store.matchConfig.topic) }} · 难度上限 L{{ store.matchConfig.cap }} · 抽题{{ strategyLabel(store.matchConfig.strategy) }}
      </div>
      <div class="head">
        <span class="round">回合 {{ store.round }}</span>
        <span class="timer" :class="{ low: remainingSec <= 10 }">{{ remainingSec }}s</span>
        <span v-if="store.viewerActed" class="solved">已行动（{{ actedLabel }}），等待结算…</span>
        <span v-else-if="lockRemaining > 0" class="locked">锁定中 {{ Math.ceil(lockRemaining / 1000) }}s</span>
        <span v-else class="hint">四选一作答：本回合只生效一个行动</span>
      </div>

      <!-- 攻击槽：双方同题竞速 -->
      <section v-if="show('attack')" class="slot">
        <header><b>⚔ 攻击</b><small>镜像题 · 双方同题竞速，越快伤害越高</small></header>
        <pre v-if="store.panel?.attack" class="statement">{{ store.panel.attack.statement }}</pre>
        <code v-if="store.panel?.attack" class="signature">{{ store.panel.attack.signature }}</code>
        <CodeEditor v-if="store.panel?.attack" v-model="codes.attack" :reset-key="store.panel.attack.id" />
        <div v-if="aimShape('attack').dir" class="targeting">
          <small>方向</small>
          <button v-for="d in DIRS" :key="d.v" class="dir" :class="{ on: cur.direction === d.v }" @click="aimDir('attack', d.v)">{{ d.a }}</button>
          <template v-if="aimShape('attack').dist">
            <small>距离</small>
            <button class="dir" @click="aimDist('attack', -1)">−</button>
            <span class="dist">{{ cur.distance || "自动" }}</span>
            <button class="dir" @click="aimDist('attack', 1)">＋</button>
          </template>
        </div>
        <button class="submit atk" :disabled="!canSubmit" @click="submitAttack">提交攻击</button>
      </section>

      <!-- 移动槽：私有题 + 网格点选目标格 -->
      <section v-if="show('move')" class="slot">
        <header><b>🏃 移动</b><small>私有题 · 解出后移动到所选格子（直线正交 1-2 格）</small></header>
        <pre v-if="store.panel?.move" class="statement">{{ store.panel.move.statement }}</pre>
        <code v-if="store.panel?.move" class="signature">{{ store.panel.move.signature }}</code>
        <CodeEditor v-if="store.panel?.move" v-model="codes.move" :reset-key="store.panel.move.id" />
        <div class="move-row">
          <span class="target">
            目标格：{{ store.pendingMoveTo ? `(${store.pendingMoveTo.row}, ${store.pendingMoveTo.col})` : "在左侧网格点击选择" }}
          </span>
          <button v-if="store.pendingMoveTo" class="clear" @click="store.setMoveTo(null)">清除</button>
        </div>
        <button class="submit mov" :disabled="!canSubmit || !store.pendingMoveTo" @click="submitMove">
          提交移动
        </button>
      </section>

      <!-- 技能槽：私有题 + 冷却灰置（06 号票） -->
      <section v-if="show('skill') && hasSkillSlot" class="slot skill" :class="{ cooldown: skillCd > 0 }">
        <header>
          <b>✨ 技能</b>
          <small>{{ mySkill?.name ?? "已装备" }} · {{ skillCd > 0 ? `冷却中（还剩 ${skillCd} 回合）` : "就绪" }}</small>
        </header>
        <pre v-if="store.panel?.skill" class="statement">{{ store.panel.skill.statement }}</pre>
        <code v-if="store.panel?.skill" class="signature">{{ store.panel.skill.signature }}</code>
        <CodeEditor v-if="store.panel?.skill" v-model="codes.skill" :reset-key="store.panel.skill.id" />
        <div v-if="skillCd === 0 && aimShape('skill').dir" class="targeting">
          <small>方向</small>
          <button v-for="d in DIRS" :key="d.v" class="dir" :class="{ on: cur.direction === d.v }" @click="aimDir('skill', d.v)">{{ d.a }}</button>
          <template v-if="aimShape('skill').dist">
            <small>距离</small>
            <button class="dir" @click="aimDist('skill', -1)">−</button>
            <span class="dist">{{ cur.distance || "自动" }}</span>
            <button class="dir" @click="aimDist('skill', 1)">＋</button>
          </template>
        </div>
        <button class="submit skl" :disabled="!canSubmit || skillCd > 0" @click="submitSkill">释放技能</button>
      </section>

      <!-- 防御槽：私有题 -->
      <section v-if="show('defend')" class="slot">
        <header><b>🛡 防御</b><small>私有题 · 解出后本回合减伤 60%</small></header>
        <pre v-if="store.panel?.defend" class="statement">{{ store.panel.defend.statement }}</pre>
        <code v-if="store.panel?.defend" class="signature">{{ store.panel.defend.signature }}</code>
        <CodeEditor v-if="store.panel?.defend" v-model="codes.defend" :reset-key="store.panel.defend.id" />
        <button class="submit def" :disabled="!canSubmit" @click="submitDefend">提交防御</button>
      </section>
    </template>
  </div>
</template>

<style scoped>
.panel {
  background: var(--cw-color-bg-surface); border-radius: 10px; padding: 14px;
  display: flex; flex-direction: column; gap: 12px;
}
.placeholder { color: var(--cw-color-text-muted); margin: auto; }
.match-cfg { font-size: var(--cw-font-size-xs); color: var(--cw-color-text-muted); }
.head { display: flex; gap: 14px; align-items: center; flex-wrap: wrap; }
.round { font-weight: 700; color: var(--cw-color-text-primary); }
.timer { font-variant-numeric: tabular-nums; color: var(--cw-color-state-success); font-weight: 700; }
.timer.low { color: var(--cw-color-state-danger); }
.solved, .locked, .hint { font-size: var(--cw-font-size-sm); color: var(--cw-color-state-warning); }
.hint { color: var(--cw-color-text-secondary); }
.slot {
  border: 1px solid var(--cw-color-border); border-radius: 8px; padding: 10px;
  display: flex; flex-direction: column; gap: 8px;
}
.slot header { display: flex; gap: 8px; align-items: baseline; }
.slot header b { color: var(--cw-color-text-primary); }
.slot header small { color: var(--cw-color-text-muted); }
.statement {
  /* 22 号票：题面字号提档 xs → sm（12px → 14px，token 阶无 13px 档） */
  white-space: pre-wrap; color: var(--cw-color-text-primary); font-size: var(--cw-font-size-sm);
  margin: 0; font-family: inherit; max-height: 96px; overflow: auto;
}
.signature {
  display: block; background: var(--cw-color-bg-raised); border-radius: 6px;
  padding: 4px 8px; color: var(--cw-color-state-success); font-size: var(--cw-font-size-xs);
  font-family: var(--cw-font-mono);
}
.move-row { display: flex; gap: 10px; align-items: center; font-size: var(--cw-font-size-xs); color: var(--cw-color-text-primary); }
.target { color: var(--cw-color-state-warning); }
.clear {
  background: none; border: 1px solid var(--cw-color-border); color: var(--cw-color-text-secondary);
  border-radius: 6px; padding: 2px 8px; cursor: pointer;
}
.submit {
  padding: 8px 0; width: 100%; color: var(--cw-color-text-primary); font-weight: 700;
  border: none; border-radius: 8px; cursor: pointer;
}
.submit.atk { background: var(--cw-color-class-warrior); }
.submit.mov { background: var(--cw-color-accent); }
.submit.def { background: var(--cw-color-state-success); }
.submit.skl { background: var(--cw-color-class-mage); }
.targeting {
  display: flex; gap: 6px; align-items: center; font-size: var(--cw-font-size-xs); color: var(--cw-color-text-secondary);
}
.targeting .dir {
  background: var(--cw-color-bg-inset); border: 1px solid var(--cw-color-border); color: var(--cw-color-text-primary);
  border-radius: 6px; padding: 2px 8px; cursor: pointer;
}
.targeting .dir.on { border-color: var(--cw-color-state-success); color: var(--cw-color-state-success); }
.targeting .dist { color: var(--cw-color-state-warning); min-width: 34px; text-align: center; }
.slot.cooldown { opacity: 0.55; }
.submit:disabled { background: var(--cw-color-border-strong); color: var(--cw-color-text-muted); cursor: not-allowed; }
</style>
