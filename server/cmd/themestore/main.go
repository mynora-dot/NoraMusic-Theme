package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mynora/noramusic-theme-server/internal/server"
)

func main() {
	cfg, err := server.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	app, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	httpServer := &http.Server{Addr: cfg.ListenAddr, Handler: app.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute, IdleTimeout: 2 * time.Minute}
	go func() {
		log.Printf("theme store listening on %s (mode=%s)", cfg.ListenAddr, cfg.Mode)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}
