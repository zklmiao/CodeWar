// 账号 API：服务端不透明令牌 + sessionStorage（按标签页隔离，支持同机双账号热座）。
// 无 token = 匿名（对局照玩，只是不结算 ELO）；HttpOnly cookie 评估归 08 号票。
const TOKEN_KEY = "cw_token";

export function getToken(): string {
  return sessionStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(t: string): void {
  if (t) sessionStorage.setItem(TOKEN_KEY, t);
  else sessionStorage.removeItem(TOKEN_KEY);
}

async function req<T>(path: string, init?: RequestInit, token?: string): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const tk = token ?? getToken();
  if (tk) headers["Authorization"] = `Bearer ${tk}`;
  const res = await fetch(path, { ...init, headers });
  if (res.status === 204) return undefined as T;
  const data = (await res.json().catch(() => ({}))) as { error?: string };
  if (!res.ok) throw new Error(data.error ?? `请求失败（${res.status}）`);
  return data as T;
}

export interface Session {
  id: string;
  username: string;
  elo: number;
  token: string;
}

export interface Me {
  id: string;
  username: string;
  elo: number;
}

export interface RecentItem {
  match_id: string;
  finished_at: string;
  player: string; // 本场站位 p1/p2
  opponent: string;
  outcome: string; // win | loss | draw（查看者视角）
  reason: string;
  elo_before: number;
  elo_after: number;
}

export interface Profile {
  user_id: string;
  username: string;
  elo: number;
  total: number;
  wins: number;
  losses: number;
  draws: number;
  win_rate: number;
  recent: RecentItem[];
}

export const apiRegister = (username: string, password: string): Promise<Session> =>
  req<Session>("/api/register", { method: "POST", body: JSON.stringify({ username, password }) }, "");

export const apiLogin = (username: string, password: string): Promise<Session> =>
  req<Session>("/api/login", { method: "POST", body: JSON.stringify({ username, password }) }, "");

export const apiMe = (): Promise<Me> => req<Me>("/api/me");

export const apiProfile = (): Promise<Profile> => req<Profile>("/api/profile");

export const apiLogout = (): Promise<void> => req<void>("/api/logout", { method: "POST" });
