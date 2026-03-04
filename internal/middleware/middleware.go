package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"marketplace/internal/auth"
	"marketplace/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserContextKey      contextKey = "user"
	RequestIDContextKey contextKey = "request_id"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs all requests in JSON format
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		rw.Header().Set("X-Request-Id", requestID)

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Milliseconds()

		var userID string
		if user, ok := r.Context().Value(UserContextKey).(*auth.Claims); ok {
			userID = user.UserID
		}

		logEntry := map[string]interface{}{
			"request_id":  requestID,
			"method":      r.Method,
			"endpoint":    r.URL.Path,
			"status_code": rw.statusCode,
			"duration_ms": duration,
			"user_id":     userID,
			"timestamp":   time.Now().Format(time.RFC3339),
		}

		logJSON, _ := json.Marshal(logEntry)
		log.Println(string(logJSON))
	})
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, models.ErrorCodeTOKEN_INVALID, "Authorization header required", nil)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, models.ErrorCodeTOKEN_INVALID, "Invalid authorization header format", nil)
				return
			}

			claims, err := jwtManager.ValidateToken(parts[1])
			if err != nil {
				if strings.Contains(err.Error(), "expired") {
					writeError(w, http.StatusUnauthorized, models.ErrorCodeTOKEN_EXPIRED, "Token has expired", nil)
				} else {
					writeError(w, http.StatusUnauthorized, models.ErrorCodeTOKEN_INVALID, "Invalid token", nil)
				}
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, code models.ErrorCode, message string, details map[string]interface{}) {
	errorResp := models.ErrorResponse{
		ErrorCode: code,
		Message:   message,
		Timestamp: time.Now(),
	}
	if details != nil {
		errorResp.Details = make(map[string]string)
		for k, v := range details {
			errorResp.Details[k] = fmt.Sprint(v)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResp)
}
