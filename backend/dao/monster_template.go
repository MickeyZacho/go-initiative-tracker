package dao

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
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
	// Source is where an imported template came from ("srd-5.2"); empty for
	// templates made in the app.
	Source string
	// Type, Size and CR come from an imported template's details; empty / nil
	// for templates made in the app.
	Type string
	Size string
	CR   *float64
}

// TemplateFilter narrows a template search. Zero values mean "no filter".
// Type, Size and the CR range only match imported templates, since app-made
// ones have no details.
type TemplateFilter struct {
	Query  string
	Type   string
	Size   string
	MinCR  *float64
	MaxCR  *float64
	Source string // "", "custom" (app-made) or "srd" (any SRD import)
	SortBy string // "" / "name" or "cr"
	Limit  int
	Offset int
}

// TemplateFacets are the distinct filter values present in the data, for
// building filter dropdowns.
type TemplateFacets struct {
	Types []string
	Sizes []string
}

type NpcTemplateDAO interface {
	Search(f TemplateFilter) ([]NpcTemplate, error)
	Facets() (TemplateFacets, error)
	GetByID(id int) (NpcTemplate, error)
	Create(template NpcTemplate) (int, error)
	UpdateByOwner(template NpcTemplate, ownerID string) (bool, error)
	DeleteByOwner(id int, ownerID string) (bool, error)
	AddCharactersToEncounterFromTemplate(templateID, encounterID, count int, staging Staging, generateNames bool) ([]Character, error)
	UpsertImported(template NpcTemplate, source, sourceKey string, details []byte) (inserted bool, err error)
}

type npcTemplateDAOImpl struct {
	db *sql.DB
}

func NewNpcTemplateDAO(db *sql.DB) NpcTemplateDAO {
	return &npcTemplateDAOImpl{db: db}
}

// crExpr is an imported template's challenge rating (NULL for app-made ones).
const crExpr = `(details->>'challenge_rating')::float8`

const npcTemplateColumns = `id, name, description, base_stats, armor_class, max_hp, COALESCE(owner_id, ''), COALESCE(source, ''), ` +
	`COALESCE(details->>'type', ''), COALESCE(details->>'size', ''), ` + crExpr

// Search lists templates one page at a time. Every filter is optional; with
// none it returns everything, app-made templates first. A query matches name
// or description (case-insensitive substring), name-prefix matches first.
func (dao *npcTemplateDAOImpl) Search(f TemplateFilter) ([]NpcTemplate, error) {
	var where, order []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.Query != "" {
		p := arg(likeEscaper.Replace(f.Query))
		where = append(where, fmt.Sprintf(`(name ILIKE '%%' || %[1]s || '%%' OR description ILIKE '%%' || %[1]s || '%%')`, p))
		order = append(order, fmt.Sprintf(`(name ILIKE %s || '%%') DESC`, p))
	}
	if f.Type != "" {
		where = append(where, `lower(details->>'type') = lower(`+arg(f.Type)+`)`)
	}
	if f.Size != "" {
		where = append(where, `lower(details->>'size') = lower(`+arg(f.Size)+`)`)
	}
	if f.MinCR != nil {
		where = append(where, crExpr+` >= `+arg(*f.MinCR))
	}
	if f.MaxCR != nil {
		where = append(where, crExpr+` <= `+arg(*f.MaxCR))
	}
	switch f.Source {
	case "custom":
		where = append(where, `source IS NULL`)
	case "srd":
		where = append(where, `source LIKE 'srd-%'`)
	}
	order = append(order, `(source IS NULL) DESC`)
	if f.SortBy == "cr" {
		order = append(order, crExpr)
	}
	order = append(order, `name`, `id`)

	q := `SELECT ` + npcTemplateColumns + ` FROM npc_templates`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, ` AND `)
	}
	q += ` ORDER BY ` + strings.Join(order, `, `)
	q += ` LIMIT ` + arg(f.Limit) + ` OFFSET ` + arg(f.Offset)

	rows, err := dao.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []NpcTemplate
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanTemplate(r rowScanner) (NpcTemplate, error) {
	var t NpcTemplate
	var statsStr string
	var cr sql.NullFloat64
	if err := r.Scan(&t.ID, &t.Name, &t.Description, &statsStr, &t.ArmorClass, &t.MaxHP, &t.OwnerID, &t.Source, &t.Type, &t.Size, &cr); err != nil {
		return t, err
	}
	stats, err := parseStatBlock(statsStr)
	if err != nil {
		return t, fmt.Errorf("failed to parse stat_block: %w", err)
	}
	t.BaseStats = stats
	if cr.Valid {
		t.CR = &cr.Float64
	}
	return t, nil
}

// Facets returns the distinct creature types and sizes among imported
// templates, lower-cased and sorted, for filter dropdowns.
func (dao *npcTemplateDAOImpl) Facets() (TemplateFacets, error) {
	f := TemplateFacets{Types: []string{}, Sizes: []string{}}
	rows, err := dao.db.Query(
		`SELECT DISTINCT 'type', lower(details->>'type') FROM npc_templates WHERE details->>'type' <> ''
		 UNION
		 SELECT DISTINCT 'size', lower(details->>'size') FROM npc_templates WHERE details->>'size' <> ''
		 ORDER BY 1, 2`,
	)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, v string
		if err := rows.Scan(&kind, &v); err != nil {
			return f, err
		}
		if kind == "type" {
			f.Types = append(f.Types, v)
		} else {
			f.Sizes = append(f.Sizes, v)
		}
	}
	return f, rows.Err()
}

// likeEscaper neutralises LIKE wildcards in user input (Postgres' default
// escape character is backslash).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (dao *npcTemplateDAOImpl) GetByID(id int) (NpcTemplate, error) {
	return scanTemplate(dao.db.QueryRow(`SELECT `+npcTemplateColumns+` FROM npc_templates WHERE id = $1`, id))
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
// have a NULL owner and so are read-only to signed-in users; imported
// templates (source set) are read-only to everyone, logged in or not.
func (dao *npcTemplateDAOImpl) UpdateByOwner(template NpcTemplate, ownerID string) (bool, error) {
	result, err := dao.db.Exec(
		`UPDATE npc_templates SET name = $1, description = $2, base_stats = $3, armor_class = $4, max_hp = $5 WHERE id = $6 AND COALESCE(owner_id, '') = $7 AND source IS NULL`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP, template.ID, ownerID,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (dao *npcTemplateDAOImpl) DeleteByOwner(id int, ownerID string) (bool, error) {
	result, err := dao.db.Exec(`DELETE FROM npc_templates WHERE id = $1 AND COALESCE(owner_id, '') = $2 AND source IS NULL`, id, ownerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// UpsertImported inserts or refreshes a shared (owner-less) template that came
// from an external source, keyed by (source, sourceKey). details is the raw
// source record stored as JSONB. Returns true when a new row was inserted and
// false when an existing one was updated (xmax = 0 only for fresh inserts).
func (dao *npcTemplateDAOImpl) UpsertImported(template NpcTemplate, source, sourceKey string, details []byte) (bool, error) {
	var inserted bool
	err := dao.db.QueryRow(
		`INSERT INTO npc_templates (name, description, base_stats, armor_class, max_hp, details, source, source_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (source, source_key) WHERE source IS NOT NULL DO UPDATE SET
		   name = EXCLUDED.name, description = EXCLUDED.description, base_stats = EXCLUDED.base_stats,
		   armor_class = EXCLUDED.armor_class, max_hp = EXCLUDED.max_hp, details = EXCLUDED.details
		 RETURNING (xmax = 0)`,
		template.Name, template.Description, template.BaseStats.String(), template.ArmorClass, template.MaxHP,
		string(details), source, sourceKey,
	).Scan(&inserted)
	return inserted, err
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
//   - is numbered after the highest-numbered copy of the same template
//     already in the encounter ("Goblin 3", "Goblin 4") so a pack is
//     distinguishable; a lone first copy keeps the plain template name;
//   - with generateNames, also gets a nickname that is unique in the
//     encounter ("Goblin 3 · Snagtooth"), see nicknamePool;
//   - starts at full HP with initiative rolled as d20 + DEX modifier.
func (dao *npcTemplateDAOImpl) AddCharactersToEncounterFromTemplate(templateID, encounterID, count int, staging Staging, generateNames bool) ([]Character, error) {
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
	// Everyone already in the encounter: copies of this template set the next
	// number, and every name's nickname is taken.
	rows, err := tx.Query(
		"SELECT c.name, COALESCE(c.npc_template_id, 0) FROM characters c JOIN encounter_characters ec ON ec.character_id = c.id WHERE ec.encounter_id = $1",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	var allNames, copyNames []string
	for rows.Next() {
		var name string
		var tid int
		if err := rows.Scan(&name, &tid); err != nil {
			rows.Close()
			return nil, err
		}
		allNames = append(allNames, name)
		if tid == templateID {
			copyNames = append(copyNames, name)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	highest := highestCopyNumber(t.Name, copyNames)
	used := usedNicknames(allNames)
	pool := nicknamePool(t.Name, t.Type)

	dexMod := AbilityModifier(t.BaseStats.Dexterity)
	created := make([]Character, 0, count)
	for i := 1; i <= count; i++ {
		name := t.Name
		if len(copyNames) > 0 || count > 1 {
			name = fmt.Sprintf("%s %d", t.Name, highest+i)
		}
		if generateNames {
			name += NicknameSeparator + pickNickname(pool, used)
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
