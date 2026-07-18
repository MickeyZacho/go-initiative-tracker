package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-initiative-tracker/dao"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import the PostgreSQL driver
	"golang.org/x/oauth2"
)

var db *sql.DB
var characterDAO dao.CharacterDAO
var npcTemplateDAO dao.NpcTemplateDAO
var encounterCharacterDAO dao.EncounterCharacterDAO
var encounterLedgerDAO dao.EncounterLedgerDAO
var encounterDAO dao.EncounterDAO
var friendshipDAO dao.FriendshipDAO
var userDAO dao.UserDAO
var frontendURL string
var allowedOrigins map[string]bool

func isAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(strings.TrimRight(origin, "/"))
	if origin == "" {
		return false
	}
	if allowedOrigins[origin] {
		return true
	}
	if strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:") {
		return true
	}
	return false
}

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/api/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

var discordOAuthConfig *oauth2.Config
var secureCookies bool

func initializeApp(db *sql.DB) {
	characterDAO = dao.NewCharacterDAO(db)
	encounterDAO = dao.NewEncounterDAO(db)
	encounterCharacterDAO = dao.NewEncounterCharacterDAO(db)
	encounterLedgerDAO = dao.NewEncounterLedgerDAO(db)
	npcTemplateDAO = dao.NewNpcTemplateDAO(db)
	friendshipDAO = dao.NewFriendshipDAO(db)
	userDAO = dao.NewUserDAO(db)
}

func loggingMiddleware(next http.Handler) http.Handler {
	log.Printf("Logging middleware initialized")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !isAllowedOrigin(origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		start := time.Now()
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic occurred: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Load variables from a local .env file if present. Existing environment
	// variables (e.g. those set by docker-compose) take precedence, so this is
	// purely a convenience for local development.
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded (%v); relying on environment variables", err)
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslmode := os.Getenv("SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	frontendURL = os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	frontendURL = strings.TrimRight(frontendURL, "/")
	secureCookies = os.Getenv("SECURE_COOKIES") == "true"
	initSessionSecret()
	allowedOrigins = map[string]bool{
		frontendURL:             true,
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
		"http://localhost:4173": true,
		"http://127.0.0.1:4173": true,
	}
	discordRedirectURL := os.Getenv("DISCORD_REDIRECT_URL")
	if discordRedirectURL == "" {
		discordRedirectURL = "http://localhost:8080/auth/discord/callback"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslmode,
	)
	var errDB error
	db, errDB = sql.Open("postgres", connStr)
	if errDB != nil {
		log.Fatalf("Error opening database connection: %v", errDB)
	}
	// sql.Open does not actually connect; verify the database is reachable at
	// startup (with a short retry, since depends_on does not wait for readiness)
	// so we fail fast instead of on the first request.
	if err := waitForDB(db, 15, 2*time.Second); err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}
	// Apply any pending schema migrations before serving, so the running schema
	// always matches what the code expects.
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	discordOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		RedirectURL:  discordRedirectURL,
		Scopes:       []string{"identify"},
		Endpoint:     discordEndpoint,
	}

	log.Printf("Connected to database %s as user %s on host %s:%s", dbName, dbUser, dbHost, dbPort)

	initializeApp(db)

	http.Handle("/", loggingMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/save-character", loggingMiddleware(http.HandlerFunc(saveCharacterHandler)))
	http.Handle("/add-character-to-encounter", loggingMiddleware(http.HandlerFunc(addCharacterToEncounterHandler)))
	http.Handle("/remove-character-from-encounter", loggingMiddleware(http.HandlerFunc(removeCharacterFromEncounterHandler)))
	http.Handle("/encounters", loggingMiddleware(http.HandlerFunc(apiEncountersHandler)))
	http.Handle("/encounters/save", loggingMiddleware(http.HandlerFunc(apiSaveEncounterHandler)))
	http.Handle("/encounters/delete", loggingMiddleware(http.HandlerFunc(apiDeleteEncounterHandler)))
	http.Handle("/characters", loggingMiddleware(http.HandlerFunc(apiCharactersHandler)))
	http.Handle("/characters/library", loggingMiddleware(http.HandlerFunc(apiLibraryCharactersHandler)))
	http.Handle("/characters/library/save", loggingMiddleware(http.HandlerFunc(apiSaveLibraryCharacterHandler)))
	http.Handle("/characters/library/delete", loggingMiddleware(http.HandlerFunc(apiDeleteLibraryCharacterHandler)))
	http.Handle("/me", loggingMiddleware(http.HandlerFunc(apiMeHandler)))
	http.Handle("/version", loggingMiddleware(http.HandlerFunc(apiVersionHandler)))
	http.Handle("/encounters/combat/start", loggingMiddleware(http.HandlerFunc(apiStartCombatHandler)))
	http.Handle("/encounters/combat/setup", loggingMiddleware(http.HandlerFunc(apiResetCombatHandler)))
	http.Handle("/encounters/combat/next-turn", loggingMiddleware(http.HandlerFunc(apiNextTurnHandler)))
	http.Handle("/encounters/combat/set-active", loggingMiddleware(http.HandlerFunc(apiSetActiveHandler)))
	http.Handle("/encounters/ledger", loggingMiddleware(http.HandlerFunc(apiEncounterLedgerHandler)))
	http.Handle("/encounters/events", loggingMiddleware(http.HandlerFunc(apiEncounterEventsHandler)))
	http.Handle("/encounters/ledger/add", loggingMiddleware(http.HandlerFunc(apiAddEncounterLedgerHandler)))
	// NPC Template API endpoints
	http.Handle("/npcs/templates", loggingMiddleware(http.HandlerFunc(apiNpcTemplatesHandler)))
	http.Handle("/npcs/templates/save", loggingMiddleware(http.HandlerFunc(apiSaveNpcTemplateHandler)))
	http.Handle("/npcs/templates/delete", loggingMiddleware(http.HandlerFunc(apiDeleteNpcTemplateHandler)))
	http.Handle("/npcs/templates/create-character", loggingMiddleware(http.HandlerFunc(apiCreateCharacterFromTemplateHandler)))
	// Friends API
	http.Handle("/friends", loggingMiddleware(http.HandlerFunc(apiFriendsHandler)))
	http.Handle("/friends/requests", loggingMiddleware(http.HandlerFunc(apiFriendRequestsHandler)))
	http.Handle("/friends/request", loggingMiddleware(http.HandlerFunc(apiSendFriendRequestHandler)))
	http.Handle("/friends/accept", loggingMiddleware(http.HandlerFunc(apiAcceptFriendHandler)))
	http.Handle("/friends/decline", loggingMiddleware(http.HandlerFunc(apiRemoveFriendHandler)))
	http.Handle("/friends/remove", loggingMiddleware(http.HandlerFunc(apiRemoveFriendHandler)))
	// Encounter sharing (members)
	http.Handle("/encounters/members", loggingMiddleware(http.HandlerFunc(apiEncounterMembersHandler)))
	http.Handle("/encounters/members/add", loggingMiddleware(http.HandlerFunc(apiAddEncounterMemberHandler)))
	http.Handle("/encounters/members/remove", loggingMiddleware(http.HandlerFunc(apiRemoveEncounterMemberHandler)))

	http.HandleFunc("/login/discord", discordLoginHandler)
	http.HandleFunc("/auth/discord/callback", discordCallbackHandler)
	http.HandleFunc("/logout", logoutHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Explicit timeouts guard against slow-client (Slowloris-style) stalls that
	// the zero-value http.Server leaves wide open. WriteTimeout is generous
	// because the OAuth callback makes outbound calls to Discord.
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Printf("Shutdown signal received; draining connections...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}
	log.Printf("Server stopped")
}

// waitForDB pings the database up to attempts times, sleeping delay between
// tries, so startup survives the database coming up a moment after the app.
func waitForDB(db *sql.DB, attempts int, delay time.Duration) error {
	var err error
	for i := range attempts {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("Database not ready (attempt %d/%d): %v", i+1, attempts, err)
		time.Sleep(delay)
	}
	return err
}

func getDiscordIDFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("discord_id")
	if err != nil {
		return ""
	}
	// The cookie is HMAC-signed at login; a missing or forged signature is
	// treated as logged-out so a caller can never spoof another user's id.
	id, ok := verifyValue(cookie.Value)
	if !ok {
		return ""
	}
	return id
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}
