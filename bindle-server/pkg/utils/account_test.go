package utils

import (
	"regexp"
	"testing"
)

func TestGenerateAccountId(t *testing.T) {
	acceptedRegex := regexp.MustCompile("^[a-zA-Z0-9]{22}$")

	id := GenerateAccountId()
	if !acceptedRegex.MatchString(id) {
		t.Errorf("Generated id does not match the required regex %s\n", acceptedRegex.String())
	}
}

func TestAccountIdIsValid(t *testing.T) {
	id := GenerateAccountId()
	if !AccountIdIsValid(id) {
		t.Errorf("Generated id is not valid")
	}
}
