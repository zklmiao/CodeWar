// CodeWar 主服务入口：WS 对局 + 静态资源托管（ADR-0001）。
// 端口约定（AGENTS.md）：本票 :3001；judge gRPC :50051。
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"codewar/server/gen/judgepb"
	"codewar/server/internal/accounts"
	"codewar/server/internal/classes"
	"codewar/server/internal/hub"
	"codewar/server/internal/judgeclient"
	"codewar/server/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":3001"), "HTTP/WS 监听地址")
	judgeAddr := flag.String("judge", envOr("JUDGE_ADDR", "localhost:50051"), "judge gRPC 地址")
	staticDir := flag.String("static", envOr("STATIC_DIR", "../client/dist"), "客户端构建产物目录（托管 / ）")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	conn, err := grpc.NewClient(*judgeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("judge gRPC 客户端创建失败", "err", err)
		os.Exit(1)
	}
	judge := judgeclient.NewGRPCJudge(judgepb.NewJudgeServiceClient(conn), 8000)

	// 07 号票：DATABASE_URL 为空 = 无库模式（纯匿名热座，账号接口 503）；
	// 配置了但连不上/迁不上 = 直接退出（运维意图明确，静默降级会掩盖故障）。
	var st *store.Store
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		s, err := store.Open(context.Background(), dsn)
		if err != nil {
			logger.Error("数据库连接/迁移失败", "err", err)
			os.Exit(1)
		}
		defer s.Close()
		st = s
		logger.Info("账号与战绩存储已启用（迁移自动执行完毕）")
	} else {
		logger.Warn("DATABASE_URL 未配置：账号接口不可用，仅匿名热座")
	}

	h := hub.New(judge, logger, st)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// 06 号票：职业数据同源 JSON（客户端职业选择的唯一权威，裁决 1）
	r.Get("/api/classes", classes.HTTPHandler())
	accounts.New(st, logger).Mount(r)
	r.Get("/ws", h.ServeWS)
	mountStatic(r, *staticDir)

	logger.Info("codewar server 启动", "addr", *addr, "judge", *judgeAddr, "static", *staticDir)
	if err := http.ListenAndServe(*addr, r); err != nil {
		logger.Error("HTTP 服务退出", "err", err)
		os.Exit(1)
	}
}

// mountStatic：托管客户端构建产物；SPA 回退到 index.html。目录不存在时仅告警（纯 WS 开发模式）。
func mountStatic(r chi.Router, dir string) {
	fs := http.FileServer(http.Dir(dir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			req.URL.Path = "/" // 深链回退 index.html
		}
		fs.ServeHTTP(w, req)
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
