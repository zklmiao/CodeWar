// 注册表：模板全集（黄金样板法，设计 002 §7；12 号票扩容至 ~124，
// 数量与矩阵覆盖目标见 coverage_test.go）。
package templates

import (
	"codewar/server/internal/question"
)

// All：返回全部模板（04 首批 26 + 12 号票分批扩容）。
func All() []question.Template {
	return []question.Template{
		// 语法与运算
		syntaxAddL1(),
		syntaxSumEvensL2(),
		syntaxCountDivL3(),
		syntaxBracketsL4(),
		syntaxReverseDigitsL3(),
		syntaxBigAddL4(),
		// 字符串处理
		stringPalindromeL2(),
		stringVowelCountL3(),
		stringRunLengthL4(),
		stringPalindromeSubstrL5(),
		stringShoutL1(),
		stringReverseWordsL3(),
		stringVersionCompareL4(),
		stringUniqueWindowL5(),
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
		// 技能（06 号票消费；L1/L2 由 12 号票批 1 补 cap<L3 缺口）
		skillClampL1(),
		skillChainHealL2(),
		skillBlastL3(),
		skillHealL3(),
		// 字符串主题私有题（12 号票批 3）
		moveFirstStepL1(),
		moveCountForwardL2(),
		moveFacingL4(),
		moveBeaconL5(),
		defendBadgeL1(),
		defendMaskL2(),
		defendCensorL3(),
		defendScrollL4(),
		defendBarrierL5(),
		skillEchoL1(),
		skillOddL2(),
		skillDrawL3(),
		skillDecompressL4(),
		skillWardL5(),
		// 语法主题私有题（12 号票批 2）
		moveVectorL2(),
		moveCorridorL3(),
		moveParseL4(),
		defendRepairL2(),
		defendHpAfterL3(),
		defendSiegeL4(),
		skillManaL2(),
		skillArcanaL4(),
	}
}
