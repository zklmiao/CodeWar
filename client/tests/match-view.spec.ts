// 09 号票：入口页挂载级回归——客户端 store 测试捕获不到组件 setup 期的
// 运行时错误（曾因 playerFromUrl 早于 isPve 初始化触发 const TDZ，导致
// router-view 整体空白）。这里真实挂载 MatchView，钉住 PvE 区块与链接生成。
// @vitest-environment jsdom
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter } from "vue-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MatchView from "../src/views/MatchView.vue";

beforeEach(() => {
  setActivePinia(createPinia());
  // 职业数据在 onMounted 拉取；入口页测试不需要真实后端
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => ({ json: async () => ({ classes: [] }) })),
  );
});

afterEach(() => vi.unstubAllGlobals());

async function mountAt(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/", component: MatchView }],
  });
  await router.push(path);
  await router.isReady();
  return mount(MatchView, { global: { plugins: [router] } });
}

describe("09 人机练习入口", () => {
  it("渲染难度选择与 PvE 链接（默认普通）", async () => {
    const wrapper = await mountAt("/");
    expect(wrapper.find('[data-testid="pve-entry"]').exists()).toBe(true);
    const link = wrapper.find('[data-testid="join-pve"]');
    expect(link.exists()).toBe(true);
    expect(link.attributes("href")).toContain("mode=pve");
    expect(link.attributes("href")).toContain("bot=normal");
    expect(link.attributes("href")).toContain("player=1");
    wrapper.unmount();
  });

  it("切换难度后链接随选择更新", async () => {
    const wrapper = await mountAt("/");
    await wrapper.find('[data-testid="bot-difficulty"]').setValue("hard");
    const href = wrapper.find('[data-testid="join-pve"]').attributes("href") ?? "";
    expect(href).toContain("bot=hard");
    expect(wrapper.find('[data-testid="join-pve"]').text()).toContain("困难");
    wrapper.unmount();
  });

  it("PvE 链接沿用题型/难度上限/抽题策略；PvP 链接不带 mode", async () => {
    const wrapper = await mountAt("/?topic=hash&cap=L5&strategy=uniform");
    const pve = wrapper.find('[data-testid="join-pve"]').attributes("href") ?? "";
    expect(pve).toContain("topic=hash");
    expect(pve).toContain("cap=L5");
    expect(pve).toContain("strategy=uniform");
    const pvp = wrapper.find('[data-testid="join-p1"]').attributes("href") ?? "";
    expect(pvp).not.toContain("mode=");
    expect(pvp).toContain("player=1");
    wrapper.unmount();
  });
});

// 22 号票 review 修正：join-p2 此前在全套件零断言（票面称 match-view.spec.ts:59
// 依赖，实际该行只 pin join-p1）——P2 链接丢 player=2 或混入 mode= 时测试仍绿。
// 这里按外部行为契约补 pin：href 参数与复制按钮随卡片渲染。
describe("22 接入卡片链接契约", () => {
  it("join-p2 携带 player=2 且不带 mode；copy-link 随 P2 卡片渲染", async () => {
    const wrapper = await mountAt("/?topic=hash&cap=L5&strategy=uniform");
    const link = wrapper.find('[data-testid="join-p2"]');
    expect(link.exists()).toBe(true);
    const href = link.attributes("href") ?? "";
    expect(href).toContain("player=2");
    expect(href).not.toContain("mode=");
    expect(href).toContain("topic=hash");
    expect(href).toContain("cap=L5");
    expect(wrapper.find('[data-testid="copy-link"]').exists()).toBe(true);
    wrapper.unmount();
  });
});
