package biz

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"login-page/internal/data"
)

var (
	ErrUserExists       = errors.New("用户名已存在")
	ErrInvalidCredential = errors.New("用户名或密码错误")
)

type User struct {
	Username string
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

	return &User{Username: username}, nil
}