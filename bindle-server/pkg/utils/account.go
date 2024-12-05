package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

func AccountIdIsValid(accountId string) bool {
	pattern := `^[a-zA-Z0-9]{22}$`
	matched, err := regexp.MatchString(pattern, strings.ToUpper(accountId))
	return err == nil && matched
}

func GenerateAccountId() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsLength := big.NewInt(int64(len(chars)))
	length := 22

	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsLength)
		if err != nil {
			fmt.Println("Could not get random number for generating an id")
			return ""
		}
		result[i] = chars[num.Int64()]
	}

	return string(result)
}
