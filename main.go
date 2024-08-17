package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
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
var currentCharacterIndex = -1
var templates *template.Template
var nextID = 1

func init() {
	templates = template.Must(template.ParseFiles("templates/index.html", "templates/character-list.html"))
}

func main() {
	characters = []Character{
		{ID: 1, Name: "Aragorn", ArmorClass: 18, MaxHP: 50, CurrentHP: 45, Initiative: 15, Order: 0},
		{ID: 2, Name: "Gandalf", ArmorClass: 15, MaxHP: 40, CurrentHP: 35, Initiative: 18, Order: 1},
		{ID: 3, Name: "Legolas", ArmorClass: 16, MaxHP: 45, CurrentHP: 40, Initiative: 20, Order: 2},
	}
	nextID++

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
	sort.Slice(characters, func(i, j int) bool {
		return characters[i].Order < characters[j].Order
	})
	templates.ExecuteTemplate(w, "character-list.html", characters)
}

func nextCharacterHandler(w http.ResponseWriter, r *http.Request) {
	currentCharacterIndex = (currentCharacterIndex + 1) % len(characters)
	for i := range characters {
		characters[i].IsActive = (i == currentCharacterIndex)
	}
	characterListHandler(w, r)
}

func sortCharactersHandler(w http.ResponseWriter, r *http.Request) {
	sort.Slice(characters, func(i, j int) bool {
		return characters[i].Initiative > characters[j].Initiative
	})
	for i := range characters {
		characters[i].Order = i
	}
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

	// Perform the reordering
	character := characters[reorderRequest.OldIndex]
	if reorderRequest.NewIndex > reorderRequest.OldIndex {
		for i := reorderRequest.OldIndex; i < reorderRequest.NewIndex; i++ {
			characters[i].Order = characters[i+1].Order
		}
	} else {
		for i := reorderRequest.OldIndex; i > reorderRequest.NewIndex; i-- {
			characters[i].Order = characters[i-1].Order
		}
	}
	character.Order = reorderRequest.NewIndex

	sort.Slice(characters, func(i, j int) bool {
		return characters[i].Order < characters[j].Order
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func addCharacterHandler(w http.ResponseWriter, r *http.Request) {
	newCharacter := Character{
		//ID:    nextID,
		Order: len(characters),
	}
	//nextID++
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
