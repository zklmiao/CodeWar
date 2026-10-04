<script setup lang="ts">
// 5×5 网格：本票双方固定站位且相邻（ruling #7）——攻击必然命中，空间博弈是 03 号票。
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
// 行列从 0 计：p1 在 (3,2)，p2 在 (2,2)，垂直相邻
const cells = Array.from({ length: 25 }, (_, i) => ({ row: Math.floor(i / 5), col: i % 5 }));
function occupant(row: number, col: number): string {
  if (row === 3 && col === 2) return "p1";
  if (row === 2 && col === 2) return "p2";
  return "";
}
</script>

<template>
  <div class="grid-wrap">
    <div class="grid">
      <div
        v-for="c in cells"
        :key="`${c.row}-${c.col}`"
        class="cell"
        :class="occupant(c.row, c.col)"
      >
        <span v-if="occupant(c.row, c.col) === 'p1'" class="pawn p1" title="P1">P1</span>
        <span v-else-if="occupant(c.row, c.col) === 'p2'" class="pawn p2" title="P2">P2</span>
      </div>
    </div>
    <div class="legend">P1（你={{ store.viewerId === "p1" ? "是" : "否" }}）· P2（你={{ store.viewerId === "p2" ? "是" : "否" }}）</div>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(5, 56px);
  grid-template-rows: repeat(5, 56px);
  gap: 4px;
  background: #1a2030;
  padding: 10px;
  border-radius: 10px;
}
.cell {
  background: #232b40;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.pawn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  box-shadow: 0 0 12px rgba(0, 0, 0, 0.4);
}
.pawn.p1 { background: #c2554f; }
.pawn.p2 { background: #5b7fd4; }
.legend { color: #8a93a8; font-size: 12px; margin-top: 8px; }
</style>
