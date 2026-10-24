<script setup lang="ts">
// 对局页（06 号票）：玩家选择 → 职业盲选（本地选完再 join）→ 接入 → 渲染对局。
// 职业与技能经 URL query 携带（?player=1&class=warrior&skill=cleave）——两个窗口
// 各自选择、互不可见（设计 003 §1）；缺省 = legacy 锚点（匿名/旧链接兼容）。
// 05 号票：对局配置表单（题型/难度上限/抽题策略）——配置随接入链接携带，
// 服务端按首个 join 锁定整场；「复制接入链接」供热座第二窗口直接粘贴。
// 断线/序号断裂：关闭连接重连，快照重建状态（ADR-0002）。
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
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
import TutorialGoals from "../components/TutorialGoals.vue";
import TutorialOverlay from "../components/TutorialOverlay.vue";

const store = useMatchStore();
const user = useUserStore();
const classesStore = useClassesStore();
const route = useRoute();
const disconnected = ref(false);
// 服务端 join 拒绝原因（close frame reason）：非空 = 不自动重连，直接展示。
const joinError = ref("");
// 09 号票：PvE 模式（?mode=pve&bot=easy|normal|hard）。必须在 playerFromUrl()
// 之前初始化——viewer 的初值会读 isPve（const TDZ 会导致整个组件渲染失败）。
// 10 号票：教程模式（?mode=tutorial&level=1..4[&cap=L3|L5]）复用同一单人拓扑。
const isPve = route.query.mode === "pve";
const isTutorial = route.query.mode === "tutorial";
const solo = isPve || isTutorial;
const tutorialLevel = ref(parseTutorialLevel(route.query.level));
const botDifficulty = ref(
  BOT_OPTIONS.some((b) => b.key === route.query.bot) ? (route.query.bot as string) : "normal",
);
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
function playerFromUrl(): string {
  // 单人模式（PvE/教程）人类固定 p1（链接里 player 可省略）
  if (solo) return "p1";
  const p = route.query.player;
  return p === "1" ? "p1" : p === "2" ? "p2" : "";
}

// 教程关号：URL 非法时回退第 1 关（服务端还是最终权威，会按进度再校验）。
function parseTutorialLevel(raw: unknown): number {
  const n = Number(raw);
  return Number.isInteger(n) && n >= 1 && n <= 4 ? n : 1;
}

// 入口页：教程 L4 难度上限（L3 ×1.00 / L5 ×1.20，缺省 L5——越高越疼）。
const tutorialCap = ref(route.query.cap === "L3" ? "L3" : "L5");

// 教程入口链接：已登录用户按服务端进度继续；未登录点击会引导登录。
const tutorialNext = computed(() => {
  const t = user.tutorial;
  if (!t || t.completed) return 1;
  return Math.min(t.total_levels, t.next_level || t.level + 1);
});
function tutorialHref(level: number): string {
  const q = new URLSearchParams();
  q.set("mode", "tutorial");
  q.set("level", String(level));
  if (level === 4) q.set("cap", tutorialCap.value);
  return `/?${q.toString()}`;
}
// PvP 门禁的客户端镜像：已登录但进度未知/未完成时保守禁用（服务端仍会校验）。
const pvpLocked = computed(() => user.loggedIn && (!user.tutorial || !user.tutorial.completed));

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
  if (pvpLocked.value) return; // 未完成教程：不复制一个必然被服务端拒绝的链接
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
        joinError.value = "";
        store.applyServerMessage(data, viewer.value);
      },
      onClose: (reason) => {
        disconnected.value = true;
        if (reason) {
          // 被服务端拒绝（教程门禁/会话失效/席位占用）：停止重连并展示原因。
          joinError.value = `接入被拒绝：${reason}`;
          return;
        }
        setTimeout(connect, 500); // 普通断线：接入即收快照，状态自愈
      },
    },
    {
      token: user.token,
      class: (route.query.class as string) || "",
      skill: (route.query.skill as string) || "",
      // 05 号票：配置随 join 注入（重连同参——服务端按首个 join 锁定，幂等）
      topic: isTutorial ? "" : cfgTopic.value,
      cap: isTutorial ? (tutorialLevel.value === 4 ? tutorialCap.value : "") : cfgCap.value,
      strategy: isTutorial ? "" : cfgStrategy.value,
      // 09 号票：PvE 模式与难度（人类固定 p1；服务端首个 join 锁定）
      mode: isTutorial ? "tutorial" : isPve ? "pve" : "",
      bot: isPve ? botDifficulty.value : "",
      // 10 号票：教程关号（服务端校验登录态与解锁进度）
      level: isTutorial ? String(tutorialLevel.value) : "",
    },
  );
}

onMounted(() => {
  if (viewer.value) connect();
  classesStore.fetch(); // 职业选择与 targeting 控件的同源数据
});

// 10 号票：登录态恢复后拉取教程进度（入口页门禁与"继续第 N 关"的数据源）。
watch(
  () => user.loggedIn,
  (v) => {
    if (v) user.fetchTutorial();
  },
  { immediate: true },
);

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
      <div v-if="joinError" class="banner error">{{ joinError }}（<a href="/">返回入口</a>）</div>
      <div v-else-if="disconnected" class="banner">连接断开，重连中…</div>
      <div class="cols">
        <GameGrid />
        <div class="right-col">
          <!-- 10 号票：教程目标清单（服务端事件驱动点亮；非教程模式不渲染） -->
          <TutorialGoals v-if="isTutorial" />
          <QuestionPanel :slots="isTutorial ? store.tutorial?.slots ?? [] : undefined" />
        </div>
      </div>
      <EventLog />
    </main>

    <main v-else class="choose">
      <p>本机热座对战：用两个浏览器窗口分别扮演双方。先配置对局，再选职业与技能（盲选——对方开赛前看不到）。</p>

      <!-- 10 号票：新手教程入口（登录后进度保存在账号；PvP 门禁的数据源） -->
      <div class="tutorial-entry" data-testid="tutorial-entry">
        <template v-if="!user.loggedIn">
          <b>新手教程</b>
          <span>登录后可开始 4 关交互式教程，进度自动保存。教程不要求匿名玩家，但也不会记录进度。</span>
          <a href="/login">登录后开始</a>
        </template>
        <template v-else-if="user.tutorial?.completed">
          <b>新手教程已完成 ✓</b>
          <span>1v1 匹配已解锁。可回炉重练：</span>
          <span class="tut-replay">
            <a v-for="n in 4" :key="n" :href="tutorialHref(n)">第 {{ n }} 关</a>
          </span>
        </template>
        <template v-else>
          <b>新手教程</b>
          <span>依次学习：读题作答 → 走位闪避 → 防御与技能冷却 → 难度收益。完成前 1v1 匹配保持锁定。</span>
          <a :href="tutorialHref(tutorialNext)" data-testid="join-tutorial">
            {{ (user.tutorial?.level ?? 0) > 0 ? `继续第 ${tutorialNext} 关` : "开始第 1 关" }}
          </a>
          <label>
            第 4 关难度上限
            <select v-model="tutorialCap" data-testid="tutorial-cap">
              <option value="L3">L3（伤害 ×1.00）</option>
              <option value="L5">L5（伤害 ×1.20，越高越疼）</option>
            </select>
          </label>
        </template>
      </div>

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

      <!-- 10 号票：已登录且未完成教程 → PvP 入口禁用（服务端仍会校验，客户端只是先行提示） -->
      <div v-if="pvpLocked" class="pvp-locked" data-testid="pvp-locked">
        🔒 先完成新手教程才能进入 1v1 匹配。
        <a :href="tutorialHref(tutorialNext)">去完成教程（第 {{ tutorialNext }} 关）</a>
      </div>
      <!-- 22 号票：P1/P2 接入升级为按钮卡片；「复制接入链接」并入 P2 卡（它复制的正是 P2 链接）。
           join-p1/join-p2 的 data-testid 与 href 语义保留（tests/match-view.spec.ts 依赖）。 -->
      <div v-else class="links">
        <a class="join-card" :href="joinHref('1')" data-testid="join-p1">
          <b>以 P1 接入</b>
          <small>本窗口先入座，等待对手加入</small>
        </a>
        <div class="join-card">
          <a class="join-main" :href="joinHref('2')" data-testid="join-p2">
            <b>以 P2 接入</b>
            <small>第二窗口入座，与本窗口热座对战</small>
          </a>
          <button class="copy" data-testid="copy-link" :disabled="pvpLocked" @click="copyJoinLink">
            {{ copied ? "已复制 ✓" : "复制接入链接（第二窗口）" }}
          </button>
        </div>
      </div>
    </main>

    <EndOverlay />
    <!-- 10 号票：教程通关完成页（下一关 / 返回入口；服务端 completed 才显示） -->
    <TutorialOverlay v-if="isTutorial" />
  </div>
</template>

<style>
.cols { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 14px; }
.banner {
  background: var(--cw-color-state-warning); color: var(--cw-color-bg-inset); padding: var(--cw-space-2) var(--cw-space-4);
  border-radius: var(--cw-radius-md); margin-bottom: var(--cw-space-3); font-size: var(--cw-font-size-xs);
}
.banner.error { background: var(--cw-color-state-danger); color: var(--cw-color-text-primary); }
.banner a { color: inherit; }
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
/* 10 号票：新手教程入口 + PvP 门禁提示 */
.right-col { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: var(--cw-space-3); }
.tutorial-entry {
  display: flex; gap: var(--cw-space-3); align-items: center; justify-content: center; flex-wrap: wrap;
  margin: var(--cw-space-4) auto 0; max-width: 720px; padding: var(--cw-space-3);
  background: var(--cw-color-bg-surface); border: 1px solid var(--cw-color-border);
  border-radius: var(--cw-radius-md); font-size: var(--cw-font-size-sm);
}
.tutorial-entry b { color: var(--cw-color-accent); }
.tutorial-entry a { color: var(--cw-color-state-success); text-decoration: none; }
.tutorial-entry label {
  display: flex; gap: var(--cw-space-2); align-items: center;
  color: var(--cw-color-text-muted); font-size: var(--cw-font-size-xs);
}
.tutorial-entry select {
  background: var(--cw-color-bg-inset); color: var(--cw-color-text-primary);
  border: 1px solid var(--cw-color-border); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-1) var(--cw-space-2); font-size: var(--cw-font-size-xs); cursor: pointer;
}
.tut-replay { display: flex; gap: var(--cw-space-3); }
.pvp-locked {
  margin: var(--cw-space-5) auto 0; max-width: 720px; padding: var(--cw-space-3);
  border: 1px dashed var(--cw-color-state-warning); border-radius: var(--cw-radius-md);
  color: var(--cw-color-state-warning); font-size: var(--cw-font-size-sm);
}
.pvp-locked a { color: var(--cw-color-accent); margin-left: var(--cw-space-2); }
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
/* 22 号票：P1/P2 接入按钮卡片（复制按钮并入 P2 卡） */
.links { display: flex; gap: var(--cw-space-4); justify-content: center; align-items: stretch; margin-top: var(--cw-space-5); }
.join-card {
  display: flex; flex-direction: column; gap: var(--cw-space-2); justify-content: center;
  background: var(--cw-color-bg-surface); border: 1px solid var(--cw-color-border);
  border-radius: var(--cw-radius-lg); padding: var(--cw-space-4); min-width: 240px; text-align: left;
}
a.join-card { text-decoration: none; cursor: pointer; }
a.join-card:hover { background: var(--cw-color-bg-raised); border-color: var(--cw-color-border-strong); }
.join-card b { color: var(--cw-color-state-success); font-size: var(--cw-font-size-lg); }
.join-card small { color: var(--cw-color-text-secondary); font-size: var(--cw-font-size-sm); }
.join-main { display: flex; flex-direction: column; gap: var(--cw-space-2); text-decoration: none; }
.join-main:hover b { text-decoration: underline; text-underline-offset: 4px; }
.join-card .copy {
  background: var(--cw-color-bg-inset); color: var(--cw-color-state-success);
  border: 1px solid var(--cw-color-state-success); border-radius: var(--cw-radius-sm);
  padding: var(--cw-space-2) var(--cw-space-3); cursor: pointer; font-size: var(--cw-font-size-sm);
}
.join-card .copy:hover { background: var(--cw-color-bg-raised); }
.join-card .copy:disabled { opacity: 0.5; cursor: not-allowed; }

/* 22 号票：全站唯一一条 @media——窄窗口防横向溢出的防崩守护（spec 范围外的移动端
   不做支持，见 001「范围外」）：.cols 纵向堆叠后日志自然跟随；棋盘格缩放由
   GameGrid 的 min(--cell-size) 无条件承担，不占媒体查询。 */
@media (max-width: 860px) {
  .cols { flex-direction: column; }
  .class-grid { flex-wrap: wrap; }
  .links { flex-wrap: wrap; }
}
</style>
