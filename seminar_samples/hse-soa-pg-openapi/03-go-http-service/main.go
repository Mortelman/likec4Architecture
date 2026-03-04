package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// --- Модели данных ---

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

// --- In-memory хранилище ---

type Store struct {
	mu      sync.RWMutex
	users   map[int64]*User
	counter int64
}

func NewStore() *Store {
	return &Store{
		users: make(map[int64]*User),
	}
}

func (s *Store) Create(req CreateUserRequest) *User {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	user := &User{
		ID:        s.counter,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	s.users[user.ID] = user
	return user
}

func (s *Store) GetAll() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

func (s *Store) GetByID(id int64) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Store) Delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

// --- HTTP обработчики ---

type Handler struct {
	store *Store
}

// writeJSON записывает JSON ответ с заданным статус кодом
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError записывает JSON ошибку
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Code: code, Message: message})
}

// GET /health
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /users
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.GetAll()
	// Возвращаем пустой массив, а не null
	if users == nil {
		users = []*User{}
	}
	writeJSON(w, http.StatusOK, users)
}

// POST /users
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Field 'name' is required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Field 'email' is required")
		return
	}

	user := h.store.Create(req)
	writeJSON(w, http.StatusCreated, user)
}

// GET /users/{id}
func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "ID must be an integer")
		return
	}

	user, ok := h.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// DELETE /users/{id}
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "ID must be an integer")
		return
	}

	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- main ---

func main() {
	store := NewStore()
	h := &Handler{store: store}

	// Роутер из стандартной библиотеки Go 1.22+
	// Поддерживает методы и path параметры {id}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.healthCheck)
	mux.HandleFunc("GET /users", h.listUsers)
	mux.HandleFunc("POST /users", h.createUser)
	mux.HandleFunc("GET /users/{id}", h.getUser)
	mux.HandleFunc("DELETE /users/{id}", h.deleteUser)

	log.Println("Users service started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
