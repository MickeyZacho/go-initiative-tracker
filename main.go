package main

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"slices"

	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

var db *sql.DB
var characterDAO CharacterDAO
var characters []Character
var templates *template.Template

func init() {
	var err error
	connStr := "user=postgres password=12Taller dbname=postgres sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	characterDAO = NewCharacterDAO(db)

	templates = template.Must(template.ParseFiles("templates/index.html", "templates/character-list.html"))

	loadCharactersFromDB()
}

func loadCharactersFromDB() {
	var err error
	characters, err = characterDAO.GetAll()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/characters", characterListHandler)
	http.HandleFunc("/next", nextCharacterHandler)
	http.HandleFunc("/sort", sortCharactersHandler)
	http.HandleFunc("/reorder", reorderCharactersHandler)
	http.HandleFunc("/add-character", addCharacterHandler)
	http.HandleFunc("/save-character", saveCharacterHandler)
	http.HandleFunc("/select-character", selectCharacterHandler)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", characters)
}

func characterListHandler(w http.ResponseWriter, r *http.Request) {
	// fresh_character_list, err := characterDAO.GetAll()
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// characters = fresh_character_list
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
		ID: nextID,
	}
	characters = append(characters, newCharacter)
	characterListHandler(w, r)
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	var char Character
	err := json.NewDecoder(r.Body).Decode(&char)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for i, c := range characters {
		if c.ID == char.ID {
			characters[i] = char
			break
		}
	}

	templates.ExecuteTemplate(w, "character-list.html", []Character{char})
}
