package dao

import (
	"database/sql"
	"errors"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ErrFriendshipExists is returned by SendRequest when a friendship or pending
// request already exists between the two users (in either direction).
var ErrFriendshipExists = errors.New("a friendship or request already exists between these users")

// Friend is a resolved view of the "other" user in a friendship or request,
// joined to the users table for display (username/avatar).
type Friend struct {
	DiscordID string `json:"discord_id"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
}

type FriendshipDAO interface {
	SendRequest(requesterID, addresseeID string) error
	Accept(addresseeID, requesterID string) (bool, error)
	Remove(userA, userB string) (bool, error)
	AreFriends(userA, userB string) (bool, error)
	ListFriends(discordID string) ([]Friend, error)
	ListIncoming(discordID string) ([]Friend, error)
	ListOutgoing(discordID string) ([]Friend, error)
}

type friendshipDAOImpl struct {
	db *sql.DB
}

func NewFriendshipDAO(db *sql.DB) FriendshipDAO {
	return &friendshipDAOImpl{db: db}
}

// SendRequest creates a pending request from requester to addressee. It refuses
// when any row already exists between the pair in either direction, so it never
// produces a duplicate or a reverse request.
func (dao *friendshipDAOImpl) SendRequest(requesterID, addresseeID string) error {
	var count int
	err := dao.db.QueryRow(
		`SELECT COUNT(*) FROM friendships
		 WHERE (requester_id = $1 AND addressee_id = $2)
		    OR (requester_id = $2 AND addressee_id = $1)`,
		requesterID, addresseeID,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrFriendshipExists
	}
	_, err = dao.db.Exec(
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'pending')`,
		requesterID, addresseeID,
	)
	return err
}

// Accept marks a pending request (requester -> addressee) as accepted. It
// reports whether a matching pending row was updated.
func (dao *friendshipDAOImpl) Accept(addresseeID, requesterID string) (bool, error) {
	result, err := dao.db.Exec(
		`UPDATE friendships SET status = 'accepted'
		 WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'`,
		requesterID, addresseeID,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// Remove deletes the relationship between the two users in either direction. It
// backs decline, cancel, and unfriend.
func (dao *friendshipDAOImpl) Remove(userA, userB string) (bool, error) {
	result, err := dao.db.Exec(
		`DELETE FROM friendships
		 WHERE (requester_id = $1 AND addressee_id = $2)
		    OR (requester_id = $2 AND addressee_id = $1)`,
		userA, userB,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// AreFriends reports whether the two users have an accepted friendship.
func (dao *friendshipDAOImpl) AreFriends(userA, userB string) (bool, error) {
	var count int
	err := dao.db.QueryRow(
		`SELECT COUNT(*) FROM friendships
		 WHERE status = 'accepted'
		   AND ((requester_id = $1 AND addressee_id = $2)
		     OR (requester_id = $2 AND addressee_id = $1))`,
		userA, userB,
	).Scan(&count)
	return count > 0, err
}

// ListFriends returns the accepted friends of discordID, resolving the "other"
// side of each relationship to its user row.
func (dao *friendshipDAOImpl) ListFriends(discordID string) ([]Friend, error) {
	rows, err := dao.db.Query(
		`SELECT other.discord_id, other.username, COALESCE(other.avatar, '')
		 FROM friendships f
		 JOIN users other ON other.discord_id =
		     CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		 WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
		 ORDER BY other.username`,
		discordID,
	)
	if err != nil {
		return nil, err
	}
	return scanFriends(rows)
}

// ListIncoming returns pending requests addressed to discordID (they sent, I
// decide), resolved to the requester's user row.
func (dao *friendshipDAOImpl) ListIncoming(discordID string) ([]Friend, error) {
	rows, err := dao.db.Query(
		`SELECT u.discord_id, u.username, COALESCE(u.avatar, '')
		 FROM friendships f
		 JOIN users u ON u.discord_id = f.requester_id
		 WHERE f.status = 'pending' AND f.addressee_id = $1
		 ORDER BY u.username`,
		discordID,
	)
	if err != nil {
		return nil, err
	}
	return scanFriends(rows)
}

// ListOutgoing returns pending requests discordID has sent, resolved to the
// addressee's user row.
func (dao *friendshipDAOImpl) ListOutgoing(discordID string) ([]Friend, error) {
	rows, err := dao.db.Query(
		`SELECT u.discord_id, u.username, COALESCE(u.avatar, '')
		 FROM friendships f
		 JOIN users u ON u.discord_id = f.addressee_id
		 WHERE f.status = 'pending' AND f.requester_id = $1
		 ORDER BY u.username`,
		discordID,
	)
	if err != nil {
		return nil, err
	}
	return scanFriends(rows)
}

func scanFriends(rows *sql.Rows) ([]Friend, error) {
	defer rows.Close()
	var friends []Friend
	for rows.Next() {
		var f Friend
		if err := rows.Scan(&f.DiscordID, &f.Username, &f.Avatar); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}
