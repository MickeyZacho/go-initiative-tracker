package dao

import (
	"database/sql"
	"fmt"
)

type EncounterDAO interface {
	GetAllEncounters() ([]Encounter, error)
	AddCharacterToEncounter(encounterID, characterID int) error
	RemoveCharacterFromEncounter(encounterID, characterID int) error
	GetEncountersByOwnerDiscordID(discordID string) ([]Encounter, error)
}
type Encounter struct {
	ID            int
	Name          string
	OwnerID       int
	Description   string
	CreatedAt     string
	UpdatedAt     string
	EncounterType string
	CampaignID    int
}

type encounterDAOImpl struct {
	db *sql.DB
}

func NewEncounterDAO(db *sql.DB) EncounterDAO {
	return &encounterDAOImpl{db: db}
}

func (dao *encounterDAOImpl) GetAllEncounters() ([]Encounter, error) {
	rows, err := dao.db.Query("SELECT id, name FROM encounters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Name)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

func (dao *encounterDAOImpl) AddCharacterToEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("INSERT INTO encounter_characters (encounter_id, character_id) VALUES ($1, $2)", encounterID, characterID)
	return err
}

func (dao *encounterDAOImpl) RemoveCharacterFromEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("DELETE FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2", encounterID, characterID)
	return err
}

// Get all encounters for a given Discord user
func (dao *encounterDAOImpl) GetEncountersByOwnerDiscordID(discordID string) ([]Encounter, error) {
	fmt.Println("Fetching encounters for Discord ID:", discordID)
	rows, err := dao.db.Query("SELECT id, name FROM encounters WHERE owner_id = $1", discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Name)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}
