<script setup lang="ts">
// 题卡 + CodeMirror 6 编辑器 + 提交（ADR-0001 前端栈；锁定与已解状态由服务端事件驱动）
import { computed, ref, watch } from "vue";
import { useMatchStore } from "../stores/match";
import { socket } from "../api/socket";
import CodeEditor from "./CodeEditor.vue";

const store = useMatchStore();
const code = ref("");
// 题目切换时，编辑器内容重置为新题的 starter code（而非残留上一题的作答）
watch(
  () => store.problem && store.problem.id,
  () => {
    code.value = (store.problem && store.problem.starterCode) || "";
  },
);
const now = ref(Date.now());
setInterval(() => (now.value = Date.now()), 100);

const lockRemaining = computed(() =>
  Math.max(0, store.viewerLockUntil - (now.value + store.serverOffsetMs)),
);
const canSubmit = computed(
  () =>
    store.isRoundActive &&
    !!store.problem &&
    !store.viewerSolved &&
    lockRemaining.value <= 0,
);
const remainingSec = computed(() => Math.ceil(store.remainingMs / 1000));

function submit() {
  if (!canSubmit.value) return;
  socket.submit(code.value);
}
</script>

<template>
  <div class="panel">
    <div v-if="!store.isRoundActive" class="placeholder">等待对局开始…</div>
    <template v-else>
      <div class="head">
        <span class="round">回合 {{ store.round }}</span>
        <span class="timer" :class="{ low: remainingSec <= 10 }">{{ remainingSec }}s</span>
        <span v-if="store.viewerSolved" class="solved">已解出，等待结算…</span>
        <span v-else-if="lockRemaining > 0" class="locked">锁定中 {{ Math.ceil(lockRemaining / 1000) }}s</span>
      </div>
      <div v-if="store.problem" class="body">
        <pre class="statement">{{ store.problem.statement }}</pre>
        <code class="signature">{{ store.problem.signature }}</code>
        <CodeEditor v-model="code" :reset-key="store.problem.id" />
        <button class="submit" :disabled="!canSubmit" @click="submit">提交攻击</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.panel {
  background: #1a2030; border-radius: 10px; padding: 14px;
  min-height: 380px; display: flex; flex-direction: column;
}
.placeholder { color: #56607a; margin: auto; }
.head { display: flex; gap: 14px; align-items: center; margin-bottom: 10px; }
.round { font-weight: 700; color: #dfe4ee; }
.timer { font-variant-numeric: tabular-nums; color: #7fd1a8; font-weight: 700; }
.timer.low { color: #e07a6a; }
.solved, .locked { font-size: 13px; color: #e0b45f; }
.statement {
  white-space: pre-wrap; color: #c7cede; font-size: 13px;
  margin: 0 0 10px; font-family: inherit;
}
.signature {
  display: block; background: #232b40; border-radius: 6px;
  padding: 6px 10px; color: #9fd0a0; font-size: 13px; margin-bottom: 10px;
}
.submit {
  margin-top: 12px; padding: 10px 0; width: 100%;
  background: #c2554f; color: #fff; font-weight: 700;
  border: none; border-radius: 8px; cursor: pointer;
}
.submit:disabled { background: #333c52; color: #56607a; cursor: not-allowed; }
</style>
