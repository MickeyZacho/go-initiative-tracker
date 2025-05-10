package dao

type EncounterDAO interface {
	GetAllEncounters() ([]Encounter, error)
	GetCharactersByEncounter(encounterID int) ([]Character, error)
	AddCharacterToEncounter(encounterID, characterID int) error
	RemoveCharacterFromEncounter(encounterID, characterID int) error
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

func (dao *characterDAOImpl) GetAllEncounters() ([]Encounter, error) {
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

func (dao *characterDAOImpl) GetCharactersByEncounter(encounterID int) ([]Character, error) {
	rows, err := dao.db.Query(`
        SELECT c.id, c.name, c.armor_class, c.max_hp, c.current_hp, c.initiative
        FROM characters c
        JOIN encounter_characters ec ON c.id = ec.character_id
        WHERE ec.encounter_id = $1
    `, encounterID)
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

func (dao *characterDAOImpl) AddCharacterToEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("INSERT INTO encounter_characters (encounter_id, character_id) VALUES ($1, $2)", encounterID, characterID)
	return err
}

func (dao *characterDAOImpl) RemoveCharacterFromEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("DELETE FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2", encounterID, characterID)
	return err
}
