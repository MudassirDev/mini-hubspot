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

func NewContactResponse(c database.Contact) ContactResponse {
	return ContactResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Name:      c.Name,
		Email:     c.Email.String,
		Phone:     c.Phone.String,
		Company:   c.Company.String,
		Position:  c.Position.String,
		Notes:     c.Notes.String,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func NewContactResponseList(cs []database.Contact) []ContactResponse {
	res := make([]ContactResponse, len(cs))
	for i, c := range cs {
		res[i] = NewContactResponse(c)
	}
	return res
}

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

		json.NewEncoder(w).Encode(NewContactResponse(contact))
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

		json.NewEncoder(w).Encode(NewContactResponseList(contacts))
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

		json.NewEncoder(w).Encode(NewContactResponse(contact))
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

		var req PatchContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		// Get the existing contact
		existing, err := db.GetContactByID(r.Context(), database.GetContactByIDParams{
			ID:     contactID,
			UserID: user.ID,
		})
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "Contact not found")
			return
		}

		// Merge fields: only overwrite if provided
		name := existing.Name
		if req.Name != nil {
			name = strings.TrimSpace(*req.Name)
		}

		// Helper to choose between new or existing
		choose := func(newVal *string, oldVal sql.NullString) sql.NullString {
			if newVal != nil {
				return ToNullString(*newVal)
			}
			return oldVal
		}

		updated, err := db.UpdateContact(r.Context(), database.UpdateContactParams{
			ID:       contactID,
			UserID:   user.ID,
			Name:     name,
			Email:    choose(req.Email, existing.Email),
			Phone:    choose(req.Phone, existing.Phone),
			Company:  choose(req.Company, existing.Company),
			Position: choose(req.Position, existing.Position),
			Notes:    choose(req.Notes, existing.Notes),
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not update contact")
			return
		}

		json.NewEncoder(w).Encode(NewContactResponse(updated))
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
