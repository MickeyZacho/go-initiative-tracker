package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"go-initiative-tracker/dao"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

func apiMeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	username := ""
	avatar := ""
	discordID := getDiscordIDFromRequest(r)
	if cookie, err := r.Cookie("discord_user"); err == nil {
		username = cookie.Value
	}
	if cookie, err := r.Cookie("discord_avatar"); err == nil {
		avatar = cookie.Value
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"loggedIn":  username != "" && discordID != "",
		"username":  username,
		"discordID": discordID,
		"avatar":    avatar,
	})
}

func generateState() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "state"
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Redirects user to Discord's OAuth2 login page
func discordLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	// Store state in a cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
	})
	url := discordOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Handles Discord's callback and retrieves user info
func discordCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "No code in request", http.StatusBadRequest)
		return
	}
	// Validate state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		log.Printf("OAuth callback rejected: invalid state parameter")
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	token, err := discordOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("OAuth token exchange failed: %v", err)
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	client := discordOAuthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var userInfo struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
		Avatar        string `json:"avatar"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_user",
		Value:    userInfo.Username + "#" + userInfo.Discriminator,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_id",
		Value:    signValue(userInfo.ID),
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_avatar",
		Value:    userInfo.Avatar,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	// Save user to database
	user := dao.User{
		DiscordID:     userInfo.ID,
		Username:      userInfo.Username,
		Discriminator: userInfo.Discriminator,
		Avatar:        userInfo.Avatar,
	}
	err = dao.NewUserDAO(db).UpsertUser(user)
	if err != nil {
		http.Error(w, "Failed to save user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_user",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "discord_avatar",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}
