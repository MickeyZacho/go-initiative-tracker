package dao

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
)

type StatBlock struct {
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
}

// String returns the StatBlock as a Postgres composite string, e.g., (8,14,10,10,8,8)
func (s StatBlock) String() string {
	return fmt.Sprintf("(%d,%d,%d,%d,%d,%d)",
		s.Strength, s.Dexterity, s.Constitution, s.Intelligence, s.Wisdom, s.Charisma)
}

type NpcTemplate struct {
	ID          int
	Name        string
	Description string
	BaseStats   StatBlock
	ArmorClass  int
	MaxHP       int
}

type NpcTemplateDAO interface {
	GetAll() ([]NpcTemplate, error)
	GetByID(id int) (NpcTemplate, error)
	Create(template NpcTemplate) (int, error)
	Update(template NpcTemplate) error
	Delete(id int) error
	AddCharacterToEncounterFromTemplate(templateID int, encounterID int) (Character, error)
}

type npcTemplateDAOImpl struct {
	db *sql.DB
}

func NewNpcTemplateDAO(db *sql.DB) NpcTemplateDAO {
	return &npcTemplateDAOImpl{db: db}
}

func (dao *npcTemplateDAOImpl) GetAll() ([]NpcTemplate, error) {
	rows, err := dao.db.Query(`SELECT id, name, description, base_stats, armor_class, max_hp FROM npc_templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []NpcTemplate
	for rows.Next() {
		var t NpcTemplate
		var statsStr string
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP); err != nil {
			return nil, err
		}
		stats, err := parseStatBlock(statsStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stat_block: %w", err)
		}
		t.BaseStats = stats
		templates = append(templates, t)
	}
	return templates, nil
}

func (dao *npcTemplateDAOImpl) GetByID(id int) (NpcTemplate, error) {
	var t NpcTemplate
	var statsStr string
	err := dao.db.QueryRow(`SELECT id, name, description, base_stats, armor_class, max_hp FROM npc_templates WHERE id = $1`, id).Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP)
	if err != nil {
		return t, err
	}
	stats, err := parseStatBlock(statsStr)
	if err != nil {
		return t, fmt.Errorf("failed to parse stat_block: %w", err)
	}
	t.BaseStats = stats
	return t, nil
}

// parseStatBlock parses a Postgres composite type stat_block string like (8,14,10,10,8,8)
func parseStatBlock(s string) (StatBlock, error) {
	// Remove parentheses
	re := regexp.MustCompile(`^\((.*)\)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 2 {
		return StatBlock{}, fmt.Errorf("invalid stat_block format: %s", s)
	}
	parts := regexp.MustCompile(`,`).Split(matches[1], -1)
	if len(parts) != 6 {
		return StatBlock{}, fmt.Errorf("expected 6 fields in stat_block, got %d", len(parts))
	}
	vals := make([]int, 6)
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return StatBlock{}, fmt.Errorf("invalid int in stat_block: %w", err)
		}
		vals[i] = v
	}
	return StatBlock{
		Strength:     vals[0],
		Dexterity:    vals[1],
		Constitution: vals[2],
		Intelligence: vals[3],
		Wisdom:       vals[4],
		Charisma:     vals[5],
	}, nil
}

func (dao *npcTemplateDAOImpl) Create(template NpcTemplate) (int, error) {
	var id int
	log.Printf("Creating npc template with base stats: %+v", template.BaseStats)
	err := dao.db.QueryRow(
		`INSERT INTO npc_templates (name, description, base_stats, armor_class, max_hp) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP,
	).Scan(&id)
	return id, err
}

func (dao *npcTemplateDAOImpl) Update(template NpcTemplate) error {
	_, err := dao.db.Exec(
		`UPDATE npc_templates SET name = $1, description = $2, base_stats = $3, armor_class = $4, max_hp = $5 WHERE id = $6`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP, template.ID,
	)
	return err
}

func (dao *npcTemplateDAOImpl) Delete(id int) error {
	_, err := dao.db.Exec(`DELETE FROM npc_templates WHERE id = $1`, id)
	return err
}

func (dao *npcTemplateDAOImpl) AddCharacterToEncounterFromTemplate(templateID int, encounterID int) (Character, error) {
	t, err := dao.GetByID(templateID)
	if err != nil {
		return Character{}, err
	}
	encounterDAO := NewEncounterDAO(dao.db)
	encounter, err := encounterDAO.GetByID(encounterID)
	if err != nil {
		return Character{}, err
	}
	character := Character{
		Name:          t.Name,
		ArmorClass:    t.ArmorClass,
		ToHitModifier: 0,
		MaxHP:         t.MaxHP,
		CurrentHP:     t.MaxHP,
		Initiative:    0,
		OwnerID:       encounter.OwnerID,
		NpcTemplateID: &t.ID,
		// Add other fields as needed
	}
	characterDAO := NewCharacterDAO(dao.db)
	newID, err := characterDAO.CreateCharacter(character)
	if err != nil {
		return Character{}, err
	}
	character.ID = newID

	err = encounterDAO.AddCharacterToEncounter(encounterID, newID)
	if err != nil {
		return Character{}, err
	}

	return character, nil
}
