package middleware

import (
	"context"
	"net/http"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
)

type contextKey string

const UserContextKey = contextKey("user")

// GetUserFromContext retrieves user from context
func GetUserFromContext(ctx context.Context) (*database.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*database.User)
	return user, ok
}

// AuthMiddleware verifies JWT from cookie and attaches user to context
func AuthMiddleware(db *database.Queries, jwtSecret string, redirectOnFail bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				if redirectOnFail {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
				}
				// If not redirecting, just pass request as is
				next.ServeHTTP(w, r)
				return
			}

			userID, err := auth.VerifyJWT(cookie.Value, jwtSecret)
			if err != nil {
				if redirectOnFail {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
				}
				next.ServeHTTP(w, r)
				return
			}

			user, err := db.GetUserByID(r.Context(), userID)
			if err != nil {
				if redirectOnFail {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
				}
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
