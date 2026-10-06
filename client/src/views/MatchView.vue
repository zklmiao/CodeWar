<script setup lang="ts">
// 对局页（06 号票）：玩家选择 → 职业盲选（本地选完再 join）→ 接入 → 渲染对局。
// 职业与技能经 URL query 携带（?player=1&class=warrior&skill=cleave）——两个窗口
// 各自选择、互不可见（设计 003 §1 盲选）；缺省 = legacy 锚点（匿名/旧链接兼容）。
// 断线/序号断裂：关闭连接重连，快照重建状态（ADR-0002）。
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useMatchStore } from "../stores/match";
import { useUserStore } from "../stores/user";
import { useClassesStore } from "../stores/classes";
import { socket } from "../api/socket";
import GameGrid from "../components/GameGrid.vue";
import HpBar from "../components/HpBar.vue";
import QuestionPanel from "../components/QuestionPanel.vue";
import EventLog from "../components/EventLog.vue";
import EndOverlay from "../components/EndOverlay.vue";

const store = useMatchStore();
const user = useUserStore();
const classesStore = useClassesStore();
const route = useRoute();
const disconnected = ref(false);
const viewer = ref(playerFromUrl());
// 盲选暂存（choose 屏用；进入对局后由 URL query 权威）
const pickedClass = ref((route.query.class as string) || "");
const pickedSkill = ref((route.query.skill as string) || "");

function playerFromUrl(): string {
  const p = route.query.player;
  return p === "1" ? "p1" : p === "2" ? "p2" : "";
}

function joinHref(player: string): string {
  const q = new URLSearchParams();
  q.set("player", player);
  if (pickedClass.value) q.set("class", pickedClass.value);
  if (pickedSkill.value) q.set("skill", pickedSkill.value);
  return `/?${q.toString()}`;
}

function connect() {
  socket.connect(
    viewer.value,
    {
      onMessage: (data) => {
        disconnected.value = false;
        store.applyServerMessage(data, viewer.value);
      },
      onClose: () => {
        disconnected.value = true;
        setTimeout(connect, 500); // 自动重连：接入即收快照，状态自愈
      },
    },
    {
      token: user.token,
      class: (route.query.class as string) || "",
      skill: (route.query.skill as string) || "",
    },
  );
}

onMounted(() => {
  if (viewer.value) connect();
  classesStore.fetch(); // 职业选择与 targeting 控件的同源数据
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
      <p>本机热座对战：用两个浏览器窗口分别扮演双方。先选职业与技能（盲选——对方开赛前看不到）。</p>
      <div class="class-grid">
        <button
          v-for="c in classesStore.classes"
          :key="c.id"
          class="class-card"
          :class="{ picked: pickedClass === c.id }"
          :style="{ borderColor: pickedClass === c.id ? c.color : undefined }"
          @click="pickedClass = c.id; pickedSkill = c.skills[0]?.id ?? ''"
        >
          <b :style="{ color: c.color }">{{ c.name }}</b>
          <small>HP {{ c.hp }} · 攻 {{ c.attackDamage }} · 移动 ≤{{ c.moveMax }}</small>
          <div class="skills">
            <button
              v-for="s in c.skills"
              :key="s.id"
              class="skill-pick"
              :class="{ on: pickedClass === c.id && pickedSkill === s.id }"
              @click.stop="pickedClass = c.id; pickedSkill = s.id"
            >
              {{ s.name }}（{{ s.kind === "heal" ? `治${s.heal}` : s.kind === "drain" ? `伤${s.power}+治${s.heal}` : `伤${s.power}` }} / CD{{ s.cooldown }}）
            </button>
          </div>
        </button>
      </div>
      <div class="links">
        <a :href="joinHref('1')" data-testid="join-p1">以 P1 接入</a>
        <a :href="joinHref('2')" data-testid="join-p2">以 P2 接入</a>
      </div>
    </main>

    <EndOverlay />
  </div>
</template>

<style>
.cols { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 14px; }
.banner {
  background: var(--cw-color-state-warning); color: var(--cw-color-bg-inset); padding: var(--cw-space-2) var(--cw-space-4);
  border-radius: var(--cw-radius-md); margin-bottom: var(--cw-space-3); font-size: var(--cw-font-size-xs);
}
.choose { text-align: center; color: var(--cw-color-text-secondary); padding: var(--cw-space-6) 0; }
.class-grid { display: flex; gap: var(--cw-space-3); justify-content: center; margin: var(--cw-space-5) 0; }
.class-card {
  background: var(--cw-color-bg-surface); border: 2px solid var(--cw-color-border); border-radius: var(--cw-radius-lg);
  padding: var(--cw-space-3); color: var(--cw-color-text-primary); cursor: pointer; display: flex;
  flex-direction: column; gap: var(--cw-space-2); min-width: 200px; text-align: left;
}
.class-card.picked { background: var(--cw-color-bg-raised); border-color: var(--cw-color-border-strong); }
.class-card small { color: var(--cw-color-text-muted); }
.skills { display: flex; flex-direction: column; gap: var(--cw-space-1); }
.skill-pick {
  background: var(--cw-color-bg-inset); border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-sm);
  color: var(--cw-color-text-secondary); padding: var(--cw-space-1) var(--cw-space-2); cursor: pointer; font-size: var(--cw-font-size-xs); text-align: left;
}
.skill-pick.on { border-color: var(--cw-color-state-success); color: var(--cw-color-text-primary); }
.links { display: flex; gap: var(--cw-space-6); justify-content: center; margin-top: var(--cw-space-5); }
.links a {
  color: var(--cw-color-state-success); text-decoration: none; font-size: var(--cw-font-size-lg);
  border: 1px solid var(--cw-color-border); padding: var(--cw-space-3) var(--cw-space-6); border-radius: var(--cw-radius-lg);
}
.links a:hover { background: var(--cw-color-bg-surface); }
</style>
