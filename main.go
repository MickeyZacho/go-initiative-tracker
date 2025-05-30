package main

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-initiative-tracker/dao"
	"html/template"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

var db *sql.DB
var characterDAO dao.CharacterDAO
var characters []dao.Character
var encounterDAO dao.EncounterDAO
var encounters []dao.Encounter
var selectedEncounterID int
var templates *template.Template

func initializeApp(db *sql.DB) {
	characterDAO = dao.NewCharacterDAO(db)
	encounterDAO = dao.NewEncounterDAO(db)
	templates = template.Must(template.ParseFiles("templates/index.html", "templates/character-list.html", "templates/encounter-list.html"))
	loadEncountersFromDB()
	loadCharactersFromDB()
}

func loadEncountersFromDB() {
	var err error
	encounters, err = encounterDAO.GetAllEncounters()
	if err != nil {
		log.Fatalf("Error in loadEncountersFromDB: %v", err)
	}
	if len(encounters) > 0 {
		selectedEncounterID = encounters[0].ID
	}
}

func loadCharactersFromDB() {
	var err error
	log.Printf("Loading characters for encounter ID: %d", selectedEncounterID)
	if selectedEncounterID > 0 {
		characters, err = characterDAO.GetCharactersByEncounterID(selectedEncounterID)
	} else {
		characters, err = characterDAO.GetAllCharacters()
	}
	if err != nil {
		log.Fatalf("Error in loadCharactersFromDB: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	log.Printf("Logging middleware initialized")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")
	sslmode := os.Getenv("SSLMODE")

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", user, password, dbname, sslmode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}

	log.Printf("Connected to database %s as user %s", dbname, user)

	initializeApp(db)

	http.Handle("/", loggingMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/encounters", loggingMiddleware(http.HandlerFunc(encounterListHandler)))
	http.Handle("/select-encounter", loggingMiddleware(http.HandlerFunc(selectEncounterHandler)))
	http.Handle("/characters", loggingMiddleware(http.HandlerFunc(characterListHandler)))
	http.Handle("/next", loggingMiddleware(http.HandlerFunc(nextCharacterHandler)))
	http.Handle("/sort", loggingMiddleware(http.HandlerFunc(sortCharactersHandler)))
	http.Handle("/reorder", loggingMiddleware(http.HandlerFunc(reorderCharactersHandler)))
	http.Handle("/add-character", loggingMiddleware(http.HandlerFunc(addCharacterHandler)))
	http.Handle("/save-character", loggingMiddleware(http.HandlerFunc(saveCharacterHandler)))
	http.Handle("/select-character", loggingMiddleware(http.HandlerFunc(selectCharacterHandler)))
	http.Handle("/search-characters", loggingMiddleware(http.HandlerFunc(searchCharactersHandler)))
	http.Handle("/add-character-to-encounter", loggingMiddleware(http.HandlerFunc(addCharacterToEncounterHandler)))
	http.Handle("/remove-character-from-encounter", loggingMiddleware(http.HandlerFunc(removeCharacterFromEncounterHandler)))

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", characters)
}

func encounterListHandler(w http.ResponseWriter, r *http.Request) {
	type EncounterView struct {
		ID         int
		Name       string
		IsSelected bool
	}
	var viewData []EncounterView
	for _, e := range encounters {
		viewData = append(viewData, EncounterView{
			ID:         e.ID,
			Name:       e.Name,
			IsSelected: e.ID == selectedEncounterID,
		})
	}
	templates.ExecuteTemplate(w, "encounter-list.html", viewData)
}

func characterListHandler(w http.ResponseWriter, r *http.Request) {
	type EditCharacterView struct {
		ID         int
		Name       string
		ArmorClass int
		MaxHP      int
		CurrentHP  int
		Initiative int
		IsActive   bool
		OwnerID    *int
		EditMode   bool
	}
	var tmplData []EditCharacterView
	for _, c := range characters {
		tmplData = append(tmplData, EditCharacterView{
			ID:         c.ID,
			Name:       c.Name,
			ArmorClass: c.ArmorClass,
			MaxHP:      c.MaxHP,
			CurrentHP:  c.CurrentHP,
			Initiative: c.Initiative,
			IsActive:   c.IsActive,
			OwnerID:    c.OwnerID,
			EditMode:   false,
		})
	}
	templates.ExecuteTemplate(w, "character-list.html", tmplData)
}

func nextCharacterHandler(w http.ResponseWriter, r *http.Request) {
	// Find the currently active character
	selectedCharacter := -1
	for i := range characters {
		if characters[i].IsActive {
			selectedCharacter = i
			break
		}
	}
	// If no character is currently active, select the first character
	if selectedCharacter == -1 {
		characters[0].IsActive = true
		selectedCharacter = 0
	} else {
		// Otherwise, find the next character
		characters[selectedCharacter].IsActive = false
		selectedCharacter++
		selectedCharacter %= len(characters)
		characters[selectedCharacter].IsActive = true
	}

	characterListHandler(w, r)
}

func selectCharacterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var selectRequest struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&selectRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i := range characters {
		characters[i].IsActive = characters[i].ID == selectRequest.ID
	}

	characterListHandler(w, r)
}

func sortCharactersHandler(w http.ResponseWriter, r *http.Request) {
	slices.SortFunc(characters, func(a, b dao.Character) int {
		return cmp.Compare(b.Initiative, a.Initiative)
	})

	characterListHandler(w, r)
}

func reorderCharactersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var reorderRequest struct {
		OldIndex int `json:"oldIndex"`
		NewIndex int `json:"newIndex"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reorderRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if reorderRequest.OldIndex < reorderRequest.NewIndex {
		low := reorderRequest.OldIndex
		high := reorderRequest.NewIndex
		for i := low; i < high; i++ {
			characters[i], characters[i+1] = characters[i+1], characters[i]
		}
	} else {
		low := reorderRequest.NewIndex
		high := reorderRequest.OldIndex
		for i := high; i > low; i-- {
			characters[i], characters[i-1] = characters[i-1], characters[i]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func addCharacterHandler(w http.ResponseWriter, r *http.Request) {
	// Render a blank character in edit mode, do not persist to DB yet
	type EditCharacterView struct {
		ID         int
		Name       string
		ArmorClass int
		MaxHP      int
		CurrentHP  int
		Initiative int
		IsActive   bool
		OwnerID    *int
		EditMode   bool
	}
	// Copy existing characters
	var tmplData []EditCharacterView
	for _, c := range characters {
		tmplData = append(tmplData, EditCharacterView{
			ID:         c.ID,
			Name:       c.Name,
			ArmorClass: c.ArmorClass,
			MaxHP:      c.MaxHP,
			CurrentHP:  c.CurrentHP,
			Initiative: c.Initiative,
			IsActive:   c.IsActive,
			OwnerID:    c.OwnerID,
			EditMode:   false,
		})
	}
	// Add the new character in edit mode
	newChar := EditCharacterView{
		ID:       -1, // 0 or -1 to indicate new/unsaved
		EditMode: true,
	}
	tmplData = append(tmplData, newChar)
	templates.ExecuteTemplate(w, "character-list.html", tmplData)
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving character...")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var char dao.Character
	err := json.NewDecoder(r.Body).Decode(&char)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if char.CurrentHP < 0 || char.CurrentHP > char.MaxHP {
		http.Error(w, "Invalid HP value", http.StatusBadRequest)
		return
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
		characters = append(characters, char)
	} else {
		// Update existing character
		for i, c := range characters {
			if c.ID == char.ID {
				characters[i] = char
				break
			}
		}
		err := characterDAO.UpdateCharacter(char)
		if err != nil {
			log.Printf("Error updating character: %v", err)
			http.Error(w, "Failed to update character", http.StatusInternalServerError)
			return
		}
	}

	templates.ExecuteTemplate(w, "character-list.html", []dao.Character{char})
}

func selectEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var selectRequest struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&selectRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	selectedEncounterID = selectRequest.ID
	loadCharactersFromDB()
	characterListHandler(w, r) // This will now render the full character list
}

// Add this handler to search for characters not in the current encounter
func searchCharactersHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	allChars, err := characterDAO.GetAllCharacters()
	if err != nil {
		http.Error(w, "Failed to fetch characters", http.StatusInternalServerError)
		return
	}
	// Get IDs of characters already in the encounter
	encounterChars, err := characterDAO.GetCharactersByEncounterID(selectedEncounterID)
	if err != nil {
		http.Error(w, "Failed to fetch encounter characters", http.StatusInternalServerError)
		return
	}
	encounterCharIDs := make(map[int]bool)
	for _, c := range encounterChars {
		encounterCharIDs[c.ID] = true
	}
	// Filter out characters already in the encounter and by fuzzy, case-insensitive search
	var filtered []dao.Character
	loweredSearch := escapeAndLower(search)
	for _, c := range allChars {
		if !encounterCharIDs[c.ID] && fuzzyMatchFold(c.Name, loweredSearch) {
			filtered = append(filtered, c)
			if len(filtered) >= 10 {
				break
			}
		}
	}
	// Render as a simple HTML list with Add buttons
	w.Header().Set("Content-Type", "text/html")
	for _, c := range filtered {
		fmt.Fprintf(w, `<div>%s <button onclick="addCharacterToEncounter(%d)">Add</button></div>`, template.HTMLEscapeString(c.Name), c.ID)
	}
}

// Fuzzy, case-insensitive substring match (all chars of substr in order in s)
func fuzzyMatchFold(s, substr string) bool {
	s, substr = toLower(s), toLower(substr)
	if substr == "" {
		return true
	}
	si, subi := 0, 0
	for si < len(s) && subi < len(substr) {
		if s[si] == substr[subi] {
			subi++
		}
		si++
	}
	return subi == len(substr)
}

func escape(s string) string {
	return string([]rune(template.HTMLEscapeString(s)))
}
func escapeAndLower(s string) string {
	return strings.ToLower(escape(s))
}

// Handler to add a character to the selected encounter
func addCharacterToEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if selectedEncounterID == 0 {
		http.Error(w, "No encounter selected", http.StatusBadRequest)
		return
	}
	err := encounterDAO.AddCharacterToEncounter(selectedEncounterID, req.CharacterID)
	if err != nil {
		http.Error(w, "Failed to add character to encounter", http.StatusInternalServerError)
		return
	}
	// Reload characters for the encounter
	loadCharactersFromDB()
	characterListHandler(w, r)
}

// Handler to remove a character from the selected encounter
func removeCharacterFromEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if selectedEncounterID == 0 {
		http.Error(w, "No encounter selected", http.StatusBadRequest)
		return
	}
	err := encounterDAO.RemoveCharacterFromEncounter(selectedEncounterID, req.CharacterID)
	if err != nil {
		http.Error(w, "Failed to remove character from encounter", http.StatusInternalServerError)
		return
	}
	// Reload characters for the encounter
	loadCharactersFromDB()
	characterListHandler(w, r)
}
