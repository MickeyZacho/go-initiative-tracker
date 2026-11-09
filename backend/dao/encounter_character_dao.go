package dao

import (
	"database/sql"
	"fmt"
)

type EncounterCharacter struct {
	EncounterID  int
	CharacterID  int
	Initiative   int
	CurrentHP    int
	IsActive     bool
	ReactionUsed bool
}

type EncounterCharacterDAO interface {
	GetByEncounterAndCharacter(encounterID, characterID int) (EncounterCharacter, error)
	Update(enc EncounterCharacter) error
	Upsert(enc EncounterCharacter) error
	StartCombat(encounterID int) (int, error)
	ResetCombat(encounterID int) error
	// GetRound returns the encounter's combat round (0 = combat not started).
	GetRound(encounterID int) (int, error)
	AdvanceTurn(encounterID int) (int, error)
	SetActiveCharacter(encounterID, characterID int) error
	SetReactionUsed(encounterID, characterID int, used bool) error
}

type encounterCharacterDAOImpl struct {
	db *sql.DB
}

func NewEncounterCharacterDAO(db *sql.DB) EncounterCharacterDAO {
	return &encounterCharacterDAOImpl{db: db}
}

func (dao *encounterCharacterDAOImpl) GetByEncounterAndCharacter(encounterID, characterID int) (EncounterCharacter, error) {
	var ec EncounterCharacter
	err := dao.db.QueryRow(
		"SELECT encounter_id, character_id, initiative, current_hp, is_active, reaction_used FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2",
		encounterID, characterID,
	).Scan(&ec.EncounterID, &ec.CharacterID, &ec.Initiative, &ec.CurrentHP, &ec.IsActive, &ec.ReactionUsed)
	return ec, err
}

func (dao *encounterCharacterDAOImpl) Update(enc EncounterCharacter) error {
	_, err := dao.db.Exec(
		"UPDATE encounter_characters SET initiative = $1, current_hp = $2, is_active = $3 WHERE encounter_id = $4 AND character_id = $5",
		enc.Initiative, enc.CurrentHP, enc.IsActive, enc.EncounterID, enc.CharacterID,
	)
	return err
}

func (dao *encounterCharacterDAOImpl) Upsert(enc EncounterCharacter) error {
	_, err := dao.db.Exec(
		"INSERT INTO encounter_characters (encounter_id, character_id, initiative, current_hp, is_active) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (encounter_id, character_id) DO UPDATE SET initiative = EXCLUDED.initiative, current_hp = EXCLUDED.current_hp, is_active = EXCLUDED.is_active",
		enc.EncounterID, enc.CharacterID, enc.Initiative, enc.CurrentHP, enc.IsActive,
	)
	return err
}

func (dao *encounterCharacterDAOImpl) StartCombat(encounterID int) (int, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		"SELECT character_id FROM encounter_characters WHERE encounter_id = $1 ORDER BY COALESCE(initiative, 0) DESC, character_id ASC",
		encounterID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var orderedIDs []int
	for rows.Next() {
		var id int
		if scanErr := rows.Scan(&id); scanErr != nil {
			return 0, scanErr
		}
		orderedIDs = append(orderedIDs, id)
	}
	if len(orderedIDs) == 0 {
		return 0, fmt.Errorf("no characters in encounter")
	}
	activeID := orderedIDs[0]

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, activeID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec("UPDATE encounters SET round = 1 WHERE id = $1", encounterID); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return activeID, nil
}

// ResetCombat ends combat: nobody holds the turn and the round clock returns to
// 0, so the next Start/Next begins again at round 1.
func (dao *encounterCharacterDAOImpl) ResetCombat(encounterID int) error {
	tx, err := dao.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE encounters SET round = 0 WHERE id = $1", encounterID); err != nil {
		return err
	}
	return tx.Commit()
}

func (dao *encounterCharacterDAOImpl) GetRound(encounterID int) (int, error) {
	var round int
	err := dao.db.QueryRow("SELECT round FROM encounters WHERE id = $1", encounterID).Scan(&round)
	return round, err
}

// SetActiveCharacter makes a single character the active turn holder, clearing
// is_active on everyone else in the encounter. This lets a user click a
// character to take the turn so a subsequent AdvanceTurn continues from there.
func (dao *encounterCharacterDAOImpl) SetActiveCharacter(encounterID, characterID int) error {
	tx, err := dao.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return err
	}

	res, err := tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, characterID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("character not in encounter")
	}

	return tx.Commit()
}

// SetReactionUsed records whether a character has spent its reaction this round.
// The turn-advance logic clears this automatically when the creature's turn
// begins; this is the manual toggle a DM uses mid-round. Returns an error when the
// character is not part of the encounter so a stale id can't silently no-op.
func (dao *encounterCharacterDAOImpl) SetReactionUsed(encounterID, characterID int, used bool) error {
	res, err := dao.db.Exec(
		"UPDATE encounter_characters SET reaction_used = $1 WHERE encounter_id = $2 AND character_id = $3",
		used, encounterID, characterID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("character not in encounter")
	}
	return nil
}

func (dao *encounterCharacterDAOImpl) AdvanceTurn(encounterID int) (int, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		"SELECT character_id FROM encounter_characters WHERE encounter_id = $1 ORDER BY COALESCE(initiative, 0) DESC, character_id ASC",
		encounterID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var orderedIDs []int
	for rows.Next() {
		var id int
		if scanErr := rows.Scan(&id); scanErr != nil {
			return 0, scanErr
		}
		orderedIDs = append(orderedIDs, id)
	}
	if len(orderedIDs) == 0 {
		return 0, fmt.Errorf("no characters in encounter")
	}

	currentActiveID := 0
	if err = tx.QueryRow(
		"SELECT COALESCE(MAX(character_id), 0) FROM encounter_characters WHERE encounter_id = $1 AND is_active = TRUE",
		encounterID,
	).Scan(&currentActiveID); err != nil {
		return 0, err
	}

	nextIndex := 0
	currentIndex := -1
	if currentActiveID != 0 {
		for idx, id := range orderedIDs {
			if id == currentActiveID {
				currentIndex = idx
				nextIndex = (idx + 1) % len(orderedIDs)
				break
			}
		}
	}
	nextActiveID := orderedIDs[nextIndex]

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, nextActiveID,
	); err != nil {
		return 0, err
	}

	// Advance the round clock. With nobody active, this Next is effectively the
	// start of combat, so make sure we're in at least round 1. Otherwise a new
	// round begins whenever the turn wraps back to the top of the order.
	switch {
	case currentIndex < 0:
		if _, err = tx.Exec("UPDATE encounters SET round = GREATEST(round, 1) WHERE id = $1", encounterID); err != nil {
			return 0, err
		}
	case nextIndex <= currentIndex:
		if _, err = tx.Exec("UPDATE encounters SET round = round + 1 WHERE id = $1", encounterID); err != nil {
			return 0, err
		}
	}

	// Tick down the outgoing creature's timed conditions as its turn ends (when we
	// switch away from it), then clear any that have expired. Conditions with a
	// NULL duration ("until removed") are left untouched. currentActiveID is 0 on
	// the very first advance (no one active yet), so nothing is decremented then.
	if currentActiveID != 0 {
		if _, err = tx.Exec(
			"UPDATE encounter_character_conditions SET duration_rounds = duration_rounds - 1 WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL",
			encounterID, currentActiveID,
		); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(
			"DELETE FROM encounter_character_conditions WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL AND duration_rounds <= 0",
			encounterID, currentActiveID,
		); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return nextActiveID, nil
}
