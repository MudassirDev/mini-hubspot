package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/MudassirDev/mini-hubspot/internal/middleware"
	"github.com/google/uuid"
)

type UpdateContactRequest = CreateContactRequest

func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func CreateContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var req CreateContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "Contact name is required")
			return
		}

		contact, err := db.CreateContact(context.Background(), database.CreateContactParams{
			UserID:   user.ID,
			Name:     req.Name,
			Email:    ToNullString(req.Email),
			Phone:    ToNullString(req.Phone),
			Company:  ToNullString(req.Company),
			Position: ToNullString(req.Position),
			Notes:    ToNullString(req.Notes),
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not create contact")
			return
		}

		json.NewEncoder(w).Encode(contact)
	}
}

func GetContactsHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		contacts, err := db.GetContactsByUser(r.Context(), user.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not fetch contacts")
			return
		}

		json.NewEncoder(w).Encode(contacts)
	}
}

func GetContactByIDHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		idStr := r.PathValue("id")
		contactID, err := uuid.Parse(idStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid contact ID")
			return
		}

		contact, err := db.GetContactByID(r.Context(), database.GetContactByIDParams{
			ID:     contactID,
			UserID: user.ID,
		})
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "Contact not found")
			return
		}

		json.NewEncoder(w).Encode(contact)
	}
}

func UpdateContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		idStr := r.PathValue("id")
		contactID, err := uuid.Parse(idStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid contact ID")
			return
		}

		var req UpdateContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		updated, err := db.UpdateContact(r.Context(), database.UpdateContactParams{
			ID:       contactID,
			UserID:   user.ID,
			Name:     req.Name,
			Email:    ToNullString(req.Email),
			Phone:    ToNullString(req.Phone),
			Company:  ToNullString(req.Company),
			Position: ToNullString(req.Position),
			Notes:    ToNullString(req.Notes),
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not update contact")
			return
		}

		json.NewEncoder(w).Encode(updated)
	}
}

func DeleteContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		idStr := r.PathValue("id")
		contactID, err := uuid.Parse(idStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid contact ID")
			return
		}

		err = db.DeleteContact(r.Context(), database.DeleteContactParams{
			ID:     contactID,
			UserID: user.ID,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not delete contact")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
