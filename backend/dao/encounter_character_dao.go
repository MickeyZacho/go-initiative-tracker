package dao

import (
	"database/sql"
	"fmt"
)

type EncounterCharacter struct {
	EncounterID int
	CharacterID int
	Initiative  int
	CurrentHP   int
	IsActive    bool
}

type EncounterCharacterDAO interface {
	GetByEncounterAndCharacter(encounterID, characterID int) (EncounterCharacter, error)
	Update(enc EncounterCharacter) error
	Upsert(enc EncounterCharacter) error
	StartCombat(encounterID int) (int, error)
	ResetCombat(encounterID int) error
	AdvanceTurn(encounterID int) (int, error)
	SetActiveCharacter(encounterID, characterID int) error
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
		"SELECT encounter_id, character_id, initiative, current_hp, is_active FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2",
		encounterID, characterID,
	).Scan(&ec.EncounterID, &ec.CharacterID, &ec.Initiative, &ec.CurrentHP, &ec.IsActive)
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
		"UPDATE encounter_characters SET is_active = TRUE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, activeID,
	); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return activeID, nil
}

func (dao *encounterCharacterDAOImpl) ResetCombat(encounterID int) error {
	_, err := dao.db.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	)
	return err
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
		"UPDATE encounter_characters SET is_active = TRUE WHERE encounter_id = $1 AND character_id = $2",
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
	if currentActiveID != 0 {
		for idx, id := range orderedIDs {
			if id == currentActiveID {
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
		"UPDATE encounter_characters SET is_active = TRUE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, nextActiveID,
	); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return nextActiveID, nil
}
