<script setup lang="ts">
// 三槽题面板（03 号票）：攻击（镜像竞速）/ 移动（私有 + 网格点选目标格）/ 防御（私有）。
// 各槽独立编辑器与提交按钮——提交即选择该槽对应的行动（一回合仅一题生效）。
import { computed, ref, watch } from "vue";
import { useMatchStore } from "../stores/match";
import { socket } from "../api/socket";
import CodeEditor from "./CodeEditor.vue";

const store = useMatchStore();

// 各槽独立编辑器内容：新题就绪时以 starter code 初始化
const codes = ref({ attack: "", move: "", defend: "" });
watch(
  () => store.panel,
  (p) => {
    codes.value = {
      attack: p?.attack?.starterCode ?? "",
      move: p?.move?.starterCode ?? "",
      defend: p?.defend?.starterCode ?? "",
    };
  },
);

const now = ref(Date.now());
setInterval(() => (now.value = Date.now()), 100);

const lockRemaining = computed(() =>
  Math.max(0, store.viewerLockUntil - (now.value + store.serverOffsetMs)),
);
const canSubmit = computed(
  () => store.isRoundActive && !store.viewerActed && lockRemaining.value <= 0,
);
const remainingSec = computed(() => Math.ceil(store.remainingMs / 1000));
const actedLabel = computed(() =>
  store.actedKind === "move" ? "移动" : store.actedKind === "defend" ? "防御" : "攻击",
);

function submitAttack() {
  if (!canSubmit.value || !store.panel?.attack) return;
  socket.submit("attack", codes.value.attack);
}
function submitMove() {
  if (!canSubmit.value || !store.panel?.move || !store.pendingMoveTo) return;
  socket.submit("move", codes.value.move, store.pendingMoveTo);
}
function submitDefend() {
  if (!canSubmit.value || !store.panel?.defend) return;
  socket.submit("defend", codes.value.defend);
}
</script>

<template>
  <div class="panel">
    <div v-if="!store.isRoundActive" class="placeholder">等待对局开始…</div>
    <template v-else>
      <div class="head">
        <span class="round">回合 {{ store.round }}</span>
        <span class="timer" :class="{ low: remainingSec <= 10 }">{{ remainingSec }}s</span>
        <span v-if="store.viewerActed" class="solved">已行动（{{ actedLabel }}），等待结算…</span>
        <span v-else-if="lockRemaining > 0" class="locked">锁定中 {{ Math.ceil(lockRemaining / 1000) }}s</span>
        <span v-else class="hint">三选一作答：本回合只生效一个行动</span>
      </div>

      <!-- 攻击槽：双方同题竞速 -->
      <section class="slot">
        <header><b>⚔ 攻击</b><small>镜像题 · 双方同题竞速，越快伤害越高</small></header>
        <pre v-if="store.panel?.attack" class="statement">{{ store.panel.attack.statement }}</pre>
        <code v-if="store.panel?.attack" class="signature">{{ store.panel.attack.signature }}</code>
        <CodeEditor v-if="store.panel?.attack" v-model="codes.attack" :reset-key="store.panel.attack.id" />
        <button class="submit atk" :disabled="!canSubmit" @click="submitAttack">提交攻击</button>
      </section>

      <!-- 移动槽：私有题 + 网格点选目标格 -->
      <section class="slot">
        <header><b>🏃 移动</b><small>私有题 · 解出后移动到所选格子（直线正交 1-2 格）</small></header>
        <pre v-if="store.panel?.move" class="statement">{{ store.panel.move.statement }}</pre>
        <code v-if="store.panel?.move" class="signature">{{ store.panel.move.signature }}</code>
        <CodeEditor v-if="store.panel?.move" v-model="codes.move" :reset-key="store.panel.move.id" />
        <div class="move-row">
          <span class="target">
            目标格：{{ store.pendingMoveTo ? `(${store.pendingMoveTo.row}, ${store.pendingMoveTo.col})` : "在左侧网格点击选择" }}
          </span>
          <button v-if="store.pendingMoveTo" class="clear" @click="store.setMoveTo(null)">清除</button>
        </div>
        <button class="submit mov" :disabled="!canSubmit || !store.pendingMoveTo" @click="submitMove">
          提交移动
        </button>
      </section>

      <!-- 防御槽：私有题 -->
      <section class="slot">
        <header><b>🛡 防御</b><small>私有题 · 解出后本回合减伤 60%</small></header>
        <pre v-if="store.panel?.defend" class="statement">{{ store.panel.defend.statement }}</pre>
        <code v-if="store.panel?.defend" class="signature">{{ store.panel.defend.signature }}</code>
        <CodeEditor v-if="store.panel?.defend" v-model="codes.defend" :reset-key="store.panel.defend.id" />
        <button class="submit def" :disabled="!canSubmit" @click="submitDefend">提交防御</button>
      </section>
    </template>
  </div>
</template>

<style scoped>
.panel {
  background: #1a2030; border-radius: 10px; padding: 14px;
  display: flex; flex-direction: column; gap: 12px;
}
.placeholder { color: #56607a; margin: auto; }
.head { display: flex; gap: 14px; align-items: center; flex-wrap: wrap; }
.round { font-weight: 700; color: #dfe4ee; }
.timer { font-variant-numeric: tabular-nums; color: #7fd1a8; font-weight: 700; }
.timer.low { color: #e07a6a; }
.solved, .locked, .hint { font-size: 13px; color: #e0b45f; }
.hint { color: #8a93a8; }
.slot {
  border: 1px solid #2c3650; border-radius: 8px; padding: 10px;
  display: flex; flex-direction: column; gap: 8px;
}
.slot header { display: flex; gap: 8px; align-items: baseline; }
.slot header b { color: #dfe4ee; }
.slot header small { color: #56607a; }
.statement {
  white-space: pre-wrap; color: #c7cede; font-size: 12px;
  margin: 0; font-family: inherit; max-height: 96px; overflow: auto;
}
.signature {
  display: block; background: #232b40; border-radius: 6px;
  padding: 4px 8px; color: #9fd0a0; font-size: 12px;
}
.move-row { display: flex; gap: 10px; align-items: center; font-size: 12px; color: #c7cede; }
.target { color: #e0b45f; }
.clear {
  background: none; border: 1px solid #2c3650; color: #8a93a8;
  border-radius: 6px; padding: 2px 8px; cursor: pointer;
}
.submit {
  padding: 8px 0; width: 100%; color: #fff; font-weight: 700;
  border: none; border-radius: 8px; cursor: pointer;
}
.submit.atk { background: #c2554f; }
.submit.mov { background: #3f6ea8; }
.submit.def { background: #4f8a5c; }
.submit:disabled { background: #333c52; color: #56607a; cursor: not-allowed; }
</style>
