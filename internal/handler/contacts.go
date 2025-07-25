package handler

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/MudassirDev/mini-hubspot/internal/middleware"
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

func parseBoolQuery(val string) bool {
	b, err := strconv.ParseBool(val)
	return err == nil && b
}

func CreateContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		count, err := db.CountContactsByUser(r.Context(), user.ID)
		if err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Failed to fetch count")
			return
		}

		if user.Plan == "free" && count >= 100 {
			http.Error(w, "Contact limit reached. Upgrade to Pro.", http.StatusForbidden)
			return
		}

		var req CreateContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			WriteJSONError(w, http.StatusBadRequest, "Contact name is required")
			return
		}

		contact, err := db.CreateContact(r.Context(), database.CreateContactParams{
			UserID:   user.ID,
			Name:     req.Name,
			Email:    ToNullString(req.Email),
			Phone:    ToNullString(req.Phone),
			Company:  ToNullString(req.Company),
			Position: ToNullString(req.Position),
			Notes:    ToNullString(req.Notes),
		})
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Could not create contact")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NewContactResponse(contact))
	}
}

func GetContactsHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		query := r.URL.Query()

		// Parse limit
		limitStr := query.Get("limit")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 20
		}

		// Parse cursor (after ID)
		afterStr := query.Get("after")
		var after int64
		if afterStr != "" {
			if afterID, err := strconv.ParseInt(afterStr, 10, 64); err == nil {
				after = afterID
			}
		}
		search := query.Get("search")

		requireNonEmptyCompany := parseBoolQuery(query.Get("require_non_empty_company"))
		requireNonEmptyPhone := parseBoolQuery(query.Get("require_non_empty_phone"))
		requireNonEmptyEmail := parseBoolQuery(query.Get("require_non_empty_email"))
		requireNonEmptyPosition := parseBoolQuery(query.Get("require_non_empty_position"))

		contacts, err := db.GetContactsPaginated(r.Context(), database.GetContactsPaginatedParams{
			UserID:                  user.ID,
			After:                   after,
			Limit:                   int32(limit),
			Search:                  search,
			RequireNonEmptyCompany:  requireNonEmptyCompany,
			RequireNonEmptyPhone:    requireNonEmptyPhone,
			RequireNonEmptyEmail:    requireNonEmptyEmail,
			RequireNonEmptyPosition: requireNonEmptyPosition,
		})
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Could not fetch contacts")
			return
		}

		// Determine next cursor
		var nextCursor *int64
		if len(contacts) == int(limit) {
			last := contacts[len(contacts)-1]
			nextCursor = &last.ID
		}

		// Build response
		resp := map[string]any{
			"contacts":    NewContactResponseList(contacts),
			"next_cursor": nextCursor,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func UpdateContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		idStr := r.PathValue("id")
		contactID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Invalid contact ID")
			return
		}

		var req PatchContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		// Get the existing contact
		existing, err := db.GetContactByID(r.Context(), database.GetContactByIDParams{
			ID:     contactID,
			UserID: user.ID,
		})
		if err != nil {
			WriteJSONError(w, http.StatusNotFound, "Contact not found")
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
			WriteJSONError(w, http.StatusInternalServerError, "Could not update contact")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NewContactResponse(updated))
	}
}

func DeleteContactHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		idStr := r.PathValue("id")
		contactID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			WriteJSONError(w, http.StatusBadRequest, "Invalid contact ID")
			return
		}

		err = db.DeleteContact(r.Context(), database.DeleteContactParams{
			ID:     contactID,
			UserID: user.ID,
		})
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "Could not delete contact")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ExportContactsCSVHandler(q *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user.Plan != "pro" {
			http.Error(w, "CSV export is only available to Pro users", http.StatusForbidden)
			return
		}

		contacts, err := q.GetContactsByUser(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="contacts.csv"`)

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"Name", "Email", "Phone", "Company", "Position", "Notes", "CreatedAt"})

		for _, c := range contacts {
			writer.Write([]string{
				c.Name,
				nullString(c.Email),
				nullString(c.Phone),
				nullString(c.Company),
				nullString(c.Position),
				nullString(c.Notes),
				c.CreatedAt.Format("2006-01-02 15:04"),
			})
		}
	}
}

func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
