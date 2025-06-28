package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MudassirDev/mini-hubspot/internal/auth"
	"github.com/MudassirDev/mini-hubspot/internal/database"
)

func CreateUserHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON input", http.StatusBadRequest)
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))
		if req.Email == "" || req.Password == "" || req.Username == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		user, err := db.CreateUser(context.Background(), database.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			PasswordHash: hashedPassword,
			Role:         "user",
			Plan:         "free",
		})

		if err != nil {
			http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
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
