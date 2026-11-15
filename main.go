package main

import (
	"GoMusic/handler"
	"GoMusic/internal/config"
	"GoMusic/logic"
	"GoMusic/misc/httputil"
	"GoMusic/misc/log"
	"GoMusic/repo/cache"
	"GoMusic/repo/db"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Errorf("GoMusic stopped: %v", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var redisRepo, sqlRepo logic.Repository
	checks := map[string]handler.HealthCheck{}
	if cfg.MySQLDSN != "" {
		store, err := db.Open(ctx, cfg.MySQLDSN, cfg.CacheTTL)
		if err != nil {
			return fmt.Errorf("MySQL startup failed: %w", err)
		}
		defer store.Close()
		sqlRepo = store
		checks["mysql"] = store.Ping
	}
	if cfg.RedisAddr != "" {
		store, err := cache.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
		if err != nil {
			return fmt.Errorf("Redis startup failed: %w", err)
		}
		defer store.Close()
		redisRepo = store
		checks["redis"] = store.Ping
	}
	service := logic.NewService(httputil.NewClient(&http.Client{Timeout: cfg.UpstreamTimeout}), redisRepo, sqlRepo, cfg.MaxSongs, cfg.Concurrency)
	server := &http.Server{Addr: cfg.Addr, Handler: handler.NewRouterWithService(service, cfg, checks),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: cfg.RequestTimeout + 5*time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failure := make(chan error, 1)
	go func() { failure <- server.ListenAndServe() }()
	log.Infof("GoMusic listening on %s (MySQL=%t Redis=%t)", cfg.Addr, sqlRepo != nil, redisRepo != nil)
	select {
	case err := <-failure:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-stopped.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
