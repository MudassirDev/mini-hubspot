package middleware

import (
	"context"
	"net/http"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
)

type contextKey string

const userContextKey = contextKey("user")

// GetUserFromContext retrieves user from context
func GetUserFromContext(ctx context.Context) (*database.User, bool) {
	user, ok := ctx.Value(userContextKey).(*database.User)
	return user, ok
}

// AuthMiddleware verifies JWT from cookie and attaches user to context
func AuthMiddleware(db *database.Queries, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
			}

			userID, err := auth.VerifyJWT(cookie.Value, jwtSecret)
			if err != nil {
				http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
				return
			}

			user, err := db.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
