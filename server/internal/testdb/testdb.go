// Package testdb：真实 Postgres 集成测试的库隔离辅助（17 号票）。
//
// 每个测试包使用独立数据库（codewar_test_<suffix>），避免 `go test ./...`
// 并行跑不同包时互相 TRUNCATE。源 DSN 由 CODEWAR_TEST_DSN 提供，且源库名
// 必须含 "test"（破坏性护栏）；目标库不存在时自动 CREATE DATABASE。
//
// 只被测试 import，不进入生产链路。
package testdb

import (
	"database/sql"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/lib/pq"
)

// DSN：返回指向本包专属测试库的 DSN；变量缺失/源库名不合规时 t.Skip。
// 目标库已存在时直接复用（迁移由调用方 store.Open 幂等执行）。
func DSN(t testing.TB, suffix string) string {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("CODEWAR_TEST_DSN"))
	if raw == "" {
		t.Skip("CODEWAR_TEST_DSN 未设置，跳过 DB 集成测试（CI 无 DB 保持绿）")
	}
	src, ok := dbNameOf(raw)
	if !ok {
		t.Skipf("CODEWAR_TEST_DSN 无法解析，跳过：%q", raw)
	}
	if !strings.Contains(strings.ToLower(src), "test") {
		t.Skip("测试库名必须含 test（防误清共享库），当前 DSN 被拒绝")
	}
	name := "codewar_test_" + sanitize(suffix)
	target, ok := replaceDBName(raw, name)
	if !ok {
		t.Skipf("CODEWAR_TEST_DSN 无法改写库名，跳过：%q", raw)
	}

	admin, err := sql.Open("postgres", raw)
	if err != nil {
		t.Fatalf("测试库管理连接失败：%v", err)
	}
	defer admin.Close()
	var exists bool
	if err := admin.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name,
	).Scan(&exists); err != nil {
		t.Fatalf("查询测试库存在性失败：%v", err)
	}
	if !exists {
		if _, err := admin.Exec("CREATE DATABASE " + quoteIdent(name)); err != nil {
			// 并发创建同一库：另一测试进程刚建好，视为成功。
			var pqErr *pq.Error
			if !(errors.As(err, &pqErr) && pqErr.Code == "42P04") &&
				!strings.Contains(err.Error(), "already exists") {
				t.Fatalf("创建测试库 %s 失败：%v", name, err)
			}
		}
	}
	return target
}

// sanitize：库名后缀只保留小写字母/数字/下划线，避免 DDL 注入。
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

// quoteIdent：Postgres 标识符转义（CREATE DATABASE 不支持参数占位符）。
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// dbNameOf：从 URL 或 key=value DSN 提取库名。
func dbNameOf(raw string) (string, bool) {
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		u, err := url.Parse(raw)
		if err != nil || u.Path == "" {
			return "", false
		}
		return strings.TrimPrefix(u.Path, "/"), true
	}
	for _, kv := range strings.Fields(raw) {
		if name, ok := strings.CutPrefix(kv, "dbname="); ok {
			return name, true
		}
	}
	return "", false
}

// replaceDBName：返回指向 name 的同源 DSN（URL 与 key=value 两种形态）。
func replaceDBName(raw, name string) (string, bool) {
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", false
		}
		u.Path = "/" + name
		return u.String(), true
	}
	fields := strings.Fields(raw)
	found := false
	for i, kv := range fields {
		if strings.HasPrefix(kv, "dbname=") {
			fields[i] = "dbname=" + name
			found = true
		}
	}
	if !found {
		fields = append(fields, "dbname="+name)
	}
	return strings.Join(fields, " "), true
}
