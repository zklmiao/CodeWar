package match

// 硬编码题桩（ruling #5）：4 道补全函数题，按回合轮换。
// 04 号票将用出题工厂替换本桩；题面/用例格式与设计 002 对齐（call = Python 表达式，expected = repr）。

var DemoQuestions = []Question{
	{
		ID:          "palindrome-L2",
		Statement:   "判断字符串是否为回文（区分大小写，空串视为回文）。\n示例：is_palindrome(\"level\") → True；is_palindrome(\"hello\") → False",
		Signature:   "def is_palindrome(s: str) -> bool:",
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
		ID:          "two-sum-L3",
		Statement:   "在数组中找出和为目标值的两个数的下标（恰好一个解，返回升序下标列表）。\n示例：two_sum([2, 7, 11, 15], 9) → [0, 1]",
		Signature:   "def two_sum(nums: list, target: int) -> list:",
		StarterCode: "def two_sum(nums: list, target: int) -> list:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: "two_sum([2, 7, 11, 15], 9)", Expected: "[0, 1]"},
			{Name: "middle", Call: "two_sum([3, 2, 4], 6)", Expected: "[1, 2]"},
			{Name: "duplicates", Call: "two_sum([3, 3], 6)", Expected: "[0, 1]"},
			{Name: "negative", Call: "two_sum([-1, 5, 10, -7], 3)", Expected: "[2, 3]"},
		},
	},
	{
		ID:          "run-length-L3",
		Statement:   "字符串压缩：连续字符按「字符+次数」输出（次数为 1 也要输出）。\n示例：compress(\"aabcc\") → \"a2b1c2\"",
		Signature:   "def compress(s: str) -> str:",
		StarterCode: "def compress(s: str) -> str:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: `compress("aabcc")`, Expected: `'a2b1c2'`},
			{Name: "single", Call: `compress("abc")`, Expected: `'a1b1c1'`},
			{Name: "empty", Call: `compress("")`, Expected: `''`},
			{Name: "long-run", Call: `compress("aaab")`, Expected: `'a3b1'`},
		},
	},
	{
		ID:          "dedup-L2",
		Statement:   "列表去重并保持首次出现的顺序。\n示例：dedup([1, 2, 2, 3, 1]) → [1, 2, 3]",
		Signature:   "def dedup(nums: list) -> list:",
		StarterCode: "def dedup(nums: list) -> list:\n    # 在这里补全函数体\n    raise NotImplementedError\n",
		Tests: []TestCase{
			{Name: "basic", Call: "dedup([1, 2, 2, 3, 1])", Expected: "[1, 2, 3]"},
			{Name: "empty", Call: "dedup([])", Expected: "[]"},
			{Name: "all-same", Call: "dedup([5, 5, 5])", Expected: "[5]"},
			{Name: "no-dup", Call: "dedup([1, 2, 3])", Expected: "[1, 2, 3]"},
		},
	},
}

// StubQuestions：测试桩——按回合轮换（ruling #5）。
// 04 号票后生产流量走出题工厂（hub 装配）；本桩保留供 match 包测试作 fake。
// 03 号票：移动/防御题带玩家维度 ID（证明面板私有性），判题用例与攻击桩一致
// （引擎行为测试只关心判定结果，不关心题面语义）。
type StubQuestions struct{}

func (StubQuestions) AttackQuestion(round int) Question {
	i := ((round-1)%len(DemoQuestions) + len(DemoQuestions)) % len(DemoQuestions)
	return DemoQuestions[i]
}

// MoveQuestion：私有移动题（ID 含玩家维度，供面板私有性断言；额外探针用例
// 供"判题用对题"断言——spy 判题器看到 move-only-probe 即说明用的是移动题）。
func (StubQuestions) MoveQuestion(round int, player Player) Question {
	q := StubQuestions{}.AttackQuestion(round)
	q.ID = "move-stub-" + player
	q.Tests = append(append([]TestCase{}, q.Tests...),
		TestCase{Name: "move-only-probe", Call: "1", Expected: "1"})
	return q
}

// DefendQuestion：私有防御题（同上，探针名不同）。
func (StubQuestions) DefendQuestion(round int, player Player) Question {
	q := StubQuestions{}.AttackQuestion(round)
	q.ID = "defend-stub-" + player
	q.Tests = append(append([]TestCase{}, q.Tests...),
		TestCase{Name: "defend-only-probe", Call: "1", Expected: "1"})
	return q
}

// SkillQuestion：技能题桩（06 号票）——轮换攻击题桩充当技能题。
// 真实实现见 question.Factory.SkillQuestion（难度 = 所选上限档，裁决 4）；
// 桩不模拟档位语义，只保证 QuestionPort 接口完整与对局可跑。
func (StubQuestions) SkillQuestion(round int, player Player) Question {
	i := ((round-1)%len(DemoQuestions) + len(DemoQuestions)) % len(DemoQuestions)
	q := DemoQuestions[i]
	q.ID = "skill-" + q.ID
	return q
}
