package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JoyElliot/zwOA/internal/platform/httpserver"
)

func main() {
	if err := run(); err != nil {
		log.Printf("API 服务退出：%v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := httpserver.New(httpserver.ConfigFromEnv())
	if err := server.Start(); err != nil {
		return err
	}
	<-ctx.Done()
	return server.Shutdown()
}
