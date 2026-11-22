package dao

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpsertImported_KeysOnSourceAndStoresDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	d := NewNpcTemplateDAO(db)

	tmpl := NpcTemplate{Name: "Goblin", Description: "Small humanoid", BaseStats: StatBlock{8, 14, 10, 10, 8, 8}, ArmorClass: 15, MaxHP: 7}
	details := []byte(`{"index":"goblin"}`)
	mock.ExpectQuery(`INSERT INTO npc_templates .* ON CONFLICT \(source, source_key\) WHERE source IS NOT NULL DO UPDATE`).
		WithArgs("Goblin", "Small humanoid", "(8,14,10,10,8,8)", 15, 7, `{"index":"goblin"}`, "srd-5.2", "goblin").
		WillReturnRows(sqlmock.NewRows([]string{"inserted"}).AddRow(true))

	inserted, err := d.UpsertImported(tmpl, "srd-5.2", "goblin", details)
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
