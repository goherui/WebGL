# Redis 安装指南

## Windows 安装 Redis

### 方法一：使用 Memurai（推荐）

1. 下载 Memurai：https://www.memurai.com/download
2. 安装并启动服务
3. Redis 默认会在 localhost:6379 运行

### 方法二：使用 WSL

1. 安装 WSL2
2. 在 WSL 终端运行：
```bash
sudo apt update
sudo apt install redis-server
sudo redis-server
```

### 方法三：使用 Docker

```bash
docker run -d -p 6379:6379 redis:latest
```

## 配置环境变量（可选）

在启动服务器前，可以设置以下环境变量：

```powershell
# Redis 配置
$env:REDIS_ADDR = "localhost:6379"
$env:REDIS_PASSWORD = ""

# SMTP 邮件配置
$env:SMTP_HOST = "smtp.qq.com"      # QQ邮箱 SMTP 服务器
$env:SMTP_USER = "your-email@qq.com"
$env:SMTP_PASSWORD = "your-smtp-password"  # QQ邮箱授权码
$env:SMTP_FROM = "your-email@qq.com"
```

## QQ邮箱 SMTP 授权码获取

1. 登录 QQ 邮箱网页版
2. 进入 设置 → 账户
3. 找到 "POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务"
4. 开启 SMTP 服务
5. 生成授权码（16位）
6. 使用授权码作为 SMTP_PASSWORD

## 启动顺序

1. 启动 Redis
2. 配置邮件服务（获取授权码）
3. 启动 WebGL 服务器

```powershell
# 启动 Redis（Memurai）
net start memurai

# 或者在 WSL 中
redis-server

# 设置环境变量
$env:SMTP_HOST = "smtp.qq.com"
$env:SMTP_USER = "123456789@qq.com"
$env:SMTP_PASSWORD = "abcdefghijklmnop"
$env:SMTP_FROM = "123456789@qq.com"

# 启动服务器
cd d:\桌面\WebGL
.\kratos\kratos-server.exe
```

## 常见问题

### Redis 连接失败

确保 Redis 正在运行：
```powershell
netstat -ano | findstr 6379
```

如果没有输出，Redis 没有运行。启动 Redis：
```powershell
net start memurai  # Memurai
# 或
redis-server       # WSL/Linux
```

### 邮件发送失败

1. 检查 SMTP 配置是否正确
2. 确认授权码是否正确（不是登录密码）
3. 确认邮箱是否开启了 SMTP 服务
4. 部分邮箱需要开启 "低安全应用" 或 "授权码登录"
