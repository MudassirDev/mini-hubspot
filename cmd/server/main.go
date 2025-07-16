package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/MudassirDev/mini-hubspot/internal/database"
	"github.com/MudassirDev/mini-hubspot/internal/email"
	appHandler "github.com/MudassirDev/mini-hubspot/internal/handler"
	appMiddleware "github.com/MudassirDev/mini-hubspot/internal/middleware"
)

type APIConfig struct {
	JwtSecret   string
	JwtExpiry   time.Duration
	EmailSender *email.MailtrapEmailSender
}

func main() {
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
		JwtSecret:   jwtSecret,
		JwtExpiry:   1 * time.Hour,
		EmailSender: email.NewMailtrapSender(),
	}

	server := &http.Server{Addr: port, Handler: service(apiCfg, queries)}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	defer serverStopCtx()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	log.Printf("Server starting on http://localhost%v", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-serverCtx.Done()
	log.Println("Server shutdown gracefully.")
}

func service(apiCfg APIConfig, queries *database.Queries) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	cwd, _ := os.Getwd()
	fs := http.StripPrefix("/static/", http.FileServer(http.Dir(cwd+"/frontend/static")))

	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.AuthMiddleware(queries, apiCfg.JwtSecret, false))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			loggedIn := false
			user, ok := appMiddleware.GetUserFromContext(r.Context())
			if ok {
				loggedIn = true
			}

			RenderTemplate(w, "index", map[string]any{
				"Title":    "Home",
				"Year":     time.Now().Year(),
				"LoggedIn": loggedIn,
				"User":     user,
			})
		})
		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			RenderTemplate(w, "login", map[string]any{
				"Title": "Login",
				"Year":  time.Now().Year(),
			})
		})
		r.Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			RenderTemplate(w, "signup", map[string]any{
				"Title": "Sign Up",
				"Year":  time.Now().Year(),
			})
		})
		r.Get("/plans", func(w http.ResponseWriter, r *http.Request) {
			loggedIn := false
			user, ok := appMiddleware.GetUserFromContext(r.Context())
			if ok {
				loggedIn = true
			}
			RenderTemplate(w, "plans", map[string]any{
				"Title":    "Plans",
				"Year":     time.Now().Year(),
				"LoggedIn": loggedIn,
				"User":     user,
			})
		})
	})
	r.Get("/logout", appHandler.LogoutHandler())
	r.Get("/verify-email", appHandler.VerifyEmailHandler(queries))
	r.Post("/webhook/stripe", appHandler.StripeWebhookHandler(queries))
	r.Mount("/static/", fs)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/create-account", appHandler.CreateUserHandler(queries, *apiCfg.EmailSender))
		r.Post("/login", appHandler.LoginHandler(queries, apiCfg.JwtSecret, apiCfg.JwtExpiry))
	})

	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.AuthMiddleware(queries, apiCfg.JwtSecret, true))

		r.Route("/contacts", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				user, ok := appMiddleware.GetUserFromContext(r.Context())
				if !ok {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				RenderTemplate(w, "contacts", map[string]any{
					"Title":    "Contacts",
					"Year":     time.Now().Year(),
					"LoggedIn": true,
					"User":     user,
				})
			})
			r.Get("/all", appHandler.GetContactsHandler(queries))
			r.Post("/new", appHandler.CreateContactHandler(queries))
			r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				user, ok := appMiddleware.GetUserFromContext(r.Context())
				if !ok {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}

				idStr := chi.URLParam(r, "id")
				contactID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					RenderTemplate(w, "error", map[string]any{
						"Title":      "Invalid Contact ID",
						"Year":       time.Now().Year(),
						"LoggedIn":   true,
						"User":       user,
						"Message":    "The contact ID provided is invalid. Please check the URL.",
						"StatusCode": http.StatusBadRequest,
					})
					return
				}

				contact, err := queries.GetContactByID(r.Context(), database.GetContactByIDParams{
					ID:     contactID,
					UserID: user.ID,
				})
				if err != nil {
					if err == sql.ErrNoRows {
						RenderTemplate(w, "error", map[string]any{
							"Title":      "Contact Not Found",
							"Year":       time.Now().Year(),
							"LoggedIn":   true,
							"User":       user,
							"Message":    "The contact you are looking for was not found or does not belong to your account.",
							"StatusCode": http.StatusNotFound,
						})
						return
					}
					RenderTemplate(w, "error", map[string]any{
						"Title":      "Server Error",
						"Year":       time.Now().Year(),
						"LoggedIn":   true,
						"User":       user,
						"Message":    fmt.Sprintf("An unexpected error occurred while fetching the contact: %v", err),
						"StatusCode": http.StatusInternalServerError,
					})
					return
				}

				RenderTemplate(w, "contact", map[string]any{
					"Title":    contact.Name,
					"Year":     time.Now().Year(),
					"LoggedIn": true,
					"User":     user,
					"Contact":  contact,
					"IsEdit":   true,
				})
			})
			r.Patch("/{id}", appHandler.UpdateContactHandler(queries))
			r.Delete("/{id}", appHandler.DeleteContactHandler(queries))
			r.Get("/export", appHandler.ExportContactsCSVHandler(queries))
		})
	})

	return r
}
