package service

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (s *LoginService) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, response{Code: 1, Msg: "参数错误"})
		return
	}

	err := s.uc.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, biz.ErrUserExists) {
			writeJSON(w, 409, response{Code: 1, Msg: "用户名已存在"})
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

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    user.Username,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	redirectTo := "/welcome"
	if isAdminUser(user.Username) {
		redirectTo = "/admin"
	}

	writeJSON(w, 200, response{Code: 0, Msg: "登录成功", Data: map[string]string{"username": user.Username, "redirectTo": redirectTo}})
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
	redirectTo := "/welcome"
	role := "user"
	if isAdminUser(cookie.Value) {
		redirectTo = "/admin"
		role = "admin"
	}
	writeJSON(w, 200, response{Code: 0, Msg: "已登录", Data: map[string]string{"username": cookie.Value, "role": role, "redirectTo": redirectTo}})
}
