package dao

import (
	"database/sql"
)

type EncounterLedgerEntry struct {
	ID          int    `json:"id"`
	EncounterID int    `json:"encounter_id"`
	ActorID     int    `json:"actor_id"`
	ActorName   string `json:"actor_name"`
	TargetID    int    `json:"target_id"`
	TargetName  string `json:"target_name"`
	ActionType  string `json:"action_type"`
	HPChange    int    `json:"hp_change"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type EncounterLedgerInsert struct {
	EncounterID int
	ActorID     int
	TargetID    int
	ActionType  string
	HPChange    int
	Description string
}

type EncounterLedgerDAO interface {
	ListByEncounterID(encounterID int, limit int) ([]EncounterLedgerEntry, error)
	Create(entry EncounterLedgerInsert) (EncounterLedgerEntry, error)
}

type encounterLedgerDAOImpl struct {
	db *sql.DB
}

func NewEncounterLedgerDAO(db *sql.DB) EncounterLedgerDAO {
	return &encounterLedgerDAOImpl{db: db}
}

func (dao *encounterLedgerDAOImpl) ListByEncounterID(encounterID int, limit int) ([]EncounterLedgerEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := dao.db.Query(
		`SELECT l.id, l.encounter_id, COALESCE(l.actor_id, 0), COALESCE(a.name, ''), COALESCE(l.target_id, 0), COALESCE(t.name, ''), COALESCE(l.action_type, ''), COALESCE(l.hp_change, 0), COALESCE(l.description, ''), TO_CHAR(l.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM encounter_ledger l LEFT JOIN characters a ON a.id = l.actor_id LEFT JOIN characters t ON t.id = l.target_id WHERE l.encounter_id = $1 ORDER BY l.created_at DESC, l.id DESC LIMIT $2`,
		encounterID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []EncounterLedgerEntry{}
	for rows.Next() {
		var entry EncounterLedgerEntry
		if scanErr := rows.Scan(
			&entry.ID,
			&entry.EncounterID,
			&entry.ActorID,
			&entry.ActorName,
			&entry.TargetID,
			&entry.TargetName,
			&entry.ActionType,
			&entry.HPChange,
			&entry.Description,
			&entry.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (dao *encounterLedgerDAOImpl) Create(entry EncounterLedgerInsert) (EncounterLedgerEntry, error) {
	created := EncounterLedgerEntry{}
	row := dao.db.QueryRow(
		`WITH inserted AS (INSERT INTO encounter_ledger (encounter_id, actor_id, target_id, action_type, hp_change, description) VALUES ($1, NULLIF($2, 0), NULLIF($3, 0), $4, $5, $6) RETURNING id, encounter_id, actor_id, target_id, action_type, hp_change, description, created_at) SELECT i.id, i.encounter_id, COALESCE(i.actor_id, 0), COALESCE(a.name, ''), COALESCE(i.target_id, 0), COALESCE(t.name, ''), COALESCE(i.action_type, ''), COALESCE(i.hp_change, 0), COALESCE(i.description, ''), TO_CHAR(i.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM inserted i LEFT JOIN characters a ON a.id = i.actor_id LEFT JOIN characters t ON t.id = i.target_id`,
		entry.EncounterID,
		entry.ActorID,
		entry.TargetID,
		entry.ActionType,
		entry.HPChange,
		entry.Description,
	)
	if err := row.Scan(
		&created.ID,
		&created.EncounterID,
		&created.ActorID,
		&created.ActorName,
		&created.TargetID,
		&created.TargetName,
		&created.ActionType,
		&created.HPChange,
		&created.Description,
		&created.CreatedAt,
	); err != nil {
		return EncounterLedgerEntry{}, err
	}
	return created, nil
}
