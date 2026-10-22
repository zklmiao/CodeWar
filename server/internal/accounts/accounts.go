// Package accounts：账号 HTTP 接口（07 号票）：注册 / 登录 / 当前用户 / 个人页 / 注销。
// 认证形态：服务端不透明令牌 + 客户端 sessionStorage，请求头 Authorization: Bearer 携带。
// 无 token = 匿名热座（WS 行为与 01 号票一致）；HttpOnly cookie 评估归 08 号票。
package accounts

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"codewar/server/internal/store"
	"codewar/server/internal/tutorial"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// Handler：账号路由（st 为 nil = 无库模式，所有接口 503，匿名热座不受影响）。
type Handler struct {
	st     *store.Store
	logger *slog.Logger
}

func New(st *store.Store, logger *slog.Logger) *Handler {
	return &Handler{st: st, logger: logger}
}

// Mount：注册 /api/* 路由。
func (h *Handler) Mount(r chi.Router) {
	r.Post("/api/register", h.register)
	r.Post("/api/login", h.login)
	r.Get("/api/me", h.me)
	r.Get("/api/profile", h.profile)
	// 10 号票：教程进度只读接口（客户端没有任何写入/advance 端点）。
	r.Get("/api/tutorial", h.tutorial)
	r.Post("/api/logout", h.logout)
}

type credentialBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionJSON struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Elo      int32  `json:"elo"`
	Token    string `json:"token"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// requireStore：无库模式统一 503（调用方继续前先判 false）。
func (h *Handler) requireStore(w http.ResponseWriter) bool {
	if h.st == nil {
		writeErr(w, http.StatusServiceUnavailable, "账号服务未启用（服务端未接数据库），可匿名热座")
		return false
	}
	return true
}

// bearer：解析 Authorization: Bearer <token>，缺失/形态不对即 401。
func bearer(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok || strings.TrimSpace(tok) == "" {
		return "", false
	}
	return strings.TrimSpace(tok), true
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	var body credentialBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	u, tok, err := h.st.Register(r.Context(), body.Username, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUsernameTaken):
			writeErr(w, http.StatusConflict, err.Error())
		case errors.Is(err, store.ErrInvalidUsername) || isValidation(err):
			writeErr(w, http.StatusBadRequest, "用户名 2~32 字符（不含空白）、密码 6~72 字符")
		default:
			h.logger.Error("注册失败", "err", err)
			writeErr(w, http.StatusInternalServerError, "服务端异常，稍后重试")
		}
		return
	}
	writeJSON(w, http.StatusCreated, sessionJSON{ID: u.ID.String(), Username: u.Username, Elo: u.Elo, Token: tok})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	var body credentialBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	u, tok, err := h.st.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		// 登录失败统一 401，不区分用户不存在/密码错误（防枚举），
		// 参数不合法也归一为 401（与 store 约定一致）。
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		h.logger.Error("登录失败", "err", err)
		writeErr(w, http.StatusInternalServerError, "服务端异常，稍后重试")
		return
	}
	writeJSON(w, http.StatusOK, sessionJSON{ID: u.ID.String(), Username: u.Username, Elo: u.Elo, Token: tok})
}

// current：Bearer 鉴权，失败写 401 并返回 false。
func (h *Handler) current(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	if !h.requireStore(w) {
		return store.User{}, false
	}
	tok, ok := bearer(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "需要登录（Authorization: Bearer <token>）")
		return store.User{}, false
	}
	u, err := h.st.Authenticate(r.Context(), tok)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "未登录或会话已过期")
		return store.User{}, false
	}
	return u, true
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, ok := h.current(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": u.ID.String(), "username": u.Username, "elo": u.Elo})
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	u, ok := h.current(w, r)
	if !ok {
		return
	}
	p, err := h.st.GetProfile(r.Context(), u.ID, store.DefaultRecentLimit)
	if err != nil {
		h.logger.Error("个人页聚合失败", "err", err)
		writeErr(w, http.StatusInternalServerError, "服务端异常，稍后重试")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// tutorial：GET /api/tutorial —— 当前用户的教程进度（只读）。
// 进度由服务端目标判定器在关卡达成后写入；本接口不提供任何推进能力。
func (h *Handler) tutorial(w http.ResponseWriter, r *http.Request) {
	u, ok := h.current(w, r)
	if !ok {
		return
	}
	prog, err := h.st.GetTutorialProgress(r.Context(), u.ID)
	if err != nil {
		h.logger.Error("教程进度读取失败", "err", err)
		writeErr(w, http.StatusInternalServerError, "服务端异常，稍后重试")
		return
	}
	next := int(prog.Level) + 1
	if prog.Completed || next > tutorial.MaxLevel {
		next = 0
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"level":        prog.Level,
		"completed":    prog.Completed,
		"completed_at": prog.CompletedAt,
		"total_levels": tutorial.MaxLevel,
		"next_level":   next,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	tok, ok := bearer(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "需要登录（Authorization: Bearer <token>）")
		return
	}
	// 幂等：未知令牌也视为已登出（204）。
	if err := h.st.Logout(r.Context(), tok); err != nil {
		h.logger.Error("注销失败", "err", err)
		writeErr(w, http.StatusInternalServerError, "服务端异常，稍后重试")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isValidation(err error) bool {
	var ves validator.ValidationErrors
	return errors.As(err, &ves)
}
