package dao

import (
	"database/sql"
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
