# 未来实验室 WebGL 登录系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现一个完整的用户认证系统，用户登录后进入沉浸式的"未来实验室"WebGL体验页面

**Architecture:** 使用Go后端+gorilla/sessions管理Session+Redis存储，前端三个页面（登录、欢迎、实验室）使用原生HTML/CSS/JS+Three.js构建WebGL场景

**Tech Stack:** Go 1.22+, gorilla/sessions, go-redis/redis/v8, Three.js, MySQL, Redis

---

## 项目文件结构

```
d:\桌面\WebGL/
├── main.go                      # Go后端主程序 (重构)
├── go.mod                       # Go模块定义 (更新依赖)
├── static/
│   ├── index.html               # 登录/注册页面 (修改)
│   ├── welcome.html             # 欢迎页面 (新增)
│   ├── lab.html                # WebGL实验室页面 (新增)
│   ├── css/
│   │   └── style.css          # 通用样式 (新增)
│   └── js/
│       └── auth.js            # 认证逻辑 (新增)
└── docs/
    └── specs/
        └── 2026-05-21-future-lab-design.md  # 设计文档
```

---

## 阶段一：后端Session支持

### Task 1: 添加Redis和Session依赖

**Files:**
- Modify: `d:\桌面\WebGL\go.mod`
- Modify: `d:\桌面\WebGL\go.sum` (自动生成)

- [ ] **Step 1: 更新go.mod添加新依赖**

```go
module login-page

go 1.22.0

require (
	github.com/go-sql-driver/mysql v1.8.1
	golang.org/x/crypto v0.28.0
	github.com/gorilla/sessions v1.2.2
	github.com/go-redis/redis/v8 v8.11.5
)
```

- [ ] **Step 2: 运行go mod tidy下载依赖**

Run: `cd "d:\桌面\WebGL"; go mod tidy`
Expected: 成功下载github.com/gorilla/sessions和github.com/go-redis/redis/v8

- [ ] **Step 3: 验证依赖是否正确**

Run: `cd "d:\桌面\WebGL"; go list -m all | findstr "gorilla redis"`
Expected: 显示gorilla/sessions和redis相关包

---

### Task 2: 实现Redis连接和Session管理

**Files:**
- Modify: `d:\桌面\WebGL\main.go` (添加Redis连接和Session初始化)

- [ ] **Step 1: 添加Redis配置常量和全局变量**

在main.go文件开头（import之后，全局变量之前）添加：

```go
const (
    redisAddr = "localhost:6379"  // Redis地址，可根据环境修改
    sessionName = "session_id"
    sessionMaxAge = 86400  // 24小时
    sessionKey = "session:"
)
```

在全局变量区域添加：

```go
var (
    db *sql.DB
    store *sessions.CookieStore  // 新增
    rdb *redis.Client            // 新增
)
```

- [ ] **Step 2: 在init()函数之后添加initRedis()函数**

```go
func initRedis() error {
    rdb = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: "", // Redis密码，根据实际情况修改
        DB:       0,
    })

    ctx := context.Background()
    _, err := rdb.Ping(ctx).Result()
    if err != nil {
        log.Printf("Redis连接失败: %v，将使用内存存储", err)
        // 降级到内存存储
        store = sessions.NewCookieStore([]byte("something-very-secret-256bit"))
        return nil
    }

    // 使用Redis存储Session
    store = sessions.NewCookieStore([]byte("something-very-secret-256bit"))
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   sessionMaxAge,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
        Secure:   false, // 开发环境，生产环境设为true
    }

    log.Println("Redis已连接，Session将存储在Redis中")
    return nil
}
```

- [ ] **Step 3: 在main()函数中调用initRedis()**

在数据库初始化之后，路由配置之前添加：

```go
// 初始化Redis
if err := initRedis(); err != nil {
    log.Fatal("Redis初始化失败:", err)
}
```

- [ ] **Step 4: 测试Redis连接**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe . && ./login-server.exe`
Expected: 显示"MySQL connected"和"Redis已连接，Session将存储在Redis中"或"Redis连接失败: xxx，将使用内存存储"

---

### Task 3: 实现Session存储和认证中间件

**Files:**
- Modify: `d:\桌面\WebGL\main.go` (添加Session保存和认证函数)

- [ ] **Step 1: 添加Session保存函数**

在cors()函数之后添加：

```go
func saveSession(w http.ResponseWriter, r *http.Request, username string, userId int) error {
    session, err := store.Get(r, sessionName)
    if err != nil {
        // Session获取失败，创建新的
        session, _ = store.New(r, sessionName)
    }

    session.Values["username"] = username
    session.Values["user_id"] = userId
    session.Values["login_time"] = time.Now().Unix()

    return session.Save(r, w)
}

func clearSession(w http.ResponseWriter, r *http.Request) error {
    session, err := store.Get(r, sessionName)
    if err != nil {
        return nil
    }
    session.Values = make(map[interface{}]interface{})
    session.Options.MaxAge = -1
    return session.Save(r, w)
}

func getSessionUsername(r *http.Request) (string, bool) {
    session, err := store.Get(r, sessionName)
    if err != nil {
        return "", false
    }

    username, ok := session.Values["username"].(string)
    if !ok {
        return "", false
    }

    return username, true
}
```

- [ ] **Step 2: 添加认证检查中间件**

在saveSession函数之后添加：

```go
func requireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        username, ok := getSessionUsername(r)
        if !ok {
            http.Redirect(w, r, "/", http.StatusFound)
            return
        }
        // 将username传递给下一个handler
        ctx := context.WithValue(r.Context(), "username", username)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

- [ ] **Step 3: 验证代码编译**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功，无错误

---

### Task 4: 修改登录接口集成Session

**Files:**
- Modify: `d:\桌面\WebGL\main.go:135-168` (handleLogin函数)

- [ ] **Step 1: 修改handleLogin函数**

找到handleLogin函数，替换整个函数体：

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        writeJSON(w, 405, Resp{Code: 1, Msg: "Method not allowed"})
        return
    }
    var req Req
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, 400, Resp{Code: 1, Msg: "参数错误"})
        return
    }
    req.Username = strings.TrimSpace(req.Username)
    if req.Username == "" || req.Password == "" {
        writeJSON(w, 400, Resp{Code: 1, Msg: "用户名密码不能为空"})
        return
    }

    db.Exec("USE login_app")
    var hashedPwd string
    var userId int
    err := db.QueryRow("SELECT password, id FROM users WHERE username = ?", req.Username).Scan(&hashedPwd, &userId)
    if err == sql.ErrNoRows {
        writeJSON(w, 401, Resp{Code: 1, Msg: "用户名或密码错误"})
        return
    }
    if err != nil {
        writeJSON(w, 500, Resp{Code: 1, Msg: "查询失败"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.Password)); err != nil {
        writeJSON(w, 401, Resp{Code: 1, Msg: "用户名或密码错误"})
        return
    }

    // 保存Session
    if err := saveSession(w, r, req.Username, userId); err != nil {
        writeJSON(w, 500, Resp{Code: 1, Msg: "Session保存失败"})
        return
    }

    writeJSON(w, 200, Resp{Code: 0, Msg: "登录成功", Data: map[string]interface{}{"username": req.Username, "user_id": userId}})
}
```

- [ ] **Step 2: 编译测试**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功

---

### Task 5: 添加退出登录和检查认证接口

**Files:**
- Modify: `d:\桌面\WebGL\main.go` (添加新接口)

- [ ] **Step 1: 添加退出登录接口处理函数**

在handleLogin函数之后添加：

```go
func handleLogout(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" && r.Method != "GET" {
        writeJSON(w, 405, Resp{Code: 1, Msg: "Method not allowed"})
        return
    }

    if err := clearSession(w, r); err != nil {
        log.Printf("清除Session失败: %v", err)
    }

    writeJSON(w, 200, Resp{Code: 0, Msg: "退出成功"})
}

func handleCheckAuth(w http.ResponseWriter, r *http.Request) {
    username, ok := getSessionUsername(r)
    if !ok {
        writeJSON(w, 401, Resp{Code: 1, Msg: "未登录"})
        return
    }

    writeJSON(w, 200, Resp{Code: 0, Msg: "已登录", Data: map[string]string{"username": username}})
}
```

- [ ] **Step 2: 在main()函数中注册新路由**

找到现有的路由注册代码，添加新路由：

```go
http.HandleFunc("/api/register", cors(handleRegister))
http.HandleFunc("/api/login", cors(handleLogin))
http.HandleFunc("/api/logout", cors(handleLogout))      // 新增
http.HandleFunc("/api/check-auth", cors(handleCheckAuth))  // 新增
```

- [ ] **Step 3: 添加context包的导入**

在import部分添加：

```go
import (
    "context"  // 新增
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "golang.org/x/crypto/bcrypt"
    "github.com/gorilla/sessions"  // 新增
    "github.com/go-redis/redis/v8"  // 新增
)
```

- [ ] **Step 4: 编译并验证**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功，无错误

---

### Task 6: 添加页面路由保护

**Files:**
- Modify: `d:\桌面\WebGL\main.go` (添加路由保护)

- [ ] **Step 1: 在main()函数中添加欢迎页和实验室页路由**

找到静态文件路由部分，修改为：

```go
// 欢迎页（需要认证）
http.Handle("/welcome", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./static/welcome.html")
})))

// 实验室页（需要认证）
http.Handle("/lab", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./static/lab.html")
})))

// 静态文件服务
fs := http.FileServer(http.Dir("./static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

- [ ] **Step 2: 修改根路径处理逻辑**

替换现有的根路径处理，添加登录检查：

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // 如果已登录且访问根路径，重定向到欢迎页
    if r.URL.Path == "/" {
        if _, ok := getSessionUsername(r); ok {
            http.Redirect(w, r, "/welcome", http.StatusFound)
            return
        }
        http.ServeFile(w, r, "./static/index.html")
        return
    }
    http.NotFound(w, r)
})
```

- [ ] **Step 3: 编译测试**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功

---

## 阶段二：前端页面开发

### Task 7: 创建欢迎页面

**Files:**
- Create: `d:\桌面\WebGL\static\welcome.html`

- [ ] **Step 1: 创建欢迎页面HTML结构**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>欢迎 - 未来实验室</title>
    <link rel="stylesheet" href="/static/css/style.css">
</head>
<body class="welcome-page">
    <div class="welcome-container">
        <div class="welcome-card">
            <div class="avatar" id="userAvatar">?</div>
            <h1 class="welcome-title" id="welcomeTitle">欢迎回来</h1>
            <p class="welcome-subtitle" id="welcomeSubtitle">正在加载...</p>
            
            <div class="welcome-actions">
                <button class="btn-primary" id="enterLab">
                    进入实验室
                    <span class="arrow">→</span>
                </button>
                <button class="btn-secondary" id="logoutBtn">
                    退出登录
                </button>
            </div>
        </div>
    </div>

    <div class="loading-overlay" id="loadingOverlay">
        <div class="loading-spinner"></div>
        <p>正在验证登录状态...</p>
    </div>

    <script src="/static/js/auth.js"></script>
    <script>
        // 页面初始化
        document.addEventListener('DOMContentLoaded', async function() {
            const overlay = document.getElementById('loadingOverlay');
            
            try {
                // 检查登录状态
                const user = await checkAuth();
                if (!user) {
                    window.location.href = '/';
                    return;
                }

                // 显示用户信息
                const initial = user.username ? user.username[0].toUpperCase() : '?';
                document.getElementById('userAvatar').textContent = initial;
                document.getElementById('welcomeTitle').textContent = `欢迎回来，${user.username}`;
                document.getElementById('welcomeSubtitle').textContent = '准备开始你的实验室之旅';

                // 隐藏加载动画
                overlay.classList.add('hidden');

                // 绑定事件
                document.getElementById('enterLab').addEventListener('click', function() {
                    window.location.href = '/lab';
                });

                document.getElementById('logoutBtn').addEventListener('click', handleLogout);

            } catch (error) {
                console.error('验证失败:', error);
                window.location.href = '/';
            }
        });
    </script>
</body>
</html>
```

- [ ] **Step 2: 验证文件创建成功**

Run: `Get-Content "d:\桌面\WebGL\static\welcome.html" | Select-Object -First 20`
Expected: 显示HTML内容

---

### Task 8: 创建通用样式文件

**Files:**
- Create: `d:\桌面\WebGL\static\css\style.css`

- [ ] **Step 1: 创建CSS样式**

```css
/* 通用样式重置 */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

:root {
    --primary-1: #2dd4bf;
    --primary-2: #3b82f6;
    --bg-dark: #0a0a0f;
    --text-white: #ffffff;
    --glass-bg: rgba(0, 0, 0, 0.5);
    --shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
}

body {
    font-family: 'PingFang SC', 'Microsoft YaHei', 'Helvetica Neue', sans-serif;
    background: linear-gradient(135deg, var(--primary-1) 0%, var(--primary-2) 100%);
    min-height: 100vh;
    color: var(--text-white);
}

/* 欢迎页面样式 */
.welcome-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: linear-gradient(135deg, var(--primary-1) 0%, var(--primary-2) 100%);
    position: relative;
    overflow: hidden;
}

.welcome-page::before {
    content: '';
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 70%);
    animation: pulse 4s ease-in-out infinite;
}

@keyframes pulse {
    0%, 100% { transform: scale(1); opacity: 0.5; }
    50% { transform: scale(1.1); opacity: 0.8; }
}

.welcome-container {
    position: relative;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 20px;
}

.welcome-card {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(20px);
    border-radius: 24px;
    padding: 48px 56px;
    text-align: center;
    box-shadow: var(--shadow);
    max-width: 420px;
    width: 100%;
    animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
    from {
        opacity: 0;
        transform: translateY(30px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.avatar {
    width: 96px;
    height: 96px;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--primary-1), var(--primary-2));
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 42px;
    font-weight: 700;
    color: white;
    margin: 0 auto 24px;
    box-shadow: 0 8px 24px rgba(45, 212, 191, 0.4);
}

.welcome-title {
    font-size: 28px;
    font-weight: 700;
    color: #1a1a2e;
    margin-bottom: 8px;
}

.welcome-subtitle {
    font-size: 14px;
    color: #666;
    margin-bottom: 32px;
}

.welcome-actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.btn-primary {
    width: 100%;
    padding: 14px 32px;
    border: none;
    border-radius: 8px;
    background: linear-gradient(135deg, var(--primary-1), var(--primary-2));
    color: white;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
}

.btn-primary:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(45, 212, 191, 0.4);
}

.btn-primary:active {
    transform: translateY(0);
}

.btn-primary .arrow {
    transition: transform 0.3s ease;
}

.btn-primary:hover .arrow {
    transform: translateX(4px);
}

.btn-secondary {
    width: 100%;
    padding: 14px 32px;
    border: 2px solid #ddd;
    border-radius: 8px;
    background: transparent;
    color: #666;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.3s ease;
}

.btn-secondary:hover {
    border-color: #ef4444;
    color: #ef4444;
}

/* 加载动画 */
.loading-overlay {
    position: fixed;
    inset: 0;
    background: linear-gradient(135deg, var(--primary-1) 0%, var(--primary-2) 100%);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    transition: opacity 0.5s ease;
}

.loading-overlay.hidden {
    opacity: 0;
    pointer-events: none;
}

.loading-spinner {
    width: 48px;
    height: 48px;
    border: 4px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 16px;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}

.loading-overlay p {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.9);
}

/* Toast提示 */
.toast {
    position: fixed;
    top: 28px;
    left: 50%;
    transform: translateX(-50%) translateY(-120px);
    padding: 10px 24px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    z-index: 10000;
    opacity: 0;
    transition: all 0.4s ease;
    pointer-events: none;
    white-space: nowrap;
}

.toast.show {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
}

.toast.success {
    background: #10b981;
    color: white;
}

.toast.error {
    background: #ef4444;
    color: white;
}

/* 响应式 */
@media (max-width: 480px) {
    .welcome-card {
        padding: 32px 24px;
        margin: 16px;
    }

    .avatar {
        width: 80px;
        height: 80px;
        font-size: 36px;
    }

    .welcome-title {
        font-size: 24px;
    }
}
```

- [ ] **Step 2: 验证文件创建成功**

Run: `Get-Content "d:\桌面\WebGL\static\css\style.css" | Select-Object -First 30`
Expected: 显示CSS样式内容

---

### Task 9: 创建认证逻辑JavaScript

**Files:**
- Create: `d:\桌面\WebGL\static\js\auth.js`

- [ ] **Step 1: 创建auth.js文件**

```javascript
// 认证状态检查
async function checkAuth() {
    try {
        const response = await fetch('/api/check-auth', {
            method: 'GET',
            credentials: 'include'
        });
        const data = await response.json();
        
        if (data.code === 0 && data.data) {
            return data.data;
        }
        return null;
    } catch (error) {
        console.error('检查认证失败:', error);
        return null;
    }
}

// 退出登录
async function handleLogout() {
    try {
        const response = await fetch('/api/logout', {
            method: 'POST',
            credentials: 'include'
        });
        
        // 无论成功失败都重定向到登录页
        window.location.href = '/';
    } catch (error) {
        console.error('退出失败:', error);
        // 仍然重定向
        window.location.href = '/';
    }
}

// Toast提示
function showToast(message, type = 'success') {
    // 移除已存在的toast
    const existingToast = document.querySelector('.toast');
    if (existingToast) {
        existingToast.remove();
    }

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    // 触发动画
    requestAnimationFrame(() => {
        toast.classList.add('show');
    });

    // 3秒后移除
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 400);
    }, 3000);
}

// 导出函数供全局使用
window.checkAuth = checkAuth;
window.handleLogout = handleLogout;
window.showToast = showToast;
```

- [ ] **Step 2: 验证文件创建成功**

Run: `Get-Content "d:\桌面\WebGL\static\js\auth.js" | Select-Object -First 20`
Expected: 显示JavaScript内容

---

### Task 10: 创建WebGL实验室页面

**Files:**
- Create: `d:\桌面\WebGL\static\lab.html`

- [ ] **Step 1: 创建lab.html基础结构**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>未来实验室</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <style>
        /* 实验室页面特定样式 */
        body.lab-page {
            background: #0a0a0f;
            overflow: hidden;
        }

        #canvas-container {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            z-index: 1;
        }

        .lab-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            pointer-events: none;
            z-index: 100;
        }

        /* 顶部导航栏 */
        .lab-header {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 60px;
            background: rgba(10, 10, 15, 0.8);
            backdrop-filter: blur(10px);
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 32px;
            z-index: 101;
            pointer-events: auto;
        }

        .lab-logo {
            font-family: 'Orbitron', monospace;
            font-size: 20px;
            font-weight: 900;
            background: linear-gradient(135deg, var(--primary-1), var(--primary-2));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            letter-spacing: 0.1em;
        }

        .lab-user {
            display: flex;
            align-items: center;
            gap: 16px;
        }

        .lab-user-name {
            font-size: 14px;
            color: rgba(255, 255, 255, 0.7);
        }

        .lab-settings {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background: rgba(255, 255, 255, 0.1);
            border: none;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.3s ease;
            pointer-events: auto;
        }

        .lab-settings:hover {
            background: rgba(255, 255, 255, 0.2);
        }

        /* 扫描线效果 */
        .scan-line {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 2px;
            background: linear-gradient(90deg, transparent, rgba(0, 245, 255, 0.3), transparent);
            animation: scan 4s linear infinite;
            pointer-events: none;
        }

        @keyframes scan {
            0% { top: 0%; }
            100% { top: 100%; }
        }

        /* 底部导航 */
        .lab-nav {
            position: fixed;
            bottom: 30px;
            left: 50%;
            transform: translateX(-50%);
            display: flex;
            gap: 2rem;
            font-family: 'Roboto Mono', monospace;
            font-size: 0.8rem;
            letter-spacing: 0.2em;
            z-index: 101;
            pointer-events: auto;
        }

        .lab-nav-item {
            color: rgba(255, 255, 255, 0.4);
            cursor: pointer;
            transition: all 0.3s ease-out;
            padding: 8px 16px;
            border: 1px solid transparent;
            border-radius: 4px;
        }

        .lab-nav-item:hover {
            color: var(--primary-1);
            text-shadow: 0 0 20px rgba(0, 245, 255, 0.8);
            border-color: rgba(0, 245, 255, 0.3);
        }

        /* 退出按钮 */
        .lab-logout {
            position: fixed;
            bottom: 30px;
            right: 30px;
            padding: 8px 20px;
            border: 1px solid rgba(239, 68, 68, 0.5);
            border-radius: 6px;
            background: transparent;
            color: rgba(239, 68, 68, 0.8);
            font-size: 12px;
            cursor: pointer;
            transition: all 0.3s ease;
            z-index: 101;
            pointer-events: auto;
        }

        .lab-logout:hover {
            background: rgba(239, 68, 68, 0.1);
            border-color: #ef4444;
            color: #ef4444;
        }

        /* 巨型标题 */
        .lab-title {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            z-index: 102;
            pointer-events: none;
        }

        .lab-title h1 {
            font-family: 'Orbitron', monospace;
            font-size: clamp(4rem, 15vw, 12rem);
            font-weight: 900;
            color: transparent;
            background: linear-gradient(135deg, var(--primary-1) 0%, var(--primary-2) 100%);
            background-clip: text;
            -webkit-background-clip: text;
            letter-spacing: 0.1em;
            opacity: 0.9;
            text-shadow: 0 0 100px rgba(0, 245, 255, 0.5);
            animation: titlePulse 3s ease-in-out infinite;
        }

        .lab-title p {
            font-family: 'Roboto Mono', monospace;
            font-size: clamp(0.8rem, 2vw, 1.5rem);
            color: rgba(255, 255, 255, 0.6);
            letter-spacing: 0.3em;
            margin-top: 1rem;
        }

        @keyframes titlePulse {
            0%, 100% { opacity: 0.9; }
            50% { opacity: 0.7; }
        }

        /* 加载动画 */
        .lab-loading {
            position: fixed;
            inset: 0;
            background: #0a0a0f;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            z-index: 9999;
            transition: opacity 0.5s ease;
        }

        .lab-loading.hidden {
            opacity: 0;
            pointer-events: none;
        }

        .lab-loading h2 {
            font-family: 'Orbitron', monospace;
            font-size: 24px;
            color: var(--primary-1);
            margin-bottom: 20px;
        }

        .lab-loading-spinner {
            width: 40px;
            height: 40px;
            border: 3px solid rgba(0, 245, 255, 0.2);
            border-top-color: var(--primary-1);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
    </style>
    <link href="https://fonts.googleapis.com/css2?family=Orbitron:wght@400;700;900&family=Roboto+Mono:wght@300;400;500&display=swap" rel="stylesheet">
</head>
<body class="lab-page">
    <!-- 加载动画 -->
    <div class="lab-loading" id="labLoading">
        <h2>INITIALIZING</h2>
        <div class="lab-loading-spinner"></div>
    </div>

    <!-- WebGL Canvas容器 -->
    <div id="canvas-container"></div>

    <!-- 实验室内容覆盖层 -->
    <div class="lab-overlay">
        <div class="scan-line"></div>
        
        <!-- 顶部导航 -->
        <header class="lab-header">
            <div class="lab-logo">FUTURE LAB</div>
            <div class="lab-user">
                <span class="lab-user-name" id="userName">USER</span>
                <button class="lab-settings" title="设置">
                    <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="8" cy="8" r="2"></circle>
                        <path d="M8 1v2M8 15v2M1 8h2M15 8h2M2.9 2.9l1.4 1.4M13.7 13.7l1.4 1.4M2.9 15.1l1.4-1.4M13.7 4.3l1.4-1.4"></path>
                    </svg>
                </button>
            </div>
        </header>

        <!-- 巨型标题 -->
        <div class="lab-title">
            <h1>FUTURE</h1>
            <p>LABORATORY</p>
        </div>

        <!-- 底部导航 -->
        <nav class="lab-nav">
            <div class="lab-nav-item">INIT</div>
            <div class="lab-nav-item">EXPLORE</div>
            <div class="lab-nav-item">ANALYZE</div>
            <div class="lab-nav-item">SYSTEM</div>
        </nav>

        <!-- 退出按钮 -->
        <button class="lab-logout" id="logoutBtn">退出登录</button>
    </div>

    <script src="/static/js/auth.js"></script>
    <script type="module">
        import * as THREE from 'https://cdn.jsdelivr.net/npm/three@0.150.1/build/three.module.js';
        import { OrbitControls } from 'https://cdn.jsdelivr.net/npm/three@0.150.1/examples/jsm/controls/OrbitControls.js';

        // 页面初始化
        document.addEventListener('DOMContentLoaded', async function() {
            try {
                // 检查登录状态
                const user = await checkAuth();
                if (!user) {
                    window.location.href = '/';
                    return;
                }

                // 显示用户名
                document.getElementById('userName').textContent = user.username.toUpperCase();

                // 绑定退出按钮
                document.getElementById('logoutBtn').addEventListener('click', handleLogout);

                // 初始化WebGL场景
                await initLab();

                // 隐藏加载动画
                document.getElementById('labLoading').classList.add('hidden');

            } catch (error) {
                console.error('初始化失败:', error);
                window.location.href = '/';
            }
        });

        // Three.js场景类
        class FutureLab {
            constructor() {
                this.scene = null;
                this.camera = null;
                this.renderer = null;
                this.controls = null;
                this.grid = null;
                this.meshGroup = null;
                this.particles = null;
                this.mouseX = 0;
                this.mouseY = 0;
                this.targetMouseX = 0;
                this.targetMouseY = 0;
                this.time = 0;
            }

            init() {
                this.setupScene();
                this.setupLighting();
                this.createGrid();
                this.createFloatingGeometry();
                this.createParticles();
                this.setupEventListeners();
            }

            setupScene() {
                this.scene = new THREE.Scene();
                this.scene.background = new THREE.Color(0x0a0a0f);
                this.scene.fog = new THREE.Fog(0x0a0a0f, 50, 200);

                const aspect = window.innerWidth / window.innerHeight;
                this.camera = new THREE.PerspectiveCamera(75, aspect, 0.1, 200);
                this.camera.position.set(0, 10, 25);

                this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
                this.renderer.setSize(window.innerWidth, window.innerHeight);
                this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
                this.renderer.shadowMap.enabled = true;
                this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
                document.getElementById('canvas-container').appendChild(this.renderer.domElement);

                this.controls = new OrbitControls(this.camera, this.renderer.domElement);
                this.controls.enableDamping = true;
                this.controls.dampingFactor = 0.05;
                this.controls.minDistance = 10;
                this.controls.maxDistance = 100;
                this.controls.enablePan = false;
            }

            setupLighting() {
                const ambientLight = new THREE.AmbientLight(0x404080, 0.3);
                this.scene.add(ambientLight);

                const directionalLight = new THREE.DirectionalLight(0x2dd4bf, 1);
                directionalLight.position.set(50, 50, 50);
                directionalLight.castShadow = true;
                directionalLight.shadow.mapSize.width = 2048;
                directionalLight.shadow.mapSize.height = 2048;
                this.scene.add(directionalLight);

                const pointLight1 = new THREE.PointLight(0x3b82f6, 0.8, 100);
                pointLight1.position.set(-30, 20, -30);
                this.scene.add(pointLight1);

                const pointLight2 = new THREE.PointLight(0x2dd4bf, 0.5, 80);
                pointLight2.position.set(30, -10, 30);
                this.scene.add(pointLight2);
            }

            createGrid() {
                const gridSize = 100;
                const gridDivisions = 100;

                const gridGeometry = new THREE.BufferGeometry();
                const gridVertices = [];
                const gridColors = [];

                const color1 = new THREE.Color(0x2dd4bf);
                const color2 = new THREE.Color(0x3b82f6);

                for (let i = -gridSize / 2; i <= gridSize / 2; i += gridSize / gridDivisions) {
                    gridVertices.push(-gridSize / 2, 0, i);
                    gridVertices.push(gridSize / 2, 0, i);
                    gridVertices.push(i, 0, -gridSize / 2);
                    gridVertices.push(i, 0, gridSize / 2);

                    const ratio = Math.abs(i) / (gridSize / 2);
                    const mixedColor = color1.clone().lerp(color2, ratio);
                    gridColors.push(mixedColor.r, mixedColor.g, mixedColor.b);
                    gridColors.push(mixedColor.r, mixedColor.g, mixedColor.b);
                    gridColors.push(mixedColor.r, mixedColor.g, mixedColor.b);
                    gridColors.push(mixedColor.r, mixedColor.g, mixedColor.b);
                }

                gridGeometry.setAttribute('position', new THREE.Float32BufferAttribute(gridVertices, 3));
                gridGeometry.setAttribute('color', new THREE.Float32BufferAttribute(gridColors, 3));

                const gridMaterial = new THREE.LineBasicMaterial({
                    vertexColors: true,
                    transparent: true,
                    opacity: 0.4
                });

                this.grid = new THREE.LineSegments(gridGeometry, gridMaterial);
                this.scene.add(this.grid);
            }

            createFloatingGeometry() {
                this.meshGroup = new THREE.Group();

                const geometries = [
                    new THREE.TorusKnotGeometry(3, 1, 128, 32),
                    new THREE.IcosahedronGeometry(2.5, 0),
                    new THREE.OctahedronGeometry(2, 0),
                    new THREE.TetrahedronGeometry(2.2, 0),
                    new THREE.DodecahedronGeometry(1.8, 0)
                ];

                const positions = [
                    { x: -15, y: 8, z: -10 },
                    { x: 12, y: 12, z: 5 },
                    { x: 0, y: 6, z: 15 },
                    { x: -10, y: 15, z: 8 },
                    { x: 18, y: 5, z: -15 }
                ];

                geometries.forEach((geometry, index) => {
                    const material = new THREE.MeshPhysicalMaterial({
                        color: new THREE.Color().setHSL(0.5 + Math.random() * 0.3, 0.8, 0.6),
                        metalness: 0.9,
                        roughness: 0.1,
                        transparent: true,
                        opacity: 0.8,
                        emissive: new THREE.Color().setHSL(0.5 + Math.random() * 0.3, 0.8, 0.3),
                        emissiveIntensity: 0.5
                    });

                    const mesh = new THREE.Mesh(geometry, material);
                    mesh.position.set(positions[index].x, positions[index].y, positions[index].z);
                    mesh.castShadow = true;
                    mesh.receiveShadow = true;
                    mesh.userData = {
                        rotationSpeed: {
                            x: (Math.random() - 0.5) * 0.01,
                            y: (Math.random() - 0.5) * 0.01,
                            z: (Math.random() - 0.5) * 0.01
                        },
                        floatOffset: index * Math.PI * 0.4,
                        floatSpeed: 0.5 + Math.random() * 0.5
                    };

                    this.meshGroup.add(mesh);
                });

                this.scene.add(this.meshGroup);
            }

            createParticles() {
                const particleCount = 2000;
                const positions = new Float32Array(particleCount * 3);
                const colors = new Float32Array(particleCount * 3);

                const color1 = new THREE.Color(0x2dd4bf);
                const color2 = new THREE.Color(0x3b82f6);
                const color3 = new THREE.Color(0xff0080);

                for (let i = 0; i < particleCount; i++) {
                    const i3 = i * 3;
                    positions[i3] = (Math.random() - 0.5) * 200;
                    positions[i3 + 1] = (Math.random() - 0.5) * 100;
                    positions[i3 + 2] = (Math.random() - 0.5) * 200;

                    const colorChoice = Math.random();
                    const color = colorChoice < 0.5 ? color1 : colorChoice < 0.8 ? color2 : color3;
                    colors[i3] = color.r;
                    colors[i3 + 1] = color.g;
                    colors[i3 + 2] = color.b;
                }

                const particleGeometry = new THREE.BufferGeometry();
                particleGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
                particleGeometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

                const particleMaterial = new THREE.PointsMaterial({
                    size: 2,
                    vertexColors: true,
                    transparent: true,
                    opacity: 0.8,
                    blending: THREE.AdditiveBlending,
                    depthWrite: false
                });

                this.particles = new THREE.Points(particleGeometry, particleMaterial);
                this.scene.add(this.particles);
            }

            setupEventListeners() {
                window.addEventListener('resize', () => this.onWindowResize());
                window.addEventListener('mousemove', (e) => this.onMouseMove(e));
            }

            onWindowResize() {
                this.camera.aspect = window.innerWidth / window.innerHeight;
                this.camera.updateProjectionMatrix();
                this.renderer.setSize(window.innerWidth, window.innerHeight);
            }

            onMouseMove(event) {
                this.targetMouseX = (event.clientX / window.innerWidth - 0.5) * 2;
                this.targetMouseY = -(event.clientY / window.innerHeight - 0.5) * 2;
            }

            animate() {
                requestAnimationFrame(() => this.animate());

                this.time += 0.016;

                this.mouseX += (this.targetMouseX - this.mouseX) * 0.05;
                this.mouseY += (this.targetMouseY - this.mouseY) * 0.05;

                if (this.grid) {
                    this.grid.rotation.x = Math.sin(this.time * 0.1) * 0.02 + this.mouseY * 0.1;
                    this.grid.position.x = Math.sin(this.time * 0.05) * 2 + this.mouseX * 3;
                    this.grid.position.z = Math.cos(this.time * 0.05) * 2 + this.mouseY * 3;
                }

                if (this.meshGroup) {
                    this.meshGroup.children.forEach(mesh => {
                        mesh.rotation.x += mesh.userData.rotationSpeed.x;
                        mesh.rotation.y += mesh.userData.rotationSpeed.y;
                        mesh.rotation.z += mesh.userData.rotationSpeed.z;
                        mesh.position.y += Math.sin(this.time * mesh.userData.floatSpeed + mesh.userData.floatOffset) * 0.05;
                    });
                }

                if (this.particles) {
                    this.particles.rotation.y += 0.002;
                }

                this.camera.position.x += (this.mouseX * 5 - this.camera.position.x) * 0.02;
                this.camera.position.y += (-this.mouseY * 3 + 10 - this.camera.position.y) * 0.02;

                this.controls.update();
                this.renderer.render(this.scene, this.camera);
            }
        }

        // 初始化实验室
        async function initLab() {
            return new Promise((resolve) => {
                const lab = new FutureLab();
                lab.init();
                lab.animate();
                
                // 等待场景初始化
                setTimeout(resolve, 1000);
            });
        }
    </script>
</body>
</html>
```

- [ ] **Step 2: 验证文件创建成功**

Run: `Get-Content "d:\桌面\WebGL\static\lab.html" | Select-Object -First 30`
Expected: 显示HTML内容

---

### Task 11: 修改登录页面跳转逻辑

**Files:**
- Modify: `d:\桌面\WebGL\static\index.html` (修改登录成功后的跳转)

- [ ] **Step 1: 找到登录成功处理逻辑**

在index.html的JavaScript部分，找到`doLogin`函数中的登录成功处理：

```javascript
// 修改这部分代码
if (data.code === 0) {
    toast('登录成功！', 'success');
    localStorage.setItem('login_user', JSON.stringify({ username: data.data.username }));
    setTimeout(function() {
        showWelcome(data.data.username);
    }, 600);
}
```

替换为：

```javascript
// 登录成功，跳转到欢迎页
if (data.code === 0) {
    toast('登录成功！', 'success');
    setTimeout(function() {
        window.location.href = '/welcome';
    }, 800);
}
```

- [ ] **Step 2: 编译并验证**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功

---

## 阶段三：功能完善与测试

### Task 12: 创建目录结构验证脚本

**Files:**
- Create: `d:\桌面\WebGL\verify-structure.sh` (Windows用cmd)

- [ ] **Step 1: 创建验证脚本**

```bash
@echo off
echo 验证项目文件结构...

set "errors=0"

:: 检查后端文件
if not exist "main.go" (
    echo [ERROR] main.go 不存在
    set /a errors+=1
)

:: 检查前端目录
if not exist "static\index.html" (
    echo [ERROR] static\index.html 不存在
    set /a errors+=1
)

if not exist "static\welcome.html" (
    echo [ERROR] static\welcome.html 不存在
    set /a errors+=1
)

if not exist "static\lab.html" (
    echo [ERROR] static\lab.html 不存在
    set /a errors+=1
)

if not exist "static\css\style.css" (
    echo [ERROR] static\css\style.css 不存在
    set /a errors+=1
)

if not exist "static\js\auth.js" (
    echo [ERROR] static\js\auth.js 不存在
    set /a errors+=1
)

if %errors% equ 0 (
    echo [SUCCESS] 所有文件存在
) else (
    echo [FAIL] 发现 %errors% 个错误
)

pause
```

- [ ] **Step 2: 运行验证脚本**

Run: `cd "d:\桌面\WebGL"; cmd /c verify-structure.cmd`
Expected: 显示所有文件验证结果

---

### Task 13: 编译和基础运行测试

**Files:**
- Build: `d:\桌面\WebGL\login-server.exe`

- [ ] **Step 1: 完整编译项目**

Run: `cd "d:\桌面\WebGL"; go build -o login-server.exe .`
Expected: 编译成功，无错误

- [ ] **Step 2: 尝试启动服务器（如果Redis可用）**

Run: `cd "d:\桌面\WebGL"; Start-Process -FilePath ".\login-server.exe" -NoNewWindow -PassThru`
Expected: 服务器启动，日志显示MySQL连接成功

- [ ] **Step 3: 测试API接口**

使用curl或Postman测试：
```
POST /api/login
POST /api/register  
GET /api/check-auth
POST /api/logout
```

Expected: 接口正常响应

---

## 实现检查清单

### 功能验收
- [ ] 用户可以注册新账户
- [ ] 用户可以登录
- [ ] 登录后跳转到欢迎页
- [ ] 欢迎页显示用户名
- [ ] 可以进入实验室
- [ ] WebGL场景正常显示
- [ ] 可以退出登录
- [ ] Session过期后需要重新登录

### 视觉验收
- [ ] 登录页保持现有风格
- [ ] 欢迎页配色统一（青蓝渐变）
- [ ] 实验室页WebGL效果流畅
- [ ] 鼠标交互响应正常
- [ ] UI风格整体协调

### 性能验收
- [ ] 页面加载时间 < 3秒
- [ ] WebGL帧率 > 30FPS
- [ ] 登录响应时间 < 500ms
- [ ] 内存占用合理

---

## 故障排除指南

### 问题1: Redis连接失败
**症状**: 服务器启动时显示"Redis连接失败"  
**解决**: 系统会自动降级到内存存储Session，功能不受影响

### 问题2: 编译错误
**症状**: `go build` 报错  
**解决**: 
1. 确保所有依赖已下载：`go mod tidy`
2. 检查Go版本：`go version`（需要1.22+）

### 问题3: 前端页面404
**症状**: 访问/welcome或/lab返回404  
**解决**: 确保所有HTML文件在`static/`目录下，且没有语法错误

### 问题4: Session不生效
**症状**: 登录成功但刷新页面又需要登录  
**解决**: 
1. 检查浏览器Cookie是否启用
2. 检查`credentials: 'include'`是否在fetch请求中
3. 查看浏览器控制台是否有CORS错误

---

## 后续优化建议

1. **性能优化**: 
   - 实现WebGL场景的LOD（Level of Detail）
   - 添加粒子系统的实例化渲染
   - 实现视锥剔除

2. **功能增强**:
   - 添加用户个人中心
   - 实现"记住我"功能
   - 添加社交登录（微信、GitHub）

3. **安全加固**:
   - 添加CSRF Token
   - 实现请求频率限制
   - 配置HTTPS和Secure Cookie

4. **用户体验**:
   - 添加页面过渡动画
   - 实现键盘快捷键导航
   - 添加声音效果（可选）

---

**计划版本**: 1.0  
**创建日期**: 2026-05-21  
**基于设计文档**: docs/specs/2026-05-21-future-lab-design.md
