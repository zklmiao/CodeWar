// 6 大题型常量（05 号票，spec 001 用户故事 7）：配置表单与对局内只读展示共用。
// 服务端权威定义在 server/internal/question（Topic/TopicLabel），此处为展示端
// 镜像——题型 key 非法时服务端回退默认（全部题型），客户端只提供合法选择。
export interface TopicOption {
  key: string; // URL 参数 / join 契约形态（空 = 全部题型）
  label: string;
}

export const TOPIC_OPTIONS: TopicOption[] = [
  { key: "", label: "全部题型" },
  { key: "syntax", label: "语法与运算" },
  { key: "string", label: "字符串处理" },
  { key: "array", label: "数组与列表" },
  { key: "hash", label: "哈希与计数" },
  { key: "sort", label: "排序与贪心" },
  { key: "recursion", label: "递归与简单 DP" },
];

export const CAP_OPTIONS = ["L1", "L2", "L3", "L4", "L5"];

export const STRATEGY_OPTIONS = [
  { key: "climb", label: "爬坡（开局低档逐回合逼近上限）" },
  { key: "uniform", label: "均匀随机" },
];

export function topicLabel(key: string): string {
  return TOPIC_OPTIONS.find((t) => t.key === key)?.label ?? "全部题型";
}

export function strategyLabel(key: string): string {
  return key === "uniform" ? "均匀随机" : "爬坡";
}
