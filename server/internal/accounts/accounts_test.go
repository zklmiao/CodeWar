package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codewar/server/internal/store"
	"codewar/server/internal/testdb"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// 无库模式：所有账号接口 503，匿名热座不受影响（本测试无需 DB，常开）。
func TestNilStoreUnavailable(t *testing.T) {
	h := New(nil, slog.Default())
	r := chi.NewRouter()
	h.Mount(r)

	cases := []struct {
		method, target, body string
	}{
		{"POST", "/api/register", `{"username":"a","password":"b"}`},
		{"POST", "/api/login", `{"username":"a","password":"b"}`},
		{"GET", "/api/me", ""},
		{"GET", "/api/profile", ""},
		{"GET", "/api/tutorial", ""},
		{"POST", "/api/logout", ""},
	}
	for _, c := range cases {
		var body *bytes.Reader
		if c.body != "" {
			body = bytes.NewReader([]byte(c.body))
		} else {
			body = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(c.method, c.target, body)
		if c.method == "GET" {
			req.Header.Set("Authorization", "Bearer x")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s 无库模式应 503，实际 %d", c.method, c.target, rec.Code)
		}
	}
}

// 门控：HTTP 全流程（注册 201 → me → profile → logout → 401；409/401/400 错误路径）。
// 17 号票：使用 accounts 专属测试库（codewar_test_accounts），与 store/hub 并行不互删。
func TestHTTPFlow(t *testing.T) {
	dsn := testdb.DSN(t, "accounts")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("测试库连接失败：%v", err)
	}
	defer st.Close()
	for _, tbl := range []string{"elo_history", "matches", "sessions", "users"} {
		if _, err := st.DB().Exec("TRUNCATE " + tbl + " CASCADE"); err != nil {
			t.Fatal(err)
		}
	}

	h := New(st, slog.Default())
	r := chi.NewRouter()
	h.Mount(r)
	call := func(method, target, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 注册
	rec := call("POST", "/api/register", `{"username":"HttpUser","password":"pw-1234"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("注册应 201，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var sess struct {
		ID, Username, Token string
		Elo                 int32
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil || sess.Token == "" || sess.Elo != 1000 {
		t.Fatalf("注册返回异常：%s err=%v", rec.Body.String(), err)
	}

	// 10 号票：GET /api/tutorial 是唯一读进度入口（无写入端点）。
	rec = call("GET", "/api/tutorial", "", sess.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("教程进度应 200，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var tut struct {
		Level       int32 `json:"level"`
		Completed   bool  `json:"completed"`
		TotalLevels int32 `json:"total_levels"`
		NextLevel   int32 `json:"next_level"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tut); err != nil {
		t.Fatal(err)
	}
	if tut.Level != 0 || tut.Completed || tut.TotalLevels != 4 || tut.NextLevel != 1 {
		t.Fatalf("新账号教程进度不符：%+v", tut)
	}
	uid, err := uuid.Parse(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AdvanceTutorial(ctx, uid, 2, 4); err != nil {
		t.Fatal(err)
	}
	rec = call("GET", "/api/tutorial", "", sess.Token)
	if err := json.Unmarshal(rec.Body.Bytes(), &tut); err != nil {
		t.Fatal(err)
	}
	if tut.Level != 2 || tut.Completed || tut.NextLevel != 3 {
		t.Fatalf("推进后只读接口应反映进度：%+v", tut)
	}
	// 未登录：401（不泄露进度）。
	if rec := call("GET", "/api/tutorial", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应 401，实际 %d", rec.Code)
	}

	// 重名 409
	if rec := call("POST", "/api/register", `{"username":"httpuser","password":"pw-1234"}`, ""); rec.Code != http.StatusConflict {
		t.Fatalf("重名注册应 409，实际 %d", rec.Code)
	}
	// 参数 400
	if rec := call("POST", "/api/register", `{"username":"x","password":"pw-1234"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("非法参数应 400，实际 %d", rec.Code)
	}
	// 含空白用户名 400（P1：曾误判 500——store 哨兵错误未被 handler 映射）
	if rec := call("POST", "/api/register", `{"username":"bad name","password":"pw-1234"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("含空白用户名应 400，实际 %d：%s", rec.Code, rec.Body.String())
	}
	// 非法 JSON 体 → 400（解码分支覆盖）
	if rec := call("POST", "/api/register", `{"username":`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 注册应 400，实际 %d", rec.Code)
	}
	if rec := call("POST", "/api/login", `not-json`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 登录应 400，实际 %d", rec.Code)
	}
	// 注销不带令牌 → 401
	if rec := call("POST", "/api/logout", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("无令牌 logout 应 401，实际 %d", rec.Code)
	}
	// 错误密码 401（与用户不存在同错，响应体同文）
	rec = call("POST", "/api/login", `{"username":"HttpUser","password":"nope-nope"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码应 401，实际 %d", rec.Code)
	}

	// me（未带令牌 401）
	if rec := call("GET", "/api/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("无令牌 me 应 401，实际 %d", rec.Code)
	}
	rec = call("GET", "/api/me", "", sess.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me 应 200，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// profile（零场次：胜率 0，recent 空）
	rec = call("GET", "/api/profile", "", sess.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile 应 200，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var prof struct {
		Username string
		Total    int32
		WinRate  float64
		Recent   []any
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &prof); err != nil || prof.Username != "HttpUser" || prof.Total != 0 || prof.WinRate != 0 {
		t.Fatalf("profile 聚合异常：%s err=%v", rec.Body.String(), err)
	}

	// logout（204，幂等：再登出一次仍 204；之后 me 变 401）
	if rec := call("POST", "/api/logout", "", sess.Token); rec.Code != http.StatusNoContent {
		t.Fatalf("logout 应 204，实际 %d", rec.Code)
	}
	if rec := call("POST", "/api/logout", "", sess.Token); rec.Code != http.StatusNoContent {
		t.Fatalf("重复 logout 应仍 204，实际 %d", rec.Code)
	}
	if rec := call("GET", "/api/me", "", sess.Token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("登出后 me 应 401，实际 %d", rec.Code)
	}
}

// 关库后的 handler：DB 错误映射 500 而非 panic（500 分支覆盖）。
func TestClosedStoreMaps500(t *testing.T) {
	dsn := testdb.DSN(t, "accounts")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("测试库连接失败：%v", err)
	}
	st.Close()
	h := New(st, slog.Default())
	r := chi.NewRouter()
	h.Mount(r)
	call := func(method, target, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	// token 形态合法但库已关：鉴权查库失败 → 401（current 统一口径，不 500 不 panic）
	if rec := call("GET", "/api/me", "tok"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("关库后 me 应 401，实际 %d", rec.Code)
	}
	// 注册/登录的库错误 → 500（default 分支覆盖；注意用合法长度参数，
	// 否则先倒在参数校验 400，到不了存储层）
	body := strings.NewReader(`{"username":"closeduser","password":"pw-1234"}`)
	req := httptest.NewRequest("POST", "/api/register", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("关库后 register 应 500，实际 %d", rec.Code)
	}
	body = strings.NewReader(`{"username":"closeduser","password":"pw-1234"}`)
	req = httptest.NewRequest("POST", "/api/login", body)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("关库后 login 应 500，实际 %d", rec.Code)
	}
	// 注销直调存储层 → 500（logout 的 err 分支覆盖）
	req = httptest.NewRequest("POST", "/api/logout", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("关库后 logout 应 500，实际 %d", rec.Code)
	}
	// 注：profile 的 500 需"鉴权成功但聚合失败"的瞬间库故障，常规手段
	// 无法稳定构造（关库会先倒在鉴权），该 2 行分支留作已知缺口，见报告。
}
