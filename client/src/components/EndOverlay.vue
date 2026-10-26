<script setup lang="ts">
// 对局终局覆盖层：胜负/平局 + 结束原因
import { computed } from "vue";
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
const title = computed(() => {
  if (!store.result) return "";
  if (!store.result.winner) return "🤝 平局";
  return store.result.winner === store.viewerId ? "🏆 你赢了" : "💀 你输了";
});
const reasonText = computed(
  () =>
    ({
      ko: "击杀对手",
      "self-ko": "对手错误提交自损殆尽",
      "rounds-cap": "10 回合打满，按剩余 HP 裁决",
      resign: "对手主动认输（31 号票）",
    })[store.result?.reason ?? ""] ?? store.result?.reason,
);
// ELO 变化行：before → after（+delta）；匿名对局 elo 为空，不显示本区块。
function deltaText(before: number, after: number): string {
  const d = after - before;
  return d > 0 ? `+${d}` : `${d}`;
}
</script>

<template>
  <div v-if="store.isEnded" class="overlay" data-testid="match-end">
    <div class="card">
      <h2>{{ title }}</h2>
      <p>{{ reasonText }}</p>
      <div v-if="store.result?.elo?.length" class="elo">
        <div v-for="e in store.result.elo" :key="e.player" class="elorow">
          <span class="who">{{ e.player }}</span>
          <span>{{ e.eloBefore }} → {{ e.eloAfter }}</span>
          <span class="delta">（{{ deltaText(e.eloBefore, e.eloAfter) }}）</span>
        </div>
      </div>
      <p class="hint">刷新两个窗口可再来一局（本票单机热座，在线匹配归 08 号票）</p>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed; inset: 0; background: var(--cw-color-bg-scrim);
  display: flex; align-items: center; justify-content: center;
  /* 22 号票：补齐 z-index（教程局终局时须压过 z-10 的教程完成页，见 --cw-z-modal） */
  z-index: var(--cw-z-modal);
}
.card {
  background: var(--cw-color-bg-surface); padding: 32px 48px; border-radius: 14px;
  text-align: center; color: var(--cw-color-text-primary);
}
.card h2 { font-size: var(--cw-font-size-xl); margin: 0 0 10px; }
.elo { margin: 12px 0 4px; font-size: var(--cw-font-size-md); }
.elorow { display: flex; gap: 10px; justify-content: center; padding: 3px 0; }
.who { color: var(--cw-color-text-secondary); }
.delta { color: var(--cw-color-state-success); }
.hint { color: var(--cw-color-text-muted); font-size: var(--cw-font-size-xs); }
</style>
