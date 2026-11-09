package main

import (
	"database/sql"
	"encoding/json"
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
	// Nobody is scheduled to arrive in round 1.
	m.ExpectQuery("AND ec.hidden AND").WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"character_id", "initiative", "dex"}))
	m.ExpectQuery("SELECT ec.character_id, c.type").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"character_id", "type", "current_hp"}).AddRow(7, "pc", 10))
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectExec("UPDATE encounter_characters SET is_active = TRUE").WithArgs(1, 7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectExec("UPDATE encounters SET round = 1").WithArgs(1).
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
	// Members may only edit PCs, so the character's type is checked first.
	m.ExpectQuery("FROM characters WHERE id").WithArgs(5).
		WillReturnRows(characterRow(5, "Rogue", "dm1", "pc"))
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

// characterRow builds a GetCharacterByID result row.
func characterRow(id int, name, owner, typ string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "armor_class", "to_hit_modifier", "max_hp",
		"current_hp", "initiative", "is_active", "owner_id", "type", "npc_template_id",
	}).AddRow(id, name, 12, 3, 10, 10, 0, false, owner, typ, nil)
}

// expectMember queues the access check for a member (not the owner).
func expectMember(m sqlmock.Sqlmock, encounterID int, owner, member string) {
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(encounterID).
		WillReturnRows(encounterRow(encounterID, owner))
	m.ExpectQuery("FROM encounter_users").WithArgs(encounterID, member).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
}

func TestSaveCharacterInEncounterRejectsMemberEditingNpc(t *testing.T) {
	m, restore := newEncounterAndCharacterMock(t)
	defer restore()
	expectMember(m, 1, "dm1", "player2")
	m.ExpectQuery("FROM characters WHERE id").WithArgs(9).
		WillReturnRows(characterRow(9, "Goblin", "dm1", "npc"))

	rr, req := postJSON("/save-character", `{"id":9,"name":"Goblin","maxHP":7,"currentHP":0,"encounter_id":1}`)
	authed(req, "player2")
	saveCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestSaveCharacterInEncounterRejectsMemberCreating(t *testing.T) {
	m, restore := newEncounterAndCharacterMock(t)
	defer restore()
	expectMember(m, 1, "dm1", "player2")

	rr, req := postJSON("/save-character", `{"id":0,"name":"Dragon","maxHP":300,"encounter_id":1}`)
	authed(req, "player2")
	saveCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestCharactersListRedactsNpcsForMember(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	expectMember(m, 1, "dm1", "player2")
	m.ExpectQuery("FROM characters c JOIN encounter_characters").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "armor_class", "to_hit_modifier", "max_hp", "current_hp",
			"initiative", "is_active", "owner_id", "type", "npc_template_id", "reaction_used", "temp_hp",
			"death_save_successes", "death_save_failures", "hidden", "reveal_round",
		}).
			AddRow(1, "Aria", 16, 5, 30, 30, 18, false, "player2", "pc", nil, false, 0, 0, 0, false, nil).
			AddRow(2, "Goblin 1", 15, 4, 8, 3, 12, false, "dm1", "npc", 4, false, 0, 0, 0, false, nil).
			// Staged by the DM: players must not see it at all.
			AddRow(3, "Ambusher", 14, 5, 20, 20, 0, false, "dm1", "npc", nil, false, 0, 0, 0, true, 3))
	m.ExpectQuery("FROM encounter_character_conditions").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "encounter_id", "character_id", "condition", "duration_rounds", "level", "note"}))

	rr, req := getReq("/characters?encounter_id=1")
	authed(req, "player2")
	apiCharactersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
	var chars []dao.Character
	if err := json.Unmarshal(rr.Body.Bytes(), &chars); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("got %d characters, want 2 (hidden combatant must be dropped)", len(chars))
	}
	pc, npc := chars[0], chars[1]
	if pc.Redacted || pc.ArmorClass != 16 || pc.CurrentHP != 30 {
		t.Errorf("PC should be untouched, got %+v", pc)
	}
	if !npc.Redacted || npc.ArmorClass != 0 || npc.CurrentHP != 0 || npc.MaxHP != 0 || npc.ToHitModifier != 0 {
		t.Errorf("NPC numbers should be hidden, got %+v", npc)
	}
	if npc.HealthStatus != "Bloodied" {
		t.Errorf("HealthStatus = %q, want Bloodied (3/8)", npc.HealthStatus)
	}
}

func TestCharactersListOwnerSeesNpcStats(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("FROM characters c JOIN encounter_characters").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "armor_class", "to_hit_modifier", "max_hp", "current_hp",
			"initiative", "is_active", "owner_id", "type", "npc_template_id", "reaction_used", "temp_hp",
			"death_save_successes", "death_save_failures", "hidden", "reveal_round",
		}).AddRow(2, "Goblin 1", 15, 4, 8, 3, 12, false, "dm1", "npc", 4, false, 0, 0, 0, true, nil))
	m.ExpectQuery("FROM encounter_character_conditions").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "encounter_id", "character_id", "condition", "duration_rounds", "level", "note"}))

	rr, req := getReq("/characters?encounter_id=1")
	authed(req, "dm1")
	apiCharactersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	var chars []dao.Character
	if err := json.Unmarshal(rr.Body.Bytes(), &chars); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(chars) != 1 || chars[0].Redacted || chars[0].ArmorClass != 15 || !chars[0].Hidden {
		t.Errorf("owner should see full NPC stats, got %+v", chars)
	}
}

func TestEncounterLedgerRedactsNpcHPForMember(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	expectMember(m, 1, "dm1", "player2")
	m.ExpectQuery("FROM encounter_ledger").WithArgs(1, 50).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "encounter_id", "actor_id", "actor_name", "actor_type", "target_id",
			"target_name", "action_type", "hp_change", "description", "created_at", "actor_hidden",
		}).
			AddRow(1, 1, 2, "Goblin 1", "npc", 0, "", "damage", -5, "", "2026-01-01T00:00:00Z", false).
			AddRow(3, 1, 9, "Ambusher", "npc", 0, "", "damage", -2, "", "2026-01-01T00:00:00Z", true).
			AddRow(2, 1, 1, "Aria", "pc", 0, "", "damage", -4, "", "2026-01-01T00:00:01Z", false),
	)

	rr, req := getReq("/encounters/ledger?encounter_id=1")
	authed(req, "player2")
	apiEncounterLedgerHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	var body struct {
		Entries []dao.EncounterLedgerEntry `json:"entries"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Entries) != 2 || body.Entries[0].HPChange != 0 || body.Entries[1].HPChange != -4 {
		t.Errorf("NPC hp_change should be hidden, PC kept; got %+v", body.Entries)
	}
}

func TestStartCombatRejectsMember(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	// Owner-only: a member is refused without even checking membership.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := postJSON("/encounters/combat/start", `{"encounter_id":1}`)
	authed(req, "player2")
	apiStartCombatHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestHealthStatus(t *testing.T) {
	cases := []struct {
		cur, max int
		want     string
	}{
		{10, 10, "Healthy"}, {9, 10, "Wounded"}, {6, 10, "Wounded"},
		{5, 10, "Bloodied"}, {1, 10, "Bloodied"}, {0, 10, "Down"},
	}
	for _, c := range cases {
		if got := healthStatus(c.cur, c.max); got != c.want {
			t.Errorf("healthStatus(%d, %d) = %q, want %q", c.cur, c.max, got, c.want)
		}
	}
}
