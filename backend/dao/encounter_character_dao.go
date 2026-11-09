package dao

import (
	"database/sql"
	"fmt"
)

type EncounterCharacter struct {
	EncounterID  int
	CharacterID  int
	Initiative   int
	CurrentHP    int
	IsActive     bool
	ReactionUsed bool
	TempHP       int
	// Death saving throws (0-3 each).
	DeathSaveSuccesses int
	DeathSaveFailures  int
}

type EncounterCharacterDAO interface {
	GetByEncounterAndCharacter(encounterID, characterID int) (EncounterCharacter, error)
	Update(enc EncounterCharacter) error
	Upsert(enc EncounterCharacter) error
	StartCombat(encounterID int) (int, error)
	ResetCombat(encounterID int) error
	// GetRound returns the encounter's combat round (0 = combat not started).
	GetRound(encounterID int) (int, error)
	AdvanceTurn(encounterID int) (int, error)
	SetActiveCharacter(encounterID, characterID int) error
	SetReactionUsed(encounterID, characterID int, used bool) error
	// RollNpcInitiative re-rolls initiative (d20 + DEX modifier) for every NPC
	// in the encounter and returns how many were rolled. PCs roll their own.
	RollNpcInitiative(encounterID int) (int, error)
	// SetDeathSaves records a downed PC's death saving throws (0-3 each).
	SetDeathSaves(encounterID, characterID, successes, failures int) error
}

type encounterCharacterDAOImpl struct {
	db *sql.DB
}

func NewEncounterCharacterDAO(db *sql.DB) EncounterCharacterDAO {
	return &encounterCharacterDAOImpl{db: db}
}

func (dao *encounterCharacterDAOImpl) GetByEncounterAndCharacter(encounterID, characterID int) (EncounterCharacter, error) {
	var ec EncounterCharacter
	err := dao.db.QueryRow(
		"SELECT encounter_id, character_id, initiative, current_hp, is_active, reaction_used, temp_hp, death_save_successes, death_save_failures FROM encounter_characters WHERE encounter_id = $1 AND character_id = $2",
		encounterID, characterID,
	).Scan(&ec.EncounterID, &ec.CharacterID, &ec.Initiative, &ec.CurrentHP, &ec.IsActive, &ec.ReactionUsed, &ec.TempHP, &ec.DeathSaveSuccesses, &ec.DeathSaveFailures)
	return ec, err
}

func (dao *encounterCharacterDAOImpl) Update(enc EncounterCharacter) error {
	_, err := dao.db.Exec(
		"UPDATE encounter_characters SET initiative = $1, current_hp = $2, is_active = $3 WHERE encounter_id = $4 AND character_id = $5",
		enc.Initiative, enc.CurrentHP, enc.IsActive, enc.EncounterID, enc.CharacterID,
	)
	return err
}

func (dao *encounterCharacterDAOImpl) Upsert(enc EncounterCharacter) error {
	_, err := dao.db.Exec(
		"INSERT INTO encounter_characters (encounter_id, character_id, initiative, current_hp, is_active, temp_hp) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (encounter_id, character_id) DO UPDATE SET initiative = EXCLUDED.initiative, current_hp = EXCLUDED.current_hp, is_active = EXCLUDED.is_active, temp_hp = EXCLUDED.temp_hp, death_save_successes = CASE WHEN EXCLUDED.current_hp > 0 THEN 0 ELSE encounter_characters.death_save_successes END, death_save_failures = CASE WHEN EXCLUDED.current_hp > 0 THEN 0 ELSE encounter_characters.death_save_failures END",
		enc.EncounterID, enc.CharacterID, enc.Initiative, enc.CurrentHP, enc.IsActive, enc.TempHP,
	)
	return err
}

func (dao *encounterCharacterDAOImpl) StartCombat(encounterID int) (int, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	order, err := loadTurnOrder(tx, encounterID)
	if err != nil {
		return 0, err
	}
	// The first creature still in the fight takes the opening turn.
	first, _ := nextTurnIndex(order, -1)
	activeID := order[first].id

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, activeID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec("UPDATE encounters SET round = 1 WHERE id = $1", encounterID); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return activeID, nil
}

// ResetCombat ends combat: nobody holds the turn and the round clock returns to
// 0, so the next Start/Next begins again at round 1.
func (dao *encounterCharacterDAOImpl) ResetCombat(encounterID int) error {
	tx, err := dao.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE encounters SET round = 0 WHERE id = $1", encounterID); err != nil {
		return err
	}
	return tx.Commit()
}

func (dao *encounterCharacterDAOImpl) GetRound(encounterID int) (int, error) {
	var round int
	err := dao.db.QueryRow("SELECT round FROM encounters WHERE id = $1", encounterID).Scan(&round)
	return round, err
}

// SetActiveCharacter makes a single character the active turn holder, clearing
// is_active on everyone else in the encounter. This lets a user click a
// character to take the turn so a subsequent AdvanceTurn continues from there.
func (dao *encounterCharacterDAOImpl) SetActiveCharacter(encounterID, characterID int) error {
	tx, err := dao.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return err
	}

	res, err := tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, characterID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("character not in encounter")
	}

	return tx.Commit()
}

// SetReactionUsed records whether a character has spent its reaction this round.
// The turn-advance logic clears this automatically when the creature's turn
// begins; this is the manual toggle a DM uses mid-round. Returns an error when the
// character is not part of the encounter so a stale id can't silently no-op.
func (dao *encounterCharacterDAOImpl) SetReactionUsed(encounterID, characterID int, used bool) error {
	res, err := dao.db.Exec(
		"UPDATE encounter_characters SET reaction_used = $1 WHERE encounter_id = $2 AND character_id = $3",
		used, encounterID, characterID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("character not in encounter")
	}
	return nil
}

func (dao *encounterCharacterDAOImpl) AdvanceTurn(encounterID int) (int, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	order, err := loadTurnOrder(tx, encounterID)
	if err != nil {
		return 0, err
	}

	currentActiveID := 0
	if err = tx.QueryRow(
		"SELECT COALESCE(MAX(character_id), 0) FROM encounter_characters WHERE encounter_id = $1 AND is_active = TRUE",
		encounterID,
	).Scan(&currentActiveID); err != nil {
		return 0, err
	}

	currentIndex := -1
	for idx, slot := range order {
		if slot.id == currentActiveID {
			currentIndex = idx
			break
		}
	}
	nextIndex, wrapped := nextTurnIndex(order, currentIndex)
	nextActiveID := order[nextIndex].id

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = FALSE WHERE encounter_id = $1",
		encounterID,
	); err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		"UPDATE encounter_characters SET is_active = TRUE, reaction_used = FALSE WHERE encounter_id = $1 AND character_id = $2",
		encounterID, nextActiveID,
	); err != nil {
		return 0, err
	}

	// Advance the round clock. With nobody active, this Next is effectively the
	// start of combat, so make sure we're in at least round 1. Otherwise a new
	// round begins whenever the turn wraps back to the top of the order.
	switch {
	case currentIndex < 0:
		if _, err = tx.Exec("UPDATE encounters SET round = GREATEST(round, 1) WHERE id = $1", encounterID); err != nil {
			return 0, err
		}
	case wrapped:
		if _, err = tx.Exec("UPDATE encounters SET round = round + 1 WHERE id = $1", encounterID); err != nil {
			return 0, err
		}
	}

	// Tick down the outgoing creature's timed conditions as its turn ends (when we
	// switch away from it), then clear any that have expired. Conditions with a
	// NULL duration ("until removed") are left untouched. currentActiveID is 0 on
	// the very first advance (no one active yet), so nothing is decremented then.
	if currentActiveID != 0 {
		if _, err = tx.Exec(
			"UPDATE encounter_character_conditions SET duration_rounds = duration_rounds - 1 WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL",
			encounterID, currentActiveID,
		); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(
			"DELETE FROM encounter_character_conditions WHERE encounter_id = $1 AND character_id = $2 AND duration_rounds IS NOT NULL AND duration_rounds <= 0",
			encounterID, currentActiveID,
		); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return nextActiveID, nil
}

func (dao *encounterCharacterDAOImpl) RollNpcInitiative(encounterID int) (int, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// NPCs without a template (quick-added customs) roll a flat d20.
	rows, err := tx.Query(
		"SELECT ec.character_id, COALESCE((t.base_stats).dexterity, 10) FROM encounter_characters ec JOIN characters c ON c.id = ec.character_id LEFT JOIN npc_templates t ON t.id = c.npc_template_id WHERE ec.encounter_id = $1 AND c.type = 'npc' ORDER BY ec.character_id",
		encounterID,
	)
	if err != nil {
		return 0, err
	}
	type npc struct{ id, dex int }
	var npcs []npc
	for rows.Next() {
		var n npc
		if err := rows.Scan(&n.id, &n.dex); err != nil {
			rows.Close()
			return 0, err
		}
		npcs = append(npcs, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, n := range npcs {
		if _, err := tx.Exec(
			"UPDATE encounter_characters SET initiative = $1 WHERE encounter_id = $2 AND character_id = $3",
			RollD20()+AbilityModifier(n.dex), encounterID, n.id,
		); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(npcs), nil
}

// turnSlot is one combatant in initiative order.
type turnSlot struct {
	id      int
	skipped bool // a downed NPC: dead monsters don't take turns
}

// loadTurnOrder reads the encounter's combatants in initiative order
// (initiative DESC, character_id ASC). It errors when there are none.
func loadTurnOrder(tx *sql.Tx, encounterID int) ([]turnSlot, error) {
	rows, err := tx.Query(
		"SELECT ec.character_id, c.type, COALESCE(ec.current_hp, c.max_hp) FROM encounter_characters ec JOIN characters c ON c.id = ec.character_id WHERE ec.encounter_id = $1 ORDER BY COALESCE(ec.initiative, 0) DESC, ec.character_id ASC",
		encounterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var order []turnSlot
	for rows.Next() {
		var (
			id, hp int
			typ    string
		)
		if err := rows.Scan(&id, &typ, &hp); err != nil {
			return nil, err
		}
		// PCs at 0 HP keep their turn — that's when they roll death saves.
		order = append(order, turnSlot{id: id, skipped: typ == "npc" && hp <= 0})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("no characters in encounter")
	}
	return order, nil
}

// nextTurnIndex finds the next combatant after current (-1 = before the top)
// that is still in the fight, and reports whether the search wrapped past the
// end of the order (i.e. a new round began). If everyone is skipped it falls
// back to the plain next slot so the turn still moves.
func nextTurnIndex(order []turnSlot, current int) (int, bool) {
	n := len(order)
	for step := 1; step <= n; step++ {
		pos := current + step
		if !order[pos%n].skipped {
			return pos % n, current >= 0 && pos >= n
		}
	}
	pos := current + 1
	return pos % n, current >= 0 && pos >= n
}

func (dao *encounterCharacterDAOImpl) SetDeathSaves(encounterID, characterID, successes, failures int) error {
	res, err := dao.db.Exec(
		"UPDATE encounter_characters SET death_save_successes = $1, death_save_failures = $2 WHERE encounter_id = $3 AND character_id = $4",
		successes, failures, encounterID, characterID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("character not in encounter")
	}
	return nil
}
