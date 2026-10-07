# 模板登记表（04 号票首批 26 + 12 号票扩容；矩阵口径见 templates/coverage_test.go）

> 四维 = 题型 × 难度档 × 动作类型 × 题面形式；陷阱清单与隐藏用例一一对应（`checks.invariants` 强制）。
> 黄金样板 20 个（攻击 18 全覆盖 + 移动/防御各 1 模式样板）+ 仿写 6 个（移动 2、防御 2、技能 2）。

## 攻击题（18，形式 both：同一 Instance 双渲染）

| 模板 ID | 题型 | 难度 | 形式 | 黄金 | 陷阱 |
|---|---|---|---|---|---|
| syntax-add-L1-attack | 语法与运算 | L1 | both | 是 | — |
| syntax-sum-evens-L2-attack | 语法与运算 | L2 | both | 是 | empty-input |
| syntax-count-div-L3-attack | 语法与运算 | L3 | both | 是 | empty-input, no-match |
| syntax-brackets-L4-attack | 语法与运算 | L4 | both | 是 | empty-input, wrong-order, unmatched-open, unmatched-close |
| syntax-reverse-digits-L3-attack | 语法与运算 | L3 | both | 是 | trailing-zero, negative, single-digit, zero |
| syntax-big-add-L4-attack | 语法与运算 | L4 | both | 是 | carry-chain, different-length, zero-value |
| syntax-brackets-L4-attack | 语法与运算 | L4 | both | 是 | empty-input, wrong-order, unmatched-open, unmatched-close |
| syntax-reverse-digits-L3-attack | 语法与运算 | L3 | both | 是 | trailing-zero, negative, single-digit, zero |
| syntax-big-add-L4-attack | 语法与运算 | L4 | both | 是 | carry-chain, different-length, zero-value |
| string-palindrome-L2-attack | 字符串处理 | L2 | both | 是 | empty-input, case-sensitive |
| string-vowel-count-L3-attack | 字符串处理 | L3 | both | 是 | empty-input, no-vowel |
| string-run-length-L4-attack | 字符串处理 | L4 | both | 是 | empty-input, single-char, long-run |
| string-palindrome-substr-L5-attack | 字符串处理 | L5 | both | 是 | empty-input, single-char, all-same, tie-first |
| string-palindrome-substr-L5-attack | 字符串处理 | L5 | both | 是 | empty-input, single-char, all-same, tie-first |
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

## 移动题（6，私有，形式 complete；12 批 2 起带题型标注）

| 模板 ID | 题型 | 难度 | 黄金 | 陷阱 |
|---|---|---|---|---|
| move-manhattan-L1-move | 语法与运算 | L1 | 是 | — |
| move-step-L2-move | 数组与列表 | L2 | 否 | already-there |
| move-follow-L3-move | 字符串处理 | L3 | 否 | empty-input, invalid-char |
| move-vector-L2-move | 语法与运算 | L2 | 否 | unknown-char |
| move-corridor-L3-move | 语法与运算 | L3 | 否 | empty-input, negative-steps, full-loop, wrap-around |
| move-parse-L4-move | 语法与运算 | L4 | 否 | empty-input, no-number, multi-digit |
| move-firststep-L1-move | 字符串处理 | L1 | 否 | — |
| move-count-forward-L2-move | 字符串处理 | L2 | 否 | empty-input, none-forward |
| move-facing-L4-move | 字符串处理 | L4 | 否 | empty-input, full-circle, unknown-char |
| move-beacon-L5-move | 字符串处理 | L5 | 否 | empty-input, undo-empty, undo-back, all-back |

## 防御题（6，私有，形式 complete；12 批 2 起带题型标注）

| 模板 ID | 题型 | 难度 | 黄金 | 陷阱 |
|---|---|---|---|---|
| defend-reduce-L1-defend | 语法与运算 | L1 | 是 | — |
| defend-shield-L2-defend | 数组与列表 | L2 | 否 | empty-input, overkill |
| defend-survive-L3-defend | 哈希与计数 | L3 | 否 | empty-input, lethal, full-block |
| defend-repair-L2-defend | 语法与运算 | L2 | 否 | over-cap, empty-input |
| defend-hp-after-L3-defend | 语法与运算 | L3 | 否 | empty-input, shield-eats-all, overkill, exact-zero |
| defend-siege-L4-defend | 语法与运算 | L4 | 否 | empty-input, full-block, degrade-chain, lethal |
| defend-badge-L1-defend | 字符串处理 | L1 | 否 | — |
| defend-mask-L2-defend | 字符串处理 | L2 | 否 | empty-input, short-input |
| defend-censor-L3-defend | 字符串处理 | L3 | 否 | empty-input, no-match, all-match |
| defend-scroll-L4-defend | 字符串处理 | L4 | 否 | empty-input, punctuation-only, mixed-case, odd-length |
| defend-barrier-L5-defend | 字符串处理 | L5 | 否 | empty-input, all-open, all-close, concatenated |

## 技能题（6，06 号票消费，形式 complete；L1/L2 为 12 号票批 1 补 cap<L3 缺口）

| 模板 ID | 题型 | 难度 | 陷阱 |
|---|---|---|---|
| skill-clamp-L1-skill | 语法与运算 | L1 | — |
| skill-chain-heal-L2-skill | 数组与列表 | L2 | empty-input |
| skill-blast-L3-skill | 语法与运算 | L3 | —（overcap/negative 为普通边界用例） |
| skill-heal-L3-skill | 哈希与计数 | L3 | no-missing, over-heal |
| skill-mana-L2-skill | 语法与运算 | L2 | over-cap |
| skill-arcana-L4-skill | 语法与运算 | L4 | zero-targets, decay-floor, overcap |
| skill-echo-L1-skill | 字符串处理 | L1 | — |
| skill-odd-L2-skill | 字符串处理 | L2 | empty-input, single-char |
| skill-draw-L3-skill | 字符串处理 | L3 | empty-input, no-digits |
| skill-decompress-L4-skill | 字符串处理 | L4 | empty-input, multi-digit-count, no-count |
| skill-ward-L5-skill | 字符串处理 | L5 | empty-input, single-char, all-same, whole-palindrome, worst-case |

## 档位覆盖

- 攻击：L1×1、L2×6、L3×6、L4×6（12 批1 +语法 L4）、L5×1（12 批1 +字符串）；矩阵口径见 templates/coverage_test.go（108 有效格 / 关键格双模板 / 总量 124）。
- 移动/防御：L1~L4 各题型推进中（12 批 2 补语法主题）；技能：L1/L2（12 批1）+ L3 样板 + L4（12 批 2）（技能题难度=所选上限档，06 号票消费）。
