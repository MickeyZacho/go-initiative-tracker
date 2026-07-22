package dao

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// The DAO relies on SQL to order combatants; these constants mirror the exact
// statements so the tests lock in the ordering contract and argument wiring.
const (
	selectOrderedQ = "SELECT character_id FROM encounter_characters WHERE encounter_id = $1 ORDER BY COALESCE(initiative, 0) DESC, character_id ASC"
	selectActiveQ  = "SELECT COALESCE(MAX(character_id), 0) FROM encounter_characters WHERE encounter_id = $1 AND is_active = TRUE"
	updateAllOffQ  = "UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1"
	updateOnQ      = "UPDATE encounter_characters SET is_active = TRUE WHERE encounter_id = $1 AND character_id = $2"
	// AdvanceTurn ticks the newly-active creature's timed conditions at the start
	// of its turn and clears expired ones; these mirror those two statements.
	tickConditionsQ   = "UPDATE encounter_character_conditions SET duration_rounds = duration_rounds - 1 WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL"
	expireConditionsQ = "DELETE FROM encounter_character_conditions WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL AND duration_rounds <= 0"
)

func q(s string) string { return regexp.QuoteMeta(s) }

func orderedRows(ids ...int) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"character_id"})
	for _, id := range ids {
		rows.AddRow(id)
	}
	return rows
}

func TestStartCombat_ActivatesTopOfInitiativeOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// Rows arrive already sorted by the SELECT (initiative DESC, id ASC). The
	// query expectation includes that ORDER BY, so the ordering contract is
	// asserted here; the DAO must activate the first row (character 5).
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5, 2, 8))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	active, err := dao.StartCombat(7)
	if err != nil {
		t.Fatalf("StartCombat returned error: %v", err)
	}
	if active != 5 {
		t.Errorf("active character = %d, want 5", active)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestStartCombat_NoCharactersErrorsAndRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows())
	mock.ExpectRollback()

	if _, err := dao.StartCombat(7); err == nil {
		t.Fatal("expected an error when the encounter has no characters")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestStartCombat_RollsBackWhenUpdateFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5, 2))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnError(errors.New("boom"))
	mock.ExpectRollback()

	if _, err := dao.StartCombat(7); err == nil {
		t.Fatal("expected StartCombat to surface the update error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_MovesToNextCombatant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// Order [5,2,8], currently active 5 -> next is 2.
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5, 2, 8))
	mock.ExpectQuery(q(selectActiveQ)).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(5))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 2).WillReturnResult(sqlmock.NewResult(0, 1))
	// Conditions tick on the outgoing creature (5), whose turn is ending.
	mock.ExpectExec(q(tickConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q(expireConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	next, err := dao.AdvanceTurn(7)
	if err != nil {
		t.Fatalf("AdvanceTurn returned error: %v", err)
	}
	if next != 2 {
		t.Errorf("next character = %d, want 2", next)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_WrapsFromLastToFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// Order [5,2,8], currently active 8 (last) -> wraps to 5.
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5, 2, 8))
	mock.ExpectQuery(q(selectActiveQ)).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(8))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 1))
	// Conditions tick on the outgoing creature (8), whose turn is ending.
	mock.ExpectExec(q(tickConditionsQ)).WithArgs(7, 8).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q(expireConditionsQ)).WithArgs(7, 8).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	next, err := dao.AdvanceTurn(7)
	if err != nil {
		t.Fatalf("AdvanceTurn returned error: %v", err)
	}
	if next != 5 {
		t.Errorf("next character = %d, want 5 (wrap-around)", next)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_NoActiveStartsAtFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// No one active yet (MAX returns 0) -> first combatant becomes active. With no
	// outgoing creature, no conditions are ticked, so none are expected here.
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5, 2, 8))
	mock.ExpectQuery(q(selectActiveQ)).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	next, err := dao.AdvanceTurn(7)
	if err != nil {
		t.Fatalf("AdvanceTurn returned error: %v", err)
	}
	if next != 5 {
		t.Errorf("next character = %d, want 5", next)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_NoCharactersErrorsAndRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows())
	mock.ExpectRollback()

	if _, err := dao.AdvanceTurn(7); err == nil {
		t.Fatal("expected an error when the encounter has no characters")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSetActiveCharacter_ActivatesOneClearsRest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 2).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.SetActiveCharacter(7, 2); err != nil {
		t.Fatalf("SetActiveCharacter returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSetActiveCharacter_NotInEncounterErrorsAndRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// The target isn't a combatant, so the activating UPDATE hits no rows and the
	// transaction must roll back rather than leave the encounter with no active.
	mock.ExpectBegin()
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 99).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	if err := dao.SetActiveCharacter(7, 99); err == nil {
		t.Fatal("expected an error when the character is not in the encounter")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSetActiveCharacter_RollsBackWhenClearFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnError(errors.New("boom"))
	mock.ExpectRollback()

	if err := dao.SetActiveCharacter(7, 2); err == nil {
		t.Fatal("expected SetActiveCharacter to surface the update error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestResetCombat_DeactivatesEveryone(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 4))

	if err := dao.ResetCombat(7); err != nil {
		t.Fatalf("ResetCombat returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
