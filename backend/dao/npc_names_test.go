package dao

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNicknamePool(t *testing.T) {
	cases := []struct {
		name, typ string
		want      []string
	}{
		{"Goblin", "", goblinoidNames}, // app-made, matched by name
		{"Goblin Boss", "humanoid", goblinoidNames},
		{"Hobgoblin Captain", "humanoid", goblinoidNames},
		{"Orc War Chief", "Humanoid", goblinoidNames}, // type is case-insensitive
		{"Hill Giant", "giant", giantNames},
		{"Ogre", "", giantNames},
		{"Giant Rat", "beast", descriptors}, // beast, not a giant
		{"Giant Rat", "", commonNames},      // app-made and "giant" isn't last
		{"Adult Red Dragon", "dragon", dragonNames},
		{"Bandit", "humanoid", commonNames},
		{"Wolf", "beast", descriptors},
		{"Zombie", "undead", descriptors},
	}
	for _, c := range cases {
		if got := nicknamePool(c.name, c.typ); &got[0] != &c.want[0] {
			t.Errorf("nicknamePool(%q, %q) picked the wrong list (first entry %q)", c.name, c.typ, got[0])
		}
	}
}

func TestHighestCopyNumber(t *testing.T) {
	names := []string{"Goblin 1", "Goblin 3 · Snagtooth", "Goblin Boss 7", "Grumpy"}
	if got := highestCopyNumber("Goblin", names); got != 3 {
		t.Errorf("highest = %d, want 3", got)
	}
	if got := highestCopyNumber("Goblin", []string{"Goblin"}); got != 1 {
		t.Errorf("plain name should count as 1, got %d", got)
	}
	if got := highestCopyNumber("Goblin", nil); got != 0 {
		t.Errorf("no copies should be 0, got %d", got)
	}
}

func TestPickNicknameSkipsUsedAndRepeatsWhenExhausted(t *testing.T) {
	prev := pickIndex
	pickIndex = func(int) int { return 0 }
	defer func() { pickIndex = prev }()

	pool := []string{"A", "B"}
	used := map[string]bool{"A": true}
	if got := pickNickname(pool, used); got != "B" {
		t.Errorf("got %q, want B", got)
	}
	if got := pickNickname(pool, used); got != "A" {
		t.Errorf("exhausted pool should repeat, got %q", got)
	}
}

// After a removed copy leaves a gap (1 and 3 present), new copies continue
// from 3 rather than reusing it, and generated nicknames skip ones already
// taken anywhere in the encounter.
func TestAddCharactersFromTemplate_NumbersPastGapsWithNicknames(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	prevRoll, prevPick := RollD20, pickIndex
	RollD20 = func() int { return 10 }
	pickIndex = func(int) int { return 0 }
	defer func() { RollD20, pickIndex = prevRoll, prevPick }()
	dao := NewNpcTemplateDAO(db)

	mock.ExpectQuery("FROM npc_templates WHERE id").WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "base_stats", "armor_class", "max_hp", "owner_id", "source", "type", "size", "cr"}).
			AddRow(2, "Goblin", "", "(8,14,10,10,8,8)", 15, 7, "", "srd-5.2", "humanoid", "Small", 0.25))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM encounters WHERE id").WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow("dm1"))
	mock.ExpectQuery("SELECT c.name").WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"name", "npc_template_id"}).
			AddRow("Goblin 1", 2).AddRow("Goblin 3 · Snagtooth", 2).AddRow("Wolf 1 · Grimjaw", 9))
	for i, want := range []string{"Goblin 4 · Muckfist", "Goblin 5 · Skrag"} {
		mock.ExpectQuery("INSERT INTO characters").
			WithArgs(want, 15, 0, 7, "dm1", "npc", 2).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(90 + i))
		mock.ExpectExec("INSERT INTO encounter_characters").WithArgs(3, 90+i, 12, 7, false, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	created, err := dao.AddCharactersToEncounterFromTemplate(2, 3, 2, Staging{}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created %d, want 2", len(created))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
