package handler

import (
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

		// Handle the event
		switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession
			if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
				http.Error(w, "Unmarshal error", http.StatusBadRequest)
				return
			}

			customerEmail := session.CustomerDetails.Email
			log.Printf("Checkout completed for customer: %s", customerEmail)
			err := db.UpgradeUserPlanByEmail(r.Context(), session.CustomerDetails.Email)
			if err != nil {
				log.Printf("Failed to upgrade plan for %s: %v", customerEmail, err)
			} else {
				log.Printf("Plan upgraded to 'pro' for %s", customerEmail)
			}
		default:
			log.Printf("Unhandled event type: %s", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	}
}
