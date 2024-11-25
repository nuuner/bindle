package utils

import (
	"regexp"
	"strings"
)

func AccountIdIsValid(accountId string) bool {
	pattern := `^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`
	matched, err := regexp.MatchString(pattern, strings.ToUpper(accountId))
	return err == nil && matched
}
