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

type MonsterTemplate struct {
	ID          int
	Name        string
	Description string
	BaseStats   StatBlock
	ArmorClass  int
	MaxHP       int
}

type MonsterTemplateDAO interface {
	GetAll() ([]MonsterTemplate, error)
	GetByID(id int) (MonsterTemplate, error)
	Create(template MonsterTemplate) (int, error)
	Update(template MonsterTemplate) error
	Delete(id int) error
	AddCharacterToEncounterFromTemplate(templateID int, encounterID int) (Character, error)
}

type monsterTemplateDAOImpl struct {
	db *sql.DB
}

func NewMonsterTemplateDAO(db *sql.DB) MonsterTemplateDAO {
	return &monsterTemplateDAOImpl{db: db}
}

func (dao *monsterTemplateDAOImpl) GetAll() ([]MonsterTemplate, error) {
	rows, err := dao.db.Query(`SELECT id, name, description, base_stats, armor_class, max_hp FROM monster_templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []MonsterTemplate
	for rows.Next() {
		var t MonsterTemplate
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

func (dao *monsterTemplateDAOImpl) GetByID(id int) (MonsterTemplate, error) {
	var t MonsterTemplate
	var statsStr string
	err := dao.db.QueryRow(`SELECT id, name, description, base_stats, armor_class, max_hp FROM monster_templates WHERE id = $1`, id).Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP)
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

func (dao *monsterTemplateDAOImpl) Create(template MonsterTemplate) (int, error) {
	var id int
	log.Printf("Creating monster template with base stats: %+v", template.BaseStats)
	err := dao.db.QueryRow(
		`INSERT INTO monster_templates (name, description, base_stats, armor_class, max_hp) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP,
	).Scan(&id)
	return id, err
}

func (dao *monsterTemplateDAOImpl) Update(template MonsterTemplate) error {
	_, err := dao.db.Exec(
		`UPDATE monster_templates SET name = $1, description = $2, base_stats = $3, armor_class = $4, max_hp = $5 WHERE id = $6`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP, template.ID,
	)
	return err
}

func (dao *monsterTemplateDAOImpl) Delete(id int) error {
	_, err := dao.db.Exec(`DELETE FROM monster_templates WHERE id = $1`, id)
	return err
}

func (dao *monsterTemplateDAOImpl) AddCharacterToEncounterFromTemplate(templateID int, encounterID int) (Character, error) {
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
		Name:              t.Name,
		ArmorClass:        t.ArmorClass,
		ToHitModifier:     0,
		MaxHP:             t.MaxHP,
		CurrentHP:         t.MaxHP,
		Initiative:        0,
		OwnerID:           encounter.OwnerID,
		MonsterTemplateID: &t.ID,
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
