<script setup lang="ts">
// 5×5 网格（03 号票）：动态渲染 store 中的结算位置；回合活跃且未行动时
// 点选格子作为移动题的目标格（服务端校验合法性，客户端只做选择）。
// 06 号票：棋子按职业渲染 13 号票占位资产（cw-class-*.svg）+ 职业主色。
import { useMatchStore } from "../stores/match";
import { useClassesStore } from "../stores/classes";
import { computed } from "vue";
import warriorSvg from "../../../design/components/cw-class-warrior.svg";
import mageSvg from "../../../design/components/cw-class-mage.svg";
import rangerSvg from "../../../design/components/cw-class-ranger.svg";

const store = useMatchStore();
const classesStore = useClassesStore();

const classArt: Record<string, string> = {
  warrior: warriorSvg,
  mage: mageSvg,
  ranger: rangerSvg,
};

function classArtOf(p: string): string {
  return classArt[store.classes[p]?.classId ?? ""] ?? "";
}
function classColorOf(p: string): string {
  return classesStore.byId(store.classes[p]?.classId ?? "")?.color ?? "";
}
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
// 09 号票：Bot 棋子悬停标识（PvE 的 p2）
function titleOf(p: string): string {
  return store.classes[p]?.bot ? "机器人" : p.toUpperCase();
}
// 10 号票：教程未展示移动槽时不可点选（避免误以为在选移动目标）。
const canPick = computed(
  () =>
    store.isRoundActive &&
    !store.viewerActed &&
    (!store.tutorial || store.tutorial.slots.includes("move")),
);
// 点选：回合活跃且自己未行动时可设置目标格；点击自己所在格 = 清除选择，
// 对方占用的格子不可作为目标（合法性最终由服务端校验）
function pick(row: number, col: number) {
  if (!canPick.value) return;
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
    <div class="grid" :class="{ picking: canPick }">
      <div
        v-for="c in cells"
        :key="`${c.row}-${c.col}`"
        class="cell"
        :class="{
          [occupant(c.row, c.col)]: true,
          target: isMoveTarget(c.row, c.col),
          pickable: canPick && !occupant(c.row, c.col),
        }"
        @click="pick(c.row, c.col)"
      >
        <span
          v-if="occupant(c.row, c.col)"
          class="pawn"
          :class="occupant(c.row, c.col)"
          :title="titleOf(occupant(c.row, c.col))"
          :style="{ background: classColorOf(occupant(c.row, c.col)) || undefined }"
        >
          <img v-if="classArtOf(occupant(c.row, c.col))" :src="classArtOf(occupant(c.row, c.col))" alt="" />
          <template v-else>{{ occupant(c.row, c.col).toUpperCase() }}</template>
        </span>
        <span v-if="isShielded(c.row, c.col)" class="shield" title="护盾中">🛡</span>
      </div>
    </div>
    <div class="legend">
      P1（你={{ store.viewerId === "p1" ? "是" : "否" }}）· P2（你={{ store.viewerId === "p2" ? "是" : "否" }}）
      <template v-if="canPick"> · 点选格子设定移动目标</template>
    </div>
  </div>
</template>

<style scoped>
.grid {
  /* 22 号票：格子用 min() 随视口收缩——宽屏恒为 56px，窄窗防横向溢出（唯一 @media
     只负责 .cols 纵向堆叠；扣减 96px = 页面留白 32 + 棋盘内衬/间隙 36 + 滚动条余量） */
  --cell-size: min(56px, calc((100vw - 96px) / 5));
  display: grid;
  grid-template-columns: repeat(5, var(--cell-size));
  grid-template-rows: repeat(5, var(--cell-size));
  gap: 4px;
  background: var(--cw-color-bg-surface);
  padding: 10px;
  border-radius: 10px;
}
.cell {
  background: var(--cw-color-bg-raised);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}
.cell.pickable { cursor: pointer; }
.cell.pickable:hover { background: var(--cw-color-border-strong); }
.cell.target { outline: 2px solid var(--cw-color-state-warning); background: var(--cw-color-bg-target); }
.pawn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--cw-color-text-primary);
  font-weight: 700;
  font-size: var(--cw-font-size-sm);
}
.pawn { overflow: hidden; }
.pawn img { width: 100%; height: 100%; object-fit: contain; }
.shield {
  position: absolute;
  top: 2px;
  right: 4px;
  font-size: var(--cw-font-size-xs);
}
.legend { color: var(--cw-color-text-secondary); font-size: var(--cw-font-size-xs); margin-top: 8px; }
</style>
