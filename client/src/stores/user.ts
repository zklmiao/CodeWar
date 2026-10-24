// 用户状态：令牌（sessionStorage 持久）+ 当前用户（启动时以令牌换取）。
// 对局页以本 store 的 token 随 join 上行；无 token = 匿名热座。
import { defineStore } from "pinia";
import {
  apiLogin,
  apiLogout,
  apiMe,
  apiRegister,
  apiTutorial,
  getToken,
  setToken,
  type Me,
  type TutorialProgress,
} from "../api/auth";

export const useUserStore = defineStore("user", {
  state: () => ({
    token: getToken(),
    user: null as Me | null,
    tutorial: null as TutorialProgress | null,
    lastError: "",
  }),
  getters: {
    loggedIn: (s) => s.user !== null,
  },
  actions: {
    applySession(token: string, user: Me) {
      this.token = token;
      setToken(token);
      this.user = user;
      this.tutorial = null;
      this.lastError = "";
    },
    clearSession() {
      this.token = "";
      setToken("");
      this.user = null;
      this.tutorial = null;
    },
    async register(username: string, password: string): Promise<void> {
      try {
        const s = await apiRegister(username, password);
        this.applySession(s.token, { id: s.id, username: s.username, elo: s.elo });
      } catch (e) {
        this.lastError = (e as Error).message;
        throw e;
      }
    },
    async login(username: string, password: string): Promise<void> {
      try {
        const s = await apiLogin(username, password);
        this.applySession(s.token, { id: s.id, username: s.username, elo: s.elo });
      } catch (e) {
        this.lastError = (e as Error).message;
        throw e;
      }
    },
    // 启动/刷新时恢复会话：token 无效即清掉并视为未登录（不抛）。
    async fetchMe(): Promise<boolean> {
      if (!this.token) {
        this.user = null;
        return false;
      }
      try {
        this.user = await apiMe();
        return true;
      } catch {
        this.clearSession();
        return false;
      }
    },
    // 10 号票：教程进度（入口页 PvP 门禁的展示源；服务端仍是权威）。
    // 拉取失败不清会话——不影响对局与匿名链路，只让入口页保持保守禁用。
    async fetchTutorial(): Promise<void> {
      if (!this.token) {
        this.tutorial = null;
        return;
      }
      try {
        this.tutorial = await apiTutorial();
      } catch {
        this.tutorial = null;
      }
    },
    async logout(): Promise<void> {
      try {
        await apiLogout();
      } finally {
        this.clearSession();
      }
    },
  },
});
