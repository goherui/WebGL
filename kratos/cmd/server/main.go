package main

import (
	"os"

	"github.com/go-kratos/kratos/v2/log"

	"login-page/internal/biz"
	"login-page/internal/data"
	"login-page/internal/server"
	"login-page/internal/service"
)

func main() {
	logger := log.NewStdLogger(os.Stdout)
	log.SetLogger(logger)

	d, cleanup, err := data.NewData(logger)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "115.190.54.31:6379"
	}

	redisPwd := os.Getenv("REDIS_PASSWORD")
	if redisPwd == "" {
		redisPwd = "redis_6jBRR2"
	}

	redis, _, err := data.NewRedisClientWithConfig(logger, redisAddr, redisPwd)
	if redis == nil {
		log.Fatal("Redis初始化失败")
	}
	defer redis.Close()

	emailHost := os.Getenv("SMTP_HOST")
	if emailHost == "" {
		emailHost = "smtp.example.com"
	}
	emailPort := 587
	emailUser := os.Getenv("SMTP_USER")
	if emailUser == "" {
		emailUser = "your-email@example.com"
	}
	emailPwd := os.Getenv("SMTP_PASSWORD")
	if emailPwd == "" {
		emailPwd = "your-email-password"
	}
	emailFrom := os.Getenv("SMTP_FROM")
	if emailFrom == "" {
		emailFrom = "your-email@example.com"
	}

	email := data.NewEmailService(logger, data.EmailConfig{
		Host:     emailHost,
		Port:     emailPort,
		Username: emailUser,
		Password: emailPwd,
		From:     emailFrom,
	})

	uc := biz.NewUserUseCase(d, redis, email, logger)
	svc := service.NewLoginService(uc, logger)

	app := server.NewServer(svc, logger)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
