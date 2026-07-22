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
		WithArgs(7, 2, "Poisoned", rounds, nil, "from a dagger").
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
		WithArgs(7, 2, "Prone", nil, nil, "").
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

// Exhaustion upserts on the unique key, so raising a creature from level 2 to 3
// is an UPDATE of the same row rather than a second exhaustion row.
func TestConditionAdd_ExhaustionCarriesLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewEncounterConditionDAO(db)

	level := 3
	mock.ExpectExec(q("INSERT INTO encounter_character_conditions")).
		WithArgs(7, 2, "Exhaustion", nil, level, "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = dao.Add(Condition{
		EncounterID: 7,
		CharacterID: 2,
		Condition:   "Exhaustion",
		Level:       &level,
	})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
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

func TestIsValidConditionLevel(t *testing.T) {
	level := func(n int) *int { return &n }
	cases := []struct {
		name  string
		cond  string
		level *int
		want  bool
	}{
		{"exhaustion min", "Exhaustion", level(1), true},
		{"exhaustion max", "Exhaustion", level(MaxExhaustionLevel), true},
		{"exhaustion zero", "Exhaustion", level(0), false},
		{"exhaustion over max", "Exhaustion", level(MaxExhaustionLevel + 1), false},
		{"exhaustion missing", "Exhaustion", nil, false},
		{"binary without level", "Prone", nil, true},
		{"binary with level", "Prone", level(2), false},
	}
	for _, tc := range cases {
		if got := IsValidConditionLevel(tc.cond, tc.level); got != tc.want {
			t.Errorf("%s: IsValidConditionLevel(%q, %v) = %v, want %v", tc.name, tc.cond, tc.level, got, tc.want)
		}
	}
}

// The catalog must tell the frontend which conditions are leveled; without
// MaxLevel it cannot know to prompt for one.
func TestConditionCatalogMarksExhaustionLeveled(t *testing.T) {
	catalog := ConditionCatalog()
	if len(catalog) != len(ValidConditions) {
		t.Fatalf("catalog has %d entries, want %d", len(catalog), len(ValidConditions))
	}
	for _, info := range catalog {
		if info.Name == "Exhaustion" {
			if info.MaxLevel != MaxExhaustionLevel {
				t.Errorf("Exhaustion MaxLevel = %d, want %d", info.MaxLevel, MaxExhaustionLevel)
			}
			if len(info.LevelEffects) != MaxExhaustionLevel {
				t.Errorf("Exhaustion has %d level effects, want %d", len(info.LevelEffects), MaxExhaustionLevel)
			}
			continue
		}
		if info.MaxLevel != 0 {
			t.Errorf("%s MaxLevel = %d, want 0", info.Name, info.MaxLevel)
		}
	}
}
