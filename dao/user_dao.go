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
	row := dao.db.QueryRow(`SELECT id, discord_id, username, discriminator, avatar FROM users WHERE discord_id = $1`, discordID)
	err := row.Scan(&user.DiscordID, &user.Username, &user.Discriminator, &user.Avatar)
	return user, err
}
