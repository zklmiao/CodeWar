<script setup lang="ts">
// 应用壳：导航（对局 / 登录 / 注册 / 个人）+ 路由出口。
// 启动时以 sessionStorage 令牌恢复会话；对局页逻辑在 views/MatchView.vue。
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useUserStore } from "./stores/user";

const user = useUserStore();
const router = useRouter();
const route = useRoute();

// 23 号票：对局 viewer 形态放宽 App 壳到 ~1240px（1366 屏宽度增量主要给右栏题区），
// 入口（choose）/登录/注册/个人保持 960。判定与 MatchView.playerFromUrl 同源：
// solo 模式（pve/教程）人类固定 p1 恒为 viewer；PvP 看 player 参数。
const wide = computed(() => {
  if (route.path !== "/") return false;
  const q = route.query;
  return q.mode === "pve" || q.mode === "tutorial" || q.player === "1" || q.player === "2";
});

onMounted(() => {
  user.fetchMe();
});

async function logout() {
  await user.logout();
  router.push("/");
}
</script>

<template>
  <div class="page" :class="{ 'page--wide': wide }">
    <header>
      <h1>CodeWar · 代码战争 <small>07 号票 · 账号与 ELO</small></h1>
      <nav>
        <router-link to="/">对局</router-link>
        <template v-if="user.loggedIn">
          <router-link to="/profile">{{ user.user?.username }}（{{ user.user?.elo }}）</router-link>
          <a href="#" @click.prevent="logout">退出</a>
        </template>
        <template v-else>
          <router-link to="/login">登录</router-link>
          <router-link to="/register">注册</router-link>
        </template>
      </nav>
    </header>

    <router-view />
  </div>
</template>

<style>
* { box-sizing: border-box; }
/* body 背景对齐 --cw-color-bg-base（22 号票：原 body 裸背景色退役，token 为准） */
body { margin: 0; background: var(--cw-color-bg-base); font-family: var(--cw-font-sans); }
.page { max-width: 960px; margin: 0 auto; padding: 16px; }
/* 23 号票：对局 viewer 形态放宽到 ~1240px——1366 屏右栏 ~590px → ~860px */
.page--wide { max-width: 1240px; }
header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
h1 { font-size: var(--cw-font-size-lg); color: var(--cw-color-text-primary); margin: 0; }
h1 small { color: var(--cw-color-text-muted); font-size: var(--cw-font-size-xs); font-weight: 400; margin-left: 8px; }
nav { display: flex; gap: 16px; font-size: var(--cw-font-size-sm); }
nav a { color: var(--cw-color-text-secondary); text-decoration: none; }
nav a.router-link-active { color: var(--cw-color-state-success); }
</style>
