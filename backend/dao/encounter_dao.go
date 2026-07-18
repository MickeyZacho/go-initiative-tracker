package dao

import (
	"database/sql"
)

type EncounterDAO interface {
	GetAllEncounters() ([]Encounter, error)
	// GetUnownedEncounters returns only encounters with no owner, shown to
	// logged-out visitors as public examples.
	GetUnownedEncounters() ([]Encounter, error)
	CreateEncounter(encounter Encounter) (int, error)
	DeleteEncounter(id int) error
	DeleteEncounterByOwner(id int, ownerID string) (bool, error)
	AddCharacterToEncounter(encounterID, characterID int) error
	RemoveCharacterFromEncounter(encounterID, characterID int) error
	GetByID(id int) (Encounter, error)
	GetEncountersByOwnerDiscordID(discordID string) ([]Encounter, error)
	// GetAccessibleEncounters returns encounters the user owns plus any they are
	// a shared-edit member of (via encounter_users).
	GetAccessibleEncounters(discordID string) ([]Encounter, error)
	AddMember(encounterID int, userID string) error
	RemoveMember(encounterID int, userID string) error
	IsMember(encounterID int, userID string) (bool, error)
	ListMembers(encounterID int) ([]Friend, error)
}
type Encounter struct {
	ID          int
	Name        string
	OwnerID     string
	Description string
}

type encounterDAOImpl struct {
	db *sql.DB
}

func NewEncounterDAO(db *sql.DB) EncounterDAO {
	return &encounterDAOImpl{db: db}
}

func (dao *encounterDAOImpl) GetAllEncounters() ([]Encounter, error) {
	rows, err := dao.db.Query("SELECT id, name, COALESCE(owner_id, ''), COALESCE(description, '') FROM encounters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Name, &e.OwnerID, &e.Description)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

func (dao *encounterDAOImpl) GetUnownedEncounters() ([]Encounter, error) {
	rows, err := dao.db.Query("SELECT id, name, COALESCE(owner_id, ''), COALESCE(description, '') FROM encounters WHERE owner_id IS NULL OR owner_id = '' ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Name, &e.OwnerID, &e.Description)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

func (dao *encounterDAOImpl) CreateEncounter(encounter Encounter) (int, error) {
	var newID int
	err := dao.db.QueryRow(
		"INSERT INTO encounters (name, owner_id, description) VALUES ($1, $2, $3) RETURNING id",
		encounter.Name,
		encounter.OwnerID,
		encounter.Description,
	).Scan(&newID)
	return newID, err
}

func (dao *encounterDAOImpl) DeleteEncounter(id int) error {
	_, err := dao.db.Exec("DELETE FROM encounters WHERE id = $1", id)
	return err
}

func (dao *encounterDAOImpl) DeleteEncounterByOwner(id int, ownerID string) (bool, error) {
	result, err := dao.db.Exec("DELETE FROM encounters WHERE id = $1 AND owner_id = $2", id, ownerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (dao *encounterDAOImpl) AddCharacterToEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("INSERT INTO encounter_characters (encounter_id, character_id) VALUES ($1, $2)", encounterID, characterID)
	return err
}

func (dao *encounterDAOImpl) RemoveCharacterFromEncounter(encounterID, characterID int) error {
	_, err := dao.db.Exec("DELETE FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2", encounterID, characterID)
	return err
}

func (dao *encounterDAOImpl) GetByID(id int) (Encounter, error) {
	var e Encounter
	err := dao.db.QueryRow("SELECT id, name, COALESCE(owner_id, ''), COALESCE(description, '') FROM encounters WHERE id = $1", id).Scan(&e.ID, &e.Name, &e.OwnerID, &e.Description)
	return e, err
}

// Get all encounters for a given Discord user
func (dao *encounterDAOImpl) GetEncountersByOwnerDiscordID(discordID string) ([]Encounter, error) {
	rows, err := dao.db.Query("SELECT id, name, COALESCE(owner_id, ''), COALESCE(description, '') FROM encounters WHERE owner_id = $1", discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Name, &e.OwnerID, &e.Description)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

// GetAccessibleEncounters returns the union of encounters owned by discordID and
// those they are a shared-edit member of. DISTINCT dedups the case where an owner
// is (redundantly) also listed as a member.
func (dao *encounterDAOImpl) GetAccessibleEncounters(discordID string) ([]Encounter, error) {
	rows, err := dao.db.Query(`
		SELECT DISTINCT e.id, e.name, COALESCE(e.owner_id, ''), COALESCE(e.description, '')
		FROM encounters e
		LEFT JOIN encounter_users eu ON eu.encounter_id = e.id
		WHERE e.owner_id = $1 OR eu.user_id = $1
		ORDER BY e.id`, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		if err := rows.Scan(&e.ID, &e.Name, &e.OwnerID, &e.Description); err != nil {
			return nil, err
		}
		encounters = append(encounters, e)
	}
	return encounters, rows.Err()
}

func (dao *encounterDAOImpl) AddMember(encounterID int, userID string) error {
	_, err := dao.db.Exec(
		"INSERT INTO encounter_users (encounter_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		encounterID, userID,
	)
	return err
}

func (dao *encounterDAOImpl) RemoveMember(encounterID int, userID string) error {
	_, err := dao.db.Exec(
		"DELETE FROM encounter_users WHERE encounter_id = $1 AND user_id = $2",
		encounterID, userID,
	)
	return err
}

func (dao *encounterDAOImpl) IsMember(encounterID int, userID string) (bool, error) {
	var count int
	err := dao.db.QueryRow(
		"SELECT COUNT(*) FROM encounter_users WHERE encounter_id = $1 AND user_id = $2",
		encounterID, userID,
	).Scan(&count)
	return count > 0, err
}

// ListMembers returns the shared-edit members of an encounter, resolved to their
// user rows for display.
func (dao *encounterDAOImpl) ListMembers(encounterID int) ([]Friend, error) {
	rows, err := dao.db.Query(`
		SELECT u.discord_id, u.username, COALESCE(u.avatar, '')
		FROM encounter_users eu
		JOIN users u ON u.discord_id = eu.user_id
		WHERE eu.encounter_id = $1
		ORDER BY u.username`, encounterID)
	if err != nil {
		return nil, err
	}
	return scanFriends(rows)
}
