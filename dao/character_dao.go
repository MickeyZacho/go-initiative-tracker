package dao

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

type CharacterDAO interface {
	GetAllCharacters() ([]Character, error)
	GetCharacterByID(id int) (Character, error)
	CreateCharacter(character Character) error
	UpdateCharacter(character Character) error
	DeleteCharacter(id int) error
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
