package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-initiative-tracker/dao"
	"log"
	"net/http"
	"os"
	"strconv"
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

// fetchCharacters loads characters for the given request, scoped to encounterID
// when it is > 0 and to the signed-in Discord user when a cookie is present.
// It holds no server-side state, so concurrent requests never see each other's
// selection.
func fetchCharacters(r *http.Request, encounterID int) ([]dao.Character, error) {
	discordID := getDiscordIDFromRequest(r)
	if encounterID > 0 {
		if discordID != "" {
			return characterDAO.GetCharactersByEncounterIDAndOwner(encounterID, discordID)
		}
		return characterDAO.GetCharactersByEncounterID(encounterID)
	}
	if discordID != "" {
		return characterDAO.GetAllCharactersByOwner(discordID)
	}
	return characterDAO.GetAllCharacters()
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

func apiEncountersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	var data []dao.Encounter
	var err error
	if discordID != "" {
		data, err = encounterDAO.GetEncountersByOwnerDiscordID(discordID)
	} else {
		data, err = encounterDAO.GetAllEncounters()
	}
	if err != nil {
		http.Error(w, "Failed to fetch encounters", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = []dao.Encounter{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func apiSaveEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var enc dao.Encounter
	if err := json.NewDecoder(r.Body).Decode(&enc); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(enc.Name) == "" {
		http.Error(w, "Encounter name is required", http.StatusBadRequest)
		return
	}
	if enc.OwnerID == "" {
		enc.OwnerID = getDiscordIDFromRequest(r)
	}
	newID, err := encounterDAO.CreateEncounter(enc)
	if err != nil {
		http.Error(w, "Failed to create encounter", http.StatusInternalServerError)
		return
	}
	enc.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "encounter": enc})
}

func apiDeleteEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.ID <= 0 {
		http.Error(w, "Invalid encounter id", http.StatusBadRequest)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	deleted, err := encounterDAO.DeleteEncounterByOwner(req.ID, discordID)
	if err != nil {
		http.Error(w, "Failed to delete encounter", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Encounter not found or not owned by you", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiCharactersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	encounterID := 0
	if encounterIDRaw := r.URL.Query().Get("encounter_id"); encounterIDRaw != "" {
		id, err := strconv.Atoi(encounterIDRaw)
		if err != nil || id <= 0 {
			http.Error(w, "Invalid encounter id", http.StatusBadRequest)
			return
		}
		encounterID = id
	}
	characters, err := fetchCharacters(r, encounterID)
	if err != nil {
		http.Error(w, "Failed to fetch characters", http.StatusInternalServerError)
		return
	}
	if characters == nil {
		characters = []dao.Character{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(characters)
}

func apiLibraryCharactersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	var data []dao.Character
	var err error
	if discordID != "" {
		data, err = characterDAO.GetAllCharactersByOwner(discordID)
	} else {
		data, err = characterDAO.GetAllCharacters()
	}
	if err != nil {
		http.Error(w, "Failed to fetch characters", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = []dao.Character{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func apiSaveLibraryCharacterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var char dao.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(char.Name) == "" {
		http.Error(w, "Character name is required", http.StatusBadRequest)
		return
	}
	if char.MaxHP < 1 {
		http.Error(w, "Invalid max HP value", http.StatusBadRequest)
		return
	}
	if char.OwnerID == "" {
		char.OwnerID = getDiscordIDFromRequest(r)
	}
	if char.ID == 0 {
		newID, err := characterDAO.CreateCharacter(char)
		if err != nil {
			http.Error(w, "Failed to create character", http.StatusInternalServerError)
			return
		}
		char.ID = newID
	} else {
		err := characterDAO.UpdateCharacter(char)
		if err != nil {
			http.Error(w, "Failed to update character", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": char})
}

func apiDeleteLibraryCharacterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.ID <= 0 {
		http.Error(w, "Invalid character id", http.StatusBadRequest)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	deleted, err := characterDAO.DeleteCharacterByOwner(req.ID, discordID)
	if err != nil {
		http.Error(w, "Failed to delete character", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Character not found or not owned by you", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiMeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	username := ""
	discordID := ""
	avatar := ""
	if cookie, err := r.Cookie("discord_user"); err == nil {
		username = cookie.Value
	}
	if cookie, err := r.Cookie("discord_id"); err == nil {
		discordID = cookie.Value
	}
	if cookie, err := r.Cookie("discord_avatar"); err == nil {
		avatar = cookie.Value
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"loggedIn":  username != "" && discordID != "",
		"username":  username,
		"discordID": discordID,
		"avatar":    avatar,
	})
}

func encounterIDFromRequest(r *http.Request) (int, error) {
	var req struct {
		EncounterID int `json:"encounter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return 0, err
	}
	if req.EncounterID <= 0 {
		return 0, fmt.Errorf("invalid encounter id")
	}
	return req.EncounterID, nil
}

func apiStartCombatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}

	activeCharacterID, err := encounterCharacterDAO.StartCombat(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Encounter has no characters"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to start combat"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}

func apiResetCombatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}

	if err := encounterCharacterDAO.ResetCombat(encounterID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to reset combat"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiNextTurnHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}

	activeCharacterID, err := encounterCharacterDAO.AdvanceTurn(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Encounter has no characters"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to advance turn"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}

func apiEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterIDRaw := r.URL.Query().Get("encounter_id")
	encounterID, err := strconv.Atoi(encounterIDRaw)
	if err != nil || encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid encounter id"})
		return
	}

	entries, err := encounterLedgerDAO.ListByEncounterID(encounterID, 50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to load encounter log"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entries": entries})
}

func apiAddEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	var req struct {
		EncounterID int    `json:"encounter_id"`
		ActorID     int    `json:"actor_id"`
		TargetID    int    `json:"target_id"`
		ActionType  string `json:"action_type"`
		HPChange    int    `json:"hp_change"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.EncounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid encounter id"})
		return
	}
	if req.ActorID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Actor is required"})
		return
	}
	actionType := strings.TrimSpace(req.ActionType)
	if actionType == "" {
		actionType = "note"
	}

	entry, err := encounterLedgerDAO.Create(dao.EncounterLedgerInsert{
		EncounterID: req.EncounterID,
		ActorID:     req.ActorID,
		TargetID:    req.TargetID,
		ActionType:  actionType,
		HPChange:    req.HPChange,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to add combat log entry"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entry": entry})
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving character...")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		dao.Character
		EncounterID int `json:"encounter_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	char := payload.Character

	if char.MaxHP < 1 {
		http.Error(w, "Invalid max HP value", http.StatusBadRequest)
		return
	}
	if char.CurrentHP < 0 {
		char.CurrentHP = 0
	}
	if char.CurrentHP > char.MaxHP {
		char.CurrentHP = char.MaxHP
	}
	if char.OwnerID == "" {
		char.OwnerID = getDiscordIDFromRequest(r)
	}

	if char.ID == 0 {
		// New character, insert into DB
		newID, err := characterDAO.CreateCharacter(char)
		if err != nil {
			log.Printf("Error creating character: %v", err)
			http.Error(w, "Failed to create character", http.StatusInternalServerError)
			return
		}
		char.ID = newID
	} else {
		// Update existing character
		err := characterDAO.UpdateCharacter(char)
		if err != nil {
			log.Printf("Error updating character: %v", err)
			http.Error(w, "Failed to update character", http.StatusInternalServerError)
			return
		}
	}

	if payload.EncounterID > 0 {
		encChar := dao.EncounterCharacter{
			EncounterID: payload.EncounterID,
			CharacterID: char.ID,
			Initiative:  char.Initiative,
			CurrentHP:   char.CurrentHP,
			IsActive:    char.IsActive,
		}
		err = encounterCharacterDAO.Upsert(encChar)
		if err != nil {
			log.Printf("Error upserting encounter character: %v", err)
			http.Error(w, "Failed to save encounter values", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "success",
		"character": char,
	})
}

func generateState() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "state"
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Handler to add a character to the selected encounter
func addCharacterToEncounterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}
	var req struct {
		EncounterID int `json:"encounter_id"`
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.CharacterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid character id"})
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "encounter_id is required"})
		return
	}
	err := encounterDAO.AddCharacterToEncounter(encounterID, req.CharacterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Character is already in this encounter"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to add character to encounter"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiNpcTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	templates, err := npcTemplateDAO.GetAll()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to fetch npc templates", http.StatusInternalServerError)
		return
	}
	if templates == nil {
		templates = []dao.NpcTemplate{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// POST /api/npcs/templates/save
func apiSaveNpcTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var nt dao.NpcTemplate
	if err := json.NewDecoder(r.Body).Decode(&nt); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(nt.Name) == "" {
		http.Error(w, "NPC name is required", http.StatusBadRequest)
		return
	}

	if nt.ID > 0 {
		// Update
		err := npcTemplateDAO.Update(nt)
		if err != nil {
			http.Error(w, "Failed to update npc template", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "npc": nt})
		return
	}

	// Create new template
	newID, err := npcTemplateDAO.Create(nt)
	if err != nil {
		fmt.Println(nt)
		fmt.Println(err)
		http.Error(w, "Failed to create npc template", http.StatusInternalServerError)
		return
	}
	nt.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "npc": nt})
}

// POST /api/npcs/templates/delete
func apiDeleteNpcTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.ID <= 0 {
		http.Error(w, "Invalid npc template id", http.StatusBadRequest)
		return
	}
	if err := npcTemplateDAO.Delete(req.ID); err != nil {
		http.Error(w, "Failed to delete npc template", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// POST /api/npcs/templates/create-character
func apiCreateCharacterFromTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TemplateID  int `json:"npc_template_id"`
		EncounterID int `json:"encounter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.TemplateID <= 0 {
		http.Error(w, "Invalid template id", http.StatusBadRequest)
		return
	}
	if req.EncounterID <= 0 {
		http.Error(w, "No encounter selected", http.StatusBadRequest)
		return
	}

	character, err := npcTemplateDAO.AddCharacterToEncounterFromTemplate(req.TemplateID, req.EncounterID)
	if err != nil {
		http.Error(w, "Failed to create character from template", http.StatusInternalServerError)
		return
	}
	// Optionally, insert the character into the characters table here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": character})
}

// Handler to remove a character from the selected encounter
func removeCharacterFromEncounterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}
	var req struct {
		EncounterID int `json:"encounter_id"`
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.CharacterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid character id"})
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "encounter_id is required"})
		return
	}
	err := encounterDAO.RemoveCharacterFromEncounter(encounterID, req.CharacterID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to remove character from encounter"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Redirects user to Discord's OAuth2 login page
func discordLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	// Store state in a cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
	})
	url := discordOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	log.Printf("Redirecting to Discord OAuth URL: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Handles Discord's callback and retrieves user info
func discordCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	log.Printf("[OAUTH DEBUG] code: %s, state: %s", code, state)
	if code == "" {
		http.Error(w, "No code in request", http.StatusBadRequest)
		return
	}
	// Validate state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		log.Printf("[OAUTH DEBUG] Invalid state. Cookie: %v, Query: %v", cookie, state)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	log.Printf("[OAUTH DEBUG] Exchanging token with redirect_uri: %s", discordOAuthConfig.RedirectURL)
	token, err := discordOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("[OAUTH DEBUG] Token exchange error: %v", err)
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	client := discordOAuthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var userInfo struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
		Avatar        string `json:"avatar"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_user",
		Value:    userInfo.Username + "#" + userInfo.Discriminator,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_id",
		Value:    userInfo.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_avatar",
		Value:    userInfo.Avatar,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	// Save user to database
	user := dao.User{
		DiscordID:     userInfo.ID,
		Username:      userInfo.Username,
		Discriminator: userInfo.Discriminator,
		Avatar:        userInfo.Avatar,
	}
	err = dao.NewUserDAO(db).UpsertUser(user)
	if err != nil {
		http.Error(w, "Failed to save user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_user",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_avatar",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}
