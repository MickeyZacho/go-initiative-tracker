package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-initiative-tracker/dao"
)

// srdMonster is the subset of a 5e-database monster record we map onto the
// template's own columns. Everything else rides along in details.
type srdMonster struct {
	Index      string  `json:"index"`
	Name       string  `json:"name"`
	Size       string  `json:"size"`
	Type       string  `json:"type"`
	Subtype    string  `json:"subtype"`
	Alignment  string  `json:"alignment"`
	CR         float64 `json:"challenge_rating"`
	HitPoints  int     `json:"hit_points"`
	ArmorClass []struct {
		Value int `json:"value"`
	} `json:"armor_class"`
	Strength     int `json:"strength"`
	Dexterity    int `json:"dexterity"`
	Constitution int `json:"constitution"`
	Intelligence int `json:"intelligence"`
	Wisdom       int `json:"wisdom"`
	Charisma     int `json:"charisma"`
}

type importedMonster struct {
	Key      string
	Template dao.NpcTemplate
	Details  []byte
}

func parseMonsters(raw []byte) ([]importedMonster, error) {
	var records []json.RawMessage
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, fmt.Errorf("monsters file is not a JSON array: %w", err)
	}
	out := make([]importedMonster, 0, len(records))
	for i, rec := range records {
		var m srdMonster
		if err := json.Unmarshal(rec, &m); err != nil {
			return nil, fmt.Errorf("record %d: %w", i, err)
		}
		if m.Index == "" || m.Name == "" {
			return nil, fmt.Errorf("record %d: missing index or name", i)
		}
		ac := 10
		if len(m.ArmorClass) > 0 {
			ac = m.ArmorClass[0].Value
		}
		out = append(out, importedMonster{
			Key: m.Index,
			Template: dao.NpcTemplate{
				Name:        m.Name,
				Description: describe(m),
				BaseStats: dao.StatBlock{
					Strength: m.Strength, Dexterity: m.Dexterity, Constitution: m.Constitution,
					Intelligence: m.Intelligence, Wisdom: m.Wisdom, Charisma: m.Charisma,
				},
				ArmorClass: ac,
				MaxHP:      m.HitPoints,
			},
			Details: rec,
		})
	}
	return out, nil
}

// describe builds a one-line summary like
// "Small humanoid (goblinoid), neutral evil, CR 1/4".
func describe(m srdMonster) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(m.Size + " " + m.Type))
	if m.Subtype != "" {
		fmt.Fprintf(&b, " (%s)", m.Subtype)
	}
	if m.Alignment != "" {
		fmt.Fprintf(&b, ", %s", m.Alignment)
	}
	fmt.Fprintf(&b, ", CR %s", formatCR(m.CR))
	return b.String()
}

func formatCR(cr float64) string {
	switch cr {
	case 0.125:
		return "1/8"
	case 0.25:
		return "1/4"
	case 0.5:
		return "1/2"
	}
	return fmt.Sprintf("%g", cr)
}
