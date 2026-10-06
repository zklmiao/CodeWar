# 设计 002：出题系统——模板规范、难度标定、判题管线与工作量估算

> 依据：Spec 001 + 四项已确认决策（模板=混合形态 / 用例=可见2+隐藏5-8 / 标定=Rubric+抽样实测校准 / 生产=黄金样板法）。
> 边界：本文设计管线的**流程与契约**；沙箱**机制**（容器/资源限制实现）归 02 号票 ADR。

## 1. 模板模型（混合形态）

一个模板 = 一个模块，两部分组成：

**声明式 meta 头**（工具可索引、可自动审计）：

```yaml
id: hash-word-frequency-L3-attack
topic: 哈希与计数        # 6 题型之一
difficulty: L3           # L1~L5
action: attack           # attack | move | defend | skill
form: both               # complete | fill-blank | both（攻击模板默认 both）
slots:                   # 参数槽声明：名称、原语、约束
  text_length: int[8, 20]
  word_pool: pick[topic_pools.words]
invariants:              # 跨槽不变量，生成后校验
  - text_length >= 8
traps:                   # 声明的边界陷阱，必须与隐藏用例一一对应
  - empty-input
  - repeated-max
```

**代码生成器**：`generate(rng) → Instance`，纯函数（同 seed 同实例）。共享运行库提供槽位原语，L1-L3 模板基本零自写逻辑。

**Instance**（工厂产物，MatchEngine 与 Judge 的消费单元）：
`{ statement 题面, signature 函数签名, skeleton 填空骨架, visibleExamples[2], hiddenTests[5-8], answerKey 参考解, meta, seed }`

### 槽位原语清单（共享运行库）

| 原语 | 说明 |
|---|---|
| `rng.int / pick / sample / shuffle` | 种子化随机，全库唯一随机来源 |
| `names.variable(k) / names.function()` | 命名池，防实例趋同 |
| `data.list(size, range, opts)` | opts：distinct / sorted / mustContain——边界陷阱的数据来源 |
| `data.matrix / data.string` | 二维与字符串构造器 |
| `text.scenario(poolKey)` | 情境文本池（MVP 静态池；LLM 皮肤后续接入，只替换本原语） |
| `code.render(form)` | 双形式渲染（见 §2） |
| `checks.invariants(instance)` | 跨槽不变量与陷阱追溯校验，生成时强制执行 |

## 2. 题面形式：一模板双渲染

攻击模板不区分"补全函数版/填空版"两套模板——同一 Instance 双渲染：

- **complete**：签名 + 说明 + 可见样例，玩家写整个函数体；
- **fill-blank**：参考解中标记的 1-3 个**核心行**挖空（选概念负荷最高的行），其余骨架照给；隐藏用例与 complete 完全一致。

规则：挖空行总量 ≤3 行；挖空行必须是参考解的逻辑核心（工具按行复杂度校验，不许挖花括号式的假空）。这样填空版天然更"快"，服务于节奏变化；其收益平衡（伤害系数）归 combat 数值调参，不在本设计。

## 3. 测试用例策略

- 可见样例 **2 个**（题面自解释用）+ 隐藏用例 **5-8 个**（边界密集：空输入 / 单元素 / 极值 / 重复 / 类型边角）。
- 生效判定 = 可见 + 隐藏**全部通过**（spec 的"全对才生效"）。
- **陷阱追溯**：meta 声明的每个 trap 必须至少映射一个隐藏用例，`checks.invariants` 强制——没有陷阱用例的陷阱是装饰。
- **失败反馈粒度（防作弊默认，可调）**：提交失败时只返回 失败类别（wrong-answer / runtime-error / timeout / limit-exceeded / syntax），**不回传隐藏用例通过计数**——防止逐位探测隐藏测试。

## 4. 难度标定：Rubric 定档 + 抽样实测校准

**Rubric 映射表**（三维度定档，作者自标 + 校验工具机械复核）：

| 档位 | 概念负荷 C | 参考解步骤 S | 陷阱负荷 T | 期望解题耗时 |
|---|---|---|---|---|
| L1 | 1 个语法点 | ≤3 行 | 0 | ≤30s |
| L2 | 1-2 个语法点 | 3-5 行 | ≤1 | ~45s |
| L3 | 1 个算法思想 | 5-8 行 | 1-2 | ~60s |
| L4 | 算法思想 + 边界处理 | 8-12 行 | ≥2 | ~90s |
| L5 | ≥2 思想组合（DP/贪心/模拟） | 12-18 行 | ≥3 | ~120s |

工具机械复核项：参考解行数（S）、陷阱声明与隐藏用例的追溯完整、概念标签合法性。人工判定项：C（概念负荷）。

**校准循环**：上线后每档抽 10% 模板实测（≥3 名合资格解题者，测解题耗时 vs 档位期望）；漂移 >1 档的模板强制调档并回溯同批次。实测预算按 13 个模板/批次计入估算（§7）。

## 5. 判题管线（流程与契约）

```txt
QuestionFactory.create(spec, seed) ──→ Instance
   seed 分配（MatchEngine 持有 matchSeed）：
     镜像攻击题   roundSeed = f(matchSeed, round)        ← 双方同实例
     私有行动题   roundSeed = f(matchSeed, player, round) ← 双方独立
（整场对局可由 matchSeed 完整重放：测试复现与对局审计共用此性质）

玩家提交 → ① 静态卫兵（秒级 fail-fast）：语法检查、长度上限、
            import 白名单、禁用模式
         → ② 沙箱执行（机制归 02 号票 ADR）：先可见后隐藏、
            首败即停、时间/内存限制、断网
         → ③ verdict：{ passed, category }，durationMs = 解执行耗时
         → ④ 遥测上报（见 §6）
         → ⑤ SubmissionPort → MatchEngine 结算
```

**契约要点**：
- 判题只回 verdict 与执行耗时；**解题耗时**（时间系数的输入）由 MatchEngine 的 ClockPort 测量，与执行耗时是两个量——前者反映玩家，后者反映代码。
- 判题在服务端（spec 服务端权威）；客户端不参与判定。

## 6. 反作弊遥测点（本设计埋点，11 号票消费）

| 埋点 | 来源 | 用途 |
|---|---|---|
| 键入间隔序列、粘贴事件、失焦次数 | 编辑器 | 行为指纹（11 号票） |
| 执行耗时分布（全用例瞬时通过 = 硬编码嫌疑） | Judge | 硬编码检测 |
| 内存峰值、异常类型分布 | Judge | 沙箱滥用检测 |
| (spec, seed, verdict) 日志 | Factory/Judge | 审计与复现 |
| 每回合提交次数、首次提交耗时 | Engine | 节奏指纹 |

## 7. 模板矩阵与 MVP 工作量估算

**矩阵**：6 题型 × 5 档 × 4 动作类型 = 120 格；Rubric 判定语义为空的格子约 12 个（如 递归与简单DP×L1、语法与运算×L5）→ **有效格 ≈ 108**。攻击模板双渲染（§2），不因形式翻倍。关键格（攻击 × L3-L5，18 格）配 2 个模板 → **总量 ≈ 126 个模板**。

**口径 A：纯人工**（对照基线）

| 档位 | 数量 | 单模板工时 | 小计 |
|---|---|---|---|
| L1 | 15 | 1h | 15h |
| L2 | 20 | 1.5h | 30h |
| L3 | 35 | 2.5h | 87.5h |
| L4 | 30 | 4h | 120h |
| L5 | 26 | 6h | 156h |
| **合计** | 126 | | **≈ 409h ≈ 51 人日**（区间 45-60） |

**口径 B：黄金样板法（已选定）**

| 项 | 数量 | 工时 | 小计 |
|---|---|---|---|
| 黄金样板（人工精写：攻击覆盖全部 6×5 组合 + 移/防/技各 1 个模式样板） | ≈33 | 按档位加权 ≈3h | ≈93h |
| AI 仿写扩量 + 人工逐个审查（rubric 工具 + 判题冒烟 + 人工试解 1 实例） | ≈93 | L1 20min → L5 90min | ≈72h |
| 抽样实测校准（≈13 模板 × 3 人）+ 调档修复 | — | | ≈13h |
| 整合与修复预留（+15%） | | | ≈27h |
| **合计** | ≈126 | | **≈205h ≈ 26 人日**（区间 25-30） |

**对齐票据**：
- **04 号票**的模板内容部分（首批 25 个：12-15 个黄金样板 + 10-13 个 AI 仿写）≈ **5-6 人日**（工厂运行时与校验工具的工程量另计在 04 的工程范围）。
- **12 号票**剩余 ≈ **19-24 人日**，按题型分 6 批，每批 3-4 人日独立验收。
- 日历估算：单人口径 B 全职 ≈ 5 周；2-3 个审查者并行 ≈ 3 周。

## 8. 可调项（不阻塞，标注默认值）

1. 失败反馈粒度（§3 默认：只回类别不回计数）——playtest 观察挫败感再调。
2. 填空形式的收益平衡（spec 开放问题 1，伤害系数 ×0.85 待验证）。
3. 隐藏用例数量按题型微调（模拟类题可能需要更多用例）。
4. 情境文本池 → LLM 皮肤切换点已隔离在 `text.scenario` 原语。

## 附录 E：结构体类题格式范例（15 号票判题契约 v2，12 号票直接抄）

契约见 docs/adr/0004：TestCase 可选 `setup`（题作者 Python 代码，玩家 import 后、call 前执行，绑定结构体入参）与 `compare`（`def _normalize(got)` 函数源码，worker 用它把 got 规范化为 repr 字符串回传，宿主做字符串比对；expected 永不进容器）。空串 = 现行 eval/repr 路径。构造辅助库（`list_to_nodes`/`make_cycle`/`nodes_repr`/`build_tree`/`tree_repr`）随 payload 注入，setup/compare 内可直接调用。

### E.1 反转链表（比较器范式）

| 字段 | 值 |
|---|---|
| setup | `head = list_to_nodes([1,2,3,4,5])` |
| call | `reverseList(head)` |
| expected | `[5,4,3,2,1]`（= `nodes_repr` 的紧凑风格） |
| compare | `def _normalize(got):\n    return nodes_repr(got)` |

### E.2 环形链表 II（带环构造 + 节点身份比较）

| 字段 | 值 |
|---|---|
| setup | `head = list_to_nodes([3,2,0,-4])\nmake_cycle(head, 1)` |
| call | `detectCycle(head)` |
| expected | `cycle:[2,0,-4]`（从入口出发的规范环序；pos=1 时 3 不在环上——LC 官方语义） |
| compare | `def _normalize(got):\n    return nodes_repr(got)` |

身份语义：`nodes_repr` 按 id 走环，从入口出发的旋转序列在节点值唯一时即节点身份。非唯一值环需 id 传递机制（ADR-0004 遗留项）。

### E.3 层序遍历（树构造范式）

| 字段 | 值 |
|---|---|
| setup | `root = build_tree([3,9,20,None,None,15,7])` |
| call | `levelOrder(root)` |
| expected | `[[3], [9, 20], [15, 7]]`（原生 repr——玩家返回原生列表时无需 compare） |

### E.4 LRU 缓存（有状态调用序列范式）

| 字段 | 值 |
|---|---|
| setup | `obj = LRUCache(2)`（构造玩家的有状态对象） |
| call | `[obj.put(1, 1), obj.put(2, 2), obj.get(1), obj.put(3, 3), obj.get(2), obj.get(3)]`（**call 为表达式序列**：求值结果=每步返回值的列表） |
| expected | `[None, None, 1, None, -1, 3]`（整段序列的期望 repr） |

端到端样例（真实容器）见 `judge/tests/test_struct_contract.py` 的四个 `test_e2e_*` 用例。
