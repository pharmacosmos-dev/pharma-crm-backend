package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
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
	barcodeRegex := regexp.MustCompile(`^\d{5,20}$`)
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

// Lotin -> Kirill translit map
var latinToCyrillic = map[rune]rune{
	'a': 'а', 'b': 'б', 'c': 'с', 'd': 'д', 'e': 'е', 'f': 'ф', 'g': 'г',
	'h': 'ҳ', 'i': 'и', 'j': 'ж', 'k': 'к', 'l': 'л', 'm': 'м', 'n': 'н',
	'o': 'о', 'p': 'п', 'q': 'қ', 'r': 'р', 's': 'с', 't': 'т', 'u': 'у',
	'v': 'в', 'x': 'х', 'y': 'й', 'z': 'з',
}

// Kirill -> Lotin translit map
var cyrillicToLatin = map[rune]string{
	'а': "a", 'б': "b", 'с': "s", 'д': "d", 'е': "e", 'ф': "f", 'г': "g",
	'ҳ': "h", 'и': "i", 'ж': "j", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
	'о': "o", 'п': "p", 'қ': "q", 'р': "r", 'т': "t", 'у': "u", 'в': "v",
	'х': "x", 'й': "y", 'з': "z",
	'ш': "sh", 'ч': "ch", 'ц': "ts", 'ў': "o‘", 'ғ': "g‘",
}

var multiCharMap = map[string]string{
	"sh": "ш", "ch": "ч", "ts": "ц", "o‘": "ў", "g‘": "г",
}

// Translit converts between Lotin and Kirill
func Translit(input string) string {
	var result strings.Builder
	input = strings.ToLower(input)
	runes := []rune(input)

	// Check if input contains Cyrillic (to determine conversion direction)
	isCyrillic := false
	for _, r := range runes {
		if _, ok := cyrillicToLatin[r]; ok {
			isCyrillic = true
			break
		}
	}

	for i := 0; i < len(runes); i++ {
		// Check two-character mappings first
		if i < len(runes)-1 {
			twoChar := string(runes[i : i+2])
			if val, ok := multiCharMap[twoChar]; ok {
				result.WriteString(val)
				i++ // Skip next character
				continue
			}
		}

		// Handle single-character mappings
		if isCyrillic {
			if val, ok := cyrillicToLatin[runes[i]]; ok {
				result.WriteString(val)
				continue
			}
		} else {
			if val, ok := latinToCyrillic[runes[i]]; ok {
				result.WriteRune(val)
				continue
			}
		}

		// If no mapping found, keep the character unchanged
		result.WriteRune(runes[i])
	}

	return result.String()
}

// covert before start date and end date
func BeforeDates(startDateStr, endDateStr string) (string, string) {
	if endDateStr == "" {
		endDateStr = startDateStr
	}
	startDate, _ := time.Parse("2006-01-02", startDateStr)
	endDate, _ := time.Parse("2006-01-02", endDateStr)
	diff := endDate.Sub(startDate)
	beforeStart := startDate.Add(-diff)
	beforeEnd := startDate
	return beforeStart.Format("2006-01-02"), beforeEnd.Format("2006-01-02")
}
