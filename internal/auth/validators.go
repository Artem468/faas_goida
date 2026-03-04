package auth

import (
	"regexp"
	"strings"
)

var hasLetter = regexp.MustCompile(`[a-zA-Z]`)
var hasDigit = regexp.MustCompile(`[0-9]`)
var hasSpecial = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>/?]`)

func validEmail(email string) bool {
	if len(email) < 3 {
		return false
	}
	if !strings.Contains(email, "@") {
		return false
	}
	return true
}

func validPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	_hasLetter := hasLetter.MatchString(password)
	_hasDigit := hasDigit.MatchString(password)
	_hasSpecial := hasSpecial.MatchString(password)

	return _hasLetter && _hasDigit && _hasSpecial
}
