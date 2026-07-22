package main

import (
	"database/sql"
	"encoding/json"
	"go-initiative-tracker/dao"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/oauth2"
)

// newFullMock swaps every package-level DAO for one backed by a single shared
// sqlmock connection and returns the mock plus a restore func. Sharing one
// connection lets ordered expectations that span multiple DAOs (e.g. an access
// check on encounterDAO followed by a write on characterDAO) line up.
func newFullMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	prevChar, prevEnc, prevEC, prevLedger, prevNpc, prevFriend, prevUser :=
		characterDAO, encounterDAO, encounterCharacterDAO, encounterLedgerDAO, npcTemplateDAO, friendshipDAO, userDAO
	characterDAO = dao.NewCharacterDAO(mockDB)
	encounterDAO = dao.NewEncounterDAO(mockDB)
	encounterCharacterDAO = dao.NewEncounterCharacterDAO(mockDB)
	encounterLedgerDAO = dao.NewEncounterLedgerDAO(mockDB)
	npcTemplateDAO = dao.NewNpcTemplateDAO(mockDB)
	friendshipDAO = dao.NewFriendshipDAO(mockDB)
	userDAO = dao.NewUserDAO(mockDB)
	return m, func() {
		characterDAO, encounterDAO, encounterCharacterDAO, encounterLedgerDAO, npcTemplateDAO, friendshipDAO, userDAO =
			prevChar, prevEnc, prevEC, prevLedger, prevNpc, prevFriend, prevUser
		mockDB.Close()
	}
}

// authed attaches the signed discord_id cookie so the request is treated as
// coming from the given logged-in user.
func authed(req *http.Request, discordID string) *http.Request {
	req.AddCookie(&http.Cookie{Name: "discord_id", Value: signValue(discordID)})
	return req
}

func getReq(path string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil)
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Errorf("status = %d, want %d (body: %s)", rr.Code, want, rr.Body.String())
	}
}

func assertMet(t *testing.T, m sqlmock.Sqlmock) {
	t.Helper()
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// --- Method guards -----------------------------------------------------------

// Every handler rejects the wrong HTTP method before touching state, so this
// table locks that contract in one place across the whole surface.
func TestHandlersRejectWrongMethod(t *testing.T) {
	cases := []struct {
		name   string
		h      http.HandlerFunc
		method string
	}{
		{"save-character", saveCharacterHandler, http.MethodGet},
		{"add-character", addCharacterToEncounterHandler, http.MethodGet},
		{"remove-character", removeCharacterFromEncounterHandler, http.MethodGet},
		{"encounters", apiEncountersHandler, http.MethodPost},
		{"encounters/save", apiSaveEncounterHandler, http.MethodGet},
		{"encounters/delete", apiDeleteEncounterHandler, http.MethodGet},
		{"characters", apiCharactersHandler, http.MethodPost},
		{"characters/library", apiLibraryCharactersHandler, http.MethodPost},
		{"characters/library/save", apiSaveLibraryCharacterHandler, http.MethodGet},
		{"characters/library/delete", apiDeleteLibraryCharacterHandler, http.MethodGet},
		{"combat/start", apiStartCombatHandler, http.MethodGet},
		{"combat/setup", apiResetCombatHandler, http.MethodGet},
		{"combat/next-turn", apiNextTurnHandler, http.MethodGet},
		{"combat/set-active", apiSetActiveHandler, http.MethodGet},
		{"ledger", apiEncounterLedgerHandler, http.MethodPost},
		{"ledger/add", apiAddEncounterLedgerHandler, http.MethodGet},
		{"events", apiEncounterEventsHandler, http.MethodPost},
		{"npcs/templates", apiNpcTemplatesHandler, http.MethodPost},
		{"npcs/templates/save", apiSaveNpcTemplateHandler, http.MethodGet},
		{"npcs/templates/delete", apiDeleteNpcTemplateHandler, http.MethodGet},
		{"npcs/templates/create-character", apiCreateCharacterFromTemplateHandler, http.MethodGet},
		{"friends", apiFriendsHandler, http.MethodPost},
		{"friends/requests", apiFriendRequestsHandler, http.MethodPost},
		{"friends/request", apiSendFriendRequestHandler, http.MethodGet},
		{"friends/accept", apiAcceptFriendHandler, http.MethodGet},
		{"friends/remove", apiRemoveFriendHandler, http.MethodGet},
		{"members", apiEncounterMembersHandler, http.MethodPost},
		{"members/add", apiAddEncounterMemberHandler, http.MethodGet},
		{"members/remove", apiRemoveEncounterMemberHandler, http.MethodGet},
		{"version", apiVersionHandler, http.MethodPost},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/"+tc.name, nil)
			rr := httptest.NewRecorder()
			tc.h(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// --- Payload validation (no DB) ---------------------------------------------

func TestHandlersRejectInvalidJSON(t *testing.T) {
	cases := []struct {
		name string
		h    http.HandlerFunc
		path string
	}{
		{"save-character", saveCharacterHandler, "/save-character"},
		{"add-character", addCharacterToEncounterHandler, "/add-character-to-encounter"},
		{"remove-character", removeCharacterFromEncounterHandler, "/remove-character-from-encounter"},
		{"encounters/save", apiSaveEncounterHandler, "/encounters/save"},
		{"encounters/delete", apiDeleteEncounterHandler, "/encounters/delete"},
		{"library/save", apiSaveLibraryCharacterHandler, "/characters/library/save"},
		{"library/delete", apiDeleteLibraryCharacterHandler, "/characters/library/delete"},
		{"combat/start", apiStartCombatHandler, "/encounters/combat/start"},
		{"combat/set-active", apiSetActiveHandler, "/encounters/combat/set-active"},
		{"ledger/add", apiAddEncounterLedgerHandler, "/encounters/ledger/add"},
		{"npcs/save", apiSaveNpcTemplateHandler, "/npcs/templates/save"},
		{"npcs/delete", apiDeleteNpcTemplateHandler, "/npcs/templates/delete"},
		{"npcs/create-character", apiCreateCharacterFromTemplateHandler, "/npcs/templates/create-character"},
		{"members/add", apiAddEncounterMemberHandler, "/encounters/members/add"},
		{"members/remove", apiRemoveEncounterMemberHandler, "/encounters/members/remove"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr, req := postJSON(tc.path, "not json")
			// Some handlers require a login before decoding; give them one so the
			// failure they surface is specifically the bad payload (400).
			authed(req, "u1")
			tc.h(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestSetActiveRejectsNonPositiveIDs(t *testing.T) {
	rr, req := postJSON("/encounters/combat/set-active", `{"encounter_id":1,"character_id":0}`)
	apiSetActiveHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestNextTurnRejectsMissingEncounterID(t *testing.T) {
	rr, req := postJSON("/encounters/combat/next-turn", `{"encounter_id":0}`)
	apiNextTurnHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestAddCharacterRequiresCharacterID(t *testing.T) {
	rr, req := postJSON("/add-character-to-encounter", `{"encounter_id":1,"character_id":0}`)
	addCharacterToEncounterHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestAddCharacterRequiresEncounterID(t *testing.T) {
	rr, req := postJSON("/add-character-to-encounter", `{"encounter_id":0,"character_id":5}`)
	addCharacterToEncounterHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestSaveLibraryCharacterRequiresName(t *testing.T) {
	rr, req := postJSON("/characters/library/save", `{"Name":"  ","MaxHP":10}`)
	apiSaveLibraryCharacterHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestSaveLibraryCharacterRejectsInvalidMaxHP(t *testing.T) {
	rr, req := postJSON("/characters/library/save", `{"Name":"Rogue","MaxHP":0}`)
	apiSaveLibraryCharacterHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestSaveNpcTemplateRequiresName(t *testing.T) {
	rr, req := postJSON("/npcs/templates/save", `{"Name":"  "}`)
	apiSaveNpcTemplateHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestEncounterLedgerRejectsBadEncounterID(t *testing.T) {
	rr, req := getReq("/encounters/ledger?encounter_id=abc")
	apiEncounterLedgerHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestEncounterMembersRejectsBadEncounterID(t *testing.T) {
	rr, req := getReq("/encounters/members?encounter_id=0")
	apiEncounterMembersHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestCreateCharacterFromTemplateValidation(t *testing.T) {
	rr, req := postJSON("/npcs/templates/create-character", `{"npc_template_id":0,"encounter_id":1}`)
	apiCreateCharacterFromTemplateHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)

	rr, req = postJSON("/npcs/templates/create-character", `{"npc_template_id":2,"encounter_id":0}`)
	apiCreateCharacterFromTemplateHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

// --- Auth requirements (no DB) ----------------------------------------------

func TestHandlersRequireLogin(t *testing.T) {
	cases := []struct {
		name string
		h    http.HandlerFunc
		path string
		body string
	}{
		{"friends", apiFriendsHandler, "/friends", ""},
		{"friends/requests", apiFriendRequestsHandler, "/friends/requests", ""},
		{"friends/accept", apiAcceptFriendHandler, "/friends/accept", `{"discord_id":"x"}`},
		{"friends/remove", apiRemoveFriendHandler, "/friends/remove", `{"discord_id":"x"}`},
		{"encounters/delete", apiDeleteEncounterHandler, "/encounters/delete", `{"id":1}`},
		{"library/delete", apiDeleteLibraryCharacterHandler, "/characters/library/delete", `{"id":1}`},
		{"members/add", apiAddEncounterMemberHandler, "/encounters/members/add", `{"encounter_id":1,"user_id":"f1"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rr *httptest.ResponseRecorder
			var req *http.Request
			if tc.body == "" {
				rr, req = getReq(tc.path)
			} else {
				rr, req = postJSON(tc.path, tc.body)
			}
			tc.h(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestFriendActionsRejectEmptyDiscordID(t *testing.T) {
	rr, req := postJSON("/friends/accept", `{"discord_id":"  "}`)
	authed(req, "u1")
	apiAcceptFriendHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)

	rr, req = postJSON("/friends/remove", `{"discord_id":""}`)
	authed(req, "u1")
	apiRemoveFriendHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestSendFriendRequestRejectsEmptyUsername(t *testing.T) {
	rr, req := postJSON("/friends/request", `{"username":"   "}`)
	authed(req, "u1")
	apiSendFriendRequestHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

// --- Combat happy paths & access --------------------------------------------

func TestSetActiveRejectsNonOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := postJSON("/encounters/combat/set-active", `{"encounter_id":1,"character_id":7}`)
	apiSetActiveHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestSetActiveMissingEncounterIs404(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(9).WillReturnError(sql.ErrNoRows)

	rr, req := postJSON("/encounters/combat/set-active", `{"encounter_id":9,"character_id":7}`)
	apiSetActiveHandler(rr, req)

	assertStatus(t, rr, http.StatusNotFound)
	assertMet(t, m)
}

func TestSetActiveAllowsOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectBegin()
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 3))
	m.ExpectExec("UPDATE encounter_characters SET is_active = TRUE").WithArgs(1, 7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectCommit()

	rr, req := postJSON("/encounters/combat/set-active", `{"encounter_id":1,"character_id":7}`)
	authed(req, "dm1")
	apiSetActiveHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	var body struct {
		Status  string `json:"status"`
		ActiveW int    `json:"active_character_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "success" || body.ActiveW != 7 {
		t.Errorf("unexpected body: %+v", body)
	}
	assertMet(t, m)
}

func TestSetActiveCharacterNotInEncounterIs404(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectBegin()
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 3))
	// The target isn't in the encounter, so the second UPDATE affects no rows.
	m.ExpectExec("UPDATE encounter_characters SET is_active = TRUE").WithArgs(1, 999).
		WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectRollback()

	rr, req := postJSON("/encounters/combat/set-active", `{"encounter_id":1,"character_id":999}`)
	authed(req, "dm1")
	apiSetActiveHandler(rr, req)

	assertStatus(t, rr, http.StatusNotFound)
	assertMet(t, m)
}

func TestNextTurnAllowsOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectBegin()
	m.ExpectQuery("SELECT character_id FROM encounter_characters").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"character_id"}).AddRow(5).AddRow(2).AddRow(8))
	m.ExpectQuery("AND is_active = TRUE").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(5))
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 3))
	m.ExpectExec("UPDATE encounter_characters SET is_active = TRUE").WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectExec("UPDATE encounter_character_conditions SET duration_rounds").WithArgs(1, 5).
		WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectExec("DELETE FROM encounter_character_conditions").WithArgs(1, 5).
		WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectCommit()

	rr, req := postJSON("/encounters/combat/next-turn", `{"encounter_id":1}`)
	authed(req, "dm1")
	apiNextTurnHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestResetCombatAllowsOwner(t *testing.T) {
	m, restore := newEncounterMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("UPDATE encounter_characters SET is_active = FALSE").WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 4))

	rr, req := postJSON("/encounters/combat/setup", `{"encounter_id":1}`)
	authed(req, "dm1")
	apiResetCombatHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

// --- Encounters CRUD --------------------------------------------------------

func TestEncountersListLoggedOutReturnsOnlyUnowned(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	// Logged-out visitors only see owner-less encounters.
	m.ExpectQuery("owner_id IS NULL OR owner_id = ''").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "owner_id", "description"}).
			AddRow(1, "Goblins", "", "ambush"),
	)

	rr, req := getReq("/encounters")
	apiEncountersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveEncounterCreates(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("INSERT INTO encounters").WithArgs("Goblins", "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))

	rr, req := postJSON("/encounters/save", `{"Name":"Goblins"}`)
	apiSaveEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestDeleteEncounterOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("DELETE FROM encounters").WithArgs(1, "dm1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/encounters/delete", `{"id":1}`)
	authed(req, "dm1")
	apiDeleteEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestDeleteEncounterNonOwnerForbidden(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("DELETE FROM encounters").WithArgs(1, "someone").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/encounters/delete", `{"id":1}`)
	authed(req, "someone")
	apiDeleteEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

// --- Characters -------------------------------------------------------------

func TestCharactersListLoggedOut(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("SELECT id, name, armor_class").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "name", "armor_class", "to_hit_modifier", "max_hp",
			"current_hp", "initiative", "is_active", "owner_id", "type", "npc_template_id",
		}),
	)

	rr, req := getReq("/characters")
	apiCharactersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestCharactersListBadEncounterID(t *testing.T) {
	rr, req := getReq("/characters?encounter_id=-1")
	apiCharactersHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

func TestAddCharacterAllowsOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("INSERT INTO encounter_characters").WithArgs(1, 5).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/add-character-to-encounter", `{"encounter_id":1,"character_id":5}`)
	authed(req, "dm1")
	addCharacterToEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAddCharacterDuplicateConflict(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("INSERT INTO encounter_characters").WithArgs(1, 5).
		WillReturnError(errString("duplicate key value violates unique constraint"))

	rr, req := postJSON("/add-character-to-encounter", `{"encounter_id":1,"character_id":5}`)
	authed(req, "dm1")
	addCharacterToEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusConflict)
	assertMet(t, m)
}

func TestRemoveCharacterAllowsOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("DELETE FROM encounter_characters").WithArgs(1, 5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/remove-character-from-encounter", `{"encounter_id":1,"character_id":5}`)
	authed(req, "dm1")
	removeCharacterFromEncounterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveLibraryCharacterCreates(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("INSERT INTO characters").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))

	rr, req := postJSON("/characters/library/save", `{"Name":"Rogue","MaxHP":8}`)
	authed(req, "u1")
	apiSaveLibraryCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveLibraryCharacterLoggedOutRejected(t *testing.T) {
	_, restore := newFullMock(t)
	defer restore()

	rr, req := postJSON("/characters/library/save", `{"Name":"Rogue","MaxHP":8}`)
	apiSaveLibraryCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusUnauthorized)
}

func TestSaveCharacterLoggedOutRejected(t *testing.T) {
	_, restore := newFullMock(t)
	defer restore()

	rr, req := postJSON("/save-character", `{"Name":"Rogue","MaxHP":8}`)
	saveCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusUnauthorized)
}

func TestDeleteLibraryCharacterOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("DELETE FROM characters").WithArgs(5, "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/characters/library/delete", `{"id":5}`)
	authed(req, "u1")
	apiDeleteLibraryCharacterHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestLibraryCharactersLoggedInScopesToOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("WHERE c.owner_id").WithArgs("u1").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "name", "armor_class", "to_hit_modifier", "max_hp",
			"current_hp", "initiative", "is_active", "owner_id", "type",
		}),
	)

	rr, req := getReq("/characters/library")
	authed(req, "u1")
	apiLibraryCharactersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

// --- Ledger -----------------------------------------------------------------

func TestEncounterLedgerReadAllowsOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("FROM encounter_ledger").WithArgs(1, 50).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "encounter_id", "actor_id", "actor_name", "target_id",
			"target_name", "action_type", "hp_change", "description", "created_at",
		}),
	)

	rr, req := getReq("/encounters/ledger?encounter_id=1")
	authed(req, "dm1")
	apiEncounterLedgerHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAddLedgerEntryAllowsOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("INSERT INTO encounter_ledger").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "encounter_id", "actor_id", "actor_name", "target_id",
			"target_name", "action_type", "hp_change", "description", "created_at",
		}).AddRow(1, 1, 5, "Hero", 0, "", "attack", -3, "", "2026-07-18T00:00:00Z"),
	)

	rr, req := postJSON("/encounters/ledger/add", `{"encounter_id":1,"actor_id":5,"action_type":"attack","hp_change":-3}`)
	authed(req, "dm1")
	apiAddEncounterLedgerHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAddLedgerEntryRejectsBadEncounterID(t *testing.T) {
	rr, req := postJSON("/encounters/ledger/add", `{"encounter_id":0,"actor_id":5}`)
	apiAddEncounterLedgerHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

// --- NPC templates ----------------------------------------------------------

func TestNpcTemplatesList(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM npc_templates").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "base_stats", "armor_class", "max_hp", "owner_id"}),
	)

	rr, req := getReq("/npcs/templates")
	apiNpcTemplatesHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveNpcTemplateCreates(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("INSERT INTO npc_templates").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

	rr, req := postJSON("/npcs/templates/save", `{"Name":"Orc","MaxHP":10,"ArmorClass":13}`)
	apiSaveNpcTemplateHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveNpcTemplateUpdatesOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("UPDATE npc_templates SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/npcs/templates/save", `{"id":4,"Name":"Orc","MaxHP":10}`)
	authed(req, "u1")
	apiSaveNpcTemplateHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSaveNpcTemplateUpdateNonOwnerForbidden(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	// Updating a template the caller doesn't own matches no rows -> 403.
	m.ExpectExec("UPDATE npc_templates SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/npcs/templates/save", `{"id":4,"Name":"Orc","MaxHP":10}`)
	authed(req, "intruder")
	apiSaveNpcTemplateHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestCreateCharacterFromTemplateAllowsOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	// Access check on the encounter.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	// Template lookup (valid composite stat_block so parsing succeeds).
	m.ExpectQuery("FROM npc_templates WHERE id").WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "base_stats", "armor_class", "max_hp", "owner_id"}).
			AddRow(2, "Goblin", "", "(8,14,10,10,8,8)", 13, 7, "dm1"))
	// Owner is resolved from the encounter again inside the template flow.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("INSERT INTO characters").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(50))
	m.ExpectExec("INSERT INTO encounter_characters").WithArgs(1, 50).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/npcs/templates/create-character", `{"npc_template_id":2,"encounter_id":1}`)
	authed(req, "dm1")
	apiCreateCharacterFromTemplateHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

// --- Friends ----------------------------------------------------------------

func TestFriendsListLoggedIn(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("other.discord_id").WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "avatar"}))

	rr, req := getReq("/friends")
	authed(req, "u1")
	apiFriendsHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestFriendRequestsListLoggedIn(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("f.requester_id").WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "avatar"}))
	m.ExpectQuery("f.addressee_id").WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "avatar"}))

	rr, req := getReq("/friends/requests")
	authed(req, "u1")
	apiFriendRequestsHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestSendFriendRequestUnknownUserIs404(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM users WHERE username").WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	rr, req := postJSON("/friends/request", `{"username":"ghost"}`)
	authed(req, "u1")
	apiSendFriendRequestHandler(rr, req)

	assertStatus(t, rr, http.StatusNotFound)
	assertMet(t, m)
}

func TestSendFriendRequestToSelfIsRejected(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM users WHERE username").WithArgs("me").
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "discriminator", "avatar"}).
			AddRow("u1", "me", "0", ""))

	rr, req := postJSON("/friends/request", `{"username":"me"}`)
	authed(req, "u1")
	apiSendFriendRequestHandler(rr, req)

	assertStatus(t, rr, http.StatusBadRequest)
	assertMet(t, m)
}

func TestSendFriendRequestSucceeds(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM users WHERE username").WithArgs("friend").
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "discriminator", "avatar"}).
			AddRow("u2", "friend", "0", ""))
	m.ExpectQuery("FROM friendships").WithArgs("u1", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	m.ExpectExec("INSERT INTO friendships").WithArgs("u1", "u2").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/friends/request", `{"username":"friend"}`)
	authed(req, "u1")
	apiSendFriendRequestHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAcceptFriendSucceeds(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("UPDATE friendships SET status = 'accepted'").WithArgs("u2", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/friends/accept", `{"discord_id":"u2"}`)
	authed(req, "u1")
	apiAcceptFriendHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAcceptFriendNoPendingIs404(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("UPDATE friendships SET status = 'accepted'").WithArgs("u2", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/friends/accept", `{"discord_id":"u2"}`)
	authed(req, "u1")
	apiAcceptFriendHandler(rr, req)

	assertStatus(t, rr, http.StatusNotFound)
	assertMet(t, m)
}

func TestRemoveFriendSucceeds(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("DELETE FROM friendships").WithArgs("u1", "u2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/friends/remove", `{"discord_id":"u2"}`)
	authed(req, "u1")
	apiRemoveFriendHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

// --- Encounter members ------------------------------------------------------

func TestEncounterMembersListOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("JOIN users u").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"discord_id", "username", "avatar"}))

	rr, req := getReq("/encounters/members?encounter_id=1")
	authed(req, "dm1")
	apiEncounterMembersHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestEncounterMembersListRejectsNonOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := getReq("/encounters/members?encounter_id=1")
	authed(req, "intruder")
	apiEncounterMembersHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestAddEncounterMemberRequiresFriendship(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("status = 'accepted'").WithArgs("dm1", "stranger").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rr, req := postJSON("/encounters/members/add", `{"encounter_id":1,"user_id":"stranger"}`)
	authed(req, "dm1")
	apiAddEncounterMemberHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestAddEncounterMemberSucceeds(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectQuery("status = 'accepted'").WithArgs("dm1", "friend1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	m.ExpectExec("INSERT INTO encounter_users").WithArgs(1, "friend1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/encounters/members/add", `{"encounter_id":1,"user_id":"friend1"}`)
	authed(req, "dm1")
	apiAddEncounterMemberHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestRemoveEncounterMemberOwner(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("DELETE FROM encounter_users").WithArgs(1, "friend1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/encounters/members/remove", `{"encounter_id":1,"user_id":"friend1"}`)
	authed(req, "dm1")
	apiRemoveEncounterMemberHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

// --- Auth (login / logout) --------------------------------------------------

func TestLogoutClearsCookies(t *testing.T) {
	rr, req := getReq("/logout")
	logoutHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	cleared := map[string]bool{}
	for _, c := range rr.Result().Cookies() {
		if c.MaxAge < 0 && c.Value == "" {
			cleared[c.Name] = true
		}
	}
	for _, name := range []string{"discord_user", "discord_id", "discord_avatar"} {
		if !cleared[name] {
			t.Errorf("expected cookie %q to be cleared", name)
		}
	}
}

func TestDiscordLoginRedirectsWithStateCookie(t *testing.T) {
	prev := discordOAuthConfig
	discordOAuthConfig = &oauth2.Config{
		ClientID:    "test-client",
		RedirectURL: "http://localhost/auth/discord/callback",
		Scopes:      []string{"identify"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
	defer func() { discordOAuthConfig = prev }()

	rr, req := getReq("/login/discord")
	discordLoginHandler(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}
	var stateCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil || stateCookie.Value == "" {
		t.Fatal("expected a non-empty oauth_state cookie")
	}
	// The redirect target must carry the same state for CSRF protection.
	loc, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatalf("redirect location is not a valid URL: %v", err)
	}
	if got := loc.Query().Get("state"); got != stateCookie.Value {
		t.Errorf("redirect state = %q, want %q", got, stateCookie.Value)
	}
}

// --- Events -----------------------------------------------------------------

func TestEncounterEventsRejectsBadEncounterID(t *testing.T) {
	rr, req := getReq("/encounters/events?encounter_id=0")
	apiEncounterEventsHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}

// errString is a tiny error type so tests can simulate driver errors whose
// message the handler inspects (e.g. the unique-violation path).
type errString string

func (e errString) Error() string { return string(e) }
