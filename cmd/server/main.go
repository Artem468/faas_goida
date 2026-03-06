package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"faas_goida/internal/auth"
	"faas_goida/internal/executor"
	"faas_goida/internal/user"

	_ "github.com/lib/pq"
)

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)

	userRepo := user.NewRepository(db)
	tokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(userRepo, tokenRepo, auth.ServiceConfig{
		AccessTTL:    15 * time.Minute,
		RefreshTTL:   7 * 24 * time.Hour,
		AccessSecret: mustEnv("ACCESS_TOKEN_SECRET"),
	})
	authHandler := auth.NewHandler(authService)

	port := ":8080"
	log.Printf("Server listening on %s", port)

	mux := http.NewServeMux()
	mux.Handle("/run", authService.AuthMiddleware(http.HandlerFunc(executor.RunGoidaHandler)))

	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/auth/refresh", authHandler.Refresh)
	server := http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func openDB() (*sql.DB, error) {
	dsn := mustEnv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}
