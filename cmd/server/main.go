package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	appHandler "github.com/MudassirDev/mini-hubspot/internal/handler"
	appMiddleware "github.com/MudassirDev/mini-hubspot/internal/middleware"
)

type APIConfig struct {
	JwtSecret string
	DB        *database.Queries
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system env")
	}

	port := ":" + os.Getenv("PORT")
	jwtSecret := os.Getenv("JWT_SECRET")
	dbConnString := os.Getenv("DATABASE_URL")
	if dbConnString == "" {
		log.Fatal("DATABASE_URL not set")
	}
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	if !strings.Contains(dbConnString, "?sslmode=disable") {
		dbConnString += "?sslmode=disable"
	}

	// DB connection
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database unreachable:", err)
	}

	queries := database.New(db)
	apiCfg := APIConfig{
		JwtSecret: jwtSecret,
		DB:        queries,
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve frontend
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/templates/index.html")
	})

	// Public auth routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/create-account", appHandler.CreateUserHandler(apiCfg.DB))
		r.Post("/login", appHandler.LoginHandler(apiCfg.DB, apiCfg.JwtSecret))
	})

	// Authenticated contacts routes
	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.AuthMiddleware(apiCfg.DB, apiCfg.JwtSecret))

		r.Route("/contacts", func(r chi.Router) {
			r.Get("/", appHandler.GetContactsHandler(apiCfg.DB))
			r.Post("/", appHandler.CreateContactHandler(apiCfg.DB))
			r.Get("/{id}", appHandler.GetContactByIDHandler(apiCfg.DB))
			r.Patch("/{id}", appHandler.UpdateContactHandler(apiCfg.DB))
			r.Delete("/{id}", appHandler.DeleteContactHandler(apiCfg.DB))
		})
	})

	fmt.Printf("Server is running on http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
