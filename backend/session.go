package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"os"
	"strings"
)

// sessionSecret signs the identity cookies so a client cannot forge another
// user's discord_id. It is loaded from SESSION_SECRET; when that is unset (local
// dev) a random secret is generated, which invalidates existing logins on every
// restart. Production must set SESSION_SECRET to a stable, secret value.
var sessionSecret []byte

func initSessionSecret() {
	if s := os.Getenv("SESSION_SECRET"); s != "" {
		sessionSecret = []byte(s)
		return
	}
	sessionSecret = make([]byte, 32)
	if _, err := rand.Read(sessionSecret); err != nil {
		log.Fatalf("failed to generate a session secret: %v", err)
	}
	log.Printf("WARNING: SESSION_SECRET is not set; using an ephemeral secret. Logins will not survive a restart.")
}

// signValue returns "value|tag" where tag is the base64url HMAC-SHA256 of value.
func signValue(value string) string {
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(value))
	tag := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "|" + tag
}

// verifyValue validates a "value|tag" cookie and returns the value when the tag
// is a correct HMAC. The bool is false for missing, malformed, or forged input,
// so a caller can treat a failed check as "not authenticated".
func verifyValue(signed string) (string, bool) {
	idx := strings.LastIndex(signed, "|")
	if idx < 0 {
		return "", false
	}
	value, tag := signed[:idx], signed[idx+1:]
	got, err := base64.RawURLEncoding.DecodeString(tag)
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(value))
	want := mac.Sum(nil)
	if subtle.ConstantTimeCompare(want, got) != 1 {
		return "", false
	}
	return value, true
}
