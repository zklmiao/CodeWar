// 路由（07 号票）：/login、/register、/profile 三页；对局页保持 / 直达
// （/?player=N 不变，MatchView 读 route.query.player）。
import { createRouter, createWebHistory } from "vue-router";
import MatchView from "./views/MatchView.vue";
import LoginView from "./views/LoginView.vue";
import RegisterView from "./views/RegisterView.vue";
import ProfileView from "./views/ProfileView.vue";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: MatchView },
    { path: "/login", component: LoginView },
    { path: "/register", component: RegisterView },
    { path: "/profile", component: ProfileView },
  ],
});
