package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	httpserver "github.com/lutefd/baseline-api/internal/http"
	"github.com/lutefd/baseline-api/internal/storage/postgres"
)

type config struct {
	Port          string
	DatabaseURL   string
	APIToken      string
	DefaultUserID uuid.UUID
}

func loadConfig() (config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://baseline:baseline@localhost:5432/baseline?sslmode=disable"
	}

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		apiToken = "baseline-dev-token"
	}

	uid := os.Getenv("DEFAULT_USER_ID")
	if uid == "" {
		uid = "00000000-0000-0000-0000-000000000001"
	}
	parsedUID, err := uuid.Parse(uid)
	if err != nil {
		return config{}, err
	}

	return config{
		Port:          port,
		DatabaseURL:   databaseURL,
		APIToken:      apiToken,
		DefaultUserID: parsedUID,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	store, err := postgres.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer store.Close()

	srv := httpserver.NewServer(httpserver.Dependencies{
		Store:         store,
		APIToken:      cfg.APIToken,
		DefaultUserID: cfg.DefaultUserID,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
