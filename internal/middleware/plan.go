package middleware

import (
	"net/http"
)

// RequirePlan ensures the user has the required subscription plan
func RequirePlan(plan string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok || user.Plan != plan {
				http.Error(w, "Upgrade required to access this feature", http.StatusPaymentRequired) // 402
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
