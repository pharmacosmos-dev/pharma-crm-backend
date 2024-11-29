package utils

import (
	"fmt"
	"math/rand"
)

// GenerateCode generates a 6-digit code where digits can repeat and leading zeros are allowed.
func GenerateCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		code += fmt.Sprintf("%d", rand.Intn(10)) // Generate random digit (0-9)
	}
	return code
}
