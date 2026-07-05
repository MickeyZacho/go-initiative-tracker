package main

import (
	"database/sql"
	"fmt"
	"go-initiative-tracker/dao"
	"log"
	"net/http"
	"os"
	"strings"
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
	http.Handle("/encounters/combat/start", loggingMiddleware(http.HandlerFunc(apiStartCombatHandler)))
	http.Handle("/encounters/combat/setup", loggingMiddleware(http.HandlerFunc(apiResetCombatHandler)))
	http.Handle("/encounters/combat/next-turn", loggingMiddleware(http.HandlerFunc(apiNextTurnHandler)))
	http.Handle("/encounters/ledger", loggingMiddleware(http.HandlerFunc(apiEncounterLedgerHandler)))
	http.Handle("/encounters/ledger/add", loggingMiddleware(http.HandlerFunc(apiAddEncounterLedgerHandler)))
	// NPC Template API endpoints
	http.Handle("/npcs/templates", loggingMiddleware(http.HandlerFunc(apiNpcTemplatesHandler)))
	http.Handle("/npcs/templates/save", loggingMiddleware(http.HandlerFunc(apiSaveNpcTemplateHandler)))
	http.Handle("/npcs/templates/delete", loggingMiddleware(http.HandlerFunc(apiDeleteNpcTemplateHandler)))
	http.Handle("/npcs/templates/create-character", loggingMiddleware(http.HandlerFunc(apiCreateCharacterFromTemplateHandler)))

	http.HandleFunc("/login/discord", discordLoginHandler)
	http.HandleFunc("/auth/discord/callback", discordCallbackHandler)
	http.HandleFunc("/logout", logoutHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getDiscordIDFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("discord_id"); err == nil {
		log.Printf("Found discord_id cookie: %s", cookie.Value)
		return cookie.Value
	}
	return ""
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}
