package dao

import (
	"database/sql"
	"slices"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Condition is a status effect applied to a character within an encounter.
// DurationRounds is nil for "until removed" conditions; a non-nil value counts
// down at the start of the affected creature's turn (see AdvanceTurn).
// Level is non-nil only for leveled conditions (Exhaustion, 1-6).
type Condition struct {
	ID             int    `json:"ID"`
	EncounterID    int    `json:"EncounterID"`
	CharacterID    int    `json:"CharacterID"`
	Condition      string `json:"Condition"`
	DurationRounds *int   `json:"DurationRounds"`
	Level          *int   `json:"Level"`
	Note           string `json:"Note"`
}

// ValidConditions is the canonical D&D 5e condition set. Handlers validate
// incoming names against this so the stored data stays consistent and the
// frontend can render a known set of chips.
var ValidConditions = []string{
	"Blinded",
	"Charmed",
	"Deafened",
	"Exhaustion",
	"Frightened",
	"Grappled",
	"Incapacitated",
	"Invisible",
	"Paralyzed",
	"Petrified",
	"Poisoned",
	"Prone",
	"Restrained",
	"Stunned",
	"Unconscious",
}

// MaxExhaustionLevel is the 5e cap: a sixth level of exhaustion kills the creature.
const MaxExhaustionLevel = 6

// conditionMaxLevels lists the conditions that track a level and how high it
// goes. Anything absent is binary — applied or not — and must carry a nil Level.
var conditionMaxLevels = map[string]int{
	"Exhaustion": MaxExhaustionLevel,
}

// ExhaustionEffects describes what each level of exhaustion does (5e, 2014
// rules), indexed by level - 1. Served with the catalog so the frontend can
// explain a level without duplicating the table.
var ExhaustionEffects = []string{
	"Disadvantage on ability checks",
	"Speed halved",
	"Disadvantage on attack rolls and saving throws",
	"Hit point maximum halved",
	"Speed reduced to 0",
	"Death",
}

// ConditionInfo is the catalog entry for one condition. MaxLevel is 0 for
// ordinary binary conditions and >0 for leveled ones, which lets the frontend
// decide whether to prompt for a level.
type ConditionInfo struct {
	Name         string   `json:"Name"`
	MaxLevel     int      `json:"MaxLevel"`
	LevelEffects []string `json:"LevelEffects,omitempty"`
}

// ConditionCatalog returns every valid condition with its level metadata.
func ConditionCatalog() []ConditionInfo {
	catalog := make([]ConditionInfo, 0, len(ValidConditions))
	for _, name := range ValidConditions {
		info := ConditionInfo{Name: name, MaxLevel: conditionMaxLevels[name]}
		if name == "Exhaustion" {
			info.LevelEffects = ExhaustionEffects
		}
		catalog = append(catalog, info)
	}
	return catalog
}

// IsValidCondition reports whether name is a recognized 5e condition.
func IsValidCondition(name string) bool {
	return slices.Contains(ValidConditions, name)
}

// ConditionMaxLevel returns the highest level name supports, or 0 if it is a
// binary condition (or not a condition at all).
func ConditionMaxLevel(name string) int {
	return conditionMaxLevels[name]
}

// IsValidConditionLevel reports whether level is acceptable for the named
// condition: leveled conditions require 1..MaxLevel, binary ones require nil.
func IsValidConditionLevel(name string, level *int) bool {
	max := conditionMaxLevels[name]
	if max == 0 {
		return level == nil
	}
	return level != nil && *level >= 1 && *level <= max
}

type EncounterConditionDAO interface {
	ListByEncounter(encounterID int) ([]Condition, error)
	Add(cond Condition) error
	Remove(id, encounterID int) (bool, error)
}

type encounterConditionDAOImpl struct {
	db *sql.DB
}

func NewEncounterConditionDAO(db *sql.DB) EncounterConditionDAO {
	return &encounterConditionDAOImpl{db: db}
}

func (dao *encounterConditionDAOImpl) ListByEncounter(encounterID int) ([]Condition, error) {
	rows, err := dao.db.Query(
		"SELECT id, encounter_id, character_id, condition, duration_rounds, level, COALESCE(note, '') FROM encounter_character_conditions WHERE encounter_id = $1 ORDER BY character_id ASC, condition ASC",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conditions []Condition
	for rows.Next() {
		var c Condition
		if err := rows.Scan(&c.ID, &c.EncounterID, &c.CharacterID, &c.Condition, &c.DurationRounds, &c.Level, &c.Note); err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	return conditions, rows.Err()
}

// Add applies a condition to a character in an encounter. Re-applying the same
// condition updates its duration/level/note (upsert on the unique key) rather
// than erroring, so a DM can refresh a timer — or raise a creature's exhaustion
// level — without removing it first.
func (dao *encounterConditionDAOImpl) Add(cond Condition) error {
	_, err := dao.db.Exec(
		`INSERT INTO encounter_character_conditions (encounter_id, character_id, condition, duration_rounds, level, note)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		 ON CONFLICT (encounter_id, character_id, condition)
		 DO UPDATE SET duration_rounds = EXCLUDED.duration_rounds, level = EXCLUDED.level, note = EXCLUDED.note`,
		cond.EncounterID, cond.CharacterID, cond.Condition, cond.DurationRounds, cond.Level, cond.Note,
	)
	return err
}

// Remove deletes a condition by id, scoped to its encounter so a stale id from
// another fight can never delete the wrong row. Returns false when nothing matched.
func (dao *encounterConditionDAOImpl) Remove(id, encounterID int) (bool, error) {
	res, err := dao.db.Exec(
		"DELETE FROM encounter_character_conditions WHERE id = $1 AND encounter_id = $2",
		id, encounterID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}
