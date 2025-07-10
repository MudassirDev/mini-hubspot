package handler

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func StripeWebhookHandler(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		const MaxBodyBytes = int64(65536)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Read error", http.StatusServiceUnavailable)
			return
		}

		sigHeader := r.Header.Get("Stripe-Signature")
		event, err := webhook.ConstructEvent(payload, sigHeader, stripeWebhookSecret)
		if err != nil {
			http.Error(w, "Webhook verification failed", http.StatusBadRequest)
			return
		}

		switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession
			if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
				http.Error(w, "Unmarshal error", http.StatusBadRequest)
				return
			}

			email := session.CustomerDetails.Email
			customerID := session.Customer.ID

			log.Printf("Checkout completed for %s (Stripe ID: %s)", email, customerID)

			// Store Stripe customer ID now
			err := db.UpdateStripeCustomerIDByEmail(r.Context(), database.UpdateStripeCustomerIDByEmailParams{
				Email: email,
				StripeCustomerID: sql.NullString{
					String: customerID,
					Valid:  true,
				},
			})
			if err != nil {
				log.Printf("Failed to save customer ID for %s: %v", email, err)
			}

		case "invoice.paid":
			var invoice stripe.Invoice
			if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
				http.Error(w, "Unmarshal error", http.StatusBadRequest)
				return
			}

			email := invoice.CustomerEmail
			log.Printf("Invoice paid for: %s", email)

			err := db.UpgradeUserPlanByEmail(r.Context(), email)
			if err != nil {
				log.Printf("Failed to upgrade plan for %s: %v", email, err)
			} else {
				log.Printf("Plan upgraded to 'pro' for %s", email)
			}

		case "customer.subscription.deleted":
			var sub stripe.Subscription
			if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
				http.Error(w, "Unmarshal error", http.StatusBadRequest)
				return
			}

			customerID := sub.Customer.ID
			log.Printf("Subscription canceled for customer ID: %s", customerID)

			err := db.DowngradeUserPlanByStripeCustomerID(r.Context(), sql.NullString{String: customerID, Valid: true})
			if err != nil {
				log.Printf("Failed to downgrade plan for customer %s: %v", customerID, err)
			} else {
				log.Printf("Plan downgraded to 'free' for customer %s", customerID)
			}

		default:
			log.Printf("Unhandled event type: %s", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	}
}
