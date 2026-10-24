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
// modeTutorial：新手教程模式（10 号票）——PvE 管线的一种模式：
// 同一 MatchEngine + 固定题包 + 脚本化 Bot + 服务端目标判定器。
const (
	modePvE      = "pve"
	modeTutorial = "tutorial"
)

// matchConfig：解析并归一后的本场配置（首个 join 锁定；快照展示与 ELO 结算共用）。
type matchConfig struct {
	topicKey string           // 题型 key（空 = 全部题型）；快照展示用
	topics   []question.Topic // 工厂题型过滤（只约束共享攻击题）；nil = 全部
	tier     int              // 难度上限档 1..5（出题区间上限 = 风险系数档位，同源）
	strategy string           // question.StrategyUniform | StrategyClimb
	// 09 号票：模式与 Bot（与 topic/cap/strategy 同一锁定语义）。
	mode          string // "" = PvP；modePvE = 单人练习；modeTutorial = 新手教程
	botDifficulty string // "" = 非 PvE；PvE 时 easy|normal|hard
	level         int    // 教程关号 1..4（仅 modeTutorial；其余为 0）
	seed          string // 本场 matchSeed（首个 join 生成；工厂与 Bot 决策同源）
}

// joinConfig：join 携带的原始配置参数（未校验）。
type joinConfig struct {
	topic, cap, strategy, mode, bot, level string
}

// anySet：是否携带了任一配置参数（后续 join「忽略并记日志」的判定面）。
func (j joinConfig) anySet() bool {
	return j.topic != "" || j.cap != "" || j.strategy != "" || j.mode != "" || j.bot != "" || j.level != ""
}

// parseMatchConfig：非法值回退默认并 WARN（回退语义由 hub_config_test 钉住）。
// 默认锚点 = 全部题型 / cap L3 / climb——与 question.Config 归一化一致
// （04 的既有测试锚点，05 裁决 5：不得改默认值）。
func parseMatchConfig(j joinConfig, logger *slog.Logger) matchConfig {
	mc := matchConfig{tier: match.DefaultRiskTier, strategy: question.StrategyClimb}

	// 模式先解析：教程会覆盖题目/难度配置（固定题包，不走出题工厂）。
	switch m := normalize(j.mode); m {
	case "", "pvp":
		// 缺省/显式 pvp = 现行 PvP（零行为变化）
	case modePvE:
		mc.mode = modePvE
	case modeTutorial:
		mc.mode = modeTutorial
	default:
		logger.Warn("对局模式非法，回退 PvP", "mode", j.mode)
	}

	if t := normalize(j.topic); t != "" {
		if validTopic(t) {
			mc.topicKey = t
			mc.topics = []question.Topic{question.Topic(t)}
		} else {
			logger.Warn("对局配置 topic 非法，回退默认（全部题型）", "topic", j.topic)
		}
	}
	if s := normalize(j.strategy); s != "" {
		if s == question.StrategyUniform || s == question.StrategyClimb {
			mc.strategy = s
		} else {
			logger.Warn("对局配置 strategy 非法，回退默认 climb", "strategy", j.strategy)
		}
	}
	if mc.mode != modeTutorial {
		if c := normalize(j.cap); c != "" {
			if n, ok := parseCap(c); ok {
				mc.tier = n
			} else {
				logger.Warn("对局配置 cap 非法，回退默认 L3", "cap", j.cap)
			}
		}
	}

	// 09 号票：Bot 难度（同样回退而非拒连——配置不是安全边界）。
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

	// 10 号票：教程关号与难度上限（L4 允许 L3/L5，其余关固定 L3）。
	if mc.mode == modeTutorial {
		mc.level = 1
		if raw := normalize(j.level); raw != "" {
			if n, ok := parseLevel(raw); ok {
				mc.level = n
			} else {
				logger.Warn("教程关号非法，回退第 1 关", "level", j.level)
			}
		}
		mc.tier = tutorialTier(mc.level, normalize(j.cap), logger)
		// 教程题包固定：题型过滤与抽题策略不参与（出现即记日志，不静默吞掉）。
		if mc.topicKey != "" {
			logger.Warn("教程使用固定题包，topic 参数已忽略", "topic", j.topic)
		}
		if mc.strategy != question.StrategyClimb {
			logger.Warn("教程使用固定题包，strategy 参数已忽略", "strategy", j.strategy)
		}
		mc.topicKey = ""
		mc.topics = nil
		mc.strategy = question.StrategyClimb
	}
	return mc
}

// tutorialTier：教程难度上限。L4 教学关允许 L3（×1.00）或 L5（×1.20），
// 缺省 L5 并在入口页提示"越高越疼"；其余关固定 L3 以保持教学数值可复现。
func tutorialTier(level int, cap string, logger *slog.Logger) int {
	if level != tutorialMaxLevel {
		return match.DefaultRiskTier
	}
	switch cap {
	case "", "l5":
		return 5
	case "l3":
		return 3
	default:
		logger.Warn("教程 L4 难度上限只支持 L3/L5，回退 L5", "cap", cap)
		return 5
	}
}

// tutorialMaxLevel：镜像 tutorial.MaxLevel（避免 config 层依赖教程包实现）。
const tutorialMaxLevel = 4

// parseLevel：教程关号 "1".."4" → 1..4。
func parseLevel(s string) (int, bool) {
	if len(s) != 1 {
		return 0, false
	}
	n := int(s[0] - '0')
	if n < 1 || n > tutorialMaxLevel {
		return 0, false
	}
	return n, true
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
