package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type APIConfig struct {
	JwtSecret string
	DB        *sql.DB
}

func main() {
	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, falling back to system env")
	}

	// Get env vars
	port := ":" + os.Getenv("PORT")
	jwtSecret := os.Getenv("JWT_SECRET")
	dbConnString := os.Getenv("DATABASE_URL")
	if !strings.Contains(dbConnString, "sslmode=") {
		dbConnString += "?sslmode=disable"
	}

	// Connect to DB
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Optional: Ping DB to test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Database is unreachable:", err)
	}

	// Create config for handlers
	_ = APIConfig{
		JwtSecret: jwtSecret,
		DB:        db,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cwd, err := os.Getwd()
		if err != nil {
			http.Error(w, "Failed to get working directory", http.StatusInternalServerError)
			return
		}

		filePath := cwd + "/frontend/templates/index.html"
		http.ServeFile(w, r, filePath)
	})

	// Start server
	srv := http.Server{
		Addr:    port,
		Handler: mux,
	}

	fmt.Printf("Server is listening on http://localhost%v\n", port)
	log.Fatal(srv.ListenAndServe())
}
