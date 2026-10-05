<script setup lang="ts">
// 对局页：玩家选择（?player=1/2）→ 接入 → 渲染对局。
// 登录用户随 join 上行 token（无 token = 匿名热座）；终局 ELO 随 MatchEnded 下发。
// 断线/序号断裂：关闭连接重连，快照重建状态（ADR-0002）。
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useMatchStore } from "../stores/match";
import { useUserStore } from "../stores/user";
import { socket } from "../api/socket";
import GameGrid from "../components/GameGrid.vue";
import HpBar from "../components/HpBar.vue";
import QuestionPanel from "../components/QuestionPanel.vue";
import EventLog from "../components/EventLog.vue";
import EndOverlay from "../components/EndOverlay.vue";

const store = useMatchStore();
const user = useUserStore();
const route = useRoute();
const disconnected = ref(false);
// viewer 必须是响应式的（模板 v-if 依赖它）；URL 在 setup 时即确定
const viewer = ref(playerFromUrl());

function playerFromUrl(): string {
  const p = route.query.player;
  return p === "1" ? "p1" : p === "2" ? "p2" : "";
}

function connect() {
  socket.connect(viewer.value, {
    onMessage: (data) => {
      disconnected.value = false;
      store.applyServerMessage(data, viewer.value);
    },
    onClose: () => {
      disconnected.value = true;
      setTimeout(connect, 500); // 自动重连：接入即收快照，状态自愈
    },
  }, user.token);
}

onMounted(() => {
  if (viewer.value) connect();
});

onBeforeUnmount(() => socket.close());

// 序号断裂：主动重连触发快照重建（ADR-0002 事件流完整性规则）
watch(
  () => store.gapDetected,
  (g) => {
    if (g) {
      store.gapDetected = false;
      socket.close();
      connect();
    }
  },
);
</script>

<template>
  <div>
    <HpBar v-if="viewer" />

    <main v-if="viewer">
      <div v-if="disconnected" class="banner">连接断开，重连中…</div>
      <div class="cols">
        <GameGrid />
        <QuestionPanel />
      </div>
      <EventLog />
    </main>

    <main v-else class="choose">
      <p>本机热座对战：用两个浏览器窗口分别扮演双方。</p>
      <div class="links">
        <a href="/?player=1">以 P1 接入</a>
        <a href="/?player=2">以 P2 接入</a>
      </div>
    </main>

    <EndOverlay />
  </div>
</template>

<style>
.cols { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 14px; }
.banner {
  background: #4a3a20; color: #e0b45f; padding: 8px 14px;
  border-radius: 8px; margin-bottom: 12px; font-size: 13px;
}
.choose { text-align: center; color: #8a93a8; padding: 60px 0; }
.links { display: flex; gap: 24px; justify-content: center; margin-top: 20px; }
.links a {
  color: #7fd1a8; text-decoration: none; font-size: 18px;
  border: 1px solid #2c3650; padding: 12px 28px; border-radius: 10px;
}
.links a:hover { background: #1a2030; }
</style>
