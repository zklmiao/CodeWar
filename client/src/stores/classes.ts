// 职业数据（06 号票）：GET /api/classes 的同源 JSON（服务端 classes 包单一来源）。
// 纯取数 + 缓存；职业选择 UI 与 targeting 的形状/射程判断都从这里读。
import { defineStore } from "pinia";

export interface SkillView {
  id: string;
  name: string;
  kind: string; // damage | heal | drain
  power: number;
  heal: number;
  cooldown: number;
  shape: string; // melee | line | snipe | burst8 | band | self
  depth: number;
  rangeMax: number;
  // 形状语义（27 号票）：由服务端 classes 包唯一判定下发，客户端不再自行判形状。
  needsDirection: boolean;
  needsDistance: boolean;
}

export interface ClassView {
  id: string;
  name: string;
  hp: number;
  attackDamage: number;
  attackShape: string;
  attackDepth: number;
  attackRange: number;
  attackNeedsDirection: boolean; // 普攻形状语义（同上，服务端下发）
  attackNeedsDistance: boolean;
  moveMax: number;
  color: string;
  skills: SkillView[];
}

export const useClassesStore = defineStore("classes", {
  state: () => ({
    classes: [] as ClassView[],
    loaded: false,
    loading: false,
  }),
  getters: {
    byId: (s) => (id: string) => s.classes.find((c) => c.id === id),
  },
  actions: {
    // 幂等取数：多组件共用（职业选择、targeting 控件、棋盘渲染）
    async fetch() {
      if (this.loaded || this.loading) return;
      this.loading = true;
      try {
        const res = await fetch("/api/classes");
        const body = await res.json();
        this.classes = body.classes ?? [];
        this.loaded = true;
      } finally {
        this.loading = false;
      }
    },
  },
});
