// 工厂与抽题策略（设计 002 §5/§7，spec「出题系统」）。
// 配置进构造器，不动 QuestionPort 方法签名（架构师裁决 1）：
// question.NewFactory(Config{Topics, MaxDifficulty, Strategy, MatchSeed})。
package question

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"reflect"

	"codewar/server/internal/match"
)

// 抽题策略：均匀随机 / 对局爬坡（开局低档、逐回合逼近上限）。
const (
	StrategyUniform = "uniform"
	StrategyClimb   = "climb"
)

// MaxRoundsFallback：爬坡映射的分母。
// 与 match.MaxRounds 语义同步：question 包不得反向依赖 match 包
// （依赖方向 match→question，由 hub 桥接），改动 MaxRounds 时必须同步此值。
const climbRounds = 10

// Config：对局级出题配置（05 号票后续从 UI 喂真实值，本票默认即可）。
type Config struct {
	Topics        []Topic    // 攻击题题型过滤；空 = 全部 6 题型
	MaxDifficulty Difficulty // 出题区间上限 [L1, Max]；0 视为 L3
	Strategy      string     // uniform | climb；空视为 climb
	MatchSeed     string     // 整场种子（不同对局不同值，连打两局实例不同）
}

// normalize：补默认值。
func (c *Config) normalize() {
	if c.MaxDifficulty < L1 || c.MaxDifficulty > L5 {
		c.MaxDifficulty = L3
	}
	if c.Strategy != StrategyUniform && c.Strategy != StrategyClimb {
		c.Strategy = StrategyClimb
	}
	if c.MatchSeed == "" {
		c.MatchSeed = "default-match"
	}
}

// Factory：seed 确定性出题工厂（纯函数 seam）。
type Factory struct {
	cfg      Config
	registry []Template
	byID     map[string]Template
}

// NewFactory：按配置构造工厂（模板另行 Register）。
func NewFactory(cfg Config) *Factory {
	cfg.normalize()
	return &Factory{cfg: cfg, byID: map[string]Template{}}
}

// Register：注册模板（ID 重复则 panic，早暴露配置错误）。
func (f *Factory) Register(t Template) {
	m := t.Meta()
	if err := ValidateMeta(m); err != nil {
		panic(fmt.Sprintf("注册模板失败：%v", err))
	}
	if _, dup := f.byID[m.ID]; dup {
		panic(fmt.Sprintf("模板 ID 重复：%s", m.ID))
	}
	f.registry = append(f.registry, t)
	f.byID[m.ID] = t
}

// RegisterAll：批量注册。
func (f *Factory) RegisterAll(ts ...Template) {
	for _, t := range ts {
		f.Register(t)
	}
}

// Count：已注册模板数。
func (f *Factory) Count() int { return len(f.registry) }

// DeriveSeed：种子派生（裁决 4）：sha256(matchSeed | 维度串…) 前 8 字节 → int64。
// 全库唯一随机来源是本函数产出的流，禁止全局 rand。
func DeriveSeed(matchSeed string, dims ...string) int64 {
	h := sha256.New()
	h.Write([]byte(matchSeed))
	for _, d := range dims {
		h.Write([]byte("|"))
		h.Write([]byte(d))
	}
	sum := h.Sum(nil)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

// NewMatchSeed：生成一局的 matchSeed（每局不同，连打两局实例不同）。
// 生产调用；测试注入固定值以保复现。
func NewMatchSeed() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "fallback-match"
	}
	return fmt.Sprintf("match-%x", b)
}

// Create：按 spec 精确取题并实例化（同 spec+seed 必同实例）。
func (f *Factory) Create(spec Spec, seed string) (Instance, error) {
	if spec.Action == "" {
		return Instance{}, fmt.Errorf("spec 缺少动作类型")
	}
	cands := filterTemplates(f.registry, spec)
	if len(cands) == 0 {
		return Instance{}, fmt.Errorf("无匹配模板：action=%s topic=%s difficulty=%v form=%s",
			spec.Action, spec.Topic, spec.Difficulty, spec.Form)
	}
	// 多候选时按种子流确定性选用其一。
	pick := cands[0]
	if len(cands) > 1 {
		r := NewRNG(DeriveSeed(seed, "pick", specKey(spec)))
		pick = cands[r.Int(0, len(cands)-1)]
	}
	instSeed := fmt.Sprintf("%s|%s", seed, pick.Meta().ID)
	rng := NewRNG(DeriveSeed(instSeed, "generate"))
	in, err := pick.Generate(rng)
	if err != nil {
		return Instance{}, err
	}
	in.Seed = instSeed
	if err := ValidateInstance(in); err != nil {
		return Instance{}, err
	}
	return in, nil
}

// AttackQuestion：QuestionPort 镜像题（seed = f(matchSeed, round)，双方同实例）。
// 满足 match.QuestionPort 方法签名（无 error；注册表为空时返回错误占位题，不 panic）。
func (f *Factory) AttackQuestion(round int) match.Question {
	q, err := f.attackQuestion(round)
	if err != nil {
		return match.Question{ID: "error-no-template", Statement: err.Error()}
	}
	return q
}

func (f *Factory) attackQuestion(round int) (match.Question, error) {
	roundSeed := fmt.Sprintf("attack:round=%d|%s", round, f.cfg.MatchSeed)
	rng := NewRNG(DeriveSeed(f.cfg.MatchSeed, fmt.Sprintf("attack:round=%d", round)))
	d := pickDifficulty(rng, round, f.cfg.MaxDifficulty, f.cfg.Strategy)
	topic := pickTopic(rng, f.cfg.Topics)
	spec := Spec{Topic: topic, Difficulty: d, Action: ActionAttack}
	in, err := f.Create(spec, roundSeed)
	if err != nil {
		// 题型过滤硬语义（05 号票验收 2：攻击题只从所选题型抽取）：策略抽出的
		// 难度格在本题型下为空时，就近平移难度重试（不放弃题型，seed 流不变、
		// 确定性保持）；整个 [L1, cap] 区间都没有该题型攻击模板（12 号票补齐
		// 模板矩阵前的真空格）才退回跨题型——与行动题同语义的最后保底。
		if in2, ok := f.createNearestDifficulty(spec, d); ok {
			in = in2
		} else {
			spec.Topic = ""
			in, err = f.Create(spec, roundSeed)
			if err != nil {
				return match.Question{}, err
			}
		}
	}
	return in.ToQuestion(pickForm(rng, in.Meta.Form)), nil
}

// createNearestDifficulty：题型不变，从 drawn 难度向外螺旋（+1/−1/+2/−2…）
// 寻找 [L1, cap] 内首个有模板的难度档（平移距离最近，策略语义漂移最小）。
func (f *Factory) createNearestDifficulty(spec Spec, drawn Difficulty) (Instance, bool) {
	capD := f.cfg.MaxDifficulty
	for step := 1; step < int(capD); step++ {
		for _, d := range []Difficulty{drawn + Difficulty(step), drawn - Difficulty(step)} {
			if d < L1 || d > capD {
				continue
			}
			spec.Difficulty = d
			if in, err := f.Create(spec, fmt.Sprintf("%s|near=%d", specKey(spec), int(drawn))); err == nil {
				return in, true
			}
		}
	}
	return Instance{}, false
}

// MoveQuestion：私有移动题（seed = f(matchSeed, player, round)，双方独立）。
func (f *Factory) MoveQuestion(round int, player match.Player) match.Question {
	seed := fmt.Sprintf("move:%s:round=%d|%s", player, round, f.cfg.MatchSeed)
	rng := NewRNG(DeriveSeed(f.cfg.MatchSeed, fmt.Sprintf("move:%s:round=%d", player, round)))
	d := pickDifficulty(rng, round, f.cfg.MaxDifficulty, f.cfg.Strategy)
	in, err := f.Create(Spec{Difficulty: d, Action: ActionMove}, seed)
	if err != nil {
		return match.Question{ID: "error-no-template", Statement: err.Error()}
	}
	return in.ToQuestion(FormComplete)
}

// DefendQuestion：私有防御题（seed 派生同移动题）。
func (f *Factory) DefendQuestion(round int, player match.Player) match.Question {
	seed := fmt.Sprintf("defend:%s:round=%d|%s", player, round, f.cfg.MatchSeed)
	rng := NewRNG(DeriveSeed(f.cfg.MatchSeed, fmt.Sprintf("defend:%s:round=%d", player, round)))
	d := pickDifficulty(rng, round, f.cfg.MaxDifficulty, f.cfg.Strategy)
	in, err := f.Create(Spec{Difficulty: d, Action: ActionDefend}, seed)
	if err != nil {
		return match.Question{ID: "error-no-template", Statement: err.Error()}
	}
	return in.ToQuestion(FormComplete)
}

// pickDifficulty：难度抽取。均匀 = [L1,Max] 等概率；爬坡 = 开局 L1、随回合单调逼近上限。
func pickDifficulty(rng *RNG, round int, max Difficulty, strategy string) Difficulty {
	if max < L1 {
		max = L1
	}
	if max > L5 {
		max = L5
	}
	if strategy == StrategyUniform {
		return Difficulty(rng.Int(int(L1), int(max)))
	}
	// 爬坡：round=1 → L1；round≥climbRounds → max；中间线性上取整（单调不降）。
	if round <= 1 {
		return L1
	}
	if round >= climbRounds {
		return max
	}
	span := int(max) - int(L1)
	step := (span*(round-1) + (climbRounds - 2)) / (climbRounds - 1) // ceil
	d := int(L1) + step
	if d > int(max) {
		d = int(max)
	}
	return Difficulty(d)
}

// pickTopic：题型抽取（Config 为空 = 全题型均匀）。
func pickTopic(rng *RNG, topics []Topic) Topic {
	pool := topics
	if len(pool) == 0 {
		pool = AllTopics
	}
	return pool[rng.Int(0, len(pool)-1)]
}

// pickForm：both 模板随机渲染其一（节奏变化），单形式模板保持原样。
func pickForm(rng *RNG, declared Form) Form {
	if declared != FormBoth {
		return declared
	}
	if rng.Bool(0.5) {
		return FormFillBlank
	}
	return FormComplete
}

// filterTemplates：按 spec 精确过滤（Topic 空 = 不限；Form both 兼容任一请求）。
// 注：Config.Topics 的题型过滤不在此生效——实际过滤在 pickTopic 经 f.cfg.Topics
// 选定本题回合题型后，由上层以 Spec.Topic 传入。
func filterTemplates(all []Template, spec Spec) []Template {
	var out []Template
	for _, t := range all {
		m := t.Meta()
		if m.Action != spec.Action {
			continue
		}
		if spec.Difficulty != 0 && m.Difficulty != spec.Difficulty {
			continue
		}
		if spec.Topic != "" && m.Topic != spec.Topic {
			continue
		}
		if spec.Form != "" && m.Form != FormBoth && m.Form != spec.Form {
			continue
		}
		out = append(out, t)
	}
	return out
}

func specKey(s Spec) string {
	return fmt.Sprintf("%s-%d-%s-%s", s.Topic, int(s.Difficulty), s.Action, s.Form)
}

// 实例深度相等（测试用：剔除 Seed 后比较）。
func instancesEqual(a, b Instance) bool {
	a.Seed, b.Seed = "", ""
	return reflect.DeepEqual(a, b)
}

// SkillQuestion：私有技能题（06 号票，裁决 4）——难度 = 所选上限档，
// 不走爬坡/均匀策略（技能题一律贴上限，"高难高收益"的出题面语义）。
// 上限档无模板时向低档逐档回退取最近可用（12 号票补齐 L4/L5 技能模板后
// 该回退仅在 cap>L4 时触发）。
func (f *Factory) SkillQuestion(round int, player match.Player) match.Question {
	seed := fmt.Sprintf("skill:%s:round=%d|%s", player, round, f.cfg.MatchSeed)
	for d := f.cfg.MaxDifficulty; d >= L1; d-- {
		in, err := f.Create(Spec{Difficulty: d, Action: ActionSkill}, seed)
		if err == nil {
			return in.ToQuestion(FormComplete)
		}
	}
	return match.Question{ID: "error-no-template", Statement: "无可用技能题模板"}
}
