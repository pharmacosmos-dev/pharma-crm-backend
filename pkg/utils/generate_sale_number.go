package utils

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// GenerateCode generates a 6-digit code where digits can repeat and leading zeros are allowed.
func GenerateCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		code += fmt.Sprintf("%d", rand.Intn(10)) // Generate random digit (0-9)
	}
	return code
}

var (
	generatedCodes = make(map[int]bool) // Store generated codes
	mu             sync.Mutex           // Mutex for concurrency safety
)

func GenerateRandomCode() int {
	mu.Lock()
	defer mu.Unlock()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var code int
	for {
		code = 100000 + rng.Intn(900000) // Generate random number between 100000 and 999999
		if !generatedCodes[code] {       // Check if the code is unique
			generatedCodes[code] = true
			break
		}
	}
	return code
}
