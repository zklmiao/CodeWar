// 对局配置解析与锁定（05 号票）：页面 URL 参数 ?topic=&cap=&strategy= 由客户端
// 经 join 请求注入（proto JoinRequest.topic/cap/strategy），hub 在首个 join
// （创建引擎）时采纳并整场锁定；后续 join 携带的参数忽略并记日志（热座第二
// 窗口复制粘贴同一链接，参数天然一致；两窗口冲突时以先到者为准）。
// 非法值回退默认并 WARN，不拒连——配置不是安全边界（错配置只影响本局体验），
// 与职业组合的 fail-closed（错身份会污染战绩）刻意不同。
package hub

import (
	"log/slog"
	"strings"

	"codewar/server/internal/bot"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

// modePvE：单人练习模式（09 号票）。空串 = PvP（01~07 的现行行为）。
const modePvE = "pve"

// matchConfig：解析并归一后的本场配置（首个 join 锁定；快照展示与 ELO 结算共用）。
type matchConfig struct {
	topicKey string           // 题型 key（空 = 全部题型）；快照展示用
	topics   []question.Topic // 工厂题型过滤（只约束共享攻击题）；nil = 全部
	tier     int              // 难度上限档 1..5（出题区间上限 = 风险系数档位，同源）
	strategy string           // question.StrategyUniform | StrategyClimb
	// 09 号票：模式与 Bot（与 topic/cap/strategy 同一锁定语义）。
	mode          string // "" = PvP；modePvE = 单人练习
	botDifficulty string // "" = 非 PvE；PvE 时 easy|normal|hard
	seed          string // 本场 matchSeed（首个 join 生成；工厂与 Bot 决策同源）
}

// joinConfig：join 携带的原始配置参数（未校验）。
type joinConfig struct {
	topic, cap, strategy, mode, bot string
}

// anySet：是否携带了任一配置参数（后续 join「忽略并记日志」的判定面）。
func (j joinConfig) anySet() bool {
	return j.topic != "" || j.cap != "" || j.strategy != "" || j.mode != "" || j.bot != ""
}

// parseMatchConfig：非法值回退默认并 WARN（回退语义由 hub_config_test 钉住）。
// 默认锚点 = 全部题型 / cap L3 / climb——与 question.Config 归一化一致
// （04 的既有测试锚点，05 裁决 5：不得改默认值）。
func parseMatchConfig(j joinConfig, logger *slog.Logger) matchConfig {
	mc := matchConfig{tier: match.DefaultRiskTier, strategy: question.StrategyClimb}
	if t := normalize(j.topic); t != "" {
		if validTopic(t) {
			mc.topicKey = t
			mc.topics = []question.Topic{question.Topic(t)}
		} else {
			logger.Warn("对局配置 topic 非法，回退默认（全部题型）", "topic", j.topic)
		}
	}
	if c := normalize(j.cap); c != "" {
		if n, ok := parseCap(c); ok {
			mc.tier = n
		} else {
			logger.Warn("对局配置 cap 非法，回退默认 L3", "cap", j.cap)
		}
	}
	if s := normalize(j.strategy); s != "" {
		if s == question.StrategyUniform || s == question.StrategyClimb {
			mc.strategy = s
		} else {
			logger.Warn("对局配置 strategy 非法，回退默认 climb", "strategy", j.strategy)
		}
	}
	// 09 号票：模式与 Bot 难度（同样回退而非拒连——配置不是安全边界）。
	switch m := normalize(j.mode); m {
	case "", "pvp":
		// 缺省/显式 pvp = 现行 PvP（零行为变化）
	case modePvE:
		mc.mode = modePvE
	default:
		logger.Warn("对局模式非法，回退 PvP", "mode", j.mode)
	}
	if mc.mode == modePvE {
		if b := normalize(j.bot); b == "" {
			mc.botDifficulty = string(bot.DefaultDifficulty)
		} else if d, ok := bot.ParseDifficulty(b); ok {
			mc.botDifficulty = string(d)
		} else {
			logger.Warn("PvE Bot 难度非法，回退 normal", "bot", j.bot)
			mc.botDifficulty = string(bot.DefaultDifficulty)
		}
	} else if normalize(j.bot) != "" {
		logger.Warn("bot 参数仅在 mode=pve 下生效，已忽略", "bot", j.bot)
	}
	return mc
}

// topicDisplay：日志用题型展示（空 = 全部）。
func (m matchConfig) topicDisplay() string {
	if m.topicKey == "" {
		return "全部题型"
	}
	return m.topicKey
}

// normalize：URL 参数的宽松清洗（大小写/空白不敏感；合法集合本身仍严格）。
func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validTopic(t string) bool {
	for _, x := range question.AllTopics {
		if string(x) == t {
			return true
		}
	}
	return false
}

// parseCap："l1".."l5" → 1..5（URL 参数形式 cap=L3，大小写不敏感）。
func parseCap(c string) (int, bool) {
	if len(c) != 2 || c[0] != 'l' {
		return 0, false
	}
	n := int(c[1] - '0')
	if n < 1 || n > 5 {
		return 0, false
	}
	return n, true
}
