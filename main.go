package main

import (
	"database/sql"
	"log"
	"marketplace/internal/auth"
	"marketplace/internal/handlers"
	"marketplace/internal/middleware"
	"marketplace/internal/repository"
	"marketplace/internal/service"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// DATABASE CONNECTION & MIGRATIONS
// ============================================================================

func connectDB(dsn string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 1; i <= 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("✅ Connected to database")
				return db
			}
		}
		log.Printf("⏳ Waiting for database... %d/30", i)
		time.Sleep(time.Second)
	}
	log.Fatalf("❌ Cannot connect to database: %v", err)
	return nil
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("❌ Migration driver error: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("❌ Migrate init error: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("❌ Migration error: %v", err)
	}
	log.Println("✅ Migrations applied")
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Configuration
	dsn := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/marketplace_db?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	port := getEnv("PORT", "8080")
	orderRateLimitMinutes := 5

	// Database
	db := connectDB(dsn)
	defer db.Close()
	runMigrations(db)

	// Initialize layers
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, db, orderRateLimitMinutes)
	jwtManager := auth.NewJWTManager(jwtSecret, 30*time.Minute, 7*24*time.Hour)
	handler := handlers.NewHandler(svc, repo, jwtManager)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", handler.Health)

	// Auth endpoints (no auth required)
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", handler.RefreshToken)

	// Protected endpoints
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/products", handler.GetProducts)
	protected.HandleFunc("POST /api/v1/products", handler.CreateProduct)
	protected.HandleFunc("GET /api/v1/products/{id}", handler.GetProductByID)
	protected.HandleFunc("PUT /api/v1/products/{id}", handler.UpdateProduct)
	protected.HandleFunc("DELETE /api/v1/products/{id}", handler.SoftDeleteProduct)
	protected.HandleFunc("POST /api/v1/orders", handler.CreateOrder)
	protected.HandleFunc("GET /api/v1/orders/{id}", handler.GetOrderByID)
	protected.HandleFunc("PUT /api/v1/orders/{id}", handler.UpdateOrder)
	protected.HandleFunc("POST /api/v1/orders/{id}/cancel", handler.CancelOrder)
	protected.HandleFunc("POST /api/v1/promo-codes", handler.CreatePromoCode)

	// Apply middleware
	mux.Handle("/api/v1/", middleware.LoggingMiddleware(middleware.AuthMiddleware(jwtManager)(protected)))

	log.Printf("🚀 Marketplace service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
