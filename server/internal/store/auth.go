package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"codewar/server/internal/elo"
	"codewar/server/internal/store/storedb"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// 会话有效期 30 天（过期令牌一律拒绝，客户端应引导重登）。
const SessionTTL = 30 * 24 * time.Hour

var (
	// ErrInvalidCredentials：登录失败统一错误——不区分"用户不存在 / 密码错误"，防枚举。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrUsernameTaken：注册用户名冲突（大小写不敏感）。
	ErrUsernameTaken = errors.New("用户名已被占用")
	// ErrInvalidUsername：用户名形态不合法（长度/空白），注册时报 400。
	ErrInvalidUsername = errors.New("用户名不合法")
	// ErrUnauthorized：令牌缺失 / 未知 / 过期。
	ErrUnauthorized = errors.New("未登录或会话已过期")
)

var validate = validator.New()

// 注册与登录的入参校验（长度约束与 DB CHECK 对齐）。
type credentials struct {
	Username string `validate:"required,min=2,max=32"`
	Password string `validate:"required,min=6,max=72"`
}

func checkCredentials(username, password string) error {
	if strings.ContainsFunc(username, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }) {
		return ErrInvalidUsername
	}
	return validate.Struct(credentials{Username: username, Password: password})
}

// newToken：32 字节随机不透明令牌（base64url 明文下发给客户端），
// 返回（明文，SHA-256 hex 落库值）。库内永存明文。
func newToken() (raw, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b[:])
	return raw, hashToken(raw), nil
}

// hashToken：令牌的 SHA-256 落库/索引形态。
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// Register：注册并直接登录（返回首个会话令牌）。
// 用户名大小写不敏感唯一；密码 bcrypt 落库（默认 cost）。
func (s *Store) Register(ctx context.Context, username, password string) (User, string, error) {
	if err := checkCredentials(username, password); err != nil {
		return User{}, "", err
	}
	lower := strings.ToLower(username)
	if _, err := s.q.GetUserByLower(ctx, lower); err == nil {
		return User{}, "", ErrUsernameTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return User{}, "", err
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", err
	}
	row, err := s.q.CreateUser(ctx, storedb.CreateUserParams{
		Username:      username,
		UsernameLower: lower,
		PasswordHash:  string(pwHash),
		Elo:           int32(elo.Initial),
	})
	if err != nil {
		return User{}, "", err
	}
	raw, _, err := s.newSession(ctx, row.ID)
	if err != nil {
		return User{}, "", err
	}
	return User{ID: row.ID, Username: row.Username, Elo: row.Elo, CreatedAt: row.CreatedAt}, raw, nil
}

// Login：校验后签发新会话（旧会话不挤掉：同用户多会话并存，支持两窗口两账号/同账号多端）。
func (s *Store) Login(ctx context.Context, username, password string) (User, string, error) {
	if err := checkCredentials(username, password); err != nil {
		return User{}, "", ErrInvalidCredentials
	}
	u, err := s.q.GetUserByLower(ctx, strings.ToLower(username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", err
	}
	// bcrypt 对比用常量时间语义；失败与"用户不存在"返回同一错误。
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return User{}, "", ErrInvalidCredentials
	}
	raw, _, err := s.newSession(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	return User{ID: u.ID, Username: u.Username, Elo: u.Elo, CreatedAt: u.CreatedAt}, raw, nil
}

// newSession：签发会话行，返回明文令牌（仅此一次可见）。
func (s *Store) newSession(ctx context.Context, userID uuid.UUID) (string, storedb.CreateSessionRow, error) {
	raw, hash, err := newToken()
	if err != nil {
		return "", storedb.CreateSessionRow{}, err
	}
	row, err := s.q.CreateSession(ctx, storedb.CreateSessionParams{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(SessionTTL),
	})
	if err != nil {
		return "", storedb.CreateSessionRow{}, err
	}
	return raw, row, nil
}

// Authenticate：Bearer 令牌 → 用户（哈希等值查询 + 过期检查）。
// 库内只存令牌哈希：即使 DB 只读泄露，攻击者也无法直接冒用会话。
func (s *Store) Authenticate(ctx context.Context, rawToken string) (User, error) {
	if rawToken == "" {
		return User{}, ErrUnauthorized
	}
	row, err := s.q.GetSessionByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}
	if time.Now().After(row.ExpiresAt) {
		return User{}, ErrUnauthorized
	}
	return User{ID: row.UserID, Username: row.Username, Elo: row.Elo}, nil
}

// Logout：注销当前会话（幂等：令牌未知也返回 nil）。
func (s *Store) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.q.DeleteSessionByHash(ctx, hashToken(rawToken))
}
