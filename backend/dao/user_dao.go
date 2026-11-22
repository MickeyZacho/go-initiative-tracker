package dao

import (
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type User struct {
	Username      string
	DiscordID     string
	Discriminator string
	Avatar        string
}

type UserDAO struct {
	db *sql.DB
}

func NewUserDAO(db *sql.DB) UserDAO {
	return UserDAO{db: db}
}

func (dao UserDAO) UpsertUser(user User) error {
	_, err := dao.db.Exec(`
		INSERT INTO users (discord_id, username, discriminator, avatar)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (discord_id) DO UPDATE
		SET username = EXCLUDED.username,
		    discriminator = EXCLUDED.discriminator,
		    avatar = EXCLUDED.avatar
	`, user.DiscordID, user.Username, user.Discriminator, user.Avatar)
	return err
}

func (dao UserDAO) GetUserByDiscordID(discordID string) (User, error) {
	var user User
	row := dao.db.QueryRow(`SELECT discord_id, username, discriminator, avatar FROM users WHERE discord_id = $1`, discordID)
	err := row.Scan(&user.DiscordID, &user.Username, &user.Discriminator, &user.Avatar)
	return user, err
}

// GetUserByUsername resolves a Discord username to a user row. Only users who
// have logged into this app exist in the users table, so this returns
// sql.ErrNoRows for anyone who has never signed in. Usernames are unique under
// Discord's new (discriminator-less) system.
func (dao UserDAO) GetUserByUsername(username string) (User, error) {
	var user User
	row := dao.db.QueryRow(`SELECT discord_id, username, discriminator, avatar FROM users WHERE username = $1`, username)
	err := row.Scan(&user.DiscordID, &user.Username, &user.Discriminator, &user.Avatar)
	return user, err
}

// GetTheme returns the user's saved theme JSON, or nil when they have none
// (or have no users row yet).
func (dao UserDAO) GetTheme(discordID string) ([]byte, error) {
	var theme sql.NullString
	err := dao.db.QueryRow(`SELECT theme FROM users WHERE discord_id = $1`, discordID).Scan(&theme)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !theme.Valid {
		return nil, nil
	}
	return []byte(theme.String), nil
}

// SetTheme stores the user's theme JSON; nil clears it back to the default.
// Returns false when the user has no row (never logged in).
func (dao UserDAO) SetTheme(discordID string, theme []byte) (bool, error) {
	var value any
	if theme != nil {
		value = string(theme)
	}
	res, err := dao.db.Exec(`UPDATE users SET theme = $2 WHERE discord_id = $1`, discordID, value)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
