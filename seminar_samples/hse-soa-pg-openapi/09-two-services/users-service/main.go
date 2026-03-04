package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
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

type Store struct {
	mu      sync.RWMutex
	users   map[int64]*User
	counter int64
}

func NewStore() *Store {
	return &Store{users: make(map[int64]*User)}
}

func (s *Store) Create(name, email string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	u := &User{ID: s.counter, Name: name, Email: email, CreatedAt: time.Now()}
	s.users[u.ID] = u
	return u
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorResponse{Code: code, Message: msg})
}

type Handler struct{ store *Store }

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.GetAll()
	if users == nil {
		users = []*User{}
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
	writeJSON(w, 201, h.store.Create(req.Name, req.Email))
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, 400, "INVALID_ID", "ID must be an integer")
		return
	}
	user, ok := h.store.GetByID(id)
	if !ok {
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
	if !h.store.Delete(id) {
		writeError(w, 404, "NOT_FOUND", "User not found")
		return
	}
	w.WriteHeader(204)
}

func main() {
	h := &Handler{store: NewStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /users", h.listUsers)
	mux.HandleFunc("POST /users", h.createUser)
	mux.HandleFunc("GET /users/{id}", h.getUser)
	mux.HandleFunc("DELETE /users/{id}", h.deleteUser)

	log.Println("Users service started on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
