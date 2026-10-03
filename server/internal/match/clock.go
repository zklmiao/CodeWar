package match

import (
	"time"

	"codewar/server/gen/matchpb"
)

// RealClock：生产时钟。
type RealClock struct{}

func (RealClock) Now() time.Time                 { return time.Now() }
func (RealClock) NewTimer(d time.Duration) Timer { return realTimer{time.NewTimer(d)} }

type realTimer struct{ t *time.Timer }

func (r realTimer) C() <-chan time.Time { return r.t.C }
func (r realTimer) Stop() bool          { return r.t.Stop() }

// Phase：当前对局阶段（hub 判断是否可开局用）。
func (e *Engine) Phase() matchpb.MatchPhase {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.phase
}
