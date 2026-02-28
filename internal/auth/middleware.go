package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type userContextKey struct{}

type Middleware struct {
	token  string
	userID uuid.UUID
}

func NewMiddleware(token string, userID uuid.UUID) Middleware {
	return Middleware{token: token, userID: userID}
}

func (m Middleware) Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(authz, prefix) || strings.TrimPrefix(authz, prefix) != m.token {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey{}, m.userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(userContextKey{})
	id, ok := v.(uuid.UUID)
	return id, ok
}
