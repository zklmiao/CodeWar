<script setup lang="ts">
// 5×5 网格（03 号票）：动态渲染 store 中的结算位置；回合活跃且未行动时
// 点选格子作为移动题的目标格（服务端校验合法性，客户端只做选择）。
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
// 行列从 0 计
const cells = Array.from({ length: 25 }, (_, i) => ({ row: Math.floor(i / 5), col: i % 5 }));

function occupant(row: number, col: number): string {
  for (const p of ["p1", "p2"] as const) {
    const pos = store.positions[p];
    if (pos && pos.row === row && pos.col === col) return p;
  }
  return "";
}
function isMoveTarget(row: number, col: number): boolean {
  const t = store.pendingMoveTo;
  return !!t && t.row === row && t.col === col;
}
function isShielded(row: number, col: number): boolean {
  const o = occupant(row, col);
  return !!o && !!store.shield[o];
}
// 点选：回合活跃且自己未行动时可设置目标格；点击自己所在格 = 清除选择，
// 对方占用的格子不可作为目标（合法性最终由服务端校验）
function pick(row: number, col: number) {
  if (!store.isRoundActive || store.viewerActed) return;
  const cur = store.positions[store.viewerId];
  if (cur && cur.row === row && cur.col === col) {
    store.setMoveTo(null);
    return;
  }
  if (occupant(row, col)) return;
  store.setMoveTo({ row, col });
}
</script>

<template>
  <div class="grid-wrap">
    <div class="grid" :class="{ picking: store.isRoundActive && !store.viewerActed }">
      <div
        v-for="c in cells"
        :key="`${c.row}-${c.col}`"
        class="cell"
        :class="{
          [occupant(c.row, c.col)]: true,
          target: isMoveTarget(c.row, c.col),
          pickable: store.isRoundActive && !store.viewerActed && !occupant(c.row, c.col),
        }"
        @click="pick(c.row, c.col)"
      >
        <span v-if="occupant(c.row, c.col) === 'p1'" class="pawn p1" title="P1">P1</span>
        <span v-else-if="occupant(c.row, c.col) === 'p2'" class="pawn p2" title="P2">P2</span>
        <span v-if="isShielded(c.row, c.col)" class="shield" title="护盾中">🛡</span>
      </div>
    </div>
    <div class="legend">
      P1（你={{ store.viewerId === "p1" ? "是" : "否" }}）· P2（你={{ store.viewerId === "p2" ? "是" : "否" }}）
      <template v-if="store.isRoundActive && !store.viewerActed"> · 点选格子设定移动目标</template>
    </div>
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
  position: relative;
}
.cell.pickable { cursor: pointer; }
.cell.pickable:hover { background: #2c3650; }
.cell.target { outline: 2px solid #e0b45f; background: #3a3320; }
.pawn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 13px;
}
.pawn.p1 { background: #c2554f; }
.pawn.p2 { background: #5b7fd4; }
.shield {
  position: absolute;
  top: 2px;
  right: 4px;
  font-size: 12px;
}
.legend { color: #8a93a8; font-size: 12px; margin-top: 8px; }
</style>
