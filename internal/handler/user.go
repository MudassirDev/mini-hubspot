package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
)

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func CreateUserHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))
		if req.Email == "" || req.Password == "" || req.Username == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required fields: email, username or password")
			return
		}

		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		user, err := db.CreateUser(r.Context(), database.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			PasswordHash: hashedPassword,
			Role:         "user",
			Plan:         "free",
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Error creating user: "+err.Error())
			return
		}

		resp := CreateUserResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Plan:  user.Plan,
			Role:  user.Role,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func LoginHandler(db *database.Queries, jwtSecret string, expiresIn time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if req.Email == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := db.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		token, err := auth.MakeJWT(user.ID, expiresIn, jwtSecret)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		// Set JWT as an HTTP-only cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // set to true if using HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int((expiresIn).Seconds()),
		})

		// Respond with basic user info (but no token)
		resp := LoginResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Plan:  user.Plan,
			Role:  user.Role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
