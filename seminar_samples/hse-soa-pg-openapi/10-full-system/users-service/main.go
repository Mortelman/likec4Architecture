package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func connectDB(dsn string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 1; i <= 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("Connected to database")
				return db
			}
		}
		log.Printf("Waiting for database... %d/30", i)
		time.Sleep(time.Second)
	}
	log.Fatalf("Cannot connect to database: %v", err)
	return nil
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Migration driver error: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("Migrate init error: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration error: %v", err)
	}
	log.Println("Migrations applied")
}

type UserRepo struct{ db *sql.DB }

func (r *UserRepo) Create(ctx context.Context, name, email string) (*User, error) {
	u := &User{Name: name, Email: email}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at`,
		name, email).Scan(&u.ID, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) GetAll(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, email, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorResponse{Code: code, Message: msg})
}

type Handler struct{ repo *UserRepo }

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(w, 500, "DB_ERROR", err.Error())
		return
	}
	if users == nil {
		users = []User{}
	}
	writeJSON(w, 200, users)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "INVALID_JSON", "Invalid request body")
		return
	}
	if req.Name == "" || req.Email == "" {
		writeError(w, 400, "VALIDATION_ERROR", "name and email are required")
		return
	}
	user, err := h.repo.Create(r.Context(), req.Name, req.Email)
	if err != nil {
		writeError(w, 500, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, 201, user)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, 400, "INVALID_ID", "ID must be an integer")
		return
	}
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, 500, "DB_ERROR", err.Error())
		return
	}
	if user == nil {
		writeError(w, 404, "NOT_FOUND", "User not found")
		return
	}
	writeJSON(w, 200, user)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, 400, "INVALID_ID", "ID must be an integer")
		return
	}
	deleted, err := h.repo.Delete(r.Context(), id)
	if err != nil {
		writeError(w, 500, "DB_ERROR", err.Error())
		return
	}
	if !deleted {
		writeError(w, 404, "NOT_FOUND", "User not found")
		return
	}
	w.WriteHeader(204)
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/users_db?sslmode=disable"
	}

	db := connectDB(dsn)
	defer db.Close()
	runMigrations(db)

	h := &Handler{repo: &UserRepo{db: db}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /users", h.listUsers)
	mux.HandleFunc("POST /users", h.createUser)
	mux.HandleFunc("GET /users/{id}", h.getUser)
	mux.HandleFunc("DELETE /users/{id}", h.deleteUser)

	log.Println("Users service started on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
