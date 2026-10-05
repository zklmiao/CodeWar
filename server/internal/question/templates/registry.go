// 注册表：首批模板全集（黄金样板法，设计 002 §7）。
package templates

import (
	"codewar/server/internal/question"
)

// All：返回首批全部模板（攻击 18 + 移动 3 + 防御 3 + 技能 2 = 26）。
func All() []question.Template {
	return []question.Template{
		// 语法与运算
		syntaxAddL1(),
		syntaxSumEvensL2(),
		syntaxCountDivL3(),
		// 字符串处理
		stringPalindromeL2(),
		stringVowelCountL3(),
		stringRunLengthL4(),
		// 数组与列表
		arrayDedupL2(),
		arrayTwoSumL3(),
		arrayWindowSumL4(),
		// 哈希与计数
		hashCountL2(),
		hashWordFreqL3(),
		hashTopKL4(),
		// 排序与贪心
		sortParityL2(),
		sortKthL3(),
		sortInsertionL4(),
		// 递归与简单 DP（L1/L5 无语义格归 12 号票）
		recursionFactorialL2(),
		recursionBinarySearchL3(),
		recursionPascalL4(),
		// 移动（私有题）
		moveManhattanL1(),
		moveStepL2(),
		moveFollowL3(),
		// 防御（私有题）
		defendReduceL1(),
		defendShieldL2(),
		defendSurviveL3(),
		// 技能（供 06 号票参照）
		skillBlastL3(),
		skillHealL3(),
	}
}
