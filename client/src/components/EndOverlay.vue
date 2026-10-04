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
    })[store.result?.reason ?? ""] ?? store.result?.reason,
);
</script>

<template>
  <div v-if="store.isEnded" class="overlay">
    <div class="card">
      <h2>{{ title }}</h2>
      <p>{{ reasonText }}</p>
      <p class="hint">刷新两个窗口可再来一局（本票单机热座，07/08 号票引入账号与在线匹配）</p>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed; inset: 0; background: rgba(10, 13, 20, 0.8);
  display: flex; align-items: center; justify-content: center;
}
.card {
  background: #1a2030; padding: 32px 48px; border-radius: 14px;
  text-align: center; color: #dfe4ee;
}
.card h2 { font-size: 26px; margin: 0 0 10px; }
.hint { color: #56607a; font-size: 12px; }
</style>
