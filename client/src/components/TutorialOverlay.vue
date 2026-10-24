<script setup lang="ts">
// 10 号票：教程关卡完成页（只在服务端 completed 后显示）。
// 下一关用整页导航：教程对局刷新重开，进度由服务端 /api/tutorial 恢复。
import { computed } from "vue";
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
const t = computed(() => store.tutorial);

function nextHref(): string {
  const next = t.value?.nextLevel ?? 0;
  return next > 0 ? `/?mode=tutorial&level=${next}` : "/";
}
</script>

<template>
  <div v-if="t?.completed" class="overlay" data-testid="tutorial-complete">
    <div class="card">
      <h2>✅ {{ t.title }} 完成</h2>
      <ul>
        <li v-for="g in t.goals" :key="g.id" class="done">☑ {{ g.label }}</li>
      </ul>
      <p class="mechanic">{{ t.mechanic }}</p>
      <!-- 第 4 关教学难度收益：系数来自服务端（不在客户端复制数值表） -->
      <p v-if="t.level === 4" class="cap">
        本关难度上限 L{{ t.cap }}：普攻伤害系数 ×{{ t.capCoeff.toFixed(2) }}
      </p>
      <p v-if="t.allCompleted" class="unlock">🎉 新手教程全部完成，1v1 匹配已解锁。</p>
      <div class="actions">
        <a v-if="t.nextLevel > 0" :href="nextHref()" class="primary">继续第 {{ t.nextLevel }} 关</a>
        <a href="/" class="ghost">返回入口</a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(10, 13, 20, 0.82);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}
.card {
  background: var(--cw-color-bg-surface);
  border: 1px solid var(--cw-color-border-strong);
  padding: var(--cw-space-6) var(--cw-space-8);
  border-radius: var(--cw-radius-lg);
  text-align: left;
  color: var(--cw-color-text-primary);
  max-width: 560px;
}
h2 {
  font-size: var(--cw-font-size-xl);
  margin: 0 0 var(--cw-space-3);
  color: var(--cw-color-state-success);
}
ul {
  list-style: none;
  margin: 0;
  padding: 0;
}
li.done {
  color: var(--cw-color-state-success);
  font-size: var(--cw-font-size-sm);
  padding: var(--cw-space-1) 0;
}
.mechanic {
  color: var(--cw-color-text-secondary);
  font-size: var(--cw-font-size-sm);
  line-height: 1.6;
  border-top: 1px solid var(--cw-color-border);
  padding-top: var(--cw-space-3);
  margin: var(--cw-space-3) 0;
}
.cap {
  color: var(--cw-color-state-warning);
  font-size: var(--cw-font-size-sm);
}
.unlock {
  color: var(--cw-color-accent);
  font-size: var(--cw-font-size-sm);
}
.actions {
  display: flex;
  gap: var(--cw-space-3);
  margin-top: var(--cw-space-4);
}
.actions a {
  text-decoration: none;
  border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-4);
  font-size: var(--cw-font-size-sm);
}
.actions .primary {
  color: var(--cw-color-bg-base);
  background: var(--cw-color-state-success);
}
.actions .ghost {
  color: var(--cw-color-text-secondary);
  border: 1px solid var(--cw-color-border);
}
</style>
