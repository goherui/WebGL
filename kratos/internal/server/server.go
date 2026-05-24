package server

import (
	"net/http"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type LoginHandler interface {
	Register(http.ResponseWriter, *http.Request)
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	CheckAuth(http.ResponseWriter, *http.Request)
	GetUserInfo(http.ResponseWriter, *http.Request)
	UpdateProfile(http.ResponseWriter, *http.Request)
	UpdateAvatar(http.ResponseWriter, *http.Request)
	UpdateEmail(http.ResponseWriter, *http.Request)
	UpdatePhone(http.ResponseWriter, *http.Request)
	UpdatePassword(http.ResponseWriter, *http.Request)
	UpdateLanguage(http.ResponseWriter, *http.Request)
	UpdateTimezone(http.ResponseWriter, *http.Request)
	SendEmailCode(http.ResponseWriter, *http.Request)
	VerifyEmailCode(http.ResponseWriter, *http.Request)
}

func NewServer(svc LoginHandler, logger log.Logger) *kratos.App {
	httpSrv := khttp.NewServer(khttp.Address(":8088"))

	httpSrv.HandleFunc("/api/register", cors(svc.Register))
	httpSrv.HandleFunc("/api/email/send-code", cors(svc.SendEmailCode))
	httpSrv.HandleFunc("/api/email/verify-code", cors(svc.VerifyEmailCode))
	httpSrv.HandleFunc("/api/login", cors(svc.Login))
	httpSrv.HandleFunc("/api/logout", cors(svc.Logout))
	httpSrv.HandleFunc("/api/check-auth", cors(svc.CheckAuth))
	httpSrv.HandleFunc("/api/user/info", cors(svc.GetUserInfo))
	httpSrv.HandleFunc("/api/user/profile", cors(svc.UpdateProfile))
	httpSrv.HandleFunc("/api/user/avatar", cors(svc.UpdateAvatar))
	httpSrv.HandleFunc("/api/user/email", cors(svc.UpdateEmail))
	httpSrv.HandleFunc("/api/user/phone", cors(svc.UpdatePhone))
	httpSrv.HandleFunc("/api/user/password", cors(svc.UpdatePassword))
	httpSrv.HandleFunc("/api/user/language", cors(svc.UpdateLanguage))
	httpSrv.HandleFunc("/api/user/timezone", cors(svc.UpdateTimezone))

	fs := http.FileServer(http.Dir("../frontend/static"))
	httpSrv.HandlePrefix("/static/", http.StripPrefix("/static/", fs))
	httpSrv.HandleFunc("/bg.jpg", serveFile("../frontend/static/bg.jpg"))
	httpSrv.HandleFunc("/side.jpg", serveFile("../frontend/static/side.jpg"))

	httpSrv.Handle("/welcome", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/static/welcome.html")
	})))

	httpSrv.Handle("/lab", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/static/lab.html")
	})))

	httpSrv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/static/index.html")
	})

	return kratos.New(
		kratos.Name("login-page"),
		kratos.Logger(logger),
		kratos.Server(httpSrv),
	)
}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		h(w, r)
	}
}

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
