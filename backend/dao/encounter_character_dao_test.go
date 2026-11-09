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
	selectOrderedQ = "SELECT ec.character_id, c.type, COALESCE(ec.current_hp, c.max_hp) FROM encounter_characters ec JOIN characters c ON c.id = ec.character_id WHERE ec.encounter_id = $1 ORDER BY COALESCE(ec.initiative, 0) DESC, ec.character_id ASC"
	selectActiveQ  = "SELECT COALESCE(MAX(character_id), 0) FROM encounter_characters WHERE encounter_id = $1 AND is_active = TRUE"
	updateAllOffQ  = "UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1"
	// Activating a character also refreshes its reaction: a creature's reaction
	// comes back at the start of its turn, so every path that makes a character
	// active (start combat, advance turn, click-to-select) clears reaction_used.
	updateOnQ = "UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2"
	// AdvanceTurn ticks the newly-active creature's timed conditions at the start
	// of its turn and clears expired ones; these mirror those two statements.
	tickConditionsQ   = "UPDATE encounter_character_conditions SET duration_rounds = duration_rounds - 1 WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL"
	expireConditionsQ = "DELETE FROM encounter_character_conditions WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL AND duration_rounds <= 0"
	// Round clock on the encounter: start sets 1, a wrap increments, a Next with
	// nobody active ensures at least 1, and reset returns to 0.
	roundStartQ  = "UPDATE encounters SET round = 1 WHERE id = $1"
	roundEnsureQ = "UPDATE encounters SET round = GREATEST(round, 1) WHERE id = $1"
	roundIncQ    = "UPDATE encounters SET round = round + 1 WHERE id = $1"
	roundResetQ  = "UPDATE encounters SET round = 0 WHERE id = $1"
)

func q(s string) string { return regexp.QuoteMeta(s) }

// orderedRows builds turn-order rows for healthy PCs.
func orderedRows(ids ...int) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"character_id", "type", "current_hp"})
	for _, id := range ids {
		rows.AddRow(id, "pc", 10)
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
	mock.ExpectExec(q(roundStartQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
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
	// Wrapping to the top of the order starts a new round.
	mock.ExpectExec(q(roundIncQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
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
	// Nobody was active, so this Next starts combat: round becomes at least 1.
	mock.ExpectExec(q(roundEnsureQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
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

const setReactionQ = "UPDATE encounter_characters SET reaction_used = $1 WHERE encounter_id = $2 AND character_id = $3"

func TestSetReactionUsed_UpdatesFlag(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectExec(q(setReactionQ)).WithArgs(true, 7, 2).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := dao.SetReactionUsed(7, 2, true); err != nil {
		t.Fatalf("SetReactionUsed returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSetReactionUsed_NotInEncounterErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// No matching combatant row -> zero rows affected must surface as an error so a
	// stale character id can't silently no-op.
	mock.ExpectExec(q(setReactionQ)).WithArgs(false, 7, 99).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := dao.SetReactionUsed(7, 99, false); err == nil {
		t.Fatal("expected an error when the character is not in the encounter")
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

	mock.ExpectBegin()
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(q(roundResetQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := dao.ResetCombat(7); err != nil {
		t.Fatalf("ResetCombat returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_SingleCombatantStartsNewRoundEachTurn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// Only one creature: every advance wraps to itself, so every advance is a
	// new round.
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(orderedRows(5))
	mock.ExpectQuery(q(selectActiveQ)).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(5))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(roundIncQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(tickConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q(expireConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if _, err := dao.AdvanceTurn(7); err != nil {
		t.Fatalf("AdvanceTurn returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetRound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectQuery(q("SELECT round FROM encounters WHERE id = $1")).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"round"}).AddRow(3))

	round, err := dao.GetRound(7)
	if err != nil {
		t.Fatalf("GetRound returned error: %v", err)
	}
	if round != 3 {
		t.Errorf("round = %d, want 3", round)
	}
}

func TestAbilityModifier(t *testing.T) {
	cases := map[int]int{1: -5, 7: -2, 8: -1, 9: -1, 10: 0, 11: 0, 12: 1, 14: 2, 15: 2, 20: 5}
	for score, want := range cases {
		if got := AbilityModifier(score); got != want {
			t.Errorf("AbilityModifier(%d) = %d, want %d", score, got, want)
		}
	}
}

func TestAddCharactersFromTemplate_LoneFirstCopyKeepsPlainName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	prevRoll := RollD20
	RollD20 = func() int { return 7 }
	defer func() { RollD20 = prevRoll }()
	dao := NewNpcTemplateDAO(db)

	mock.ExpectQuery("FROM npc_templates WHERE id").WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "base_stats", "armor_class", "max_hp", "owner_id"}).
			AddRow(2, "Ogre", "", "(19,8,16,5,7,7)", 11, 59, ""))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM encounters WHERE id").WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow("dm1"))
	mock.ExpectQuery("SELECT COUNT").WithArgs(3, 2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("INSERT INTO characters").
		WithArgs("Ogre", 11, 0, 59, "dm1", "npc", 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(80))
	// d20 7 + DEX 8 (-1) = 6.
	mock.ExpectExec("INSERT INTO encounter_characters").WithArgs(3, 80, 6, 59).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	created, err := dao.AddCharactersToEncounterFromTemplate(2, 3, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 1 || created[0].Name != "Ogre" || created[0].Initiative != 6 {
		t.Errorf("unexpected result: %+v", created)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdvanceTurn_SkipsDownedNpcsButNotDownedPcs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	// Order [5 active, 2 dead goblin, 8 PC at 0 HP]: skip 2, land on 8 (a
	// downed PC still gets a turn for death saves).
	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(
		sqlmock.NewRows([]string{"character_id", "type", "current_hp"}).
			AddRow(5, "pc", 10).AddRow(2, "npc", 0).AddRow(8, "pc", 0))
	mock.ExpectQuery(q(selectActiveQ)).WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(5))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 8).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(tickConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q(expireConditionsQ)).WithArgs(7, 5).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	next, err := dao.AdvanceTurn(7)
	if err != nil {
		t.Fatalf("AdvanceTurn returned error: %v", err)
	}
	if next != 8 {
		t.Errorf("next character = %d, want 8", next)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestNextTurnIndex(t *testing.T) {
	order := []turnSlot{{id: 1}, {id: 2, skipped: true}, {id: 3}}
	cases := []struct {
		current, want int
		wrapped       bool
	}{
		{-1, 0, false}, // start of combat
		{0, 2, false},  // skip the dead one
		{2, 0, true},   // wrap = new round
	}
	for _, c := range cases {
		got, wrapped := nextTurnIndex(order, c.current)
		if got != c.want || wrapped != c.wrapped {
			t.Errorf("nextTurnIndex(%d) = (%d, %v), want (%d, %v)", c.current, got, wrapped, c.want, c.wrapped)
		}
	}
	// Skipping the dead creature at the end of the order still counts as a wrap.
	tail := []turnSlot{{id: 1}, {id: 2}, {id: 3, skipped: true}}
	if got, wrapped := nextTurnIndex(tail, 1); got != 0 || !wrapped {
		t.Errorf("nextTurnIndex(tail, 1) = (%d, %v), want (0, true)", got, wrapped)
	}
	// Everyone down: still advance one slot.
	allDown := []turnSlot{{id: 1, skipped: true}, {id: 2, skipped: true}}
	if got, wrapped := nextTurnIndex(allDown, 0); got != 1 || wrapped {
		t.Errorf("nextTurnIndex(allDown, 0) = (%d, %v), want (1, false)", got, wrapped)
	}
}

func TestStartCombat_SkipsDownedNpcAtTop(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterCharacterDAO(db)

	mock.ExpectBegin()
	mock.ExpectQuery(q(selectOrderedQ)).WithArgs(7).WillReturnRows(
		sqlmock.NewRows([]string{"character_id", "type", "current_hp"}).
			AddRow(3, "npc", 0).AddRow(4, "pc", 12))
	mock.ExpectExec(q(updateAllOffQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(q(updateOnQ)).WithArgs(7, 4).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(roundStartQ)).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	active, err := dao.StartCombat(7)
	if err != nil {
		t.Fatalf("StartCombat returned error: %v", err)
	}
	if active != 4 {
		t.Errorf("active = %d, want 4", active)
	}
}
