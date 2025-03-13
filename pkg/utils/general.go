package utils

import (
	"fmt"
	"math/rand"
	"regexp"
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

func GenerateMaterialCode() int {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return rng.Intn(100000)
}

func GenerateDocumentNumber() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("PN-%06d", rng.Intn(1_000_000_000))
}

func DefineProductSearchQuery(search string) string {
	barcodeRegex := regexp.MustCompile(`^\d{1,20}$`)
	nameCategoryRegex := regexp.MustCompile(`^[a-zA-Zа-яА-ЯёЁ\s-]+$`)
	markingRegex := regexp.MustCompile(`^.{31}$`)
	switch {
	case barcodeRegex.MatchString(search):
		return "barcode"
	case nameCategoryRegex.MatchString(search):
		return "name/category"
	case markingRegex.MatchString(search):
		return "marking"
	default:
		return "name/category"
	}
}
