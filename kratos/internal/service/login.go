package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"login-page/internal/biz"
)

type LoginService struct {
	uc  *biz.UserUseCase
	log *log.Helper
}

func NewLoginService(uc *biz.UserUseCase, logger log.Logger) *LoginService {
	return &LoginService{uc: uc, log: log.NewHelper(logger)}
}

type response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, data response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (s *LoginService) SendEmailCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	if req.Email == "" {
		writeJSON(w, 400, response{Code: 1, Msg: "请输入邮箱"})
		return
	}

	err := s.uc.SendEmailCode(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, biz.ErrEmailExists) {
			writeJSON(w, 409, response{Code: 1, Msg: "该邮箱已被注册"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "验证码已发送到邮箱"})
}

func (s *LoginService) VerifyEmailCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	if req.Email == "" || req.Code == "" {
		writeJSON(w, 400, response{Code: 1, Msg: "请填写完整信息"})
		return
	}

	err := s.uc.VerifyEmailCode(r.Context(), req.Email, req.Code)
	if err != nil {
		if errors.Is(err, biz.ErrCodeNotFound) {
			writeJSON(w, 400, response{Code: 1, Msg: "验证码已过期，请重新获取"})
			return
		}
		if errors.Is(err, biz.ErrCodeMismatch) {
			writeJSON(w, 400, response{Code: 1, Msg: "验证码错误"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "验证成功"})
}

func (s *LoginService) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err := s.uc.Register(r.Context(), req.Username, req.Password, req.Email, req.Phone)
	if err != nil {
		if errors.Is(err, biz.ErrUserExists) {
			writeJSON(w, 409, response{Code: 1, Msg: "用户名已存在"})
			return
		}
		if errors.Is(err, biz.ErrEmailExists) {
			writeJSON(w, 409, response{Code: 1, Msg: "邮箱已被注册"})
			return
		}
		writeJSON(w, 400, response{Code: 1, Msg: err.Error()})
		return
	}
	writeJSON(w, 200, response{Code: 0, Msg: "注册成功"})
}

func (s *LoginService) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	user, err := s.uc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredential) {
			writeJSON(w, 401, response{Code: 1, Msg: "用户名或密码错误"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}
	s.uc.UpdateLastLogin(r.Context(), user.ID, clientIP)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    user.Username,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, 200, response{Code: 0, Msg: "登录成功", Data: map[string]string{"username": user.Username}})
}

func (s *LoginService) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, 200, response{Code: 0, Msg: "退出成功"})
}

func (s *LoginService) CheckAuth(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}
	writeJSON(w, 200, response{Code: 0, Msg: "已登录", Data: map[string]string{"username": cookie.Value}})
}

func (s *LoginService) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) {
			writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	userData := map[string]interface{}{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email.String,
		"phone":          user.Phone.String,
		"nickname":       user.Nickname.String,
		"avatar":         user.Avatar.String,
		"role":           user.Role,
		"status":         user.Status,
		"created_at":     user.CreatedAt.Format(time.RFC3339),
		"updated_at":     user.UpdatedAt.Format(time.RFC3339),
		"last_login_at":  "",
		"login_count":    user.LoginCount,
		"last_ip":        user.LastIP.String,
		"email_verified": user.EmailVerified,
		"phone_verified": user.PhoneVerified,
		"bio":            user.Bio.String,
		"location":       user.Location.String,
		"birthday":       "",
		"gender":         user.Gender.String,
		"language":       user.Language,
		"timezone":       user.Timezone,
	}
	if user.LastLoginAt.Valid {
		userData["last_login_at"] = user.LastLoginAt.Time.Format(time.RFC3339)
	}
	if user.Birthday.Valid {
		userData["birthday"] = user.Birthday.Time.Format("2006-01-02")
	}

	writeJSON(w, 200, response{Code: 0, Msg: "获取成功", Data: userData})
}

func (s *LoginService) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) {
			writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Bio      string `json:"bio"`
		Location string `json:"location"`
		Gender   string `json:"gender"`
		Birthday string `json:"birthday"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	var birthday time.Time
	if req.Birthday != "" {
		birthday, err = time.Parse("2006-01-02", req.Birthday)
		if err != nil {
			writeJSON(w, 400, response{Code: 1, Msg: "日期格式错误"})
			return
		}
	}

	err = s.uc.UpdateProfile(r.Context(), user.ID, req.Nickname, req.Bio, req.Location, req.Gender, birthday)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		Avatar string `json:"avatar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdateAvatar(r.Context(), user.ID, req.Avatar)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdateEmail(r.Context(), user.ID, req.Email)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdatePhone(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdatePhone(r.Context(), user.ID, req.Phone)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdatePassword(r.Context(), user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredential) {
			writeJSON(w, 401, response{Code: 1, Msg: "旧密码错误"})
			return
		}
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdateLanguage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdateLanguage(r.Context(), user.ID, req.Language)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}

func (s *LoginService) UpdateTimezone(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		writeJSON(w, 401, response{Code: 1, Msg: "未登录"})
		return
	}

	user, err := s.uc.GetUserByUsername(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, 404, response{Code: 1, Msg: "用户不存在"})
		return
	}

	var req struct {
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err = s.uc.UpdateTimezone(r.Context(), user.ID, req.Timezone)
	if err != nil {
		writeJSON(w, 500, response{Code: 1, Msg: err.Error()})
		return
	}

	writeJSON(w, 200, response{Code: 0, Msg: "更新成功"})
}
