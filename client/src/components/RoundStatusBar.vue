<script setup lang="ts">
// 24 号票：回合状态 sticky 条（HpBar 下方）——回合号 / 倒计时 / 75s 细进度条 /
// 已行动 / 锁定 / 本场配置行（match-config-view testid 随配置行自 QuestionPanel 迁入）。
// 滚动时倒计时始终在场；数值 tabular-nums；≤10s 变红；动效遵守 prefers-reduced-motion。
// sticky 生效条件：滚动祖先是视口（页面级滚动），本条只贴在 .cols 之上、随内容滚动。
// 31 号票：认输按钮（confirm 防误触）——可见性组件内自治（isRoundActive &&
// 非教程局），终局/教程局自然隐藏；PvP 与 PvE 皆可投（PvE 由 Bot 胜收束）。
import { computed, onBeforeUnmount, ref } from "vue";
import { useMatchStore } from "../stores/match";
import { socket } from "../api/socket";
import { strategyLabel, topicLabel } from "../topics";

const store = useMatchStore();

// 倒计时与锁定剩余都是墙钟派生，100ms 心跳（与原面板头部同节奏）
const now = ref(Date.now());
const timer = setInterval(() => (now.value = Date.now()), 100);
onBeforeUnmount(() => clearInterval(timer));

// 倒计时从 deadline 与 100ms 心跳 now 本地派生：store.remainingMs 的 getter 内联
// Date.now() 无响应式依赖，只在事件流写 serverOffsetMs 时偶然重算（旧面板头部
// 潜伏缺陷）——条内必须逐帧可走，本组件不消费它。
const remainingMs = computed(() =>
  Math.max(0, store.deadlineUnixMs - (now.value + store.serverOffsetMs)),
);
const remainingSec = computed(() => Math.ceil(remainingMs.value / 1000));
// 75s 回合进度条：规格数值（001 §回合时长）；服务端 deadline 权威，本条只做视觉反馈
const ROUND_MS = 75_000;
const progressPct = computed(() =>
  Math.min(100, Math.max(0, (remainingMs.value / ROUND_MS) * 100)),
);
const lockRemaining = computed(() =>
  Math.max(0, store.viewerLockUntil - (now.value + store.serverOffsetMs)),
);
const actedLabel = computed(() => {
  const map: Record<string, string> = { move: "移动", defend: "防御", skill: "技能" };
  return map[store.actedKind ?? ""] ?? "攻击";
});

// 31 号票：可投降 = 回合活跃且非教程局。教程的教学目标是引导完成而非胜负，
// 认输会作废教学管线（服务端同样 fail-closed 拒绝）；终局后 isRoundActive 为
// false，按钮随状态条一并消失。
const canResign = computed(
  () => store.isRoundActive && store.matchConfig?.mode !== "tutorial",
);
function onResign() {
  // confirm 防误触：认输立即终局且不可撤销（对方胜 + ELO 结算），必须二次确认
  if (!canResign.value || !window.confirm("确认认输？对局将以对方获胜结束。")) return;
  socket.resign();
}
</script>

<template>
  <div class="round-bar" data-testid="round-status-bar">
    <span class="round">回合 {{ store.round }}</span>
    <span class="timer" :class="{ low: remainingSec <= 10 }">{{ remainingSec }}s</span>
    <span class="track" aria-hidden="true">
      <span class="fill" :class="{ low: remainingSec <= 10 }" :style="{ width: progressPct + '%' }"></span>
    </span>
    <span v-if="store.viewerActed" class="acted">已行动（{{ actedLabel }}），等待结算…</span>
    <span v-else-if="lockRemaining > 0" class="locked">锁定中 {{ Math.ceil(lockRemaining / 1000) }}s</span>
    <span v-else class="hint">四选一作答：本回合只生效一个行动</span>
    <!-- 05 号票：本场锁定的对局配置（快照下发，只读——两窗口参数冲突时以服务端为准） -->
    <span v-if="store.matchConfig" class="match-cfg" data-testid="match-config-view">
      题型 {{ topicLabel(store.matchConfig.topic) }} · 难度上限 L{{ store.matchConfig.cap }} · 抽题{{ strategyLabel(store.matchConfig.strategy) }}
    </span>
    <!-- 31 号票：认输（confirm 防误触；终局后随状态条消失，教程局不渲染） -->
    <button v-if="canResign" class="resign" data-testid="resign-btn" @click="onResign">认输</button>
  </div>
</template>

<style scoped>
.round-bar {
  /* sticky：页面滚动时状态条钉在视口顶；z-index 需压过棋盘 positioned 格子，
     取 overlay 档（结算/教程遮罩在 DOM 中更靠后，同档仍能盖住本条） */
  position: sticky; top: 0; z-index: var(--cw-z-overlay);
  display: flex; gap: 14px; align-items: center; flex-wrap: wrap;
  background: var(--cw-color-bg-surface); border: 1px solid var(--cw-color-border);
  border-radius: var(--cw-radius-md); padding: 8px 14px; margin-bottom: 14px;
}
.round { font-weight: 700; color: var(--cw-color-text-primary); font-variant-numeric: tabular-nums; }
.timer { font-variant-numeric: tabular-nums; color: var(--cw-color-state-success); font-weight: 700; }
.timer.low { color: var(--cw-color-state-danger); }
.track {
  position: relative; flex: 1; min-width: 120px; height: 4px;
  background: var(--cw-color-bg-inset); border-radius: var(--cw-radius-full); overflow: hidden;
}
.fill {
  position: absolute; top: 0; bottom: 0; left: 0;
  background: var(--cw-color-accent); border-radius: var(--cw-radius-full);
  transition: width 0.2s linear;
}
.fill.low { background: var(--cw-color-state-danger); }
.acted, .locked { font-size: var(--cw-font-size-sm); color: var(--cw-color-state-warning); }
.hint { font-size: var(--cw-font-size-sm); color: var(--cw-color-text-secondary); }
.match-cfg { font-size: var(--cw-font-size-xs); color: var(--cw-color-text-muted); }
/* 31 号票：认输按钮——弱化样式（次要操作，避免与行动选择抢注意力） */
.resign {
  background: none; border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-sm);
  color: var(--cw-color-text-muted); padding: var(--cw-space-1) var(--cw-space-3);
  font-size: var(--cw-font-size-xs); cursor: pointer;
}
.resign:hover { border-color: var(--cw-color-state-danger); color: var(--cw-color-state-danger); }
/* 24 号票：动效尊重系统"减少动态效果"——进度条改为跳变 */
@media (prefers-reduced-motion: reduce) {
  .fill { transition: none; }
}
</style>
