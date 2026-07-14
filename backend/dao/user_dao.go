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
