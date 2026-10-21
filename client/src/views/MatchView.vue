<script setup lang="ts">
// 对局页（06 号票）：玩家选择 → 职业盲选（本地选完再 join）→ 接入 → 渲染对局。
// 职业与技能经 URL query 携带（?player=1&class=warrior&skill=cleave）——两个窗口
// 各自选择、互不可见（设计 003 §1）；缺省 = legacy 锚点（匿名/旧链接兼容）。
// 05 号票：对局配置表单（题型/难度上限/抽题策略）——配置随接入链接携带，
// 服务端按首个 join 锁定整场；「复制接入链接」供热座第二窗口直接粘贴。
// 断线/序号断裂：关闭连接重连，快照重建状态（ADR-0002）。
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useMatchStore } from "../stores/match";
import { useUserStore } from "../stores/user";
import { useClassesStore } from "../stores/classes";
import { BOT_OPTIONS, CAP_OPTIONS, STRATEGY_OPTIONS, TOPIC_OPTIONS, botLabel } from "../topics";
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
// 05 号票：对局配置（表单值随链接携带；URL 带非法值时回退表单默认，
// 与服务端回退语义一致——服务端才是权威）
const cfgTopic = ref(
  TOPIC_OPTIONS.some((t) => t.key === route.query.topic) ? (route.query.topic as string) : "",
);
const cfgCap = ref(
  CAP_OPTIONS.includes(route.query.cap as string) ? (route.query.cap as string) : "L3",
);
const cfgStrategy = ref(
  STRATEGY_OPTIONS.some((s) => s.key === route.query.strategy)
    ? (route.query.strategy as string)
    : "climb",
);
// 09 号票：PvE 模式（?mode=pve&bot=easy|normal|hard）。人类固定 p1，对手为服务端 Bot。
const isPve = route.query.mode === "pve";
const botDifficulty = ref(
  BOT_OPTIONS.some((b) => b.key === route.query.bot) ? (route.query.bot as string) : "normal",
);

function playerFromUrl(): string {
  // PvE 人机练习人类固定 p1（链接里 player 可省略）
  if (isPve) return "p1";
  const p = route.query.player;
  return p === "1" ? "p1" : p === "2" ? "p2" : "";
}

// configParams：对局配置的 URL 参数（PvP 双窗口链接与 PvE 人机链接共用，
// 保证两种模式从同一套配置进入、服务端也走同一 question.Config 构造路径）。
function configParams(): URLSearchParams {
  const q = new URLSearchParams();
  if (pickedClass.value) q.set("class", pickedClass.value);
  if (pickedSkill.value) q.set("skill", pickedSkill.value);
  // 05 号票：配置随链接携带（topic 空 = 全部题型，省参；cap/strategy 恒携带）
  if (cfgTopic.value) q.set("topic", cfgTopic.value);
  q.set("cap", cfgCap.value);
  q.set("strategy", cfgStrategy.value);
  return q;
}

function joinHref(player: string): string {
  const q = configParams();
  q.set("player", player);
  return `/?${q.toString()}`;
}

// pveHref：人机练习接入链接（含当前难度与配置；player=p1 显式携带便于复制）。
function pveHref(): string {
  const q = configParams();
  q.set("player", "1");
  q.set("mode", "pve");
  q.set("bot", botDifficulty.value);
  return `/?${q.toString()}`;
}

// 复制接入链接（第二窗口 = P2）：clipboard 不可用（非安全上下文/旧浏览器）
// 时静默失败——下方「以 P2 接入」链接仍可右键复制
const copied = ref(false);
async function copyJoinLink() {
  try {
    await navigator.clipboard.writeText(location.origin + joinHref("2"));
    copied.value = true;
    setTimeout(() => (copied.value = false), 1500);
  } catch {
    copied.value = false;
  }
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
      // 05 号票：配置随 join 注入（重连同参——服务端按首个 join 锁定，幂等）
      topic: cfgTopic.value,
      cap: cfgCap.value,
      strategy: cfgStrategy.value,
      // 09 号票：PvE 模式与难度（人类固定 p1；服务端首个 join 锁定）
      mode: isPve ? "pve" : "",
      bot: isPve ? botDifficulty.value : "",
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
      <p>本机热座对战：用两个浏览器窗口分别扮演双方。先配置对局，再选职业与技能（盲选——对方开赛前看不到）。</p>

      <!-- 05 号票：对局配置表单（题型过滤只约束攻击题；难度上限档挂钩伤害与结算奖励） -->
      <div class="cfg-form" data-testid="match-config">
        <label>
          题型
          <select v-model="cfgTopic" data-testid="cfg-topic">
            <option v-for="t in TOPIC_OPTIONS" :key="t.key" :value="t.key">{{ t.label }}</option>
          </select>
        </label>
        <label>
          难度上限
          <select v-model="cfgCap" data-testid="cfg-cap">
            <option v-for="c in CAP_OPTIONS" :key="c" :value="c">{{ c }}</option>
          </select>
        </label>
        <label>
          抽题策略
          <select v-model="cfgStrategy" data-testid="cfg-strategy">
            <option v-for="s in STRATEGY_OPTIONS" :key="s.key" :value="s.key">{{ s.label }}</option>
          </select>
        </label>
        <button class="copy" data-testid="copy-link" @click="copyJoinLink">
          {{ copied ? "已复制 ✓" : "复制接入链接（第二窗口）" }}
        </button>
      </div>
      <p class="cfg-hint">难度上限越高，普攻伤害与结算奖励系数越高（L1 ×0.85 → L5 ×1.20）。</p>

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
      <!-- 09 号票：单人 PvE 练习（人类 p1 + 服务端 Bot；不落战绩/ELO） -->
      <div class="pve-entry" data-testid="pve-entry">
        <label>
          人机练习难度
          <select v-model="botDifficulty" data-testid="bot-difficulty">
            <option v-for="b in BOT_OPTIONS" :key="b.key" :value="b.key">{{ b.label }}</option>
          </select>
        </label>
        <a :href="pveHref()" data-testid="join-pve">开始人机练习（{{ botLabel(botDifficulty) }}）</a>
      </div>
      <p class="cfg-hint">人机练习使用与 PvP 相同的题型/难度上限/抽题策略；Bot 按难度决定解题速度与正确率。</p>

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
/* 05 号票：对局配置表单 */
.cfg-form {
  display: flex; gap: var(--cw-space-4); justify-content: center; align-items: flex-end;
  flex-wrap: wrap; margin: var(--cw-space-4) auto 0; max-width: 720px;
}
.cfg-form label {
  display: flex; flex-direction: column; gap: var(--cw-space-1); text-align: left;
  font-size: var(--cw-font-size-xs); color: var(--cw-color-text-muted);
}
.cfg-form select {
  background: var(--cw-color-bg-inset); color: var(--cw-color-text-primary);
  border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-3); font-size: var(--cw-font-size-sm); cursor: pointer;
}
.cfg-form .copy {
  background: var(--cw-color-bg-surface); color: var(--cw-color-state-success);
  border: 1px solid var(--cw-color-state-success); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-4); cursor: pointer; font-size: var(--cw-font-size-sm);
}
.cfg-form .copy:hover { background: var(--cw-color-bg-raised); }
.cfg-hint { font-size: var(--cw-font-size-xs); color: var(--cw-color-text-muted); margin: var(--cw-space-2) 0 0; }
/* 09 号票：人机练习入口 */
.pve-entry {
  display: flex; gap: var(--cw-space-4); justify-content: center; align-items: flex-end;
  flex-wrap: wrap; margin: var(--cw-space-4) auto 0; max-width: 720px;
}
.pve-entry label {
  display: flex; flex-direction: column; gap: var(--cw-space-1); text-align: left;
  font-size: var(--cw-font-size-xs); color: var(--cw-color-text-muted);
}
.pve-entry select {
  background: var(--cw-color-bg-inset); color: var(--cw-color-text-primary);
  border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-3); font-size: var(--cw-font-size-sm); cursor: pointer;
}
.pve-entry a {
  color: var(--cw-color-state-success); text-decoration: none; font-size: var(--cw-font-size-sm);
  border: 1px solid var(--cw-color-state-success); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-4);
}
.pve-entry a:hover { background: var(--cw-color-bg-raised); }
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
