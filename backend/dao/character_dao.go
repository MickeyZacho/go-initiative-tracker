package dao

import (
	"database/sql"
)

type Character struct {
	ID         int
	Name       string
	ArmorClass int
	MaxHP      int
	CurrentHP  int
	Initiative int
	IsActive   bool
	OwnerID    string
}

type CharacterDAO interface {
	GetAllCharacters() ([]Character, error)
	GetCharacterByID(id int) (Character, error)
	GetCharactersByEncounterID(encounterID int) ([]Character, error)
	CreateCharacter(character Character) (int, error)
	UpdateCharacter(character Character) error
	DeleteCharacter(id int) error
	GetAllCharactersByOwner(discordID string) ([]Character, error)
	GetCharactersByEncounterIDAndOwner(encounterID int, discordID string) ([]Character, error)
}

type characterDAOImpl struct {
	db *sql.DB
}

func NewCharacterDAO(db *sql.DB) CharacterDAO {
	return &characterDAOImpl{db: db}
}

func (dao *characterDAOImpl) GetAllCharacters() ([]Character, error) {
	rows, err := dao.db.Query("SELECT id, name, armor_class, max_hp, current_hp, initiative, owner_id FROM characters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.OwnerID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) GetCharacterByID(id int) (Character, error) {
	var c Character
	err := dao.db.QueryRow("SELECT id, name, armor_class, max_hp, current_hp, initiative, owner_id FROM characters WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (dao *characterDAOImpl) GetCharactersByEncounterID(encounterID int) ([]Character, error) {
	rows, err := dao.db.Query(
		"SELECT c.id, c.name, c.armor_class, c.max_hp, c.current_hp, c.initiative, c.owner_id FROM characters c JOIN encounter_characters ec ON c.id = ec.character_id WHERE ec.encounter_id = $1",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.OwnerID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) CreateCharacter(character Character) (int, error) {
	var newID int
	err := dao.db.QueryRow(
		"INSERT INTO characters (name, armor_class, max_hp, current_hp, initiative) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		character.Name, character.ArmorClass, character.MaxHP, character.CurrentHP, 0,
	).Scan(&newID)
	return newID, err
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

// Get all characters for a given Discord user
func (dao *characterDAOImpl) GetAllCharactersByOwner(discordID string) ([]Character, error) {
	rows, err := dao.db.Query(`SELECT c.id, c.name, c.armor_class, c.max_hp, c.current_hp, c.initiative, c.owner_id FROM characters c WHERE c.owner_id = $1`, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.OwnerID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

// Get all characters for a given encounter and Discord user
func (dao *characterDAOImpl) GetCharactersByEncounterIDAndOwner(encounterID int, discordID string) ([]Character, error) {
	rows, err := dao.db.Query(`SELECT c.id, c.name, c.armor_class, c.max_hp, c.current_hp, c.initiative, c.owner_id FROM characters c JOIN encounter_characters ec ON c.id = ec.character_id WHERE ec.encounter_id = $1 AND c.owner_id = $2`, encounterID, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.OwnerID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}
