package main

import (
	"go-initiative-tracker/dao"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// newConditionMock swaps encounterDAO and encounterConditionDAO for DAOs backed
// by a shared sqlmock connection so an ownership check and a follow-on condition
// write line up in order.
func newConditionMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, m, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	prevEnc, prevCond := encounterDAO, encounterConditionDAO
	encounterDAO = dao.NewEncounterDAO(mockDB)
	encounterConditionDAO = dao.NewEncounterConditionDAO(mockDB)
	return m, func() {
		encounterDAO, encounterConditionDAO = prevEnc, prevCond
		mockDB.Close()
	}
}

func TestAddConditionAllowsOwner(t *testing.T) {
	m, restore := newConditionMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("INSERT INTO encounter_character_conditions").
		WithArgs(1, 5, "Poisoned", 3, "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr, req := postJSON("/encounters/conditions/add",
		`{"encounter_id":1,"character_id":5,"condition":"Poisoned","duration_rounds":3}`)
	authed(req, "dm1")
	apiAddConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestAddConditionRejectsUnknownCondition(t *testing.T) {
	// Validation happens before any DB access, so no mock expectations are set.
	m, restore := newConditionMock(t)
	defer restore()

	rr, req := postJSON("/encounters/conditions/add",
		`{"encounter_id":1,"character_id":5,"condition":"Sleepy"}`)
	authed(req, "dm1")
	apiAddConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusBadRequest)
	assertMet(t, m)
}

func TestAddConditionRejectsNonPositiveDuration(t *testing.T) {
	m, restore := newConditionMock(t)
	defer restore()

	rr, req := postJSON("/encounters/conditions/add",
		`{"encounter_id":1,"character_id":5,"condition":"Poisoned","duration_rounds":0}`)
	authed(req, "dm1")
	apiAddConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusBadRequest)
	assertMet(t, m)
}

func TestAddConditionRejectsNonOwner(t *testing.T) {
	m, restore := newConditionMock(t)
	defer restore()
	// Encounter owned by dm1; caller is logged out, so access is denied and no
	// condition is written.
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))

	rr, req := postJSON("/encounters/conditions/add",
		`{"encounter_id":1,"character_id":5,"condition":"Poisoned"}`)
	apiAddConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusForbidden)
	assertMet(t, m)
}

func TestRemoveConditionAllowsOwner(t *testing.T) {
	m, restore := newConditionMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("DELETE FROM encounter_character_conditions").
		WithArgs(11, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	rr, req := postJSON("/encounters/conditions/remove",
		`{"encounter_id":1,"condition_id":11}`)
	authed(req, "dm1")
	apiRemoveConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestRemoveConditionNotFound(t *testing.T) {
	m, restore := newConditionMock(t)
	defer restore()
	m.ExpectQuery("FROM encounters WHERE id").WithArgs(1).
		WillReturnRows(encounterRow(1, "dm1"))
	m.ExpectExec("DELETE FROM encounter_character_conditions").
		WithArgs(99, 1).WillReturnResult(sqlmock.NewResult(0, 0))

	rr, req := postJSON("/encounters/conditions/remove",
		`{"encounter_id":1,"condition_id":99}`)
	authed(req, "dm1")
	apiRemoveConditionHandler(rr, req)

	assertStatus(t, rr, http.StatusNotFound)
	assertMet(t, m)
}

func TestConditionCatalogReturnsList(t *testing.T) {
	rr, req := getReq("/encounters/conditions/catalog")
	apiConditionCatalogHandler(rr, req)
	assertStatus(t, rr, http.StatusOK)
	if len(dao.ValidConditions) == 0 {
		t.Fatal("expected a non-empty condition catalog")
	}
}
