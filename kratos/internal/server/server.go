package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type LoginHandler interface {
	Register(http.ResponseWriter, *http.Request)
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	CheckAuth(http.ResponseWriter, *http.Request)
	Chat(http.ResponseWriter, *http.Request)
	CheckAdmin(http.ResponseWriter, *http.Request)
	AdminOverview(http.ResponseWriter, *http.Request)
	AdminAIConfig(http.ResponseWriter, *http.Request)
}

func NewServer(svc LoginHandler, logger log.Logger) *kratos.App {
	httpSrv := khttp.NewServer(khttp.Address(":8088"))

	httpSrv.HandleFunc("/api/register", cors(svc.Register))
	httpSrv.HandleFunc("/api/login", cors(svc.Login))
	httpSrv.HandleFunc("/api/logout", cors(svc.Logout))
	httpSrv.HandleFunc("/api/check-auth", cors(svc.CheckAuth))
	httpSrv.HandleFunc("/api/ai/chat", cors(svc.Chat))
	httpSrv.HandleFunc("/api/admin/check", cors(svc.CheckAdmin))
	httpSrv.HandleFunc("/api/admin/overview", cors(svc.AdminOverview))
	httpSrv.HandleFunc("/api/admin/ai-config", cors(svc.AdminAIConfig))

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

	httpSrv.Handle("/admin", requireAdminPage(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/static/admin.html")
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

func requireAdminPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil || !isAdminUsername(cookie.Value) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAdminUsername(username string) bool {
	username = strings.TrimSpace(username)
	if username == "" {
		return false
	}
	adminUsers := strings.TrimSpace(os.Getenv("ADMIN_USERS"))
	if adminUsers == "" {
		adminUsers = "admin"
	}
	for _, item := range strings.Split(adminUsers, ",") {
		if strings.EqualFold(strings.TrimSpace(item), username) {
			return true
		}
	}
	return false
}
