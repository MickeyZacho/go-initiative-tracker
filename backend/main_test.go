package main

import (
	"bytes"
	"encoding/json"
	"go-initiative-tracker/dao"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// mock is the shared sqlmock bound to the package-level db. initializeApp only
// wires up the DAOs now (no bootstrap queries), so tests that need query
// expectations set them on their own mock connection.
var mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	var err error
	db, mock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	// initializeApp no longer loads any server-side state at startup; it just
	// constructs the DAOs against db, so there are no queries to expect here.
	initSessionSecret()
	initializeApp(db)

	os.Exit(m.Run())
}

func TestIndexHandlerRedirects(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	indexHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("indexHandler status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
}

func TestApiMeHandlerLoggedOut(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rr := httptest.NewRecorder()

	apiMeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("apiMeHandler status = %d, want %d", rr.Code, http.StatusOK)
	}
	var body struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.LoggedIn {
		t.Errorf("expected loggedIn=false when no cookies are present")
	}
}

func TestApiMeHandlerWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	rr := httptest.NewRecorder()

	apiMeHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("apiMeHandler status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestSaveCharacterHandlerInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/save-character", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("saveCharacterHandler status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSaveCharacterHandlerRejectsInvalidMaxHP(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"name": "No HP", "maxHP": 0})
	req := httptest.NewRequest(http.MethodPost, "/save-character", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("saveCharacterHandler status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestApiSaveEncounterHandlerRequiresName(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"Name": "   "})
	req := httptest.NewRequest(http.MethodPost, "/encounters/save", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	apiSaveEncounterHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("apiSaveEncounterHandler status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestApiAddEncounterLedgerHandlerRequiresActor(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"encounter_id": 1, "actor_id": 0})
	req := httptest.NewRequest(http.MethodPost, "/encounters/ledger/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	apiAddEncounterLedgerHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("apiAddEncounterLedgerHandler status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestApiStartCombatHandlerWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/encounters/combat/start", nil)
	rr := httptest.NewRecorder()

	apiStartCombatHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("apiStartCombatHandler status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestGetAllCharacters(t *testing.T) {
	mockDB, mockConn, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer mockDB.Close()

	rows := sqlmock.NewRows([]string{
		"id", "name", "armor_class", "to_hit_modifier", "max_hp",
		"current_hp", "initiative", "is_active", "owner_id", "type", "npc_template_id",
	}).AddRow(1, "Test Character", 15, 2, 100, 100, 0, false, "owner1", "pc", nil)
	mockConn.ExpectQuery("SELECT id, name").WillReturnRows(rows)

	characterDAO := dao.NewCharacterDAO(mockDB)
	characters, err := characterDAO.GetAllCharacters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(characters) != 1 || characters[0].Name != "Test Character" {
		t.Errorf("unexpected result: %+v", characters)
	}
	if err := mockConn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetAllEncounters(t *testing.T) {
	mockDB, mockConn, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer mockDB.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "owner_id", "description"}).
		AddRow(1, "Goblin Ambush", "dm1", "A group of goblins attack the party.")
	mockConn.ExpectQuery("SELECT id, name").WillReturnRows(rows)

	encounterDAO := dao.NewEncounterDAO(mockDB)
	encounters, err := encounterDAO.GetAllEncounters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(encounters) != 1 || encounters[0].Name != "Goblin Ambush" {
		t.Errorf("unexpected result: %+v", encounters)
	}
	if err := mockConn.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
