package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/MudassirDev/mini-hubspot/internal/cron"
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
	cron.StartCronJobs(queries)

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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		RenderTemplate(w, "index", map[string]any{
			"Title": "Home",
			"Year":  time.Now().Year(),
		})
	})
	r.Get("/verify-email", appHandler.VerifyEmailHandler(queries))
	r.Post("/webhook/stripe", appHandler.StripeWebhookHandler(queries))
	r.Mount("/static/", fs)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/create-account", appHandler.CreateUserHandler(queries, *apiCfg.EmailSender))
		r.Post("/login", appHandler.LoginHandler(queries, apiCfg.JwtSecret, apiCfg.JwtExpiry))
	})

	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.AuthMiddleware(queries, apiCfg.JwtSecret))

		r.Route("/contacts", func(r chi.Router) {
			r.Get("/", appHandler.GetContactsHandler(queries))
			r.Post("/", appHandler.CreateContactHandler(queries))
			r.Get("/{id}", appHandler.GetContactByIDHandler(queries))
			r.Patch("/{id}", appHandler.UpdateContactHandler(queries))
			r.Delete("/{id}", appHandler.DeleteContactHandler(queries))
		})
	})

	return r
}
