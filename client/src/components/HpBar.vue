<script setup lang="ts">
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
const players = [
  { id: "p1", label: "P1", color: "#c2554f" },
  { id: "p2", label: "P2", color: "#5b7fd4" },
];
</script>

<template>
  <div class="bars">
    <div v-for="p in players" :key="p.id" class="bar-row">
      <span class="name" :class="{ me: store.viewerId === p.id }">{{ p.label }}</span>
      <div class="bar">
        <div class="fill" :style="{ width: store.hp[p.id] + '%', background: p.color }"></div>
        <span class="num">{{ store.hp[p.id] }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bars { display: flex; gap: 24px; align-items: center; }
.bar-row { display: flex; align-items: center; gap: 8px; }
.name { font-weight: 700; color: #dfe4ee; }
.name.me { text-decoration: underline; text-underline-offset: 4px; }
.bar {
  position: relative; width: 180px; height: 18px;
  background: #232b40; border-radius: 9px; overflow: hidden;
}
.fill { height: 100%; transition: width 0.3s; }
.num {
  position: absolute; inset: 0; text-align: center;
  font-size: 12px; line-height: 18px; color: #fff;
  text-shadow: 0 1px 2px rgba(0,0,0,.6);
}
</style>
