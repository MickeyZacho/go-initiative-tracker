package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	// Create a mock database
	var err error
	db, mock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	// Mock the query executed in loadCharactersFromDB
	mock.ExpectQuery("SELECT id, name, armor_class, max_hp, current_hp, initiative FROM characters").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "armor_class", "max_hp", "current_hp", "initiative"}).
			AddRow(1, "Test Character", 15, 100, 90, 10))

	// Initialize the app with the mock database
	initializeApp(db)

	// Run tests
	os.Exit(m.Run())
}

func TestIndexHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "Initiative Tracker" // Check for a string in the response
	if !bytes.Contains(rr.Body.Bytes(), []byte(expected)) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestSaveCharacterHandler(t *testing.T) {
	character := map[string]interface{}{
		"id":         1,
		"name":       "Test Character",
		"armorClass": 15,
		"maxHP":      100,
		"currentHP":  90,
		"initiative": 10,
	}
	body, _ := json.Marshal(character)

	mock.ExpectExec("UPDATE characters SET name = \\$1, armor_class = \\$2, max_hp = \\$3, current_hp = \\$4, initiative = \\$5 WHERE id = \\$6").
		WithArgs("Test Character", 15, 100, 90, 10, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, err := http.NewRequest("POST", "/save-character", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(saveCharacterHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestSaveCharacterHandler_InvalidInput(t *testing.T) {
	req, err := http.NewRequest("POST", "/save-character", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	mock.ExpectExec("UPDATE characters SET name = \\$1, armor_class = \\$2, max_hp = \\$3, current_hp = \\$4, initiative = \\$5 WHERE id = \\$6").
		WithArgs("Test Character", 15, 100, 90, 10, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(saveCharacterHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestSelectCharacterHandler(t *testing.T) {
	selectRequest := map[string]int{"id": 1}
	body, _ := json.Marshal(selectRequest)

	req, err := http.NewRequest("POST", "/select-character", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	mock.ExpectExec("UPDATE characters SET is_active = \\$1 WHERE id = \\$2").
		WithArgs(true, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(selectCharacterHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestSortCharactersHandler(t *testing.T) {
	req, err := http.NewRequest("POST", "/sort", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(sortCharactersHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestGetAllCharacters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "armor_class", "max_hp", "current_hp", "initiative"}).
		AddRow(1, "Test Character", 15, 100, 90, 10)
	mock.ExpectQuery("SELECT id, name, armor_class, max_hp, current_hp, initiative FROM characters").
		WillReturnRows(rows)

	dao := NewCharacterDAO(db)
	characters, err := dao.GetAllCharacters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(characters) != 1 || characters[0].Name != "Test Character" {
		t.Errorf("unexpected result: %+v", characters)
	}
}
