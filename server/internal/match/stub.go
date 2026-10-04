package match

// 硬编码题桩（ruling #5）：4 道补全函数题，按回合轮换。
// 04 号票将用出题工厂替换本桩；题面/用例格式与设计 002 对齐（call = Python 表达式，expected = repr）。

var DemoQuestions = []Question{
	{
		ID:        "palindrome-L2",
		Statement: "判断字符串是否为回文（区分大小写，空串视为回文）。\n示例：is_palindrome(\"level\") → True；is_palindrome(\"hello\") → False",
		Signature: "def is_palindrome(s: str) -> bool:",
		StarterCode: "def is_palindrome(s: str) -> bool:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "odd", Call: `is_palindrome("level")`, Expected: "True"},
			{Name: "even", Call: `is_palindrome("abba")`, Expected: "True"},
			{Name: "not-palindrome", Call: `is_palindrome("hello")`, Expected: "False"},
			{Name: "empty", Call: `is_palindrome("")`, Expected: "True"},
			{Name: "case-sensitive", Call: `is_palindrome("Abba")`, Expected: "False"},
		},
	},
	{
		ID:        "two-sum-L3",
		Statement: "在数组中找出和为目标值的两个数的下标（恰好一个解，返回升序下标列表）。\n示例：two_sum([2, 7, 11, 15], 9) → [0, 1]",
		Signature: "def two_sum(nums: list, target: int) -> list:",
		StarterCode: "def two_sum(nums: list, target: int) -> list:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: "two_sum([2, 7, 11, 15], 9)", Expected: "[0, 1]"},
			{Name: "middle", Call: "two_sum([3, 2, 4], 6)", Expected: "[1, 2]"},
			{Name: "duplicates", Call: "two_sum([3, 3], 6)", Expected: "[0, 1]"},
			{Name: "negative", Call: "two_sum([-1, 5, 10, -7], 3)", Expected: "[2, 3]"},
		},
	},
	{
		ID:        "run-length-L3",
		Statement: "字符串压缩：连续字符按「字符+次数」输出（次数为 1 也要输出）。\n示例：compress(\"aabcc\") → \"a2b1c2\"",
		Signature: "def compress(s: str) -> str:",
		StarterCode: "def compress(s: str) -> str:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: `compress("aabcc")`, Expected: `'a2b1c2'`},
			{Name: "single", Call: `compress("abc")`, Expected: `'a1b1c1'`},
			{Name: "empty", Call: `compress("")`, Expected: `''`},
			{Name: "long-run", Call: `compress("aaab")`, Expected: `'a3b1'`},
		},
	},
	{
		ID:        "dedup-L2",
		Statement: "列表去重并保持首次出现的顺序。\n示例：dedup([1, 2, 2, 3, 1]) → [1, 2, 3]",
		Signature: "def dedup(nums: list) -> list:",
		StarterCode: "def dedup(nums: list) -> list:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: "dedup([1, 2, 2, 3, 1])", Expected: "[1, 2, 3]"},
			{Name: "empty", Call: "dedup([])", Expected: "[]"},
			{Name: "all-same", Call: "dedup([5, 5, 5])", Expected: "[5]"},
			{Name: "no-dup", Call: "dedup([1, 2, 3])", Expected: "[1, 2, 3]"},
		},
	},
}

// StubQuestions：镜像题桩——按回合轮换（ruling #5）。
type StubQuestions struct{}

func (StubQuestions) AttackQuestion(round int) Question {
	i := ((round - 1) % len(DemoQuestions) + len(DemoQuestions)) % len(DemoQuestions)
	return DemoQuestions[i]
}
