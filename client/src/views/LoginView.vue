<script setup lang="ts">
// 登录页：成功后进个人页。
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useUserStore } from "../stores/user";

const user = useUserStore();
const router = useRouter();
const username = ref("");
const password = ref("");
const error = ref("");

async function submit() {
  error.value = "";
  try {
    await user.login(username.value.trim(), password.value);
    router.push("/profile");
  } catch (e) {
    error.value = (e as Error).message;
  }
}
</script>

<template>
  <main class="auth">
    <h2>登录</h2>
    <form @submit.prevent="submit">
      <label>用户名<input v-model="username" autocomplete="username" /></label>
      <label>密码<input v-model="password" type="password" autocomplete="current-password" /></label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit">登录</button>
    </form>
    <p class="hint">没有账号？<router-link to="/register">去注册</router-link></p>
  </main>
</template>

<style scoped>
.auth { max-width: 360px; margin: 40px auto; color: #dfe4ee; }
.auth h2 { margin: 0 0 16px; }
.auth label { display: block; margin-bottom: 12px; font-size: 14px; color: #8a93a8; }
.auth input {
  display: block; width: 100%; margin-top: 4px; padding: 8px 10px;
  background: #1a2030; border: 1px solid #2c3650; border-radius: 8px; color: #dfe4ee;
}
.auth button {
  padding: 8px 28px; background: #2c5f46; color: #dfe4ee;
  border: none; border-radius: 8px; cursor: pointer; font-size: 15px;
}
.error { color: #e08080; font-size: 13px; }
.hint { color: #56607a; font-size: 13px; }
.hint a { color: #7fd1a8; }
</style>
