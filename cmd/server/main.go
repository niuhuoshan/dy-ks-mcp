package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"dy-ks-mcp/internal/config"
	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/httpapi"
	"dy-ks-mcp/internal/mcp"
	"dy-ks-mcp/internal/platform/registry"
	"dy-ks-mcp/internal/service"
	"dy-ks-mcp/internal/store"
)

func main() {
	configPath := flag.String("config", "./config/config.yaml", "path to YAML config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sqliteStore, err := store.NewSQLiteStore(cfg.Store.SQLitePath)
	if err != nil {
		log.Fatalf("create sqlite store: %v", err)
	}
	defer func() {
		if err := sqliteStore.Close(); err != nil {
			log.Printf("close sqlite: %v", err)
		}
	}()

	ctx := context.Background()
	if err := sqliteStore.Init(ctx); err != nil {
		log.Fatalf("init sqlite schema: %v", err)
	}

	platformRegistry, err := registry.New(cfg.Platform)
	if err != nil {
		log.Fatalf("init platform registry: %v", err)
	}

	runner, err := engine.NewRunner(cfg.Engine, sqliteStore)
	if err != nil {
		log.Fatalf("init engine: %v", err)
	}

	svc := service.New(platformRegistry, runner, sqliteStore)
	mcpHandler := mcp.NewHandler(svc)
	httpHandler := httpapi.NewHandler(svc)

	mux := http.NewServeMux()
	httpHandler.Register(mux, mcpHandler)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      100 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("dy-ks-mcp server listening on %s", addr)
	log.Printf("REST endpoints: /health /api/v1/run /api/v1/search /api/v1/comment/prepare /api/v1/comment/submit /api/v1/comment/verify /api/v1/login/status /api/v1/login/start")
	log.Printf("MCP endpoint: /mcp")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownCtx.Done()

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctxTimeout); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(started))
	})
}
