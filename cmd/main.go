package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/mllbll/kosmohak_nn/internal/app"
	"github.com/mllbll/kosmohak_nn/internal/config"
)

func main() {
	cfg := config.Load()

	appCtx, appCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer appCancel()

	a, err := app.New(appCtx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		<-appCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
	}()

	if err := a.Run(appCtx); err != nil {
		log.Fatal(err)
	}
}
