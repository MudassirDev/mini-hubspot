package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/MudassirDev/mini-hubspot/internal/email"
)

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func CreateUserHandler(db *database.Queries, EmailSender email.MailtrapEmailSender) http.HandlerFunc {
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

		token, err := auth.GenerateVerificationToken()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		tokenSentAt := time.Now()

		user, err := db.CreateUser(r.Context(), database.CreateUserParams{
			Username:          req.Username,
			Email:             req.Email,
			FirstName:         req.FirstName,
			LastName:          req.LastName,
			PasswordHash:      hashedPassword,
			Role:              "user",
			Plan:              "free",
			VerificationToken: sql.NullString{String: token, Valid: true},
			TokenSentAt:       sql.NullTime{Time: tokenSentAt, Valid: true},
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Error creating user: "+err.Error())
			return
		}

		verifyLink := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("APP_HOST"), token)

		err = EmailSender.SendVerificationEmail(req.Email, req.FirstName, verifyLink)
		if err != nil {
			log.Println("failed to send verification email:", err)
			writeJSONError(w, http.StatusInternalServerError, "Could not send verification email")
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

func VerifyEmailHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing token")
			return
		}

		user, err := db.GetUserByVerificationToken(r.Context(), sql.NullString{String: token, Valid: true})
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid or expired token")
			return
		}

		if user.EmailVerified {
			json.NewEncoder(w).Encode(map[string]string{"message": "Email already verified"})
			return
		}

		// Optional: ensure token is not older than 30 days
		if user.TokenSentAt.Valid && time.Since(user.TokenSentAt.Time) > 30*24*time.Hour {
			writeJSONError(w, http.StatusBadRequest, "Token expired")
			return
		}

		err = db.VerifyUserEmail(r.Context(), user.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not verify email")
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Email verified successfully"})
	}
}
