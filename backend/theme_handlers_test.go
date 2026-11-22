package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserThemeGetLoggedOutIsNull(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	rr, req := getReq("/user/theme")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusOK)
	if got := strings.TrimSpace(rr.Body.String()); got != `{"loggedIn":false,"theme":null}` {
		t.Errorf("body = %s", got)
	}
	assertMet(t, m)
}

func TestUserThemeGetReturnsSaved(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectQuery("SELECT theme FROM users").WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"theme"}).AddRow(`{"borderRadius": 20}`))
	rr, req := getReq("/user/theme")
	authed(req, "u1")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusOK)
	if got := strings.TrimSpace(rr.Body.String()); got != `{"loggedIn":true,"theme":{"borderRadius":20}}` {
		t.Errorf("body = %s", got)
	}
	assertMet(t, m)
}

func TestUserThemeSave(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("UPDATE users SET theme").WithArgs("u1", `{"spacing":6}`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr, req := postJSON("/user/theme", `{"theme":{"spacing":6}}`)
	authed(req, "u1")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestUserThemeResetStoresNull(t *testing.T) {
	m, restore := newFullMock(t)
	defer restore()
	m.ExpectExec("UPDATE users SET theme").WithArgs("u1", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr, req := postJSON("/user/theme", `{"theme":null}`)
	authed(req, "u1")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertMet(t, m)
}

func TestUserThemeSaveRejects(t *testing.T) {
	_, restore := newFullMock(t)
	defer restore()

	rr, req := postJSON("/user/theme", `{"theme":{"spacing":6}}`)
	apiUserThemeHandler(rr, req) // logged out
	assertStatus(t, rr, http.StatusUnauthorized)

	rr, req = postJSON("/user/theme", `{"theme":"red"}`)
	authed(req, "u1")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)

	rr, req = postJSON("/user/theme", `{"theme":{"customCss":"`+strings.Repeat("a", maxThemeBytes)+`"}}`)
	authed(req, "u1")
	apiUserThemeHandler(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
}
