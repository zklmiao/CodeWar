<script setup lang="ts">
// 题卡 + 主工作区（24 号票）：四张紧凑题卡横排一行 + 激活制共享大编辑器。
// 游戏机制"一回合仅一题生效"→ 选卡→聚焦：点卡激活后在工作区读完整题面并作答；
// codes 按 action 分槽保留，切换/回切不丢码；同时仅挂载一个 CodeMirror（原四槽各一）。
// 槽位：攻击（镜像竞速）/ 移动（私有 + 网格点选）/ 防御（私有）/
// 技能（私有 + 冷却灰置 + 目标选择）。回合状态上移 RoundStatusBar（sticky 条）。
import { computed, ref, watch } from "vue";
import { useMatchStore } from "../stores/match";
import { useClassesStore } from "../stores/classes";
import { socket } from "../api/socket";
import CodeEditor from "./CodeEditor.vue";

const store = useMatchStore();
const classesStore = useClassesStore();

// 10 号票：教程只展示本关相关行动槽（未传 = 四槽全开，PvP/PvE 行为不变）。
const props = defineProps<{ slots?: string[] }>();
function show(action: string): boolean {
  return !props.slots || props.slots.includes(action);
}

// 各槽独立编辑器内容：新题就绪时以 starter code 初始化（切换卡片由 v-model 换装）
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

// ===== 查看者职业/技能/冷却（题卡可见性与工作区头部的前置依赖，先于卡片声明） =====
// 查看者职业与已装备技能（/api/classes 同源数据 + 快照 skill_id）
const myClass = computed(() => classesStore.byId(store.classes[store.viewerId]?.classId ?? ""));
const mySkill = computed(() => myClass.value?.skills.find((k) => k.id === store.classes[store.viewerId]?.skillId));
// 查看者技能冷却剩余回合（20 号票：对局中由事件流本地推导——judged 技能通过登记、
// roundStarted 更新回合时递减；接入/重连快照值经 store 重锚为初值。组件只读派生值，
// 不再直读快照字段 classes[..].skillCdRounds——那在对局中不更新，正是本票缺陷 B）
const skillCd = computed(() => store.skillCdRounds);
const hasSkillSlot = computed(() => !!store.classes[store.viewerId]?.skillId);

// ===== 题卡行与激活态 =====
const ACTIONS = ["attack", "move", "defend", "skill"] as const;
type Action = (typeof ACTIONS)[number];
const CARD_META: Record<Action, { icon: string; name: string; sub: string; flavor: string; submit: string }> = {
  attack: { icon: "⚔", name: "攻击", sub: "镜像 · 同题竞速", flavor: "镜像题 · 双方同题竞速，越快伤害越高", submit: "提交攻击" },
  move: { icon: "🏃", name: "移动", sub: "私有 · 网格点选目标", flavor: "私有题 · 解出后移动到所选格子（直线正交 1-2 格）", submit: "提交移动" },
  defend: { icon: "🛡", name: "防御", sub: "私有 · 减伤 60%", flavor: "私有题 · 解出后本回合减伤 60%", submit: "提交防御" },
  skill: { icon: "✨", name: "技能", sub: "私有 · 大招", flavor: "", submit: "释放技能" },
};
// 技能卡仅在已装备技能时可见（与 06 号票整槽隐藏语义一致）
const visibleActions = computed(() =>
  ACTIONS.filter((a) => show(a) && (a !== "skill" || hasSkillSlot.value)),
);
const activeAction = ref<string>("");
const activeProblem = computed(() =>
  store.panel?.[activeAction.value as Action] ?? null,
);
// 默认激活第一个可见卡（PvP/PvE 恒为攻击卡；教程 slots 可能不含攻击——L2 仅 move）。
// 新回合不清激活：用户已聚焦的卡保留（新题经 codes 重灌 + resetKey 换装，工作区无空态），
// 仅当激活卡不可见（教程切关/卸下技能）时归位首个可见卡。
watch(
  visibleActions,
  (list) => {
    if (!list.includes(activeAction.value)) activeAction.value = list[0] ?? "";
  },
  { immediate: true },
);
// 棋盘点选联动：网格点选落库（GameGrid 经 store.setMoveTo，本组件零耦合），
// 目标格非 null 即自动激活移动卡；setMoveTo(null)（清除）不触发。
watch(
  () => store.pendingMoveTo,
  (v) => {
    if (v) activeAction.value = "move";
  },
);

// ===== targeting 控件（裁决 3：解出后至结算前可随时改，免费不耗答题） =====
const DIRS = [
  { v: "up", a: "↑" },
  { v: "down", a: "↓" },
  { v: "left", a: "←" },
  { v: "right", a: "→" },
];
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
// 主工作区唯一提交按钮：按激活卡分发（已行动仅禁用提交，不禁切换/编辑）
function submitActive() {
  if (activeAction.value === "attack") submitAttack();
  else if (activeAction.value === "move") submitMove();
  else if (activeAction.value === "defend") submitDefend();
  else if (activeAction.value === "skill") submitSkill();
}
// 各卡提交按钮的独立禁用条件（移动需目标格、技能需冷却结束）
function submitDisabled(a: string): boolean {
  if (!canSubmit.value) return true;
  if (a === "move") return !store.pendingMoveTo;
  if (a === "skill") return skillCd.value > 0;
  return false;
}
// 技能卡/工作区头部文案：冷却中灰置、就绪可放
const skillStatus = computed(() =>
  skillCd.value > 0 ? `冷却中（还剩 ${skillCd.value} 回合）` : "就绪",
);
</script>

<template>
  <div class="panel">
    <div v-if="!store.isRoundActive" class="placeholder">等待对局开始…</div>
    <template v-else>
      <!-- 题卡行：紧凑卡（槽名 + 题面 clamp + 难度徽标位），点卡激活 -->
      <div class="cards">
        <button
          v-for="a in visibleActions"
          :key="a"
          class="qcard"
          :class="{
            active: activeAction === a,
            acted: store.actedKind === a,
            dimmed: store.viewerActed && store.actedKind !== a,
            cooldown: a === 'skill' && skillCd > 0,
          }"
          :data-testid="`qcard-${a}`"
          @click="activeAction = a"
        >
          <span v-if="a === 'skill' && skillCd > 0" class="cd-badge">还剩 {{ skillCd }} 回合</span>
          <span class="qcard-head">
            <b>{{ CARD_META[a as Action].icon }} {{ CARD_META[a as Action].name }}</b>
            <span class="diff-ph" title="难度徽标位（25 号票）"></span>
            <span v-if="store.actedKind === a" class="acted-mark">已提交 ✓</span>
          </span>
          <small v-if="a !== 'skill'" class="sub">{{ CARD_META[a as Action].sub }}</small>
          <small v-else class="sub">{{ mySkill?.name ?? "已装备" }} · {{ skillStatus }}</small>
          <span v-if="store.panel?.[a as Action]" class="preview">{{ store.panel?.[a as Action]?.statement }}</span>
        </button>
      </div>

      <!-- 主工作区：激活卡的完整题面 + signature + 共享编辑器 + targeting + 提交 -->
      <section v-if="activeAction" class="work" data-testid="work-area">
        <header class="work-head">
          <b>{{ CARD_META[activeAction as Action].icon }} {{ CARD_META[activeAction as Action].name }}</b>
          <small v-if="activeAction !== 'skill'">{{ CARD_META[activeAction as Action].flavor }}</small>
          <small v-else>{{ mySkill?.name ?? "已装备" }} · {{ skillStatus }}</small>
        </header>
        <pre v-if="activeProblem" class="statement">{{ activeProblem.statement }}</pre>
        <code v-if="activeProblem" class="signature">{{ activeProblem.signature }}</code>
        <CodeEditor
          v-if="activeProblem"
          v-model="codes[activeAction as Action]"
          :reset-key="activeProblem.id"
        />
        <!-- 攻击/技能的目标选择（技能冷却中不渲染，与 06 号票一致） -->
        <div
          v-if="(activeAction === 'attack' || (activeAction === 'skill' && skillCd === 0)) && aimShape(activeAction).dir"
          class="targeting"
        >
          <small>方向</small>
          <button v-for="d in DIRS" :key="d.v" class="dir" :class="{ on: cur.direction === d.v }" @click="aimDir(activeAction, d.v)">{{ d.a }}</button>
          <template v-if="aimShape(activeAction).dist">
            <small>距离</small>
            <button class="dir" @click="aimDist(activeAction, -1)">−</button>
            <span class="dist">{{ cur.distance || "自动" }}</span>
            <button class="dir" @click="aimDist(activeAction, 1)">＋</button>
          </template>
        </div>
        <!-- 移动卡：目标格回显（网格点选联动落库于 store） -->
        <div v-if="activeAction === 'move'" class="move-row">
          <span class="target">
            目标格：{{ store.pendingMoveTo ? `(${store.pendingMoveTo.row}, ${store.pendingMoveTo.col})` : "在左侧网格点击选择" }}
          </span>
          <button v-if="store.pendingMoveTo" class="clear" @click="store.setMoveTo(null)">清除</button>
        </div>
        <button
          class="submit"
          :class="`act-${activeAction}`"
          :data-testid="`submit-${activeAction}`"
          :disabled="submitDisabled(activeAction)"
          @click="submitActive"
        >
          {{ CARD_META[activeAction as Action].submit }}
        </button>
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

/* ===== 题卡行 ===== */
.cards { display: flex; gap: 10px; }
.qcard {
  position: relative; flex: 1; min-width: 0; text-align: left; cursor: pointer;
  background: var(--cw-color-bg-raised); border: 1px solid var(--cw-color-border);
  border-radius: var(--cw-radius-lg); padding: 10px;
  display: flex; flex-direction: column; gap: 6px; color: inherit; font: inherit;
}
.qcard.active { border-color: var(--cw-color-accent); background: var(--cw-color-bg-surface); }
/* 已行动：当前卡"已提交 ✓"高亮，其余卡降权（仅禁提交，不禁切换） */
.qcard.acted { border-color: var(--cw-color-state-success); }
.qcard.dimmed { opacity: 0.6; }
.qcard.cooldown { opacity: 0.55; }
.qcard-head { display: flex; gap: 6px; align-items: center; }
.qcard-head b { color: var(--cw-color-text-primary); }
.diff-ph {
  /* 25 号票占位：难度徽标位（样式归 25），先留一枚空药丸 */
  width: 18px; height: 10px; border-radius: var(--cw-radius-full);
  border: 1px dashed var(--cw-color-border-strong);
}
.acted-mark { color: var(--cw-color-state-success); font-size: var(--cw-font-size-xs); font-weight: 700; }
.cd-badge {
  position: absolute; top: 6px; right: 6px;
  background: var(--cw-color-bg-inset); color: var(--cw-color-state-warning);
  border: 1px solid var(--cw-color-state-warning); border-radius: var(--cw-radius-full);
  font-size: var(--cw-font-size-xs); padding: 1px 8px;
}
.sub { color: var(--cw-color-text-muted); font-size: var(--cw-font-size-xs); }
.preview {
  /* 题面 clamp 2 行：全文点卡激活后在工作区读 */
  display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden;
  color: var(--cw-color-text-secondary); font-size: var(--cw-font-size-xs);
  white-space: pre-wrap; font-family: inherit; margin: 0;
}

/* ===== 主工作区 ===== */
.work {
  border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-lg); padding: 12px;
  display: flex; flex-direction: column; gap: 10px;
}
.work-head { display: flex; gap: 8px; align-items: baseline; }
.work-head b { color: var(--cw-color-text-primary); }
.work-head small { color: var(--cw-color-text-muted); }
.statement {
  /* 24 号票：去除 96px 内滚，完整题面上工作区；超长在本区内部滚动（一屏验收锚点） */
  white-space: pre-wrap; color: var(--cw-color-text-primary); font-size: var(--cw-font-size-sm);
  margin: 0; font-family: inherit; max-height: 160px; overflow: auto;
}
.signature {
  display: block; background: var(--cw-color-bg-raised); border-radius: 6px;
  padding: 4px 8px; color: var(--cw-color-state-success); font-size: var(--cw-font-size-xs);
  font-family: var(--cw-font-mono);
}
/* 共享编辑器限高内滚：超长代码不把工作区撑出页面（CodeEditor 接口不动，样式外部收敛） */
.work :deep(.cm-editor) { max-height: 220px; }
.work :deep(.cm-scroller) { overflow: auto; }
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
.submit.act-attack { background: var(--cw-color-class-warrior); }
.submit.act-move { background: var(--cw-color-accent); }
.submit.act-defend { background: var(--cw-color-state-success); }
.submit.act-skill { background: var(--cw-color-class-mage); }
.targeting {
  display: flex; gap: 6px; align-items: center; font-size: var(--cw-font-size-xs); color: var(--cw-color-text-secondary);
}
.targeting .dir {
  background: var(--cw-color-bg-inset); border: 1px solid var(--cw-color-border); color: var(--cw-color-text-primary);
  border-radius: 6px; padding: 2px 8px; cursor: pointer;
}
.targeting .dir.on { border-color: var(--cw-color-state-success); color: var(--cw-color-state-success); }
.targeting .dist { color: var(--cw-color-state-warning); min-width: 34px; text-align: center; }
.submit:disabled { background: var(--cw-color-border-strong); color: var(--cw-color-text-muted); cursor: not-allowed; }
</style>
