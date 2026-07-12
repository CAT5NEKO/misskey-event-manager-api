package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"miSchedule/internal/auth"
	"miSchedule/internal/config"
	"miSchedule/internal/handler"
	"miSchedule/internal/middleware"
	"miSchedule/internal/notifier"
	"miSchedule/internal/repository"
	"miSchedule/internal/service"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connected")

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("migrations applied")

	tokenCrypto, err := auth.NewTokenCrypto(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("failed to init crypto: %v", err)
	}

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTRefreshSecret)
	csrfStore := auth.NewCSRFStore()
	miauthClient := auth.NewMiAuthClient(cfg)

	userRepo := repository.NewUserRepo(db, tokenCrypto)
	eventRepo := repository.NewEventRepo(db)
	participantRepo := repository.NewParticipantRepo(db)
	instanceRepo := repository.NewInstanceRepo(db)
	auditRepo := repository.NewAuditLogRepo(db)
	settingRepo := repository.NewSettingRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)

	authService := service.NewAuthService(
		cfg, userRepo, instanceRepo, refreshTokenRepo, settingRepo, auditRepo,
		miauthClient, jwtManager, csrfStore, tokenCrypto,
	)

	eventService := service.NewEventService(
		eventRepo, participantRepo, userRepo, auditRepo, settingRepo,
	)

	adminService := service.NewAdminService(
		cfg, userRepo, eventRepo, instanceRepo, auditRepo, settingRepo, refreshTokenRepo,
	)

	authHandler := handler.NewAuthHandler(authService)
	eventHandler := handler.NewEventHandler(eventService)
	adminHandler := handler.NewAdminHandler(adminService)

	adminService.SeedInstances()

	notificationScheduler := notifier.NewNotificationScheduler(
		cfg, eventRepo, participantRepo, userRepo, auditRepo,
	)
	notificationScheduler.Run()

	go cleanupLoop(adminService, refreshTokenRepo)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.BodyLimit(1 << 20))

	allowedOrigins := cfg.AllowedOrigins
	if cfg.IsDev() {
		allowedOrigins = append(allowedOrigins, "http://localhost:5174", "http://localhost:5175")
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	rateLimiter := middleware.NewRateLimiter()
	r.Use(rateLimiter.GeneralLimit)

	r.Route("/api", func(r chi.Router) {
		r.Get("/app/name", func(w http.ResponseWriter, r *http.Request) {
			setting, _ := settingRepo.Get("app.name")
			name := "miSchedule"
			if setting != nil && len(setting.Value) > 2 {
				var n string
				if err := json.Unmarshal(setting.Value, &n); err == nil && n != "" {
					name = n
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf(`{"name":"%s"}`, name)))
		})

		r.Get("/app/config", func(w http.ResponseWriter, r *http.Request) {
			allowSelfDelete := true
			setting, _ := settingRepo.Get("users.allow_self_delete")
			if setting != nil {
				var v bool
				if err := json.Unmarshal(setting.Value, &v); err == nil {
					allowSelfDelete = v
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf(`{"allow_self_delete":%v}`, allowSelfDelete)))
		})

		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/callback", authHandler.Callback)
		r.Post("/auth/refresh", authHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtManager))

			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/revoke", authHandler.Revoke)
			r.Delete("/auth/account", authHandler.DeleteAccount)

			r.Get("/events", eventHandler.List)
			r.Post("/events", eventHandler.Create)
			r.Get("/events/{id}", eventHandler.Get)
			r.Put("/events/{id}", eventHandler.Update)
			r.Delete("/events/{id}", eventHandler.Delete)
			r.Post("/events/{id}/join", eventHandler.Join)
			r.Delete("/events/{id}/join", eventHandler.Leave)
			r.Put("/events/{id}/join", eventHandler.Join)

			r.Group(func(r chi.Router) {
				r.Use(middleware.AdminOnly)
				r.Use(rateLimiter.AdminLimit)

				r.Get("/admin/instances", adminHandler.ListInstances)
				r.Post("/admin/instances", adminHandler.AddInstance)
				r.Put("/admin/instances/{id}", adminHandler.UpdateInstance)
				r.Delete("/admin/instances/{id}", adminHandler.DeleteInstance)

				r.Get("/admin/users", adminHandler.ListUsers)
				r.Get("/admin/users/{id}", adminHandler.GetUser)
				r.Delete("/admin/users/{id}", adminHandler.DeleteUser)
				r.Put("/admin/users/{id}/deactivate", adminHandler.DeactivateUser)

				r.Get("/admin/events", adminHandler.ListEvents)
				r.Delete("/admin/events/{id}", adminHandler.DeleteEvent)

				r.Get("/admin/audit-logs", adminHandler.ListAuditLogs)
				r.Get("/admin/settings", adminHandler.GetSettings)
				r.Put("/admin/settings", adminHandler.UpdateSetting)
			})
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := ":" + cfg.Port
	log.Printf("server starting on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func cleanupLoop(adminService *service.AdminService, refreshRepo *repository.RefreshTokenRepo) {
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		adminService.CleanupAuditLogs()
		refreshRepo.CleanExpired()
	}
}

func runMigrations(db *sql.DB) error {
	dir := "migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("no migrations directory, skipping")
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
		log.Printf("migration applied: %s", name)
	}
	return nil
}
