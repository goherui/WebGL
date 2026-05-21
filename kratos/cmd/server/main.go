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

	// 依赖注入
	d, cleanup, err := data.NewData(logger)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	uc := biz.NewUserUseCase(d, logger)
	svc := service.NewLoginService(uc, logger)

	app := server.NewServer(svc, logger)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}