package store

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"codewar/server/internal/testdb"

	"github.com/google/uuid"
)

// 门控集成测试：连接 compose 的 postgres（localhost:25432），
// 以 CODEWAR_TEST_DSN 门控——变量缺失时 t.Skip，CI（无 DB）保持绿。
// 本地全量：docker compose up postgres（deploy/）+ 设变量跑。
//
// 破坏性护栏与库隔离：源 DSN 的库名必须含 "test"；每个测试包使用独立库
// （internal/testdb，codewar_test_store），因此 `go test ./...` 可并行跑包，
// 不会与 accounts/hub 的 TRUNCATE 互相干扰。
func openTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := testdb.DSN(t, "store")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("测试库连接/迁移失败（先 docker compose up postgres）：%v", err)
	}
	t.Cleanup(func() { s.Close() })
	for _, tbl := range []string{"elo_history", "matches", "sessions", "users"} {
		if _, err := s.DB().Exec("TRUNCATE " + tbl + " CASCADE"); err != nil {
			t.Fatalf("TRUNCATE %s 失败：%v", tbl, err)
		}
	}
	return s
}

// 注册 → 登录 → 双令牌鉴权 → 注销其一 → 另一仍有效；
// 错误密码/未知用户同错；重名（大小写不敏感）拒绝；过期/无效令牌拒绝；同用户多会话并存。
func TestAuthFlow(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	u, tok1, err := s.Register(ctx, "Alice", "s3cret-pw")
	if err != nil {
		t.Fatalf("注册失败：%v", err)
	}
	if u.Elo != 1000 || u.Username != "Alice" {
		t.Fatalf("注册返回异常：%+v", u)
	}

	// 大小写不敏感重名
	if _, _, err := s.Register(ctx, "alice", "other-pw"); err != ErrUsernameTaken {
		t.Fatalf("重名注册应报 ErrUsernameTaken，实际 %v", err)
	}
	// 参数校验（store  seam 断言哨兵，HTTP 400 断言在 accounts 侧）
	if _, _, err := s.Register(ctx, "a", "s3cret-pw"); err == nil {
		t.Fatal("过短用户名应被拒绝")
	}
	if _, _, err := s.Register(ctx, "bad name", "s3cret-pw"); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("含空白用户名应报 ErrInvalidUsername，实际 %v", err)
	}
	if _, _, err := s.Register(ctx, "Bob", "123"); err == nil {
		t.Fatal("过短密码应被拒绝")
	}

	// 同错：未知用户 vs 错误密码
	if _, _, err := s.Login(ctx, "Nobody", "s3cret-pw"); err != ErrInvalidCredentials {
		t.Fatalf("未知用户应报统一凭证错误，实际 %v", err)
	}
	// 形态非法直接 401，不到库（Login 参数分支）
	if _, _, err := s.Login(ctx, "a", "b"); err != ErrInvalidCredentials {
		t.Fatalf("非法形态登录应报统一凭证错误，实际 %v", err)
	}
	if _, _, err := s.Login(ctx, "alice", "wrong-pw"); err != ErrInvalidCredentials {
		t.Fatalf("错误密码应报统一凭证错误，实际 %v", err)
	}
	// 登录大小写不敏感
	u2, tok2, err := s.Login(ctx, "ALICE", "s3cret-pw")
	if err != nil {
		t.Fatalf("登录失败：%v", err)
	}
	if u2.ID != u.ID {
		t.Fatal("登录应返回同一用户")
	}
	if tok1 == tok2 {
		t.Fatal("两次会话令牌必须不同")
	}

	// 双令牌均可鉴权（两窗口两账号场景的地基：同用户多会话并存）
	for i, tok := range []string{tok1, tok2} {
		if got, err := s.Authenticate(ctx, tok); err != nil || got.ID != u.ID {
			t.Fatalf("令牌 %d 鉴权失败：%v %+v", i, err, got)
		}
	}
	// P3：多会话计数（CountSessionsByUser 此前零调用，此处转正）
	if n, err := s.q.CountSessionsByUser(ctx, u.ID); err != nil || n != 2 {
		t.Fatalf("登录两次后会话数应为 2：n=%d err=%v", n, err)
	}

	// 注销其一，另一个不受影响
	if err := s.Logout(ctx, tok1); err != nil {
		t.Fatalf("注销失败：%v", err)
	}
	if _, err := s.Authenticate(ctx, tok1); err != ErrUnauthorized {
		t.Fatalf("已注销令牌应被拒绝，实际 %v", err)
	}
	if _, err := s.Authenticate(ctx, tok2); err != nil {
		t.Fatalf("另一会话应仍有效：%v", err)
	}

	// 无效 / 空令牌
	if _, err := s.Authenticate(ctx, "bogus-token"); err != ErrUnauthorized {
		t.Fatalf("无效令牌应被拒绝，实际 %v", err)
	}
	if _, err := s.Authenticate(ctx, ""); err != ErrUnauthorized {
		t.Fatalf("空令牌应被拒绝，实际 %v", err)
	}

	// 过期会话：直接把 tok2 置过期
	if _, err := s.DB().Exec("UPDATE sessions SET expires_at = now() - interval '1 hour' WHERE token_hash = $1", hashToken(tok2)); err != nil {
		t.Fatalf("置过期失败：%v", err)
	}
	if _, err := s.Authenticate(ctx, tok2); err != ErrUnauthorized {
		t.Fatalf("过期令牌应被拒绝，实际 %v", err)
	}
}

// 并发重名注册：恰一成一冲突（H4：撞唯一约束的输家也应报 ErrUsernameTaken，
// 而非裸 pq 错误。bcrypt 耗时在检查之后，天然放大竞态窗口，复现稳定）。
func TestConcurrentDuplicateRegister(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	type res struct {
		err error
	}
	outs := make([]res, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, err := s.Register(ctx, "racer", "password1")
			outs[i] = res{err}
		}(i)
	}
	wg.Wait()
	ok, taken := 0, 0
	for _, o := range outs {
		switch {
		case o.err == nil:
			ok++
		case errors.Is(o.err, ErrUsernameTaken):
			taken++
		default:
			t.Fatalf("并发重名注册出现非预期错误：%v", o.err)
		}
	}
	if ok != 1 || taken != 1 {
		t.Fatalf("并发重名注册应一成一冲突，实际 ok=%d taken=%d", ok, taken)
	}
}

// 坏 DSN/连不上：Open 直接报错（启动 fail-fast 的地基；无 DB 可跑）。
func TestOpenBadDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := Open(ctx, "postgres://u:p@127.0.0.1:1/nope?sslmode=disable"); err == nil {
		t.Fatal("坏 DSN 应 Open 失败")
	}
	// 驱动层直接拒收的非法 DSN（sql.Open 错误分支）
	if _, err := Open(ctx, "://"); err == nil {
		t.Fatal("非法 DSN 应 Open 失败")
	}
}

// 已关闭的库：各操作返回错误而非 panic（DB 宕机时的 fail-closed）。
func TestOpsOnClosedStore(t *testing.T) {
	s := openTestStore(t)
	id := uuid.New()
	s.Close()
	if _, err := s.Authenticate(context.Background(), "tok"); err == nil {
		t.Fatal("关库后鉴权应报错")
	}
	if err := s.Logout(context.Background(), "tok"); err == nil {
		t.Fatal("关库后注销应报错")
	}
	if _, _, err := s.RecordMatch(context.Background(), MatchResult{MatchID: uuid.New(), P1UserID: id, P2UserID: id}); err == nil {
		t.Fatal("关库后结算应报错")
	}
	if _, _, err := s.Register(context.Background(), "closeduser", "pw-1234"); err == nil {
		t.Fatal("关库后注册应报错")
	}
	if _, _, err := s.Login(context.Background(), "closeduser", "pw-1234"); err == nil {
		t.Fatal("关库后登录应报错")
	}
	if _, err := s.GetProfile(context.Background(), id, 20); err == nil {
		t.Fatal("关库后个人页应报错")
	}
}

// 关库上跑迁移：fail-fast（main 的退出分支依赖此语义）。
func TestApplyMigrationsClosedDB(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://u:p@127.0.0.1:1/nope?sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open 意外失败：%v", err)
	}
	db.Close()
	if err := ApplyMigrations(db); err == nil {
		t.Fatal("关库上迁移应报错")
	}
}

// 未知用户结算：fail-closed（hub 只传已认证用户，此处锁死契约）。
func TestRecordMatchUnknownUsers(t *testing.T) {
	s := openTestStore(t)
	if _, _, err := s.RecordMatch(context.Background(), MatchResult{
		MatchID: uuid.New(), Winner: "p1", Reason: "ko", P1UserID: uuid.New(), P2UserID: uuid.New(),
	}); err == nil {
		t.Fatal("未知用户结算应报错")
	}
	// 一真一假：第二行加锁失败同样报错（覆盖第二查询分支）
	a, _, err := s.Register(context.Background(), "halfuser", "password1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.RecordMatch(context.Background(), MatchResult{
		MatchID: uuid.New(), Winner: "p1", Reason: "ko", P1UserID: a.ID, P2UserID: uuid.New(),
	}); err == nil {
		t.Fatal("半未知用户结算应报错")
	}
}

// 会话签发撞外键：CreateSession 错误上浮（newSession 错误分支）。
func TestNewSessionUnknownUser(t *testing.T) {
	s := openTestStore(t)
	if _, _, err := s.newSession(context.Background(), uuid.New()); err == nil {
		t.Fatal("未知用户签发会话应报错")
	}
}

// 空令牌注销：幂等 nil（handler 依赖此语义报 204）。
func TestLogoutEmptyToken(t *testing.T) {
	s := openTestStore(t)
	if err := s.Logout(context.Background(), ""); err != nil {
		t.Fatalf("空令牌注销应为 nil：%v", err)
	}
}

// 落库 → 查询战绩 → ELO 历史一致性；平局 +0 且照样记行。
func TestRecordAndProfile(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a, _, err := s.Register(ctx, "p1user", "password1")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := s.Register(ctx, "p2user", "password2")
	if err != nil {
		t.Fatal(err)
	}

	mid := uuid.New()
	p1, p2, err := s.RecordMatch(ctx, MatchResult{
		MatchID: mid, Winner: "p1", Reason: "ko", DurationMs: 120000,
		P1UserID: a.ID, P2UserID: b.ID,
	})
	if err != nil {
		t.Fatalf("结算落库失败：%v", err)
	}
	// 同分 p1 胜：1016 / 984（与 elo 纯函数表一致）
	if p1.Before != 1000 || p1.After != 1016 || p2.Before != 1000 || p2.After != 984 {
		t.Fatalf("ELO 变化不符：%+v %+v", p1, p2)
	}

	// 用户当前分已推进
	ua, _ := s.q.GetUserByID(ctx, a.ID)
	ub, _ := s.q.GetUserByID(ctx, b.ID)
	if ua.Elo != 1016 || ub.Elo != 984 {
		t.Fatalf("用户 ELO 未更新：%d / %d", ua.Elo, ub.Elo)
	}
	// 历史 exactly 两行
	hist, err := s.q.ListEloHistoryByMatch(ctx, mid)
	if err != nil || len(hist) != 2 {
		t.Fatalf("ELO 历史应恰两行：%v %+v", err, hist)
	}

	// 个人页：胜者视角
	prof, err := s.GetProfile(ctx, a.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if prof.Total != 1 || prof.Wins != 1 || prof.Losses != 0 || prof.Draws != 0 || prof.WinRate != 1.0 {
		t.Fatalf("战绩聚合不符：%+v", prof)
	}
	if len(prof.Recent) != 1 {
		t.Fatalf("最近对局应 1 条：%+v", prof.Recent)
	}
	r := prof.Recent[0]
	if r.Outcome != "win" || r.Opponent != "p2user" || r.EloBefore != 1000 || r.EloAfter != 1016 {
		t.Fatalf("最近对局项不符：%+v", r)
	}
	// 败者视角
	profb, err := s.GetProfile(ctx, b.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if profb.Losses != 1 || profb.WinRate != 0.0 || len(profb.Recent) != 1 || profb.Recent[0].Outcome != "loss" {
		t.Fatalf("败者视角不符：%+v", profb)
	}

	// 平局：双方 +0，但照样记行记历史
	mid2 := uuid.New()
	d1, d2, err := s.RecordMatch(ctx, MatchResult{
		MatchID: mid2, Winner: "", Reason: "rounds-cap", DurationMs: 60000,
		P1UserID: a.ID, P2UserID: b.ID,
	})
	if err != nil {
		t.Fatalf("平局落库失败：%v", err)
	}
	if d1.Before != d1.After || d2.Before != d2.After {
		t.Fatalf("平局必须 +0：%+v %+v", d1, d2)
	}
	prof2, _ := s.GetProfile(ctx, a.ID, 20)
	if prof2.Total != 2 || prof2.Draws != 1 || len(prof2.Recent) != 2 {
		t.Fatalf("平局后聚合不符：%+v", prof2)
	}
	// 条数钳制：0 与超限都回落默认（不断言具体默认，只断言可用）
	if p0, err := s.GetProfile(ctx, a.ID, 0); err != nil || p0.Total != 2 {
		t.Fatalf("limit=0 应回落可用：%v %+v", err, p0)
	}
	if pBig, err := s.GetProfile(ctx, a.ID, 9999); err != nil || len(pBig.Recent) > DefaultRecentLimit {
		t.Fatalf("超限 limit 应钳制：%v %+v", err, pBig)
	}
	if prof2.Recent[0].Outcome != "draw" {
		t.Fatalf("最新一条应为平局：%+v", prof2.Recent[0])
	}
}

// 并发结算同一 match：恰一行结果 + 两行历史，第二个提交必须失败（唯一约束验证）。
func TestConcurrentRecordSameMatch(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a, _, _ := s.Register(ctx, "racer1", "password1")
	b, _, _ := s.Register(ctx, "racer2", "password2")
	mid := uuid.New()
	res := MatchResult{MatchID: mid, Winner: "p1", Reason: "ko", P1UserID: a.ID, P2UserID: b.ID}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, errs[i] = s.RecordMatch(ctx, res)
		}(i)
	}
	wg.Wait()
	ok, fail := 0, 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			fail++
		}
	}
	if ok != 1 || fail != 1 {
		t.Fatalf("并发同场结算应一成一败，实际 errs=%v", errs)
	}
	var n int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM matches WHERE id = $1", mid).Scan(&n); err != nil || n != 1 {
		t.Fatalf("结果行应恰 1：n=%d err=%v", n, err)
	}
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM elo_history WHERE match_id = $1", mid).Scan(&n); err != nil || n != 2 {
		t.Fatalf("ELO 历史应恰 2：n=%d err=%v", n, err)
	}
	// H5：输家事务整体回滚，ELO 只应被结算一次（双重记分会是 1031/969 量级）
	ua, _ := s.q.GetUserByID(ctx, a.ID)
	ub, _ := s.q.GetUserByID(ctx, b.ID)
	if ua.Elo != 1016 || ub.Elo != 984 {
		t.Fatalf("并发结算只应生效一次：%d / %d", ua.Elo, ub.Elo)
	}
}
