<script setup lang="ts">
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
</script>

<template>
  <div class="log">
    <div v-for="(l, i) in [...store.log].reverse()" :key="i" class="line">
      <span class="seq">{{ l.seq }}</span>{{ l.text }}
    </div>
  </div>
</template>

<style scoped>
.log {
  /* 22 号票：日志为等宽数值语境 → --cw-font-mono；字号 13px 归入 sm 档 */
  height: 150px; overflow-y: auto; background: var(--cw-color-bg-surface);
  border-radius: 8px; padding: 8px 12px; font-size: var(--cw-font-size-sm);
  font-family: var(--cw-font-mono);
  color: var(--cw-color-text-secondary);
}
.line { padding: 1px 0; }
.seq { display: inline-block; width: 34px; color: var(--cw-color-text-muted); }
</style>
