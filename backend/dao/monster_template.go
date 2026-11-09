package dao

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"
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
	OwnerID     string
}

type NpcTemplateDAO interface {
	GetAll() ([]NpcTemplate, error)
	GetByID(id int) (NpcTemplate, error)
	Create(template NpcTemplate) (int, error)
	UpdateByOwner(template NpcTemplate, ownerID string) (bool, error)
	DeleteByOwner(id int, ownerID string) (bool, error)
	AddCharactersToEncounterFromTemplate(templateID, encounterID, count int, staging Staging) ([]Character, error)
}

type npcTemplateDAOImpl struct {
	db *sql.DB
}

func NewNpcTemplateDAO(db *sql.DB) NpcTemplateDAO {
	return &npcTemplateDAOImpl{db: db}
}

func (dao *npcTemplateDAOImpl) GetAll() ([]NpcTemplate, error) {
	rows, err := dao.db.Query(`SELECT id, name, description, base_stats, armor_class, max_hp, COALESCE(owner_id, '') FROM npc_templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []NpcTemplate
	for rows.Next() {
		var t NpcTemplate
		var statsStr string
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP, &t.OwnerID); err != nil {
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
	err := dao.db.QueryRow(`SELECT id, name, description, base_stats, armor_class, max_hp, COALESCE(owner_id, '') FROM npc_templates WHERE id = $1`, id).Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP, &t.OwnerID)
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
		`INSERT INTO npc_templates (name, description, base_stats, armor_class, max_hp, owner_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP, template.OwnerID,
	).Scan(&id)
	return id, err
}

// UpdateByOwner updates a template only when it belongs to ownerID; owner_id is
// not in the SET clause so ownership can never be reassigned. Returns false when
// no row matched (wrong id or not owned by the caller). Shared seed templates
// have a NULL owner and so are read-only to signed-in users.
func (dao *npcTemplateDAOImpl) UpdateByOwner(template NpcTemplate, ownerID string) (bool, error) {
	result, err := dao.db.Exec(
		`UPDATE npc_templates SET name = $1, description = $2, base_stats = $3, armor_class = $4, max_hp = $5 WHERE id = $6 AND COALESCE(owner_id, '') = $7`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP, template.ID, ownerID,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (dao *npcTemplateDAOImpl) DeleteByOwner(id int, ownerID string) (bool, error) {
	result, err := dao.db.Exec(`DELETE FROM npc_templates WHERE id = $1 AND COALESCE(owner_id, '') = $2`, id, ownerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// RollD20 is the die used for NPC initiative. A package variable so tests can
// make rolls deterministic.
var RollD20 = func() int { return rand.IntN(20) + 1 }

// AbilityModifier is the 5e ability modifier for a score: floor((score-10)/2).
func AbilityModifier(score int) int {
	d := score - 10
	if d < 0 {
		return -((-d + 1) / 2)
	}
	return d / 2
}

// MaxTemplateCopies caps how many creatures one "add NPC" call may create.
const MaxTemplateCopies = 20

// AddCharactersToEncounterFromTemplate creates count creatures from a template
// and drops them into the encounter, in one transaction. Each copy:
//   - is owned by the encounter's owner (the DM), like any template NPC;
//   - is numbered after any copies of the same template already in the
//     encounter ("Goblin 3", "Goblin 4") so a pack is distinguishable; a lone
//     first copy keeps the plain template name;
//   - starts at full HP with initiative rolled as d20 + DEX modifier.
func (dao *npcTemplateDAOImpl) AddCharactersToEncounterFromTemplate(templateID, encounterID, count int, staging Staging) ([]Character, error) {
	if count < 1 || count > MaxTemplateCopies {
		return nil, fmt.Errorf("count must be between 1 and %d", MaxTemplateCopies)
	}
	t, err := dao.GetByID(templateID)
	if err != nil {
		return nil, err
	}

	tx, err := dao.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var ownerID string
	if err := tx.QueryRow("SELECT COALESCE(owner_id, '') FROM encounters WHERE id = $1", encounterID).Scan(&ownerID); err != nil {
		return nil, err
	}
	var existing int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM characters c JOIN encounter_characters ec ON ec.character_id = c.id WHERE ec.encounter_id = $1 AND c.npc_template_id = $2",
		encounterID, templateID,
	).Scan(&existing); err != nil {
		return nil, err
	}

	dexMod := AbilityModifier(t.BaseStats.Dexterity)
	created := make([]Character, 0, count)
	for i := 1; i <= count; i++ {
		name := t.Name
		if existing > 0 || count > 1 {
			name = fmt.Sprintf("%s %d", t.Name, existing+i)
		}
		c := Character{
			Name:          name,
			ArmorClass:    t.ArmorClass,
			MaxHP:         t.MaxHP,
			CurrentHP:     t.MaxHP,
			Initiative:    RollD20() + dexMod,
			OwnerID:       ownerID,
			Type:          "npc",
			NpcTemplateID: &t.ID,
			Hidden:        staging.Hidden,
			RevealRound:   staging.RevealRound,
		}
		if err := tx.QueryRow(
			"INSERT INTO characters (name, armor_class, to_hit_modifier, max_hp, owner_id, type, npc_template_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
			c.Name, c.ArmorClass, c.ToHitModifier, c.MaxHP, c.OwnerID, c.Type, t.ID,
		).Scan(&c.ID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(
			"INSERT INTO encounter_characters (encounter_id, character_id, initiative, current_hp, hidden, reveal_round) VALUES ($1, $2, $3, $4, $5, $6)",
			encounterID, c.ID, c.Initiative, c.CurrentHP, staging.Hidden, staging.RevealRound,
		); err != nil {
			return nil, err
		}
		created = append(created, c)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}
