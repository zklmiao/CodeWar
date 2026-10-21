// 结构体类攻击题（12 号票批 7：链表/树/LRU/调用序列，15 号票契约 v2）。
// 形式均为 complete（答案内嵌 class 定义，无法用单行签名做双渲染骨架；
// coverage_test.go 的形式门禁对攻击题允许 both 或 complete）。
// 四件套严格照设计 002 附录 E：meta 声明 / setup 构造入参 / compare 规范化 / expected 永不进容器。
package templates

import (
	"fmt"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

// ---- 链表 ----

// 链表 L3：反转链表（迭代三指针，参考解 8 行；附录 E.1 范式）。
func structReverseListL3() question.Template {
	meta := question.Meta{ID: "struct-reverse-list-L3-attack", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(4, 6), 1, 9, question.ListOpts{})
		kind := rng.Scenario("rune")
		sig := "def reverse_list(head):"
		answer := "def reverse_list(head):\n" +
			"    prev = None\n" +
			"    cur = head\n" +
			"    while cur:\n" +
			"        nxt = cur.next\n" +
			"        cur.next = prev\n" +
			"        prev = cur\n" +
			"        cur = nxt\n" +
			"    return prev\n"
		visible := []matchCase{
			{name: "example-1", call: "reverse_list(head)", expected: nodesListRepr(reverseListGo(nums)), setup: listSetup("head", nums, -1), compare: nodeCompare()},
			{name: "example-2", call: "reverse_list(head)", expected: "[5,4,3,2,1]", setup: listSetup("head", []int{1, 2, 3, 4, 5}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "reverse_list(head)", expected: "None", setup: listSetup("head", []int{}, -1), compare: nodeCompare()},
			{name: "single-node", call: "reverse_list(head)", expected: "[7]", setup: listSetup("head", []int{7}, -1), compare: nodeCompare()},
			{name: "duplicates", call: "reverse_list(head)", expected: nodesListRepr(reverseListGo([]int{2, 2, 3})), setup: listSetup("head", []int{2, 2, 3}, -1), compare: nodeCompare()},
			{name: "two-nodes", call: "reverse_list(head)", expected: "[2,1]", setup: listSetup("head", []int{1, 2}, -1), compare: nodeCompare()},
			{name: "long", call: "reverse_list(head)", expected: nodesListRepr(reverseListGo([]int{1, 2, 3, 4, 5, 6})), setup: listSetup("head", []int{1, 2, 3, 4, 5, 6}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s铭链翻转：返回反转后的链表头（空链返回 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def reverse_list(head):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 链表 L3：删除排序链表重复元素（相邻去重，参考解 7 行）。
func structDeleteDupsL3() question.Template {
	meta := question.Meta{ID: "struct-delete-dups-L3-attack", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "all-same", "last-pair"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(4, 7), 1, 6, question.ListOpts{Sorted: true})
		kind := rng.Scenario("rune")
		sig := "def delete_duplicates(head):"
		answer := "def delete_duplicates(head):\n" +
			"    cur = head\n" +
			"    while cur is not None and cur.next is not None:\n" +
			"        if cur.val == cur.next.val:\n" +
			"            cur.next = cur.next.next\n" +
			"        else:\n" +
			"            cur = cur.next\n" +
			"    return head\n"
		visible := []matchCase{
			{name: "example-1", call: "delete_duplicates(head)", expected: nodesListRepr(dedupeSortedGo(nums)), setup: listSetup("head", nums, -1), compare: nodeCompare()},
			{name: "example-2", call: "delete_duplicates(head)", expected: "[1,2]", setup: listSetup("head", []int{1, 1, 2}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "delete_duplicates(head)", expected: "None", setup: listSetup("head", []int{}, -1), compare: nodeCompare()},
			{name: "all-same", call: "delete_duplicates(head)", expected: "[2]", setup: listSetup("head", []int{2, 2, 2}, -1), compare: nodeCompare()},
			{name: "no-dups", call: "delete_duplicates(head)", expected: "[1,2,3]", setup: listSetup("head", []int{1, 2, 3}, -1), compare: nodeCompare()},
			{name: "last-pair", call: "delete_duplicates(head)", expected: "[1,2]", setup: listSetup("head", []int{1, 2, 2}, -1), compare: nodeCompare()},
			{name: "single", call: "delete_duplicates(head)", expected: "[5]", setup: listSetup("head", []int{5}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s整理铭链：删除排序链表中所有重复元素（保留一个），返回头节点（空链返回 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def delete_duplicates(head):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 链表 L5：合并两个有序链表（哑节点 + 双指针，参考解 15 行含 ListNode 定义）。
func structMergeListsL5() question.Template {
	meta := question.Meta{ID: "struct-merge-lists-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "different-length", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a := rng.IntList(rng.Int(2, 4), 1, 9, question.ListOpts{Sorted: true})
		b := rng.IntList(rng.Int(2, 4), 1, 9, question.ListOpts{Sorted: true})
		kind := rng.Scenario("rune")
		sig := "class ListNode:"
		answer := "class ListNode:\n" +
			"    def __init__(self, val=0, nxt=None):\n" +
			"        self.val = val\n" +
			"        self.next = nxt\n\n" +
			"def merge_two_lists(a, b):\n" +
			"    dummy = ListNode()\n" +
			"    tail = dummy\n" +
			"    while a and b:\n" +
			"        if a.val <= b.val:\n" +
			"            tail.next = a\n" +
			"            a = a.next\n" +
			"        else:\n" +
			"            tail.next = b\n" +
			"            b = b.next\n" +
			"        tail = tail.next\n" +
			"    tail.next = a if a else b\n" +
			"    return dummy.next\n"
		starter := "class ListNode:\n    def __init__(self, val=0, nxt=None):\n        pass\n\n" +
			"def merge_two_lists(a, b):\n    # 在这里补全实现\n    raise NotImplementedError\n"
		setup := listSetup("a", a, -1) + "\n" + listSetup("b", b, -1)
		visible := []matchCase{
			{name: "example-1", call: "merge_two_lists(a, b)", expected: nodesListRepr(mergeSortedGo(a, b)), setup: setup, compare: nodeCompare()},
			{name: "example-2", call: "merge_two_lists(a, b)", expected: "[1,2,3,4]", setup: listSetup("a", []int{1, 3}, -1) + "\n" + listSetup("b", []int{2, 4}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "merge_two_lists(a, b)", expected: "None", setup: listSetup("a", []int{}, -1) + "\n" + listSetup("b", []int{}, -1), compare: nodeCompare()},
			{name: "different-length", call: "merge_two_lists(a, b)", expected: "[1,2,3,4,5]", setup: listSetup("a", []int{1, 4, 5}, -1) + "\n" + listSetup("b", []int{2, 3}, -1), compare: nodeCompare()},
			{name: "duplicates", call: "merge_two_lists(a, b)", expected: "[1,1,2,2]", setup: listSetup("a", []int{1, 2}, -1) + "\n" + listSetup("b", []int{1, 2}, -1), compare: nodeCompare()},
			{name: "one-empty", call: "merge_two_lists(a, b)", expected: "[1,2]", setup: listSetup("a", []int{}, -1) + "\n" + listSetup("b", []int{1, 2}, -1), compare: nodeCompare()},
			{name: "all-left-smaller", call: "merge_two_lists(a, b)", expected: "[1,2,3,4]", setup: listSetup("a", []int{1, 2}, -1) + "\n" + listSetup("b", []int{3, 4}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s接续铭链：合并两个升序链表为一个升序链表，返回头节点（空链为 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// 链表 L4：环形链表判定（快慢指针，参考解 8 行）。
func structHasCycleL4() question.Template {
	meta := question.Meta{ID: "struct-has-cycle-L4-attack", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "no-cycle"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 5), 1, 9, question.ListOpts{})
		pos := rng.Int(-1, len(nums)-1)
		kind := rng.Scenario("rune")
		sig := "def has_cycle(head):"
		answer := "def has_cycle(head):\n" +
			"    slow = head\n" +
			"    fast = head\n" +
			"    while fast is not None and fast.next is not None:\n" +
			"        slow = slow.next\n" +
			"        fast = fast.next.next\n" +
			"        if slow is fast:\n" +
			"            return True\n" +
			"    return False\n"
		visible := []matchCase{
			{name: "example-1", call: "has_cycle(head)", expected: pyBool(pos >= 0), setup: listSetup("head", nums, pos)},
			{name: "example-2", call: "has_cycle(head)", expected: "True", setup: listSetup("head", []int{3, 2, 0, -4}, 1)},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "has_cycle(head)", expected: "False", setup: listSetup("head", []int{}, -1)},
			{name: "single-node", call: "has_cycle(head)", expected: "False", setup: listSetup("head", []int{7}, -1)},
			{name: "no-cycle", call: "has_cycle(head)", expected: "False", setup: listSetup("head", []int{1, 2, 3}, -1)},
			{name: "cycle-at-head", call: "has_cycle(head)", expected: "True", setup: listSetup("head", []int{1, 2, 3}, 0)},
			{name: "cycle-at-tail", call: "has_cycle(head)", expected: "True", setup: listSetup("head", []int{1, 2, 3, 4}, 2)},
		}
		stmt := fmt.Sprintf("%s铭环探测：判断链表中是否存在环（空链/单节点无环为 False）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def has_cycle(head):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 链表 L5：环形链表入口（Floyd 相遇后重定位，参考解 12 行；附录 E.2 语义）。
func structDetectCycleL5() question.Template {
	meta := question.Meta{ID: "struct-detect-cycle-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-cycle", "cycle-at-head", "single-cycle"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 5), 1, 9, question.ListOpts{})
		pos := rng.Int(-1, len(nums)-1)
		kind := rng.Scenario("rune")
		sig := "def detect_cycle(head):"
		answer := "def detect_cycle(head):\n" +
			"    slow = head\n" +
			"    fast = head\n" +
			"    while fast is not None and fast.next is not None:\n" +
			"        slow = slow.next\n" +
			"        fast = fast.next.next\n" +
			"        if slow is fast:\n" +
			"            entry = head\n" +
			"            while entry is not slow:\n" +
			"                entry = entry.next\n" +
			"                slow = slow.next\n" +
			"            return entry\n" +
			"    return None\n"
		visible := []matchCase{
			{name: "example-1", call: "detect_cycle(head)", expected: cycleRepr(nums, pos), setup: listSetup("head", nums, pos), compare: nodeCompare()},
			{name: "example-2", call: "detect_cycle(head)", expected: "cycle:[2,0,-4]", setup: listSetup("head", []int{3, 2, 0, -4}, 1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "detect_cycle(head)", expected: "None", setup: listSetup("head", []int{}, -1), compare: nodeCompare()},
			{name: "no-cycle", call: "detect_cycle(head)", expected: "None", setup: listSetup("head", []int{1, 2, 3}, -1), compare: nodeCompare()},
			{name: "cycle-at-head", call: "detect_cycle(head)", expected: cycleRepr([]int{1, 2, 3}, 0), setup: listSetup("head", []int{1, 2, 3}, 0), compare: nodeCompare()},
			{name: "single-cycle", call: "detect_cycle(head)", expected: cycleRepr([]int{7}, 0), setup: listSetup("head", []int{7}, 0), compare: nodeCompare()},
			{name: "mid-cycle", call: "detect_cycle(head)", expected: cycleRepr([]int{1, 2, 3, 4}, 2), setup: listSetup("head", []int{1, 2, 3, 4}, 2), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s铭环入口：若有环返回环入口节点（按节点身份），无环返回 None（空链为 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def detect_cycle(head):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 链表 L5：两数相加（数字逆序链表 + 进位链，参考解 16 行含 ListNode 定义）。
func structAddTwoL5() question.Template {
	meta := question.Meta{ID: "struct-add-two-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"carry-chain", "different-length", "zero-value"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a := rng.IntList(rng.Int(1, 4), 0, 9, question.ListOpts{})
		b := rng.IntList(rng.Int(1, 4), 0, 9, question.ListOpts{})
		a[len(a)-1] = rng.Int(1, 9) // 最高位非 0（数字逆序存储）
		b[len(b)-1] = rng.Int(1, 9)
		kind := rng.Scenario("rune")
		sig := "class ListNode:"
		answer := "class ListNode:\n" +
			"    def __init__(self, val=0, nxt=None):\n" +
			"        self.val = val\n" +
			"        self.next = nxt\n\n" +
			"def add_two_numbers(a, b):\n" +
			"    dummy = ListNode()\n" +
			"    tail = dummy\n" +
			"    carry = 0\n" +
			"    while a or b or carry:\n" +
			"        x = a.val if a else 0\n" +
			"        y = b.val if b else 0\n" +
			"        total = x + y + carry\n" +
			"        carry = total // 10\n" +
			"        tail.next = ListNode(total % 10)\n" +
			"        tail = tail.next\n" +
			"        a = a.next if a else None\n" +
			"        b = b.next if b else None\n" +
			"    return dummy.next\n"
		starter := "class ListNode:\n    def __init__(self, val=0, nxt=None):\n        pass\n\n" +
			"def add_two_numbers(a, b):\n    # 在这里补全实现\n    raise NotImplementedError\n"
		setup := listSetup("a", a, -1) + "\n" + listSetup("b", b, -1)
		visible := []matchCase{
			{name: "example-1", call: "add_two_numbers(a, b)", expected: nodesListRepr(addDigitsGo(a, b)), setup: setup, compare: nodeCompare()},
			{name: "example-2", call: "add_two_numbers(a, b)", expected: "[7,0,8]", setup: listSetup("a", []int{2, 4, 3}, -1) + "\n" + listSetup("b", []int{5, 6, 4}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "carry-chain", call: "add_two_numbers(a, b)", expected: "[0,0,0,1]", setup: listSetup("a", []int{9, 9, 9}, -1) + "\n" + listSetup("b", []int{1}, -1), compare: nodeCompare()},
			{name: "different-length", call: "add_two_numbers(a, b)", expected: nodesListRepr(addDigitsGo([]int{9, 9}, []int{1})), setup: listSetup("a", []int{9, 9}, -1) + "\n" + listSetup("b", []int{1}, -1), compare: nodeCompare()},
			{name: "zero-value", call: "add_two_numbers(a, b)", expected: "[0]", setup: listSetup("a", []int{0}, -1) + "\n" + listSetup("b", []int{0}, -1), compare: nodeCompare()},
			{name: "single-digit", call: "add_two_numbers(a, b)", expected: "[0,1]", setup: listSetup("a", []int{5}, -1) + "\n" + listSetup("b", []int{5}, -1), compare: nodeCompare()},
			{name: "no-carry", call: "add_two_numbers(a, b)", expected: "[4,6]", setup: listSetup("a", []int{1, 2}, -1) + "\n" + listSetup("b", []int{3, 4}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s双数铭链：两个非负整数按逆序存于链表（个位在头），返回和的逆序链表。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// 链表 L4：链表排序（取值数组排序后回填，参考解 11 行）。
func structSortListL4() question.Template {
	meta := question.Meta{ID: "struct-sort-list-L4-attack", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 9, question.ListOpts{})
		kind := rng.Scenario("rune")
		sig := "def sort_list(head):"
		answer := "def sort_list(head):\n" +
			"    values = []\n" +
			"    cur = head\n" +
			"    while cur:\n" +
			"        values.append(cur.val)\n" +
			"        cur = cur.next\n" +
			"    values.sort()\n" +
			"    cur = head\n" +
			"    for v in values:\n" +
			"        cur.val = v\n" +
			"        cur = cur.next\n" +
			"    return head\n"
		visible := []matchCase{
			{name: "example-1", call: "sort_list(head)", expected: nodesListRepr(sortedGo(nums)), setup: listSetup("head", nums, -1), compare: nodeCompare()},
			{name: "example-2", call: "sort_list(head)", expected: "[1,2,3]", setup: listSetup("head", []int{3, 1, 2}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "sort_list(head)", expected: "None", setup: listSetup("head", []int{}, -1), compare: nodeCompare()},
			{name: "single-element", call: "sort_list(head)", expected: "[5]", setup: listSetup("head", []int{5}, -1), compare: nodeCompare()},
			{name: "duplicates", call: "sort_list(head)", expected: nodesListRepr(sortedGo([]int{4, 2, 2, 4})), setup: listSetup("head", []int{4, 2, 2, 4}, -1), compare: nodeCompare()},
			{name: "already-sorted", call: "sort_list(head)", expected: "[1,2,3]", setup: listSetup("head", []int{1, 2, 3}, -1), compare: nodeCompare()},
			{name: "reverse-sorted", call: "sort_list(head)", expected: "[1,2,3,4]", setup: listSetup("head", []int{4, 3, 2, 1}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s铭链排序：将链表升序排序并返回头节点（空链返回 None；可直接交换节点值）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def sort_list(head):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 链表 L4：删除倒数第 n 个节点（快慢指针 + 头部边界，参考解 11 行）。
func structRemoveNthL4() question.Template {
	meta := question.Meta{ID: "struct-remove-nth-L4-attack", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"remove-head", "remove-tail", "single-element"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 9, question.ListOpts{})
		n := rng.Int(1, len(nums))
		kind := rng.Scenario("rune")
		sig := "def remove_nth_from_end(head, n):"
		answer := "def remove_nth_from_end(head, n):\n" +
			"    fast = head\n" +
			"    for _ in range(n):\n" +
			"        fast = fast.next\n" +
			"    if fast is None:\n" +
			"        return head.next\n" +
			"    slow = head\n" +
			"    while fast.next:\n" +
			"        fast = fast.next\n" +
			"        slow = slow.next\n" +
			"    slow.next = slow.next.next\n" +
			"    return head\n"
		starter := "def remove_nth_from_end(head, n):\n    # 在这里补全实现\n    raise NotImplementedError\n"
		visible := []matchCase{
			{name: "example-1", call: "remove_nth_from_end(head, n)", expected: nodesListRepr(removeNthGo(nums, n)), setup: listSetup("head", nums, -1) + fmt.Sprintf("\nn = %d", n), compare: nodeCompare()},
			{name: "example-2", call: "remove_nth_from_end(head, n)", expected: "[1,2,3,5]", setup: listSetup("head", []int{1, 2, 3, 4, 5}, -1) + "\nn = 2", compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "single-element", call: "remove_nth_from_end(head, n)", expected: "None", setup: listSetup("head", []int{5}, -1) + "\nn = 1", compare: nodeCompare()},
			{name: "remove-head", call: "remove_nth_from_end(head, n)", expected: "[2,3]", setup: listSetup("head", []int{1, 2, 3}, -1) + "\nn = 3", compare: nodeCompare()},
			{name: "remove-tail", call: "remove_nth_from_end(head, n)", expected: "[1,2]", setup: listSetup("head", []int{1, 2, 3}, -1) + "\nn = 1", compare: nodeCompare()},
			{name: "middle", call: "remove_nth_from_end(head, n)", expected: "[1,2,3,5]", setup: listSetup("head", []int{1, 2, 3, 4, 5}, -1) + "\nn = 2", compare: nodeCompare()},
			{name: "two-nodes", call: "remove_nth_from_end(head, n)", expected: "[1]", setup: listSetup("head", []int{1, 2}, -1) + "\nn = 1", compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s断链术：删除链表倒数第 n 个节点并返回头节点（n 保证合法；空结果返回 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// 链表 L5：重排链表（节点数组 + 首尾交替，参考解 18 行）。
func structReorderL5() question.Template {
	meta := question.Meta{ID: "struct-reorder-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element", "even-length", "odd-length"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 9, question.ListOpts{})
		kind := rng.Scenario("rune")
		sig := "def reorder_list(head):"
		answer := "def reorder_list(head):\n" +
			"    if not head:\n" +
			"        return None\n" +
			"    nodes = []\n" +
			"    cur = head\n" +
			"    while cur:\n" +
			"        nodes.append(cur)\n" +
			"        cur = cur.next\n" +
			"    left = 0\n" +
			"    right = len(nodes) - 1\n" +
			"    while left < right:\n" +
			"        nodes[left].next = nodes[right]\n" +
			"        left += 1\n" +
			"        if left == right:\n" +
			"            break\n" +
			"        nodes[right].next = nodes[left]\n" +
			"        right -= 1\n" +
			"    nodes[left].next = None\n" +
			"    return head\n"
		starter := "def reorder_list(head):\n    # 在这里补全实现\n    raise NotImplementedError\n"
		visible := []matchCase{
			{name: "example-1", call: "reorder_list(head)", expected: nodesListRepr(reorderGo(nums)), setup: listSetup("head", nums, -1), compare: nodeCompare()},
			{name: "example-2", call: "reorder_list(head)", expected: "[1,5,2,4,3]", setup: listSetup("head", []int{1, 2, 3, 4, 5}, -1), compare: nodeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "reorder_list(head)", expected: "None", setup: listSetup("head", []int{}, -1), compare: nodeCompare()},
			{name: "single-element", call: "reorder_list(head)", expected: "[7]", setup: listSetup("head", []int{7}, -1), compare: nodeCompare()},
			{name: "even-length", call: "reorder_list(head)", expected: nodesListRepr(reorderGo([]int{1, 2, 3, 4})), setup: listSetup("head", []int{1, 2, 3, 4}, -1), compare: nodeCompare()},
			{name: "odd-length", call: "reorder_list(head)", expected: nodesListRepr(reorderGo([]int{1, 2, 3, 4, 5})), setup: listSetup("head", []int{1, 2, 3, 4, 5}, -1), compare: nodeCompare()},
			{name: "two-nodes", call: "reorder_list(head)", expected: "[1,2]", setup: listSetup("head", []int{1, 2}, -1), compare: nodeCompare()},
		}
		stmt := fmt.Sprintf("%s交错铭链：把链表重排为 L0→Ln→L1→Ln−1→…（原地修改并返回头节点；空链返回 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// ---- 树 ----

var treePoolBalanced = []any{3, 9, 20, nil, nil, 15, 7}
var treePoolSkewLeft = []any{1, 2, nil, 3}
var treePoolSkewRight = []any{1, nil, 2, nil, 3}
var treePoolWide = []any{4, 2, 6, 1, 3, 5, 7}

// 树 L2：最大深度（递归基线 + 左右取大，参考解 5 行）。
func structMaxDepthL2() question.Template {
	meta := question.Meta{ID: "struct-max-depth-L2-attack", Topic: question.TopicRecursion,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := [][]any{treePoolBalanced, treePoolSkewLeft, treePoolSkewRight, treePoolWide}
		picked := pool[rng.Int(0, len(pool)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def max_depth(root):"
		answer := "def max_depth(root):\n" +
			"    if root is None:\n" +
			"        return 0\n" +
			"    left = max_depth(root.left)\n" +
			"    right = max_depth(root.right)\n" +
			"    return (left if left > right else right) + 1\n"
		visible := []matchCase{
			{name: "example-1", call: "max_depth(root)", expected: fmt.Sprintf("%d", maxDepthGo(buildTreeGo(picked))), setup: treeSetup("root", picked)},
			{name: "example-2", call: "max_depth(root)", expected: "3", setup: treeSetup("root", treePoolBalanced)},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "max_depth(root)", expected: "0", setup: treeSetup("root", []any{})},
			{name: "single-node", call: "max_depth(root)", expected: "1", setup: treeSetup("root", []any{5})},
			{name: "skew-left", call: "max_depth(root)", expected: fmt.Sprintf("%d", maxDepthGo(buildTreeGo(treePoolSkewLeft))), setup: treeSetup("root", treePoolSkewLeft)},
			{name: "skew-right", call: "max_depth(root)", expected: fmt.Sprintf("%d", maxDepthGo(buildTreeGo(treePoolSkewRight))), setup: treeSetup("root", treePoolSkewRight)},
			{name: "balanced", call: "max_depth(root)", expected: "3", setup: treeSetup("root", treePoolBalanced)},
		}
		stmt := fmt.Sprintf("%s树渊深度：返回二叉树的最大深度（空树为 0）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def max_depth(root):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 树 L3：中序遍历（递归遍历 + 收集，参考解 8 行）。
func structInorderL3() question.Template {
	meta := question.Meta{ID: "struct-inorder-L3-attack", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "skew"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := [][]any{treePoolBalanced, treePoolSkewLeft, treePoolSkewRight, treePoolWide}
		picked := pool[rng.Int(0, len(pool)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def inorder(root):"
		answer := "def inorder(root):\n" +
			"    result = []\n" +
			"    def walk(node):\n" +
			"        if node is None:\n" +
			"            return\n" +
			"        walk(node.left)\n" +
			"        result.append(node.val)\n" +
			"        walk(node.right)\n" +
			"    walk(root)\n" +
			"    return result\n"
		inorderVals := func(vals []any) []int {
			var out []int
			inorderGo(buildTreeGo(vals), &out)
			return out
		}
		visible := []matchCase{
			{name: "example-1", call: "inorder(root)", expected: pyList(inorderVals(picked)), setup: treeSetup("root", picked)},
			{name: "example-2", call: "inorder(root)", expected: pyList(inorderVals(treePoolWide)), setup: treeSetup("root", treePoolWide)},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "inorder(root)", expected: "[]", setup: treeSetup("root", []any{})},
			{name: "single-node", call: "inorder(root)", expected: "[5]", setup: treeSetup("root", []any{5})},
			{name: "skew-left", call: "inorder(root)", expected: pyList(inorderVals(treePoolSkewLeft)), setup: treeSetup("root", treePoolSkewLeft)},
			{name: "skew-right", call: "inorder(root)", expected: pyList(inorderVals(treePoolSkewRight)), setup: treeSetup("root", treePoolSkewRight)},
			{name: "balanced", call: "inorder(root)", expected: pyList(inorderVals(treePoolBalanced)), setup: treeSetup("root", treePoolBalanced)},
		}
		stmt := fmt.Sprintf("%s树铭中序：返回二叉树的中序遍历值列表（空树为空列表）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def inorder(root):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 树 L3：翻转二叉树（递归交换 + 返回根，参考解 6 行）。
func structInvertL3() question.Template {
	meta := question.Meta{ID: "struct-invert-tree-L3-attack", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "skew"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := [][]any{treePoolBalanced, treePoolSkewLeft, treePoolSkewRight, treePoolWide}
		picked := pool[rng.Int(0, len(pool)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def invert_tree(root):"
		answer := "def invert_tree(root):\n" +
			"    if root is None:\n" +
			"        return None\n" +
			"    root.left, root.right = root.right, root.left\n" +
			"    invert_tree(root.left)\n" +
			"    invert_tree(root.right)\n" +
			"    return root\n"
		invertRepr := func(vals []any) string {
			return treeReprGo(invertTreeGo(buildTreeGo(vals)))
		}
		visible := []matchCase{
			{name: "example-1", call: "invert_tree(root)", expected: invertRepr(picked), setup: treeSetup("root", picked), compare: treeCompare()},
			{name: "example-2", call: "invert_tree(root)", expected: invertRepr(treePoolWide), setup: treeSetup("root", treePoolWide), compare: treeCompare()},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "invert_tree(root)", expected: treeReprGo(nil), setup: treeSetup("root", []any{}), compare: treeCompare()},
			{name: "single-node", call: "invert_tree(root)", expected: "[5]", setup: treeSetup("root", []any{5}), compare: treeCompare()},
			{name: "skew-left", call: "invert_tree(root)", expected: invertRepr(treePoolSkewLeft), setup: treeSetup("root", treePoolSkewLeft), compare: treeCompare()},
			{name: "skew-right", call: "invert_tree(root)", expected: invertRepr(treePoolSkewRight), setup: treeSetup("root", treePoolSkewRight), compare: treeCompare()},
			{name: "balanced", call: "invert_tree(root)", expected: invertRepr(treePoolBalanced), setup: treeSetup("root", treePoolBalanced), compare: treeCompare()},
		}
		stmt := fmt.Sprintf("%s镜像树境：翻转二叉树（左右子树互换），返回翻转后的根节点（空树返回 None）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def invert_tree(root):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 树 L4：层序遍历（逐层收集 + 推导式，参考解 9 行）。
func structLevelOrderL4() question.Template {
	meta := question.Meta{ID: "struct-level-order-L4-attack", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "skew"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := [][]any{treePoolBalanced, treePoolSkewLeft, treePoolSkewRight, treePoolWide}
		picked := pool[rng.Int(0, len(pool)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def level_order(root):"
		answer := "def level_order(root):\n" +
			"    if root is None:\n" +
			"        return []\n" +
			"    result = []\n" +
			"    current = [root]\n" +
			"    while current:\n" +
			"        result.append([node.val for node in current])\n" +
			"        nxt = [n for node in current for n in (node.left, node.right) if n]\n" +
			"        current = nxt\n" +
			"    return result\n"
		levelRepr := func(vals []any) string {
			return pyNestedInts(levelOrderGo(buildTreeGo(vals)))
		}
		visible := []matchCase{
			{name: "example-1", call: "level_order(root)", expected: levelRepr(picked), setup: treeSetup("root", picked)},
			{name: "example-2", call: "level_order(root)", expected: levelRepr(treePoolBalanced), setup: treeSetup("root", treePoolBalanced)},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "level_order(root)", expected: "[]", setup: treeSetup("root", []any{})},
			{name: "single-node", call: "level_order(root)", expected: "[[5]]", setup: treeSetup("root", []any{5})},
			{name: "skew-left", call: "level_order(root)", expected: levelRepr(treePoolSkewLeft), setup: treeSetup("root", treePoolSkewLeft)},
			{name: "skew-right", call: "level_order(root)", expected: levelRepr(treePoolSkewRight), setup: treeSetup("root", treePoolSkewRight)},
			{name: "balanced", call: "level_order(root)", expected: levelRepr(treePoolBalanced), setup: treeSetup("root", treePoolBalanced)},
		}
		stmt := fmt.Sprintf("%s树层铭文：返回二叉树的层序遍历（每层一个值列表；空树为空列表）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def level_order(root):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 树 L4：右视图（逐层取最右 + 推导式，参考解 9 行）。
func structRightViewL4() question.Template {
	meta := question.Meta{ID: "struct-right-view-L4-attack", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-node", "skew"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := [][]any{treePoolBalanced, treePoolSkewLeft, treePoolSkewRight, treePoolWide}
		picked := pool[rng.Int(0, len(pool)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def right_view(root):"
		answer := "def right_view(root):\n" +
			"    if root is None:\n" +
			"        return []\n" +
			"    result = []\n" +
			"    current = [root]\n" +
			"    while current:\n" +
			"        result.append(current[-1].val)\n" +
			"        nxt = [n for node in current for n in (node.left, node.right) if n]\n" +
			"        current = nxt\n" +
			"    return result\n"
		viewRepr := func(vals []any) string {
			return pyList(rightViewGo(buildTreeGo(vals)))
		}
		visible := []matchCase{
			{name: "example-1", call: "right_view(root)", expected: viewRepr(picked), setup: treeSetup("root", picked)},
			{name: "example-2", call: "right_view(root)", expected: viewRepr(treePoolBalanced), setup: treeSetup("root", treePoolBalanced)},
		}
		hidden := []matchCase{
			{name: "empty-input", call: "right_view(root)", expected: "[]", setup: treeSetup("root", []any{})},
			{name: "single-node", call: "right_view(root)", expected: "[5]", setup: treeSetup("root", []any{5})},
			{name: "skew-left", call: "right_view(root)", expected: viewRepr(treePoolSkewLeft), setup: treeSetup("root", treePoolSkewLeft)},
			{name: "skew-right", call: "right_view(root)", expected: viewRepr(treePoolSkewRight), setup: treeSetup("root", treePoolSkewRight)},
			{name: "balanced", call: "right_view(root)", expected: viewRepr(treePoolBalanced), setup: treeSetup("root", treePoolBalanced)},
		}
		stmt := fmt.Sprintf("%s右缘视野：返回二叉树的右视图（每层最右节点的值；空树为空列表）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def right_view(root):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// 树 L4：最近公共祖先（递归三态判定，参考解 11 行）。
func structLcaL4() question.Template {
	meta := question.Meta{ID: "struct-lca-L4-attack", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"root-is-lca", "left-subtree", "right-subtree"}}
	type lcaEntry struct {
		vals         []any
		pExpr, qExpr string
		pVal, qVal   int
	}
	entries := []lcaEntry{
		{[]any{3, 5, 1, 6, 2, 0, 8, nil, nil, 7, 4}, "root.left", "root.right", 5, 1},
		{[]any{3, 5, 1, 6, 2, 0, 8, nil, nil, 7, 4}, "root.left", "root.left.right.right", 5, 4},
		{[]any{3, 5, 1, 6, 2, 0, 8, nil, nil, 7, 4}, "root.left.right.left", "root.left.right.right", 7, 4},
		{[]any{1, 2, 3}, "root.left", "root.right", 2, 3},
	}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		e := entries[rng.Int(0, len(entries)-1)]
		kind := rng.Scenario("dungeon")
		sig := "def lowest_common_ancestor(root, p, q):"
		answer := "def lowest_common_ancestor(root, p, q):\n" +
			"    if root is None:\n" +
			"        return None\n" +
			"    if root is p or root is q:\n" +
			"        return root\n" +
			"    left = lowest_common_ancestor(root.left, p, q)\n" +
			"    right = lowest_common_ancestor(root.right, p, q)\n" +
			"    if left is not None and right is not None:\n" +
			"        return root\n" +
			"    if left is not None:\n" +
			"        return left\n" +
			"    return right\n"
		call := "lowest_common_ancestor(root, p, q)"
		setupOf := func(e lcaEntry) string {
			return treeSetup("root", e.vals) + fmt.Sprintf("\np = %s\nq = %s", e.pExpr, e.qExpr)
		}
		expectedOf := func(e lcaEntry) string {
			return treeReprGo(lcaGo(buildTreeGo(e.vals), e.pVal, e.qVal))
		}
		visible := []matchCase{
			{name: "example-1", call: call, expected: expectedOf(e), setup: setupOf(e), compare: treeCompare()},
			{name: "example-2", call: call, expected: expectedOf(entries[0]), setup: setupOf(entries[0]), compare: treeCompare()},
		}
		hidden := []matchCase{
			{name: "root-is-lca", call: call, expected: expectedOf(entries[0]), setup: setupOf(entries[0]), compare: treeCompare()},
			{name: "left-subtree", call: call, expected: expectedOf(entries[1]), setup: setupOf(entries[1]), compare: treeCompare()},
			{name: "deep-nodes", call: call, expected: expectedOf(entries[2]), setup: setupOf(entries[2]), compare: treeCompare()},
			{name: "small-tree", call: call, expected: expectedOf(entries[3]), setup: setupOf(entries[3]), compare: treeCompare()},
			{name: "right-subtree", call: call, expected: expectedOf(entries[2]), setup: setupOf(entries[2]), compare: treeCompare()},
		}
		stmt := fmt.Sprintf("%s树脉溯源：返回二叉树中节点 p 与 q 的最近公共祖先（按节点身份；p、q 保证存在）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, "def lowest_common_ancestor(root, p, q):\n    # 在这里补全实现\n    raise NotImplementedError\n", answer,
			structCases(visible), structCases(hidden))
	}}
}

// ---- 有状态类 ----

// 哈希 L5：LRU 缓存（dict 有序性 + 逐出，参考解 14 行含类定义；附录 E.4 调用序列范式）。
func structLruL5() question.Template {
	meta := question.Meta{ID: "struct-lru-L5-attack", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"capacity-one", "evict-order", "miss"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		capacity := rng.Int(2, 3)
		k1 := rng.Int(1, 6)
		v1, v2, v3 := rng.Int(1, 9), rng.Int(1, 9), rng.Int(1, 9)
		kind := rng.Scenario("alchemy")
		sig := "class LRUCache:"
		answer := "class LRUCache:\n" +
			"    def __init__(self, capacity: int):\n" +
			"        self.capacity = capacity\n" +
			"        self.data = {}\n\n" +
			"    def get(self, key: int) -> int:\n" +
			"        if key not in self.data:\n" +
			"            return -1\n" +
			"        value = self.data.pop(key)\n" +
			"        self.data[key] = value\n" +
			"        return value\n\n" +
			"    def put(self, key: int, value: int) -> None:\n" +
			"        if key in self.data:\n" +
			"            self.data.pop(key)\n" +
			"        elif len(self.data) >= self.capacity:\n" +
			"            oldest = next(iter(self.data))\n" +
			"            self.data.pop(oldest)\n" +
			"        self.data[key] = value\n"
		starter := "class LRUCache:\n" +
			"    def __init__(self, capacity: int):\n        pass\n\n" +
			"    def get(self, key: int) -> int:\n        pass\n\n" +
			"    def put(self, key: int, value: int) -> None:\n        pass\n"
		callOf := func(k1, v1, v2, v3 int) string {
			return fmt.Sprintf("[obj.put(%d, %d), obj.put(%d, %d), obj.get(%d), obj.put(%d, %d), obj.get(%d), obj.get(%d)]",
				k1, v1, k1+1, v2, k1, k1+2, v3, k1+1, k1+2)
		}
		expectedOf := func(cap int, k1, v1, v2, v3 int) string {
			return pyAnyListRepr(lruSimGo(cap, k1, v1, v2, v3))
		}
		visible := []matchCase{
			{name: "example-1", call: callOf(k1, v1, v2, v3), expected: expectedOf(capacity, k1, v1, v2, v3), setup: fmt.Sprintf("obj = LRUCache(%d)", capacity)},
			{name: "example-2", call: callOf(1, 1, 2, 3), expected: expectedOf(2, 1, 1, 2, 3), setup: "obj = LRUCache(2)"},
		}
		hidden := []matchCase{
			{name: "capacity-one", call: callOf(1, 1, 2, 3), expected: expectedOf(1, 1, 1, 2, 3), setup: "obj = LRUCache(1)"},
			{name: "evict-order", call: callOf(1, 1, 2, 3), expected: expectedOf(2, 1, 1, 2, 3), setup: "obj = LRUCache(2)"},
			{name: "miss", call: "[obj.get(9), obj.get(8)]", expected: "[-1, -1]", setup: "obj = LRUCache(2)"},
			{name: "update-existing", call: "[obj.put(1, 5), obj.put(1, 6), obj.get(1)]", expected: "[None, None, 6]", setup: "obj = LRUCache(2)"},
			{name: "capacity-three", call: callOf(2, 4, 5, 6), expected: expectedOf(3, 2, 4, 5, 6), setup: "obj = LRUCache(3)"},
		}
		stmt := fmt.Sprintf("%s缓存秘境：实现容量为 capacity 的 LRU 缓存——get 未命中返回 −1，put 超容时逐出最久未使用键；每次访问都刷新最近使用序。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// 语法 L4：用栈实现队列（双栈 + 摊还转移，参考解 12 行含类定义）。
func structMyQueueL4() question.Template {
	meta := question.Meta{ID: "struct-myqueue-L4-attack", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormComplete,
		Traps: []string{"fifo-order", "empty-after"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		x1, x2, x3 := rng.Int(1, 9), rng.Int(1, 9), rng.Int(1, 9)
		kind := rng.Scenario("alchemy")
		sig := "class MyQueue:"
		answer := "class MyQueue:\n" +
			"    def __init__(self):\n" +
			"        self.inbox = []\n" +
			"        self.outbox = []\n\n" +
			"    def push(self, x: int) -> None:\n" +
			"        self.inbox.append(x)\n\n" +
			"    def pop(self) -> int:\n" +
			"        self._shift()\n" +
			"        return self.outbox.pop()\n\n" +
			"    def peek(self) -> int:\n" +
			"        self._shift()\n" +
			"        return self.outbox[-1]\n\n" +
			"    def empty(self) -> bool:\n" +
			"        return not self.inbox and not self.outbox\n\n" +
			"    def _shift(self) -> None:\n" +
			"        if not self.outbox:\n" +
			"            while self.inbox:\n" +
			"                self.outbox.append(self.inbox.pop())\n"
		starter := "class MyQueue:\n" +
			"    def __init__(self):\n        pass\n\n" +
			"    def push(self, x: int) -> None:\n        pass\n\n" +
			"    def pop(self) -> int:\n        pass\n\n" +
			"    def peek(self) -> int:\n        pass\n\n" +
			"    def empty(self) -> bool:\n        pass\n"
		callOf := func(a, b, c int) string {
			return fmt.Sprintf("[obj.push(%d), obj.push(%d), obj.pop(), obj.push(%d), obj.peek(), obj.pop(), obj.empty()]", a, b, c)
		}
		visible := []matchCase{
			{name: "example-1", call: callOf(x1, x2, x3), expected: pyAnyListRepr(myQueueSimGo(x1, x2, x3)), setup: "obj = MyQueue()"},
			{name: "example-2", call: callOf(1, 2, 3), expected: pyAnyListRepr(myQueueSimGo(1, 2, 3)), setup: "obj = MyQueue()"},
		}
		hidden := []matchCase{
			{name: "fifo-order", call: "[obj.push(1), obj.push(2), obj.pop(), obj.pop()]", expected: "[None, None, 1, 2]", setup: "obj = MyQueue()"},
			{name: "empty-after", call: "[obj.push(5), obj.pop(), obj.empty()]", expected: "[None, 5, True]", setup: "obj = MyQueue()"},
			{name: "peek-no-remove", call: "[obj.push(7), obj.peek(), obj.peek(), obj.pop()]", expected: "[None, 7, 7, 7]", setup: "obj = MyQueue()"},
			{name: "refill", call: "[obj.push(1), obj.pop(), obj.push(2), obj.pop()]", expected: "[None, 1, None, 2]", setup: "obj = MyQueue()"},
			{name: "empty-initially", call: "[obj.empty()]", expected: "[True]", setup: "obj = MyQueue()"},
		}
		stmt := fmt.Sprintf("%s栈桥队列：仅用两个栈实现先入先出队列（push/pop/peek/empty 均为均摊 O(1)）。\n%s", kind, structExamplesText(visible))
		return customInstance(meta, stmt, sig, starter, answer, structCases(visible), structCases(hidden))
	}}
}

// ---- 本文件局部类型与辅助 ----

// matchCase：模板内用例声明（可见/隐藏统一结构）。
type matchCase struct {
	name, call, expected, setup, compare string
}

// structCases：转 match.TestCase 列表。
func structCases(cs []matchCase) []match.TestCase {
	out := make([]match.TestCase, len(cs))
	for i, c := range cs {
		out[i] = structCase(c.name, c.call, c.expected, c.setup, c.compare)
	}
	return out
}

// structExamplesText：结构体题可见样例文本（call/setup 一并给出，题面自包含）。
func structExamplesText(visible []matchCase) string {
	parts := make([]string, len(visible))
	for i, c := range visible {
		if c.setup != "" {
			parts[i] = fmt.Sprintf("示例：%s；%s → %s", c.setup, c.call, c.expected)
		} else {
			parts[i] = fmt.Sprintf("示例：%s → %s", c.call, c.expected)
		}
	}
	return joinSemi(parts)
}

func joinSemi(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "；"
		}
		out += p
	}
	return out
}

// lcaGo：期望值计算（按节点值定位；题面保证值唯一、p/q 存在）。
func lcaGo(root *structTree, p, q int) *structTree {
	if root == nil || root.val == p || root.val == q {
		return root
	}
	l := lcaGo(root.left, p, q)
	r := lcaGo(root.right, p, q)
	if l != nil && r != nil {
		return root
	}
	if l != nil {
		return l
	}
	return r
}

// lruSimGo：LRU 调用序列的期望值（put 返回 None，get 未命中 −1）。
func lruSimGo(capacity, k1, v1, v2, v3 int) []any {
	order := []int{}
	data := map[int]int{}
	touch := func(k int) {
		for i, v := range order {
			if v == k {
				order = append(order[:i], order[i+1:]...)
				break
			}
		}
		order = append(order, k)
	}
	put := func(k, v int) {
		if _, ok := data[k]; ok {
			data[k] = v
			touch(k)
			return
		}
		if len(data) >= capacity {
			oldest := order[0]
			order = order[1:]
			delete(data, oldest)
		}
		data[k] = v
		order = append(order, k)
	}
	get := func(k int) any {
		v, ok := data[k]
		if !ok {
			return -1
		}
		touch(k)
		return v
	}
	out := []any{nil, nil, nil, nil, nil, nil}
	put(k1, v1)
	put(k1+1, v2)
	out[0], out[1] = nil, nil
	out[2] = get(k1)
	put(k1+2, v3)
	out[3] = nil
	out[4] = get(k1 + 1)
	out[5] = get(k1 + 2)
	return out
}

// myQueueSimGo：用栈队列调用序列的期望值。
func myQueueSimGo(x1, x2, x3 int) []any {
	queue := []int{}
	push := func(x int) { queue = append(queue, x) }
	pop := func() any {
		v := queue[0]
		queue = queue[1:]
		return v
	}
	push(x1)
	push(x2)
	p1 := pop()
	push(x3)
	peek := queue[0]
	p2 := pop()
	return []any{nil, nil, p1, nil, peek, p2, len(queue) == 0}
}
