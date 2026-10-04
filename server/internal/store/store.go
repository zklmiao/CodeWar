package store

import (
	"context"
	"database/sql"
	"time"

	"codewar/server/internal/store/storedb"

	"github.com/google/uuid"
)

// User：对外用户视图（绝不含密码哈希）。
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Elo       int32     `json:"elo"`
	CreatedAt time.Time `json:"created_at"`
}

// Store：数据访问层（sqlc 生成代码的薄封装：事务、ELO 结算、密码/令牌安全原语在此）。
// DB 为空时不允许构造——无库模式由调用方（main/hub）以 nil Store 表达，
// 此时账号接口 503、WS 只走匿名热座（向后兼容）。
type Store struct {
	db *sql.DB
	q  *storedb.Queries
}

// Open：建连 + Ping + 自动迁移（幂等），失败即返回错误（调用方决定降级还是退出）。
func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(16)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := ApplyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, q: storedb.New(db)}, nil
}

// Close：释放连接池（测试与优雅退出用）。
func (s *Store) Close() error { return s.db.Close() }

// DB：暴露底层连接（hub 测试的并发断言等需要直接 SQL 的场景用；业务代码走方法）。
func (s *Store) DB() *sql.DB { return s.db }
