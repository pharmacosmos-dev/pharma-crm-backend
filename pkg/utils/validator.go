package utils

import (
	"regexp"
)

// Phone number validator for Uzbekistan phone numbers
func IsValidPhone(phone string) bool {

	// Compile the regular expression
	re := regexp.MustCompile(`^998[1-9][0-9]\d{7}$`)

	// Check if the phone number matches the pattern
	return re.MatchString(phone)
}
