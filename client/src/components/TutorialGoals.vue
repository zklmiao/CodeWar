<script setup lang="ts">
// 10 号票：对局内教程目标清单（checkbox 式，事件驱动点亮）。
// 判定完全在服务端：本组件只渲染 ServerMessage.tutorial。
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
</script>

<template>
  <div v-if="store.tutorial" class="tut-card" data-testid="tutorial-goals">
    <header>
      <b>{{ store.tutorial.title }}</b>
      <small>本关目标</small>
    </header>
    <ul>
      <li v-for="g in store.tutorial.goals" :key="g.id" :class="{ done: g.done }">
        <span class="box">{{ g.done ? "☑" : "☐" }}</span>
        <span>{{ g.label }}</span>
      </li>
    </ul>
    <p class="mechanic">{{ store.tutorial.mechanic }}</p>
  </div>
</template>

<style scoped>
.tut-card {
  background: var(--cw-color-bg-surface);
  border: 1px solid var(--cw-color-border);
  border-radius: var(--cw-radius-md);
  padding: var(--cw-space-3);
  display: flex;
  flex-direction: column;
  gap: var(--cw-space-2);
}
header {
  display: flex;
  gap: var(--cw-space-2);
  align-items: baseline;
}
header b {
  color: var(--cw-color-accent);
  font-size: var(--cw-font-size-sm);
}
header small {
  color: var(--cw-color-text-muted);
  font-size: var(--cw-font-size-xs);
}
ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--cw-space-1);
}
li {
  display: flex;
  gap: var(--cw-space-2);
  align-items: baseline;
  color: var(--cw-color-text-secondary);
  font-size: var(--cw-font-size-sm);
}
li.done {
  color: var(--cw-color-state-success);
}
.box {
  font-variant-numeric: tabular-nums;
}
.mechanic {
  margin: 0;
  color: var(--cw-color-text-muted);
  font-size: var(--cw-font-size-xs);
  line-height: 1.5;
  border-top: 1px solid var(--cw-color-border);
  padding-top: var(--cw-space-2);
}
</style>
