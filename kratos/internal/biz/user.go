package biz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"login-page/internal/data"
)

var (
	ErrUserExists       = errors.New("用户名已存在")
	ErrEmailExists      = errors.New("邮箱已存在")
	ErrInvalidCredential = errors.New("用户名或密码错误")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrCodeNotFound     = errors.New("验证码不存在或已过期")
	ErrCodeMismatch     = errors.New("验证码错误")
)

type User struct {
	ID           int
	Username     string
	Password     string
	Email        sql.NullString
	Phone        sql.NullString
	Nickname     sql.NullString
	Avatar       sql.NullString
	Role         string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  sql.NullTime
	LoginCount   int
	LastIP       sql.NullString
	EmailVerified bool
	PhoneVerified bool
	Bio          sql.NullString
	Location     sql.NullString
	Birthday     sql.NullTime
	Gender       sql.NullString
	Language     string
	Timezone     string
}

type UserUseCase struct {
	repo          *data.Data
	redis         *data.RedisClient
	email         *data.EmailService
	log           *log.Helper
}

func NewUserUseCase(repo *data.Data, redis *data.RedisClient, email *data.EmailService, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		repo:  repo,
		redis: redis,
		email: email,
		log:  log.NewHelper(logger),
	}
}

func (uc *UserUseCase) Register(ctx context.Context, username, password, email, phone string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(password) < 6 {
		return errors.New("用户名至少3位，密码至少6位")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err = uc.repo.Exec(ctx, "INSERT INTO users (username, password, email, phone) VALUES (?, ?, ?, ?)", 
		username, string(hashed), toNullString(email), toNullString(phone))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			if strings.Contains(err.Error(), "username") {
				return ErrUserExists
			}
			if strings.Contains(err.Error(), "email") {
				return ErrEmailExists
			}
			return ErrUserExists
		}
		return err
	}
	return nil
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func (uc *UserUseCase) Login(ctx context.Context, username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredential
	}

	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return nil, err
	}

	var hashed string
	var user User
	err := uc.repo.QueryRow(ctx, "SELECT id, username, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &hashed)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredential
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		return nil, ErrInvalidCredential
	}

	return &user, nil
}

func (uc *UserUseCase) GetUserByID(ctx context.Context, id int) (*User, error) {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return nil, err
	}

	var user User
	err := uc.repo.QueryRow(ctx, `
		SELECT id, username, email, phone, nickname, avatar, role, status,
			created_at, updated_at, last_login_at, login_count, last_ip,
			email_verified, phone_verified, bio, location, birthday, gender, language, timezone
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone, &user.Nickname,
		&user.Avatar, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
		&user.LastLoginAt, &user.LoginCount, &user.LastIP, &user.EmailVerified,
		&user.PhoneVerified, &user.Bio, &user.Location, &user.Birthday, &user.Gender,
		&user.Language, &user.Timezone,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (uc *UserUseCase) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return nil, err
	}

	var user User
	err := uc.repo.QueryRow(ctx, `
		SELECT id, username, email, phone, nickname, avatar, role, status,
			created_at, updated_at, last_login_at, login_count, last_ip,
			email_verified, phone_verified, bio, location, birthday, gender, language, timezone
		FROM users WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone, &user.Nickname,
		&user.Avatar, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
		&user.LastLoginAt, &user.LoginCount, &user.LastIP, &user.EmailVerified,
		&user.PhoneVerified, &user.Bio, &user.Location, &user.Birthday, &user.Gender,
		&user.Language, &user.Timezone,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (uc *UserUseCase) UpdateLastLogin(ctx context.Context, userID int, ip string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, `
		UPDATE users 
		SET last_login_at = NOW(), login_count = login_count + 1, last_ip = ?
		WHERE id = ?
	`, ip, userID)

	return err
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID int, nickname, bio, location, gender string, birthday time.Time) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	query := "UPDATE users SET "
	args := []interface{}{}
	updates := []string{}

	if nickname != "" {
		updates = append(updates, "nickname = ?")
		args = append(args, nickname)
	}
	if bio != "" {
		updates = append(updates, "bio = ?")
		args = append(args, bio)
	}
	if location != "" {
		updates = append(updates, "location = ?")
		args = append(args, location)
	}
	if gender != "" {
		updates = append(updates, "gender = ?")
		args = append(args, gender)
	}
	if !birthday.IsZero() {
		updates = append(updates, "birthday = ?")
		args = append(args, birthday)
	}

	if len(updates) == 0 {
		return nil
	}

	query += strings.Join(updates, ", ")
	query += " WHERE id = ?"
	args = append(args, userID)

	_, err := uc.repo.Exec(ctx, query, args...)
	return err
}

func (uc *UserUseCase) UpdateAvatar(ctx context.Context, userID int, avatarURL string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET avatar = ? WHERE id = ?", avatarURL, userID)
	return err
}

func (uc *UserUseCase) UpdateEmail(ctx context.Context, userID int, email string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET email = ?, email_verified = 0 WHERE id = ?", email, userID)
	return err
}

func (uc *UserUseCase) UpdatePhone(ctx context.Context, userID int, phone string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET phone = ?, phone_verified = 0 WHERE id = ?", phone, userID)
	return err
}

func (uc *UserUseCase) UpdatePassword(ctx context.Context, userID int, oldPassword, newPassword string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	var hashedPassword string
	err := uc.repo.QueryRow(ctx, "SELECT password FROM users WHERE id = ?", userID).Scan(&hashedPassword)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword)); err != nil {
		return ErrInvalidCredential
	}

	newHashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = uc.repo.Exec(ctx, "UPDATE users SET password = ? WHERE id = ?", string(newHashed), userID)
	return err
}

func (uc *UserUseCase) UpdateLanguage(ctx context.Context, userID int, language string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET language = ? WHERE id = ?", language, userID)
	return err
}

func (uc *UserUseCase) UpdateTimezone(ctx context.Context, userID int, timezone string) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET timezone = ? WHERE id = ?", timezone, userID)
	return err
}

func (uc *UserUseCase) UpdateEmailVerified(ctx context.Context, userID int, verified bool) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET email_verified = ? WHERE id = ?", verified, userID)
	return err
}

func (uc *UserUseCase) UpdatePhoneVerified(ctx context.Context, userID int, verified bool) error {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return err
	}

	_, err := uc.repo.Exec(ctx, "UPDATE users SET phone_verified = ? WHERE id = ?", verified, userID)
	return err
}

func (uc *UserUseCase) SendEmailCode(ctx context.Context, email string) error {
	if !uc.email.ValidateEmail(email) {
		return errors.New("邮箱格式不正确")
	}

	exists, err := uc.checkEmailExists(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailExists
	}

	code, err := uc.email.GenerateCode()
	if err != nil {
		return fmt.Errorf("生成验证码失败: %w", err)
	}

	key := fmt.Sprintf("email:code:%s", email)
	if err := uc.redis.Set(ctx, key, code, 60*time.Second); err != nil {
		return fmt.Errorf("存储验证码失败: %w", err)
	}

	uc.log.Infof("验证码已发送到 %s: %s (请在服务器控制台查看)", email, code)

	if err := uc.email.SendVerificationCode(email, code); err != nil {
		uc.log.Warnf("邮件发送失败，但验证码仍然有效: %v", err)
	}

	return nil
}

func (uc *UserUseCase) VerifyEmailCode(ctx context.Context, email, code string) error {
	key := fmt.Sprintf("email:code:%s", email)
	storedCode, err := uc.redis.Get(ctx, key)
	if err != nil {
		return ErrCodeNotFound
	}

	if storedCode != code {
		return ErrCodeMismatch
	}

	uc.redis.Del(ctx, key)
	return nil
}

func (uc *UserUseCase) checkEmailExists(ctx context.Context, email string) (bool, error) {
	if _, err := uc.repo.Exec(ctx, "USE future"); err != nil {
		return false, err
	}

	var count int
	err := uc.repo.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
