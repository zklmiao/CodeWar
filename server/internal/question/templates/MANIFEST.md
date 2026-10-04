# 首批模板登记表（04 号票，26 个；L5 归 12 号票）

> 四维 = 题型 × 难度档 × 动作类型 × 题面形式；陷阱清单与隐藏用例一一对应（`checks.invariants` 强制）。
> 黄金样板 20 个（攻击 18 全覆盖 + 移动/防御各 1 模式样板）+ 仿写 6 个（移动 2、防御 2、技能 2）。

## 攻击题（18，形式 both：同一 Instance 双渲染）

| 模板 ID | 题型 | 难度 | 形式 | 黄金 | 陷阱 |
|---|---|---|---|---|---|
| syntax-add-L1-attack | 语法与运算 | L1 | both | 是 | — |
| syntax-sum-evens-L2-attack | 语法与运算 | L2 | both | 是 | empty-input |
| syntax-count-div-L3-attack | 语法与运算 | L3 | both | 是 | empty-input, no-match |
| string-palindrome-L2-attack | 字符串处理 | L2 | both | 是 | empty-input, case-sensitive |
| string-vowel-count-L3-attack | 字符串处理 | L3 | both | 是 | empty-input, no-vowel |
| string-run-length-L4-attack | 字符串处理 | L4 | both | 是 | empty-input, single-char, long-run |
| array-dedup-L2-attack | 数组与列表 | L2 | both | 是 | empty-input, all-same |
| array-two-sum-L3-attack | 数组与列表 | L3 | both | 是 | duplicates, negative |
| array-window-sum-L4-attack | 数组与列表 | L4 | both | 是 | empty-input, single-element, all-negative, full-window |
| hash-count-L2-attack | 哈希与计数 | L2 | both | 是 | empty-input, absent |
| hash-word-frequency-L3-attack | 哈希与计数 | L3 | both | 是 | repeated-max, single-distinct |
| hash-top-k-L4-attack | 哈希与计数 | L4 | both | 是 | tie-order, single-distinct, k-equals-distinct |
| sort-parity-L2-attack | 排序与贪心 | L2 | both | 是 | empty-input |
| sort-kth-L3-attack | 排序与贪心 | L3 | both | 是 | single-element, duplicates |
| sort-insertion-L4-attack | 排序与贪心 | L4 | both | 是 | empty-input, single-element, already-sorted, reverse-sorted |
| recursion-factorial-L2-attack | 递归与简单 DP | L2 | both | 是 | base-zero, base-one |
| recursion-binary-search-L3-attack | 递归与简单 DP | L3 | both | 是 | empty-input, single-element, not-found |
| recursion-pascal-L4-attack | 递归与简单 DP | L4 | both | 是 | row-zero, row-one |

注：递归题型 L1/L5 在 Rubric 下无语义（矩阵空语义格），归 12 号票。

## 移动题（3，私有，形式 complete）

| 模板 ID | 难度 | 黄金 | 陷阱 |
|---|---|---|---|
| move-manhattan-L1-move | L1 | 是 | — |
| move-step-L2-move | L2 | 否 | already-there |
| move-follow-L3-move | L3 | 否 | empty-input, invalid-char |

## 防御题（3，私有，形式 complete）

| 模板 ID | 难度 | 黄金 | 陷阱 |
|---|---|---|---|
| defend-reduce-L1-defend | L1 | 是 | — |
| defend-shield-L2-defend | L2 | 否 | empty-input, overkill |
| defend-survive-L3-defend | L3 | 否 | empty-input, lethal, full-block |

## 技能题（2，供 06 号票参照，形式 complete）

| 模板 ID | 难度 | 陷阱 |
|---|---|---|
| skill-blast-L3-skill | L3 | —（overcap/negative 为普通边界用例） |
| skill-heal-L3-skill | L3 | no-missing, over-heal |

## 档位覆盖

- 攻击：L1×1、L2×6、L3×6、L4×5；L5 缺失（12 号票补 DP/贪心）。
- 移动/防御：L1~L3；技能：L3 样板（技能题难度=所选上限档，06 号票消费）。
