//	@title			Radif API
//	@version		1.0
//	@description	Backend for Radif — social payment platform for Iran.
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Format: **Bearer {token}**

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/radif/service/internal/auth"
	"github.com/radif/service/internal/config"
	"github.com/radif/service/internal/db"
	appMiddleware "github.com/radif/service/internal/middleware"
	"github.com/radif/service/internal/storage"
	"github.com/radif/service/internal/user"

	_ "github.com/radif/service/docs/swagger"
)

func main() {
	cfg := config.Load()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	store, err := storage.NewMinioStorage(
		cfg.StorageEndpoint,
		cfg.StorageAccessKey,
		cfg.StorageSecretKey,
		cfg.StorageBucket,
		cfg.StoragePublicBase,
		cfg.StorageUseSSL,
	)
	if err != nil {
		log.Fatalf("object storage init failed: %v", err)
	}

	// Wire dependencies: repository → service → handler
	userRepo := user.NewRepository(pool)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc, store)

	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, userSvc, cfg)
	authHandler := auth.NewHandler(authSvc)

	// Router
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(appMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		MaxAge:         300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI — available at http://localhost:8080/swagger/
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/otp/send", authHandler.SendOTP)
			r.Post("/otp/verify", authHandler.VerifyOTP)
			r.Post("/otp/resend", authHandler.ResendOTP)
			r.Post("/register", authHandler.Register)
		})

		// Protected user endpoints
		r.Route("/users", func(r chi.Router) {
			r.Use(appMiddleware.RequireAuth(cfg.JWTSecret))
			r.Get("/me", userHandler.GetMe)
			r.Patch("/me", userHandler.UpdateProfile)
			r.Post("/me/avatar", userHandler.UploadAvatar)
			r.Get("/username-check", userHandler.CheckUsername)
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine; wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on :%s (env=%s)", cfg.Port, cfg.AppEnv)
		log.Printf("swagger UI at http://localhost:%s/swagger/", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
