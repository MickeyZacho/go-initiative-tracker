package dao

import (
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Condition is a status effect applied to a character within an encounter.
// DurationRounds is nil for "until removed" conditions; a non-nil value counts
// down at the start of the affected creature's turn (see AdvanceTurn).
type Condition struct {
	ID             int    `json:"ID"`
	EncounterID    int    `json:"EncounterID"`
	CharacterID    int    `json:"CharacterID"`
	Condition      string `json:"Condition"`
	DurationRounds *int   `json:"DurationRounds"`
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

var validConditionSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(ValidConditions))
	for _, c := range ValidConditions {
		m[c] = struct{}{}
	}
	return m
}()

// IsValidCondition reports whether name is a recognized 5e condition.
func IsValidCondition(name string) bool {
	_, ok := validConditionSet[name]
	return ok
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
		"SELECT id, encounter_id, character_id, condition, duration_rounds, COALESCE(note, '') FROM encounter_character_conditions WHERE encounter_id = $1 ORDER BY character_id ASC, condition ASC",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conditions []Condition
	for rows.Next() {
		var c Condition
		if err := rows.Scan(&c.ID, &c.EncounterID, &c.CharacterID, &c.Condition, &c.DurationRounds, &c.Note); err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	return conditions, rows.Err()
}

// Add applies a condition to a character in an encounter. Re-applying the same
// condition updates its duration/note (upsert on the unique key) rather than
// erroring, so a DM can refresh a timer without removing it first.
func (dao *encounterConditionDAOImpl) Add(cond Condition) error {
	_, err := dao.db.Exec(
		`INSERT INTO encounter_character_conditions (encounter_id, character_id, condition, duration_rounds, note)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		 ON CONFLICT (encounter_id, character_id, condition)
		 DO UPDATE SET duration_rounds = EXCLUDED.duration_rounds, note = EXCLUDED.note`,
		cond.EncounterID, cond.CharacterID, cond.Condition, cond.DurationRounds, cond.Note,
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
