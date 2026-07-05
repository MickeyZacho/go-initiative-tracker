package main

import "testing"

func TestSignVerifyRoundTrip(t *testing.T) {
	sessionSecret = []byte("test-secret")

	signed := signValue("123456789")
	got, ok := verifyValue(signed)
	if !ok {
		t.Fatal("verifyValue rejected a value it just signed")
	}
	if got != "123456789" {
		t.Errorf("round-tripped value = %q, want %q", got, "123456789")
	}
}

func TestVerifyRejectsTamperedValue(t *testing.T) {
	sessionSecret = []byte("test-secret")

	signed := signValue("111")
	// Swap the value while keeping the original tag: the id an attacker wants
	// to impersonate, with a signature that no longer matches it.
	forged := "999" + signed[len("111"):]
	if _, ok := verifyValue(forged); ok {
		t.Error("verifyValue accepted a value whose signature does not match")
	}
}

func TestVerifyRejectsUnsignedAndMalformed(t *testing.T) {
	sessionSecret = []byte("test-secret")

	cases := []string{
		"",                 // empty
		"123456789",        // no tag at all (a pre-signing raw id)
		"123|not-base64!!", // tag not decodable
		"123|",             // empty tag
	}
	for _, c := range cases {
		if _, ok := verifyValue(c); ok {
			t.Errorf("verifyValue accepted malformed input %q", c)
		}
	}
}

func TestVerifyRejectsForeignSecret(t *testing.T) {
	sessionSecret = []byte("secret-a")
	signed := signValue("123")

	// A token minted under a different secret must not validate.
	sessionSecret = []byte("secret-b")
	if _, ok := verifyValue(signed); ok {
		t.Error("verifyValue accepted a token signed with a different secret")
	}
}
