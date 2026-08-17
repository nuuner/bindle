package unlock

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

const secret = "correct horse battery staple"

func TestIssuedTokenIsValid(t *testing.T) {
	token := IssueToken(secret, time.Now().Add(time.Hour))

	if !TokenIsValid(secret, token, time.Now()) {
		t.Error("expected a freshly issued token to validate")
	}
}

func TestTokenIsRejectedAfterExpiry(t *testing.T) {
	token := IssueToken(secret, time.Now().Add(time.Hour))

	if TokenIsValid(secret, token, time.Now().Add(2*time.Hour)) {
		t.Error("expected an expired token to be rejected")
	}
}

// The whole point of signing the expiry: a client that edits the cookie to postpone its
// own expiry produces a value the server never signed.
func TestForgedExpiryIsRejected(t *testing.T) {
	token := IssueToken(secret, time.Now().Add(time.Hour))
	_, signature, _ := strings.Cut(token, ".")

	farFuture := strconv.FormatInt(time.Now().Add(100*time.Hour).Unix(), 10)
	forged := farFuture + "." + signature

	if TokenIsValid(secret, forged, time.Now()) {
		t.Error("expected a token with a rewritten expiry to be rejected")
	}
}

// Keying the MAC with the password is what makes changing UNLOCK_PASSWORD revoke every
// token already handed out.
func TestTokenIsRejectedAfterPasswordChange(t *testing.T) {
	token := IssueToken(secret, time.Now().Add(time.Hour))

	if TokenIsValid("a different password", token, time.Now()) {
		t.Error("expected a token signed with the old password to be rejected")
	}
}

func TestMalformedTokensAreRejected(t *testing.T) {
	malformed := []string{
		"",
		"not-a-token",
		"1.",
		".abcdef",
		"notanumber.abcdef",
		IssueToken(secret, time.Now().Add(time.Hour)) + "extra",
	}

	for _, token := range malformed {
		if TokenIsValid(secret, token, time.Now()) {
			t.Errorf("expected %q to be rejected", token)
		}
	}
}

// An empty UnlockPassword disables the feature, so no cookie may unlock anything -
// including the token an empty secret would itself produce.
func TestEmptySecretUnlocksNothing(t *testing.T) {
	if TokenIsValid("", IssueToken("", time.Now().Add(time.Hour)), time.Now()) {
		t.Error("expected an unconfigured password to reject every token")
	}
	if PasswordMatches("", "") {
		t.Error("expected an unconfigured password to match nothing")
	}
}

func TestPasswordMatches(t *testing.T) {
	if !PasswordMatches(secret, secret) {
		t.Error("expected the configured password to match")
	}
	if PasswordMatches(secret, "wrong") {
		t.Error("expected a wrong password to be rejected")
	}
	if PasswordMatches(secret, secret+"x") {
		t.Error("expected a prefix of the password to be rejected")
	}
}
