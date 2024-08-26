package main

import (
	"cmp"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"slices"
)

type Character struct {
	ID         int
	Name       string
	ArmorClass int
	MaxHP      int
	CurrentHP  int
	Initiative int
	IsActive   bool
	Order      int
}

var characters []Character
var templates *template.Template

func init() {
	templates = template.Must(template.ParseFiles("templates/index.html", "templates/character-list.html"))
}

func main() {
	characters = []Character{
		{ID: 1, Name: "Aragorn", ArmorClass: 18, MaxHP: 50, CurrentHP: 45, Initiative: 15, Order: 0},
		{ID: 2, Name: "Gandalf", ArmorClass: 15, MaxHP: 40, CurrentHP: 35, Initiative: 18, Order: 1},
		{ID: 3, Name: "Legolas", ArmorClass: 16, MaxHP: 45, CurrentHP: 40, Initiative: 20, Order: 2},
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/characters", characterListHandler)
	http.HandleFunc("/next", nextCharacterHandler)
	http.HandleFunc("/sort", sortCharactersHandler)
	http.HandleFunc("/reorder", reorderCharactersHandler)
	http.HandleFunc("/add-character", addCharacterHandler)
	http.HandleFunc("/save-character", saveCharacterHandler)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", characters)
}

func characterListHandler(w http.ResponseWriter, r *http.Request) {
	// sort.Slice(characters, func(i, j int) bool {
	// 	return characters[i].Order < characters[j].Order
	// })
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
	newCharacter := Character{
		//ID:    nextID,
		//Order: len(characters),
	}
	//characters = append(characters, newCharacter)
	templates.ExecuteTemplate(w, "character-list.html", []Character{newCharacter})
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	var char Character
	err := json.NewDecoder(r.Body).Decode(&char)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	is_new_character := true
	for _, c := range characters {
		if c.ID == char.ID {
			c = char
			is_new_character = false
			break
		}
	}
	if is_new_character {
		characters = append(characters, char)
	}

	templates.ExecuteTemplate(w, "character-list.html", []Character{char})
}
