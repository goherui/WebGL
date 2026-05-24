package biz

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"login-page/internal/data"
)

var (
	ErrUserExists        = errors.New("用户名已存在")
	ErrInvalidCredential = errors.New("用户名或密码错误")
)

type User struct {
	Username string
}

type AdminUser struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	LoginCount  int64      `json:"loginCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

type AdminStats struct {
	TotalUsers      int64 `json:"totalUsers"`
	TotalAIRequests int64 `json:"totalAIRequests"`
	RealAIRequests  int64 `json:"realAIRequests"`
	MockAIRequests  int64 `json:"mockAIRequests"`
}

type AIConfig struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
}

type UserUseCase struct {
	repo *data.Data
	log  *log.Helper
}

func NewUserUseCase(repo *data.Data, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *UserUseCase) Register(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(password) < 6 {
		return errors.New("用户名至少3位，密码至少6位")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return err
	}

	_, err = uc.repo.Exec(ctx, "INSERT INTO users (username, password) VALUES (?, ?)", username, string(hashed))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (uc *UserUseCase) Login(ctx context.Context, username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredential
	}

	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return nil, err
	}

	var hashed string
	err := uc.repo.QueryRow(ctx, "SELECT password FROM users WHERE username = ?", username).Scan(&hashed)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredential
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		return nil, ErrInvalidCredential
	}

	_, _ = uc.repo.Exec(ctx, "UPDATE users SET login_count = login_count + 1, last_login_at = NOW() WHERE username = ?", username)

	return &User{Username: username}, nil
}

func (uc *UserUseCase) ListUsers(ctx context.Context, limit int) ([]AdminUser, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return nil, err
	}
	rows, err := uc.repo.Query(ctx, `SELECT id, username, login_count, created_at, last_login_at FROM users ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]AdminUser, 0)
	for rows.Next() {
		var u AdminUser
		var last sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.LoginCount, &u.CreatedAt, &last); err != nil {
			return nil, err
		}
		if last.Valid {
			u.LastLoginAt = &last.Time
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (uc *UserUseCase) AdminStats(ctx context.Context) (*AdminStats, error) {
	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return nil, err
	}
	stats := &AdminStats{}
	_ = uc.repo.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	_ = uc.repo.QueryRow(ctx, "SELECT COUNT(*) FROM ai_usage_logs").Scan(&stats.TotalAIRequests)
	_ = uc.repo.QueryRow(ctx, "SELECT COUNT(*) FROM ai_usage_logs WHERE mock = 0").Scan(&stats.RealAIRequests)
	_ = uc.repo.QueryRow(ctx, "SELECT COUNT(*) FROM ai_usage_logs WHERE mock = 1").Scan(&stats.MockAIRequests)
	return stats, nil
}

func (uc *UserUseCase) GetAIConfig(ctx context.Context) (*AIConfig, error) {
	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return nil, err
	}
	cfg := &AIConfig{BaseURL: "https://api.deepseek.com", Model: "deepseek-chat"}
	settings, err := uc.getSettings(ctx, []string{"ai_api_key", "ai_api_base", "ai_model"})
	if err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(settings["ai_api_key"]); v != "" {
		cfg.APIKey = v
	}
	if v := strings.TrimSpace(settings["ai_api_base"]); v != "" {
		cfg.BaseURL = v
	}
	if v := strings.TrimSpace(settings["ai_model"]); v != "" {
		cfg.Model = v
	}
	return cfg, nil
}

func (uc *UserUseCase) SaveAIConfig(ctx context.Context, cfg AIConfig) error {
	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		return err
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}
	if err := uc.setSetting(ctx, "ai_api_base", cfg.BaseURL); err != nil {
		return err
	}
	if err := uc.setSetting(ctx, "ai_model", cfg.Model); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.APIKey) != "" && strings.TrimSpace(cfg.APIKey) != "********" {
		if err := uc.setSetting(ctx, "ai_api_key", strings.TrimSpace(cfg.APIKey)); err != nil {
			return err
		}
	}
	return nil
}

func (uc *UserUseCase) LogAIUsage(ctx context.Context, username, model string, mock bool) {
	if _, err := uc.repo.Exec(ctx, "USE login_app"); err != nil {
		uc.log.Warnf("use db failed when logging ai usage: %v", err)
		return
	}
	_, err := uc.repo.Exec(ctx, "INSERT INTO ai_usage_logs (username, model, mock) VALUES (?, ?, ?)", username, model, mock)
	if err != nil {
		uc.log.Warnf("log ai usage failed: %v", err)
	}
}

func (uc *UserUseCase) getSettings(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(keys)), ",")
	args := make([]interface{}, 0, len(keys))
	for _, k := range keys {
		args = append(args, k)
	}
	rows, err := uc.repo.Query(ctx, "SELECT `key`, `value` FROM app_settings WHERE `key` IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

func (uc *UserUseCase) setSetting(ctx context.Context, key, value string) error {
	_, err := uc.repo.Exec(ctx, `INSERT INTO app_settings (`+"`key`, `value`"+`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `+"`value`"+` = VALUES(`+"`value`"+`), updated_at = CURRENT_TIMESTAMP`, key, value)
	return err
}
