package dao

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConditionAdd_UpsertsWithDuration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterConditionDAO(db)

	rounds := 3
	mock.ExpectExec(q("INSERT INTO encounter_character_conditions")).
		WithArgs(7, 2, "Poisoned", rounds, "from a dagger").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = dao.Add(Condition{
		EncounterID:    7,
		CharacterID:    2,
		Condition:      "Poisoned",
		DurationRounds: &rounds,
		Note:           "from a dagger",
	})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestConditionAdd_UntilRemovedPassesNilDuration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterConditionDAO(db)

	mock.ExpectExec(q("INSERT INTO encounter_character_conditions")).
		WithArgs(7, 2, "Prone", nil, "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := dao.Add(Condition{EncounterID: 7, CharacterID: 2, Condition: "Prone"}); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestConditionRemove_ScopedToEncounter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterConditionDAO(db)

	mock.ExpectExec(q("DELETE FROM encounter_character_conditions WHERE id = $1 AND encounter_id = $2")).
		WithArgs(11, 7).WillReturnResult(sqlmock.NewResult(0, 1))

	removed, err := dao.Remove(11, 7)
	if err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if !removed {
		t.Errorf("removed = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestConditionRemove_NoMatchReturnsFalse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterConditionDAO(db)

	mock.ExpectExec(q("DELETE FROM encounter_character_conditions")).
		WithArgs(99, 7).WillReturnResult(sqlmock.NewResult(0, 0))

	removed, err := dao.Remove(99, 7)
	if err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if removed {
		t.Errorf("removed = true, want false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestIsValidCondition(t *testing.T) {
	if !IsValidCondition("Stunned") {
		t.Errorf("Stunned should be valid")
	}
	if IsValidCondition("Sleepy") {
		t.Errorf("Sleepy should not be valid")
	}
	if IsValidCondition("") {
		t.Errorf("empty string should not be valid")
	}
}
