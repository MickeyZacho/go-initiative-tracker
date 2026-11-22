package main

import (
	"encoding/json"
	"testing"
)

const sample = `[{
  "index": "goblin", "name": "Goblin", "size": "Small", "type": "humanoid",
  "subtype": "goblinoid", "alignment": "neutral evil", "challenge_rating": 0.25,
  "armor_class": [{"type": "armor", "value": 15}], "hit_points": 7,
  "strength": 8, "dexterity": 14, "constitution": 10, "intelligence": 10, "wisdom": 8, "charisma": 8,
  "actions": [{"name": "Scimitar"}]
}]`

func TestParseMonsters(t *testing.T) {
	ms, err := parseMonsters([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("got %d monsters", len(ms))
	}
	m := ms[0]
	if m.Key != "goblin" || m.Template.Name != "Goblin" {
		t.Errorf("key/name = %q/%q", m.Key, m.Template.Name)
	}
	if m.Template.ArmorClass != 15 || m.Template.MaxHP != 7 || m.Template.BaseStats.Dexterity != 14 {
		t.Errorf("unexpected template %+v", m.Template)
	}
	if want := "Small humanoid (goblinoid), neutral evil, CR 1/4"; m.Template.Description != want {
		t.Errorf("description = %q, want %q", m.Template.Description, want)
	}
	var details map[string]any
	if err := json.Unmarshal(m.Details, &details); err != nil || details["actions"] == nil {
		t.Errorf("details should keep the full record, got %s", m.Details)
	}
}

func TestParseMonstersRejectsMissingIndex(t *testing.T) {
	if _, err := parseMonsters([]byte(`[{"name": "Nameless"}]`)); err == nil {
		t.Fatal("expected error for record without index")
	}
}
