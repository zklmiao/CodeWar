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
| array-sum-L1-attack | 数组与列表 | L1 | both | 否 | — |
| array-max-index-L3-attack | 数组与列表 | L3 | both | 否 | duplicates, single-element, all-negative |
| array-rotate-L4-attack | 数组与列表 | L4 | both | 否 | empty-input, k-larger-than-n, k-zero |
| array-kadane-L5-attack | 数组与列表 | L5 | both | 否 | empty-input, single-negative, all-negative, all-positive |
| array-subarray-sum-L5-attack | 数组与列表 | L5 | both | 否 | empty-input, negative-numbers, zero-k, no-match |
| hash-count-L2-attack | 哈希与计数 | L2 | both | 是 | empty-input, absent |
| hash-word-frequency-L3-attack | 哈希与计数 | L3 | both | 是 | repeated-max, single-distinct |
| hash-top-k-L4-attack | 哈希与计数 | L4 | both | 是 | tie-order, single-distinct, k-equals-distinct |
| hash-contains-L1-attack | 哈希与计数 | L1 | both | 否 | — |
| hash-pair-sum-L3-attack | 哈希与计数 | L3 | both | 否 | empty-input, no-pair, duplicates |
| hash-anagram-L4-attack | 哈希与计数 | L4 | both | 否 | different-length, empty-input, repeated-chars, no-match |
| hash-longest-consecutive-L5-attack | 哈希与计数 | L5 | both | 否 | empty-input, all-same, no-consecutive, negative |
| hash-min-steps-L5-attack | 哈希与计数 | L5 | both | 否 | empty-input, identical, no-overlap, one-empty |
| sort-parity-L2-attack | 排序与贪心 | L2 | both | 是 | empty-input |
| sort-kth-L3-attack | 排序与贪心 | L3 | both | 是 | single-element, duplicates |
| sort-insertion-L4-attack | 排序与贪心 | L4 | both | 是 | empty-input, single-element, already-sorted, reverse-sorted |
| sort-smallest-L1-attack | 排序与贪心 | L1 | both | 否 | — |
| sort-min-gap-L3-attack | 排序与贪心 | L3 | both | 否 | duplicates, adjacent-pair, unsorted-input |
| sort-merge-intervals-L4-attack | 排序与贪心 | L4 | both | 否 | empty-input, touching, contained, unsorted-input |
| sort-sorted-squares-L5-attack | 排序与贪心 | L5 | both | 否 | all-negative, all-positive, mixed, single-element |
| sort-min-arrows-L5-attack | 排序与贪心 | L5 | both | 否 | empty-input, touching-points, disjoint, single-balloon |
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
| move-nearest-L1-move | 数组与列表 | L1 | 否 | — |
| move-clamp-L3-move | 数组与列表 | L3 | 否 | empty-input, clamp-high, clamp-low |
| move-jump-L4-move | 数组与列表 | L4 | 否 | empty-input, stuck-zero, single-element |
| move-paths-L5-move | 数组与列表 | L5 | 否 | zero-input, single-row, one-by-one, single-col |
| move-lookup-L1-move | 哈希与计数 | L1 | 否 | — |
| move-keycount-L2-move | 哈希与计数 | L2 | 否 | empty-input, absent |
| move-first-repeat-L3-move | 哈希与计数 | L3 | 否 | empty-input, no-repeat, repeat-adjacent |
| move-favorite-L4-move | 哈希与计数 | L4 | 否 | empty-input, tie-order, single-distinct |
| move-schedule-L5-move | 哈希与计数 | L5 | 否 | empty-input, cooldown-zero, single-task, all-same |
| move-sortpick-L1-move | 排序与贪心 | L1 | 否 | — |
| move-cheapest-L2-move | 排序与贪心 | L2 | 否 | k-zero, empty-input |
| move-walkall-L3-move | 排序与贪心 | L3 | 否 | empty-input, single-element, gap |
| move-ranks-L4-move | 排序与贪心 | L4 | 否 | duplicates, single-element, all-same |
| move-events-L5-move | 排序与贪心 | L5 | 否 | empty-input, chained, disjoint, all-overlapping |

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
| defend-lowest-L1-defend | 数组与列表 | L1 | 否 | — |
| defend-count-L3-defend | 数组与列表 | L3 | 否 | empty-input, no-match, all-match |
| defend-bucket-L4-defend | 数组与列表 | L4 | 否 | empty-input, boundary-five, boundary-ten |
| defend-window-L5-defend | 数组与列表 | L5 | 否 | empty-input, k-larger-than-n, all-negative, k-equals-n |
| defend-membership-L1-defend | 哈希与计数 | L1 | 否 | — |
| defend-duplicate-L2-defend | 哈希与计数 | L2 | 否 | empty-input, all-same |
| defend-shared-L4-defend | 哈希与计数 | L4 | 否 | empty-input, no-overlap, duplicates-in-b |
| defend-majority-L5-defend | 哈希与计数 | L5 | 否 | empty-input, no-majority, all-same, exactly-half |
| defend-weakest-L1-defend | 排序与贪心 | L1 | 否 | — |
| defend-topsum-L2-defend | 排序与贪心 | L2 | 否 | k-zero, empty-input |
| defend-second-L3-defend | 排序与贪心 | L3 | 否 | single-distinct, empty-input, duplicates |
| defend-cluster-L4-defend | 排序与贪心 | L4 | 否 | empty-input, single-element, all-one-group, separated |
| defend-span-L5-defend | 排序与贪心 | L5 | 否 | empty-input, sorted-input, single-element, reverse-sorted |

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
| skill-orb-L1-skill | 数组与列表 | L1 | — |
| skill-charge-L3-skill | 数组与列表 | L3 | empty-input, below-threshold |
| skill-merge-L4-skill | 数组与列表 | L4 | empty-input, single-element, no-adjacent |
| skill-rob-L5-skill | 数组与列表 | L5 | empty-input, single-house, two-houses, all-zero |
| skill-ingredient-L1-skill | 哈希与计数 | L1 | — |
| skill-cost-L2-skill | 哈希与计数 | L2 | empty-input |
| skill-stockdiff-L4-skill | 哈希与计数 | L4 | empty-input, only-left, only-right |
| skill-craftreport-L5-skill | 哈希与计数 | L5 | empty-input, exact-match, deficit, surplus |
| skill-strongest-L1-skill | 排序与贪心 | L1 | — |
| skill-shield-L2-skill | 排序与贪心 | L2 | k-zero, empty-input |
| skill-closepairs-L3-skill | 排序与贪心 | L3 | empty-input, no-close, all-close |
| skill-medal-L4-skill | 排序与贪心 | L4 | empty-input, fewer-than-three, duplicates |
| skill-swaps-L5-skill | 排序与贪心 | L5 | empty-input, sorted-input, single-cycle, reverse-sorted |

## 档位覆盖

- 攻击：12 号票批 1-5 推进中（语法/字符串/数组/哈希题型已完成攻击矩阵，含关键格副本）；矩阵口径见 templates/coverage_test.go（108 有效格 / 关键格双模板 / 总量 124）。
- 移动/防御/技能：语法/字符串/数组/哈希/排序主题 L1-L5 已补齐（12 批 2-6）；剩余递归主题（批 6b）。
