// 用户 store 测试：注册/登录/恢复/退出全走 mock fetch（只测外部行为：token 落 sessionStorage、user 状态流转）。
// @vitest-environment jsdom
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useUserStore } from "../src/stores/user";

function mockFetchOnce(status: number, body: unknown) {
  (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    status,
    ok: status >= 200 && status < 300,
    json: async () => body,
  });
}

beforeEach(() => {
  setActivePinia(createPinia());
  sessionStorage.clear();
  vi.stubGlobal("fetch", vi.fn());
});

describe("用户状态机", () => {
  it("注册成功落 token 与用户", async () => {
    mockFetchOnce(201, { id: "u1", username: "alice", elo: 1000, token: "tok-1" });
    const s = useUserStore();
    await s.register("alice", "s3cret-pw");
    // 请求契约：路由与方法与服务端 accounts.Mount 一致（防两侧漂移）
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][0]).toBe("/api/register");
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe("POST");
    expect(s.loggedIn).toBe(true);
    expect(s.user?.username).toBe("alice");
    expect(s.token).toBe("tok-1");
    expect(sessionStorage.getItem("cw_token")).toBe("tok-1");
  });

  it("登录失败透出服务端错误并不留会话", async () => {
    mockFetchOnce(401, { error: "用户名或密码错误" });
    const s = useUserStore();
    await expect(s.login("alice", "wrong")).rejects.toThrow("用户名或密码错误");
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][0]).toBe("/api/login");
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe("POST");
    expect(s.loggedIn).toBe(false);
    expect(sessionStorage.getItem("cw_token")).toBeNull();
    expect(s.lastError).toBe("用户名或密码错误");
  });

  it("启动恢复：有效 token 换回用户，无效则清空", async () => {
    sessionStorage.setItem("cw_token", "good");
    mockFetchOnce(200, { id: "u1", username: "alice", elo: 1016 });
    const s = useUserStore();
    expect(await s.fetchMe()).toBe(true);
    expect(s.user?.elo).toBe(1016);

    sessionStorage.setItem("cw_token", "stale");
    mockFetchOnce(401, { error: "未登录或会话已过期" });
    const s2 = useUserStore();
    s2.token = "stale";
    expect(await s2.fetchMe()).toBe(false);
    expect(s2.loggedIn).toBe(false);
    expect(sessionStorage.getItem("cw_token")).toBeNull();
  });

  it("退出清会话（服务端失败也清本地）", async () => {
    sessionStorage.setItem("cw_token", "tok-1");
    mockFetchOnce(204, {});
    const s = useUserStore();
    s.token = "tok-1";
    s.user = { id: "u1", username: "alice", elo: 1000 };
    await s.logout();
    expect(s.loggedIn).toBe(false);
    expect(sessionStorage.getItem("cw_token")).toBeNull();
  });
});
