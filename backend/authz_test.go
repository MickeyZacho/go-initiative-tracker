package main

import (
	"database/sql"
	"go-initiative-tracker/dao"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// newEncounterMock swaps encounterDAO (and encounterCharacterDAO) for DAOs backed
// by a fresh sqlmock connection, returning the mock and a restore func. Both DAOs
// share the connection so ordered expectations across an ownership check and a
// follow-on combat mutation line up.
func newEncounterMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	prevEnc, prevEC := encounterDAO, encounterCharacterDAO
	encounterDAO = dao.NewEncounterDAO(mockDB)
	encounterCharacterDAO = dao.NewEncounterCharacterDAO(mockDB)
	return m, func() {
		encounterDAO, encounterCharacterDAO = prevEnc, prevEC
		mockDB.Close()
	}
}

func encounterRow(id int, owner string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "name", "owner_id", "description"}).
		AddRow(id, "Test Encounter", owner, "")
}

func postJSON(path, body string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), req
}

func TestStartCombatRejectsNonOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()

	// Encounter is owned by "dm1"; the caller sends no cookie (logged out).
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := postJSON("/encounters/combat/start", `{"encounter_id":1}`)
	apiStartCombatHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestStartCombatMissingEncounterIs404(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()

	m.ExpectQuery("FROM encounters WHERE id").WithArgs(99).
		WillReturnError(sql.ErrNoRows)

	rr, req := postJSON("/encounters/combat/start", `{"encounter_id":99}`)
	apiStartCombatHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestStartCombatAllowsOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()

	// Ownership check passes (owner matches the signed cookie), then the combat
	// transaction runs to completion.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectBegin()
	m.ExpectQuery("SELECT character_id FROM encounter_characters").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"character_id"}).AddRow(7))
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectExec("UPDATE encounter_characters SET is_active = TRUE").WithArgs(1, 7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectCommit()

	rr, req := postJSON("/encounters/combat/start", `{"encounter_id":1}`)
	req.AddCookie(&http.Cookie{Name: "discord_id", Value: signValue("dm1")})
	apiStartCombatHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestEncounterLedgerReadRejectsNonOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()

	// Reading another user's combat log must be denied, not just mutating it.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	req := httptest.NewRequest(http.MethodGet, "/encounters/ledger?encounter_id=1", nil)
	rr := httptest.NewRecorder()
	apiEncounterLedgerHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAddCharacterToEncounterRejectsNonOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()

	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := postJSON("/add-character-to-encounter", `{"encounter_id":1,"character_id":5}`)
	addCharacterToEncounterHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSaveCharacterUpdateRejectsNonOwner(t *testing.T) {
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer mockDB.Close()
	prev := characterDAO
	characterDAO = dao.NewCharacterDAO(mockDB)
	defer func() { characterDAO = prev }()

	// Updating a character the caller does not own affects zero rows -> 403.
	m.ExpectExec("UPDATE characters SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/save-character", `{"id":5,"name":"Rogue","maxHP":10}`)
	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// newEncounterAndCharacterMock swaps encounterDAO, encounterCharacterDAO and
// characterDAO for DAOs sharing one mock connection, so ordered expectations
// across an access check and a follow-on character write line up.
func newEncounterAndCharacterMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	prevEnc, prevEC, prevChar := encounterDAO, encounterCharacterDAO, characterDAO
	encounterDAO = dao.NewEncounterDAO(mockDB)
	encounterCharacterDAO = dao.NewEncounterCharacterDAO(mockDB)
	characterDAO = dao.NewCharacterDAO(mockDB)
	return m, func() {
		encounterDAO, encounterCharacterDAO, characterDAO = prevEnc, prevEC, prevChar
		mockDB.Close()
	}
}

// A shared-edit member may edit a character they do not own, as long as it is in
// an encounter they have access to.
func TestSaveCharacterInEncounterAllowsMemberEditingOthersCharacter(t *testing.T) {
	m, restore := newEncounterAndCharacterMock(t)
	defer restore()

	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("FROM encounter_users").WithArgs(1, "player2").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// The character is owned by dm1, yet the member's update lands.
	m.ExpectQuery("UPDATE characters c SET").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow("dm1"))
	m.ExpectExec("INSERT INTO encounter_characters").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/save-character", `{"id":5,"name":"Rogue","maxHP":10,"encounter_id":1}`)
	req.AddCookie(&http.Cookie{Name: "discord_id", Value: signValue("player2")})
	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// Encounter access is still required: a caller with neither ownership nor
// membership cannot edit through the encounter.
func TestSaveCharacterInEncounterRejectsNonMember(t *testing.T) {
	m, restore := newEncounterAndCharacterMock(t)
	defer restore()

	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("FROM encounter_users").WithArgs(1, "stranger").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rr, req := postJSON("/save-character", `{"id":5,"name":"Rogue","maxHP":10,"encounter_id":1}`)
	req.AddCookie(&http.Cookie{Name: "discord_id", Value: signValue("stranger")})
	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// A character that is not in the encounter cannot be edited through it, even by
// a member — otherwise encounter access would be a lever on the whole table.
func TestSaveCharacterInEncounterRejectsCharacterOutsideEncounter(t *testing.T) {
	m, restore := newEncounterAndCharacterMock(t)
	defer restore()

	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "player2"))
	m.ExpectQuery("UPDATE characters c SET").WillReturnError(sql.ErrNoRows)

	rr, req := postJSON("/save-character", `{"id":99,"name":"Rogue","maxHP":10,"encounter_id":1}`)
	req.AddCookie(&http.Cookie{Name: "discord_id", Value: signValue("player2")})
	saveCharacterHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestDeleteNpcTemplateRejectsNonOwner(t *testing.T) {
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer mockDB.Close()
	prev := npcTemplateDAO
	npcTemplateDAO = dao.NewNpcTemplateDAO(mockDB)
	defer func() { npcTemplateDAO = prev }()

	// Deleting a template owned by someone else matches no rows -> 403.
	m.ExpectExec("DELETE FROM npc_templates").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/npcs/templates/delete", `{"id":1}`)
	apiDeleteNpcTemplateHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
