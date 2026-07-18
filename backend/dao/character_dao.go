package dao

import (
	"database/sql"
	"errors"
)

type Character struct {
	ID            int
	Name          string
	ArmorClass    int
	ToHitModifier int
	MaxHP         int
	CurrentHP     int
	Initiative    int
	IsActive      bool
	OwnerID       string
	Type          string // 'pc' or 'npc'
	NpcTemplateID *int   // Nullable foreign key to NpcTemplate
}

type CharacterDAO interface {
	GetAllCharacters() ([]Character, error)
	GetSampleCharacters() ([]Character, error)
	GetCharacterByID(id int) (Character, error)
	GetCharactersByEncounterID(encounterID int) ([]Character, error)
	CreateCharacter(character Character) (int, error)
	UpdateCharacterByOwner(character Character, ownerID string) (bool, error)
	UpdateCharacterInEncounter(character Character, encounterID int) (string, bool, error)
	DeleteCharacter(id int) error
	DeleteCharacterByOwner(id int, ownerID string) (bool, error)
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
	rows, err := dao.db.Query("SELECT id, name, armor_class, to_hit_modifier, max_hp, max_hp AS current_hp, 0 AS initiative, false AS is_active, COALESCE(owner_id, ''), type, npc_template_id FROM characters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type, &c.NpcTemplateID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

// GetSampleCharacters returns the unowned example characters (NPCs/monsters and
// seed rows with no owner), so logged-out visitors see a few examples rather
// than every row in the database. Ordered by id for a stable sample.
func (dao *characterDAOImpl) GetSampleCharacters() ([]Character, error) {
	rows, err := dao.db.Query("SELECT id, name, armor_class, to_hit_modifier, max_hp, max_hp AS current_hp, 0 AS initiative, false AS is_active, COALESCE(owner_id, ''), type, npc_template_id FROM characters WHERE owner_id IS NULL OR owner_id = '' ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type, &c.NpcTemplateID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) GetCharacterByID(id int) (Character, error) {
	var c Character
	err := dao.db.QueryRow("SELECT id, name, armor_class, to_hit_modifier, max_hp, max_hp AS current_hp, 0 AS initiative, false AS is_active, COALESCE(owner_id, ''), type, npc_template_id FROM characters WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type, &c.NpcTemplateID)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (dao *characterDAOImpl) GetCharactersByEncounterID(encounterID int) ([]Character, error) {
	rows, err := dao.db.Query(
		"SELECT c.id, c.name, c.armor_class, c.to_hit_modifier, c.max_hp, COALESCE(ec.current_hp, c.max_hp), COALESCE(ec.initiative, 0), COALESCE(ec.is_active, false), COALESCE(c.owner_id, ''), c.type, c.npc_template_id FROM characters c JOIN encounter_characters ec ON c.id = ec.character_id WHERE ec.encounter_id = $1",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type, &c.NpcTemplateID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

func (dao *characterDAOImpl) CreateCharacter(character Character) (int, error) {
	if character.Type == "" {
		character.Type = "pc"
	}
	var newID int
	err := dao.db.QueryRow(
		"INSERT INTO characters (name, armor_class, to_hit_modifier, max_hp, owner_id, type) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		character.Name, character.ArmorClass, character.ToHitModifier, character.MaxHP, character.OwnerID, character.Type,
	).Scan(&newID)
	return newID, err
}

// UpdateCharacterByOwner updates a character only when it belongs to ownerID.
// owner_id is intentionally not in the SET clause, so a caller can never
// reassign a character to a different owner. Returns false when no row matched
// (wrong id or not owned by the caller).
func (dao *characterDAOImpl) UpdateCharacterByOwner(character Character, ownerID string) (bool, error) {
	if character.Type == "" {
		character.Type = "pc"
	}
	result, err := dao.db.Exec("UPDATE characters SET name = $1, armor_class = $2, to_hit_modifier = $3, max_hp = $4, type = $5 WHERE id = $6 AND owner_id = $7",
		character.Name, character.ArmorClass, character.ToHitModifier, character.MaxHP, character.Type, character.ID, ownerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// UpdateCharacterInEncounter updates a character when it belongs to encounterID,
// whoever owns it: access to an encounter carries the right to edit every
// character in it. Callers must verify the requester's encounter access first.
// owner_id stays out of the SET clause, so an edit can never reassign a
// character; the current owner is returned so callers can echo it back. Returns
// false when the character is not in the encounter (or does not exist).
func (dao *characterDAOImpl) UpdateCharacterInEncounter(character Character, encounterID int) (string, bool, error) {
	if character.Type == "" {
		character.Type = "pc"
	}
	var ownerID string
	err := dao.db.QueryRow(
		`UPDATE characters c SET name = $1, armor_class = $2, to_hit_modifier = $3, max_hp = $4, type = $5
		 WHERE c.id = $6 AND EXISTS (SELECT 1 FROM encounter_characters ec WHERE ec.character_id = c.id AND ec.encounter_id = $7)
		 RETURNING COALESCE(c.owner_id, '')`,
		character.Name, character.ArmorClass, character.ToHitModifier, character.MaxHP, character.Type, character.ID, encounterID,
	).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return ownerID, true, nil
}

func (dao *characterDAOImpl) DeleteCharacter(id int) error {
	_, err := dao.db.Exec("DELETE FROM characters WHERE id = $1", id)
	return err
}

func (dao *characterDAOImpl) DeleteCharacterByOwner(id int, ownerID string) (bool, error) {
	result, err := dao.db.Exec("DELETE FROM characters WHERE id = $1 AND owner_id = $2", id, ownerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// Get all characters for a given Discord user
func (dao *characterDAOImpl) GetAllCharactersByOwner(discordID string) ([]Character, error) {
	rows, err := dao.db.Query(`SELECT c.id, c.name, c.armor_class, c.to_hit_modifier, c.max_hp, c.max_hp AS current_hp, 0 AS initiative, false AS is_active, COALESCE(c.owner_id, ''), c.type FROM characters c WHERE c.owner_id = $1`, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}

// Get all characters for a given encounter and Discord user
func (dao *characterDAOImpl) GetCharactersByEncounterIDAndOwner(encounterID int, discordID string) ([]Character, error) {
	rows, err := dao.db.Query(`SELECT c.id, c.name, c.armor_class, c.to_hit_modifier, c.max_hp, COALESCE(ec.current_hp, c.max_hp), COALESCE(ec.initiative, 0), COALESCE(ec.is_active, false), COALESCE(c.owner_id, ''), c.type FROM characters c JOIN encounter_characters ec ON c.id = ec.character_id WHERE ec.encounter_id = $1 AND c.owner_id = $2`, encounterID, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		err := rows.Scan(&c.ID, &c.Name, &c.ArmorClass, &c.ToHitModifier, &c.MaxHP, &c.CurrentHP, &c.Initiative, &c.IsActive, &c.OwnerID, &c.Type)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}
	return characters, nil
}
