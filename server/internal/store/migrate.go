// Package store：账号 / 会话 / 对局结果 / ELO 历史的数据访问层（07 号票，ADR-0001 sqlc 纪律）。
//
// 迁移（db/migrations 为权威源）通过 go:embed 内嵌进 Go 二进制，
// 应用启动时自动执行（幂等）：server/ 模块无法直接 embed 仓库根的 db/ 目录，
// 故此处放一份嵌入副本，漂移由 sync_test.go 守护（与 db/migrations 字节一致）。
package store

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// ApplyMigrations：幂等执行内嵌迁移（无新版本时返回 nil，不报错）。
func ApplyMigrations(db *sql.DB) error {
	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return err
	}
	inst, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", inst)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
