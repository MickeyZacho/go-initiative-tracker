package dao

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSendRequest_InsertsWhenNoneExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewFriendshipDAO(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM friendships`).
		WithArgs("a", "b").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`INSERT INTO friendships`).
		WithArgs("a", "b").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := dao.SendRequest("a", "b"); err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSendRequest_RejectsExistingPair(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewFriendshipDAO(db)

	// A row already exists in either direction; SendRequest must not insert.
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM friendships`).
		WithArgs("a", "b").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err = dao.SendRequest("a", "b")
	if !errors.Is(err, ErrFriendshipExists) {
		t.Fatalf("expected ErrFriendshipExists, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAccept_ReportsWhetherRowUpdated(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewFriendshipDAO(db)

	// Accept(addressee, requester) targets the pending row (requester -> addressee).
	mock.ExpectExec(`UPDATE friendships SET status = 'accepted'`).
		WithArgs("requester", "me").
		WillReturnResult(sqlmock.NewResult(0, 1))

	accepted, err := dao.Accept("me", "requester")
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if !accepted {
		t.Errorf("expected accepted=true when a pending row is updated")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAreFriends_TrueWhenAcceptedRowExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewFriendshipDAO(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM friendships`).
		WithArgs("a", "b").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := dao.AreFriends("a", "b")
	if err != nil {
		t.Fatalf("AreFriends returned error: %v", err)
	}
	if !ok {
		t.Errorf("expected AreFriends=true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestListFriends_ResolvesOtherSide(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}
	defer db.Close()
	dao := NewFriendshipDAO(db)

	rows := sqlmock.NewRows([]string{"discord_id", "username", "avatar"}).
		AddRow("friend1", "Ada", "hash1").
		AddRow("friend2", "Bo", "")
	mock.ExpectQuery(`FROM friendships f`).
		WithArgs("me").
		WillReturnRows(rows)

	friends, err := dao.ListFriends("me")
	if err != nil {
		t.Fatalf("ListFriends returned error: %v", err)
	}
	if len(friends) != 2 || friends[0].Username != "Ada" || friends[1].DiscordID != "friend2" {
		t.Errorf("unexpected friends: %+v", friends)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
