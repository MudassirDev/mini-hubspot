package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/MudassirDev/mini-hubspot/internal/email"
	"github.com/lib/pq"
)

func IsProduction() bool {
	return os.Getenv("ENV") == "production"
}

func WriteJSONError(w http.ResponseWriter, status int, message string) {
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
			WriteJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))
		if req.Email == "" || req.Password == "" || req.Username == "" {
			WriteJSONError(w, http.StatusBadRequest, "Missing required fields: email, username or password")
			return
		}

		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		token, err := auth.GenerateVerificationToken()
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Failed to generate token")
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
			var pqErr *pq.Error
			if errors.As(err, &pqErr) {
				switch pqErr.Code.Name() {
				case "unique_violation":
					if pqErr.Constraint == "users_email_key" {
						WriteJSONError(w, http.StatusBadRequest, "Email already in use")
						return
					}
					if pqErr.Constraint == "users_username_key" {
						WriteJSONError(w, http.StatusBadRequest, "Username already taken")
						return
					}
					WriteJSONError(w, http.StatusBadRequest, "Duplicate field")
					return
				default:
					WriteJSONError(w, http.StatusBadRequest, "Database error: "+pqErr.Message)
					return
				}
			}

			WriteJSONError(w, http.StatusInternalServerError, "Unexpected error: "+err.Error())
			return
		}

		verifyLink := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("APP_HOST"), token)

		err = EmailSender.SendVerificationEmail(req.Email, req.FirstName, verifyLink)
		if err != nil {
			log.Println("failed to send verification email:", err)
			WriteJSONError(w, http.StatusInternalServerError, "Could not send verification email")
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
			WriteJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if req.Email == "" || req.Password == "" {
			WriteJSONError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := db.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			// Check for sql.ErrNoRows specifically
			if errors.Is(err, sql.ErrNoRows) {
				WriteJSONError(w, http.StatusUnauthorized, "User not found")
				return
			}

			// Unexpected DB error
			log.Printf("DB error during login: %v", err)
			WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
			WriteJSONError(w, http.StatusUnauthorized, "Invalid password")
			return
		}

		token, err := auth.MakeJWT(user.ID, expiresIn, jwtSecret)
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		secure := IsProduction()
		sameSite := http.SameSiteStrictMode
		if secure {
			sameSite = http.SameSiteNoneMode
		}

		// Set JWT as an HTTP-only cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
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
			WriteJSONError(w, http.StatusBadRequest, "Missing token")
			return
		}

		user, err := db.GetUserByVerificationToken(r.Context(), sql.NullString{String: token, Valid: true})
		if err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Invalid or expired token")
			return
		}

		if user.EmailVerified {
			json.NewEncoder(w).Encode(map[string]string{"message": "Email already verified"})
			return
		}

		// Optional: ensure token is not older than 30 days
		if user.TokenSentAt.Valid && time.Since(user.TokenSentAt.Time) > 30*24*time.Hour {
			WriteJSONError(w, http.StatusBadRequest, "Token expired")
			return
		}

		err = db.VerifyUserEmail(r.Context(), user.ID)
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Could not verify email")
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Email verified successfully"})
	}
}

func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
