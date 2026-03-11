package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"faas_goida/internal/auth"
	"faas_goida/internal/executor"
	"faas_goida/internal/file"
	"faas_goida/internal/project"
	"faas_goida/internal/storage"
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

	projectRepo := project.NewRepository(db)
	projectHandler := project.NewHandler(projectRepo)

	fileRepo := file.NewRepository(db)
	s3Storage, err := storage.NewS3(storage.Config{
		Endpoint:        mustEnv("S3_ENDPOINT"),
		AccessKeyID:     mustEnv("S3_ACCESS_KEY"),
		SecretAccessKey: mustEnv("S3_SECRET_KEY"),
		UseSSL:          mustEnvBool("S3_USE_SSL"),
		Bucket:          mustEnv("S3_BUCKET"),
		PublicBaseURL:   os.Getenv("S3_PUBLIC_BASE_URL"),
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := s3Storage.EnsureBucket(context.Background()); err != nil {
		log.Fatal(err)
	}
	fileService := file.NewService(fileRepo, s3Storage)
	fileHandler := file.NewHandler(fileService)
	execCache := executor.NewCache()
	execHandler := executor.NewHandler(fileRepo, s3Storage, execCache)

	port := ":8080"
	log.Printf("Server listening on %s", port)

	mux := http.NewServeMux()
	mux.Handle("/call", authService.AuthMiddleware(http.HandlerFunc(execHandler.Call)))

	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/auth/refresh", authHandler.Refresh)

	mux.Handle("POST /projects", authService.AuthMiddleware(http.HandlerFunc(projectHandler.Create)))
	mux.Handle("GET /projects", authService.AuthMiddleware(http.HandlerFunc(projectHandler.List)))
	mux.Handle("GET /projects/{id}", authService.AuthMiddleware(http.HandlerFunc(projectHandler.GetByID)))
	mux.Handle("PUT /projects/{id}", authService.AuthMiddleware(http.HandlerFunc(projectHandler.Update)))
	mux.Handle("DELETE /projects/{id}", authService.AuthMiddleware(http.HandlerFunc(projectHandler.Delete)))

	mux.Handle("POST /projects/{project_id}/files", authService.AuthMiddleware(http.HandlerFunc(fileHandler.Create)))
	mux.Handle("GET /projects/{project_id}/files", authService.AuthMiddleware(http.HandlerFunc(fileHandler.ListByProject)))
	mux.Handle("GET /projects/{project_id}/files/{id}", authService.AuthMiddleware(http.HandlerFunc(fileHandler.GetByID)))
	mux.Handle("PUT /projects/{project_id}/files/{id}", authService.AuthMiddleware(http.HandlerFunc(fileHandler.Update)))
	mux.Handle("DELETE /projects/{project_id}/files/{id}", authService.AuthMiddleware(http.HandlerFunc(fileHandler.Delete)))

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

func mustEnvBool(key string) bool {
	value := mustEnv(key)
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatalf("%s must be true or false", key)
	}
	return parsed
}
