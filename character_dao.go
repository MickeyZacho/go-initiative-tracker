package main

import (
	"database/sql"
)

type Character struct {
	ID              int
	Name            string
	ArmorClass      int
	MaxHP           int
	CurrentHP       int
	Initiative      int
	IsActive        bool
	InitiativeOrder int
}

type Encounter struct {
	ID              int
	CharacterID     int
	CurrentHP       int
	Initiative      int
	InitiativeOrder int
}

type CharacterDAO interface {
	GetAllCharacters() ([]Character, error)
	GetCharacterByID(id int) (Character, error)
	CreateCharacter(character Character) error
	UpdateCharacter(character Character) error
	DeleteCharacter(id int) error
	GetAllEncounters() ([]Encounter, error)
	GetEncounterByID(id int) (Encounter, error)
	CreateEncounter(encounter Encounter) error
	UpdateEncounter(encounter Encounter) error
	DeleteEncounter(id int) error
	GetCharacterFromEncounter(encounterID int) (Character, error)
}

type characterDAOImpl struct {
	db *sql.DB
}

func NewCharacterDAO(db *sql.DB) CharacterDAO {
	return &characterDAOImpl{db: db}
}

func (dao *characterDAOImpl) GetAllCharacters() ([]Character, error) {
	rows, err := dao.db.Query("SELECT id, name, armor_class, max_hp, current_hp, initiative FROM characters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) GetCharacterByID(id int) (Character, error) {
	var c Character
	err := dao.db.QueryRow("SELECT id, name, armor_class, max_hp, current_hp, initiative FROM characters WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (dao *characterDAOImpl) CreateCharacter(character Character) error {
	_, err := dao.db.Exec("INSERT INTO characters (name, armor_class, max_hp, current_hp, initiative) VALUES ($1, $2, $3, $4, $5)",
		character.Name, character.ArmorClass, character.MaxHP, character.CurrentHP, `0`)
	return err
}

func (dao *characterDAOImpl) UpdateCharacter(character Character) error {
	_, err := dao.db.Exec("UPDATE characters SET name = $1, armor_class = $2, max_hp = $3, current_hp = $4, initiative = $5 WHERE id = $6",
		character.Name, character.ArmorClass, character.MaxHP, character.CurrentHP, character.Initiative, character.ID)
	return err
}

func (dao *characterDAOImpl) DeleteCharacter(id int) error {
	_, err := dao.db.Exec("DELETE FROM characters WHERE id = $1", id)
	return err
}

func (dao *characterDAOImpl) GetAllEncounters() ([]Encounter, error) {
	rows, err := dao.db.Query("SELECT id, character_id, current_hp, initiative, is_active, initiative_order FROM encounter")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.CharacterID, &e.CurrentHP, &e.Initiative, &e.InitiativeOrder)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

func (dao *characterDAOImpl) GetEncounterByID(id int) (Encounter, error) {
	var e Encounter
	err := dao.db.QueryRow("SELECT id, character_id, current_hp, initiative, is_active, initiative_order FROM encounter WHERE id = $1", id).Scan(&e.ID, &e.CharacterID, &e.CurrentHP, &e.Initiative, &e.InitiativeOrder)
	if err != nil {
		return e, err
	}
	return e, nil
}

func (dao *characterDAOImpl) CreateEncounter(encounter Encounter) error {
	_, err := dao.db.Exec("INSERT INTO encounter (character_id, current_hp, initiative, is_active, initiative_order) VALUES ($1, $2, $3, $4, $5)",
		encounter.CharacterID, encounter.CurrentHP, encounter.Initiative, encounter.InitiativeOrder)
	return err
}

func (dao *characterDAOImpl) UpdateEncounter(encounter Encounter) error {
	_, err := dao.db.Exec("UPDATE encounter SET character_id = $1, current_hp = $2, initiative = $3, is_active = $4, initiative_order = $5 WHERE id = $6",
		encounter.CharacterID, encounter.CurrentHP, encounter.Initiative, encounter.InitiativeOrder, encounter.ID)
	return err
}

func (dao *characterDAOImpl) DeleteEncounter(id int) error {
	_, err := dao.db.Exec("DELETE FROM encounter WHERE id = $1", id)
	return err
}

func (dao *characterDAOImpl) GetCharacterFromEncounter(encounterID int) (Character, error) {
	var c Character
	query := `
		SELECT 
			characters.id, characters.name, characters.armor_class, characters.max_hp,
			encounter.current_hp, encounter.initiative, encounter.is_active, encounter.initiative_order
		FROM 
			characters
		JOIN 
			encounter ON characters.id = encounter.character_id
		WHERE 
			encounter.id = $1
	`
	err := dao.db.QueryRow(query, encounterID).Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.InitiativeOrder)
	if err != nil {
		return c, err
	}
	return c, nil
}
