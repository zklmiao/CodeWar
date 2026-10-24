<script setup lang="ts">
import { botLabel } from "../topics";
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
const players = [
  // 22 号票：席位识别色归位 tokens.css（--cw-color-seat-*），内联样式只持 token 引用
  { id: "p1", color: "var(--cw-color-seat-p1)" },
  { id: "p2", color: "var(--cw-color-seat-p2)" },
];

// 09 号票：PvE 的 p2 是模拟 Bot——显示"机器人（难度）"（难度取服务端锁定配置）。
function playerLabel(id: string): string {
  if (store.classes[id]?.bot) {
    return `${id.toUpperCase()} · 机器人（${botLabel(store.matchConfig?.bot ?? "")}）`;
  }
  return id.toUpperCase();
}
</script>

<template>
  <div class="bars">
    <div v-for="p in players" :key="p.id" class="bar-row">
      <span class="name" :class="{ me: store.viewerId === p.id }">{{ playerLabel(p.id) }}</span>
      <div class="bar">
        <div class="fill" :style="{ width: store.hp[p.id] + '%', background: p.color }"></div>
        <span class="num">{{ store.hp[p.id] }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bars { display: flex; gap: 24px; align-items: center; flex-wrap: wrap; }
.bar-row { display: flex; align-items: center; gap: 8px; }
.name { font-weight: 700; color: var(--cw-color-text-primary); }
.name.me { text-decoration: underline; text-underline-offset: 4px; }
.bar {
  position: relative; width: 180px; height: 18px;
  background: var(--cw-color-bg-raised); border-radius: 9px; overflow: hidden;
}
.fill { height: 100%; transition: width 0.3s; }
.num {
  position: absolute; inset: 0; text-align: center;
  font-size: var(--cw-font-size-xs); line-height: 18px; color: var(--cw-color-text-primary);
  text-shadow: 0 1px 2px var(--cw-color-shadow);
}
</style>
