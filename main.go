package main

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

var db *sql.DB
var characterDAO CharacterDAO
var characters []Character
var templates *template.Template

func initializeApp(db *sql.DB) {
	characterDAO = NewCharacterDAO(db)
	templates = template.Must(template.ParseFiles("templates/index.html", "templates/character-list.html"))
	loadCharactersFromDB()
}

func loadCharactersFromDB() {
	var err error
	characters, err = characterDAO.GetAllCharacters()
	if err != nil {
		log.Fatal(err)
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
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")
	sslmode := os.Getenv("SSLMODE")

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", user, password, dbname, sslmode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	initializeApp(db)

	http.Handle("/", loggingMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/characters", loggingMiddleware(http.HandlerFunc(characterListHandler)))
	http.Handle("/next", loggingMiddleware(http.HandlerFunc(nextCharacterHandler)))
	http.Handle("/sort", loggingMiddleware(http.HandlerFunc(sortCharactersHandler)))
	http.Handle("/reorder", loggingMiddleware(http.HandlerFunc(reorderCharactersHandler)))
	http.Handle("/add-character", loggingMiddleware(http.HandlerFunc(addCharacterHandler)))
	http.Handle("/save-character", loggingMiddleware(http.HandlerFunc(saveCharacterHandler)))
	http.Handle("/select-character", loggingMiddleware(http.HandlerFunc(selectCharacterHandler)))

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", characters)
}

func characterListHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "character-list.html", characters)
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
	slices.SortFunc(characters, func(a, b Character) int {
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
	nextID := len(characters) + 1
	newCharacter := Character{
		ID:        nextID,
		CurrentHP: 0,
	}
	characters = append(characters, newCharacter)
	err := characterDAO.CreateCharacter(newCharacter) // Persist to the database
	if err != nil {
		log.Printf("Error creating character: %v", err)
		http.Error(w, "Failed to create character", http.StatusInternalServerError)
		return
	}
	characterListHandler(w, r)
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving character...")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var char Character
	err := json.NewDecoder(r.Body).Decode(&char)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if char.CurrentHP < 0 || char.CurrentHP > char.MaxHP {
		http.Error(w, "Invalid HP value", http.StatusBadRequest)
		return
	}

	log.Printf("Received character: %+v", char)
	for i, c := range characters {
		if c.ID == char.ID {
			// Update the character locally
			characters[i] = char
			// Update the character in the database
			err := characterDAO.UpdateCharacter(char)
			if err != nil {
				log.Printf("Error updating character: %v", err)
				http.Error(w, "Failed to update character", http.StatusInternalServerError)
				return
			}
			break
		}
	}

	templates.ExecuteTemplate(w, "character-list.html", []Character{char})
}
