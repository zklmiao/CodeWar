<script setup lang="ts">
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
</script>

<template>
  <div class="log">
    <div v-for="(l, i) in [...store.log].reverse()" :key="i" class="line" :class="`kind-${l.kind}`">
      <span class="seq">{{ l.seq }}</span>{{ l.text }}
    </div>
  </div>
</template>

<style scoped>
.log {
  /* 23 号票：固定 150px 改 flex 弹性高度——左栏内 flex:1 吃掉棋盘以下富余（右栏
     更高则日志随之变高），min-height 150px 保底，窄屏/短题面下日志头部不消失；
     内部滚动行为不变。22 号票：日志为等宽数值语境 → --cw-font-mono；字号 13px 归入 sm 档 */
  flex: 1; min-height: 150px; overflow-y: auto; background: var(--cw-color-bg-surface);
  border-radius: 8px; padding: 8px 12px; font-size: var(--cw-font-size-sm);
  font-family: var(--cw-font-mono);
  color: var(--cw-color-text-secondary);
}
.line { padding: 1px 0; }
.seq { display: inline-block; width: 34px; color: var(--cw-color-text-muted); }
/* 30 号票：kind 着色全走 token（004 §token 规则禁裸色值）。
   system 为容器默认次级色，无需覆盖；序号列单独 muted，不随 kind 染色 */
.kind-damage { color: var(--cw-color-state-danger); }
.kind-heal { color: var(--cw-color-state-success); }
.kind-dodge { color: var(--cw-color-state-warning); }
</style>
