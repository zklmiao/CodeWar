<script setup lang="ts">
// App：玩家选择（?player=1/2）→ 接入 → 渲染对局。
// 断线/序号断裂：关闭连接重连，快照重建状态（ADR-0002）。
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useMatchStore } from "./stores/match";
import { socket } from "./api/socket";
import GameGrid from "./components/GameGrid.vue";
import HpBar from "./components/HpBar.vue";
import QuestionPanel from "./components/QuestionPanel.vue";
import EventLog from "./components/EventLog.vue";
import EndOverlay from "./components/EndOverlay.vue";

const store = useMatchStore();
const disconnected = ref(false);
// viewer 必须是响应式的（模板 v-if 依赖它）；URL 在 setup 时即确定
const viewer = ref(playerFromUrl());

function playerFromUrl(): string {
  const p = new URLSearchParams(location.search).get("player");
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
  });
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
  <div class="page">
    <header>
      <h1>CodeWar · 代码战争 <small>01 号票 · 热座 Tracer</small></h1>
      <HpBar v-if="viewer" />
    </header>

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
        <a href="?player=1">以 P1 接入</a>
        <a href="?player=2">以 P2 接入</a>
      </div>
    </main>

    <EndOverlay />
  </div>
</template>

<style>
* { box-sizing: border-box; }
body { margin: 0; background: #10141f; font-family: "Segoe UI", "Microsoft YaHei", sans-serif; }
.page { max-width: 960px; margin: 0 auto; padding: 16px; }
header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
h1 { font-size: 20px; color: #dfe4ee; margin: 0; }
h1 small { color: #56607a; font-size: 12px; font-weight: 400; margin-left: 8px; }
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
