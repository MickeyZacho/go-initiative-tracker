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

type CharacterDAO interface {
	GetAll() ([]Character, error)
	GetByID(id int) (Character, error)
	Create(character Character) error
	Update(character Character) error
	Delete(id int) error
}

type characterDAOImpl struct {
	db *sql.DB
}

func NewCharacterDAO(db *sql.DB) CharacterDAO {
	return &characterDAOImpl{db: db}
}

func (dao *characterDAOImpl) GetAll() ([]Character, error) {
	rows, err := dao.db.Query("SELECT id, name, armor_class, max_hp, current_hp, initiative, is_active, initiative_order FROM characters ")
	if err != nil {
		print(err)
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.InitiativeOrder)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) GetByID(id int) (Character, error) {
	var c Character
	err := dao.db.QueryRow("SELECT id, name, armor_class, max_hp, current_hp, initiative, is_active, initiative_order FROM characters WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.ArmorClass, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.InitiativeOrder)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (dao *characterDAOImpl) Create(character Character) error {
	_, err := dao.db.Exec("INSERT INTO characters (name, armor_class, max_hp, current_hp, initiative, is_active, initiative_order) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		character.Name, character.ArmorClass, character.MaxHP, character.CurrentHP, character.Initiative, character.IsActive, character.InitiativeOrder)
	return err
}

func (dao *characterDAOImpl) Update(character Character) error {
	_, err := dao.db.Exec("UPDATE characters SET name = $1, armor_class = $2, max_hp = $3, current_hp = $4, initiative = $5, is_active = $6, initiative_order = $7 WHERE id = $8",
		character.Name, character.ArmorClass, character.MaxHP, character.CurrentHP, character.Initiative, character.IsActive, character.InitiativeOrder, character.ID)
	return err
}

func (dao *characterDAOImpl) Delete(id int) error {
	_, err := dao.db.Exec("DELETE FROM characters WHERE id = $1", id)
	return err
}
