package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbConnString := os.Getenv("DATABASE_URL")
	if dbConnString == "" {
		log.Fatal("DATABASE_URL not set")
	}
	if !strings.Contains(dbConnString, "?sslmode=disable") {
		dbConnString += "?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	queries := database.New(db)

	for {
		log.Println("Running scheduled task: delete expired unverified users")

		ctx := context.Background()
		err := queries.DeleteExpiredUnverifiedUsers(ctx)
		if err != nil {
			log.Printf("Error deleting users: %v", err)
		} else {
			log.Println("Expired unverified users deleted successfully")
		}

		time.Sleep(24 * time.Hour)
	}
}
