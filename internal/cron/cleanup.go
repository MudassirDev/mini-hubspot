package cron

import (
	"context"
	"log"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/robfig/cron/v3"
)

func StartCronJobs(db *database.Queries) {
	c := cron.New()

	// Run hourly to clean up unverified users older than 24 hours
	_, err := c.AddFunc("@every 24h", func() {
		ctx := context.Background()
		log.Println("⏰ Running cron job: delete expired unverified users")

		err := db.DeleteExpiredUnverifiedUsers(ctx)
		if err != nil {
			log.Printf("⚠️ Error deleting expired users: %v", err)
		} else {
			log.Println("✅ Expired unverified users deleted")
		}
	})

	if err != nil {
		log.Fatalf("❌ Failed to schedule cron job: %v", err)
	}

	c.Start()
	log.Println("🌀 Cron jobs started")
}
