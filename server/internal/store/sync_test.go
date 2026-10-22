package store

import (
	"os"
	"path/filepath"
	"testing"
)

// 漂移守护：嵌入副本必须与 db/migrations 权威源字节一致。
// server/ 是独立 Go 模块，go:embed 够不到仓库根，只能放副本——
// 一旦两边不一致，CI 即红，禁止手改副本（改 db/migrations 后拷过来）。
func TestMigrationsInSync(t *testing.T) {
	for _, name := range []string{
		"000001_init.up.sql", "000001_init.down.sql",
		"000002_tutorial.up.sql", "000002_tutorial.down.sql",
	} {
		embedded, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("嵌入迁移缺失 %s: %v", name, err)
		}
		// 本文件在 server/internal/store，仓库根在其上三层。
		canonical, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", name))
		if err != nil {
			t.Fatalf("权威迁移不可读 %s: %v", name, err)
		}
		if string(embedded) != string(canonical) {
			t.Fatalf("迁移副本与权威源不一致：%s（改 db/migrations 后把文件拷到 server/internal/store/migrations/）", name)
		}
	}
}
