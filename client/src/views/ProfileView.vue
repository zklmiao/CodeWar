<script setup lang="ts">
// 个人页：用户名、当前 ELO、总场次/胜率、最近对局（结果 + ELO 变化）。
// 未登录提示去登录；数据来自 GET /api/profile。
import { onMounted, ref } from "vue";
import { apiProfile, type Profile } from "../api/auth";
import { useUserStore } from "../stores/user";

const user = useUserStore();
const profile = ref<Profile | null>(null);
const error = ref("");

const outcomeText: Record<string, string> = { win: "胜", loss: "负", draw: "平" };

function delta(r: { elo_before: number; elo_after: number }): string {
  const d = r.elo_after - r.elo_before;
  return d > 0 ? `+${d}` : `${d}`;
}

onMounted(async () => {
  if (!user.loggedIn) return;
  try {
    profile.value = await apiProfile();
  } catch (e) {
    error.value = (e as Error).message;
  }
});
</script>

<template>
  <main class="profile">
    <template v-if="!user.loggedIn">
      <p>登录后查看战绩与段位。<router-link to="/login">去登录</router-link></p>
    </template>
    <template v-else-if="profile">
      <h2>{{ profile.username }}</h2>
      <p class="elo">ELO {{ profile.elo }}</p>
      <p class="stats">
        共 {{ profile.total }} 场 · 胜 {{ profile.wins }} · 负 {{ profile.losses }} · 平 {{ profile.draws }}
        · 胜率 {{ (profile.win_rate * 100).toFixed(1) }}%
      </p>
      <h3>最近对局</h3>
      <p v-if="profile.recent.length === 0" class="empty">还没有已结算的对局——登录后与另一登录用户对战即结算 ELO。</p>
      <ul v-else class="recent">
        <li v-for="r in profile.recent" :key="r.match_id">
          <span class="outcome">{{ outcomeText[r.outcome] ?? r.outcome }}</span>
          <span>vs {{ r.opponent || "?" }}（{{ r.reason }}）</span>
          <span class="elo-delta">{{ r.elo_before }} → {{ r.elo_after }}（{{ delta(r) }}）</span>
        </li>
      </ul>
      <p v-if="error" class="error">{{ error }}</p>
    </template>
    <template v-else>
      <p>加载中…</p>
      <p v-if="error" class="error">{{ error }}</p>
    </template>
  </main>
</template>

<style scoped>
.profile { max-width: 640px; margin: 24px auto; color: #dfe4ee; }
.profile h2 { margin: 0; }
.elo { font-size: 22px; color: #7fd1a8; margin: 6px 0; }
.stats { color: #8a93a8; font-size: 14px; }
.profile h3 { margin-top: 20px; font-size: 16px; }
.empty { color: #56607a; font-size: 13px; }
.recent { list-style: none; padding: 0; }
.recent li {
  display: flex; gap: 12px; padding: 8px 12px; margin-bottom: 6px;
  background: #1a2030; border-radius: 8px; font-size: 14px;
}
.outcome { font-weight: bold; min-width: 1.2em; }
.elo-delta { margin-left: auto; color: #8a93a8; }
.error { color: #e08080; font-size: 13px; }
a { color: #7fd1a8; }
</style>
