package utils

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/logger"
)

// Phone number validator for Uzbekistan phone numbers
func IsValidPhone(phone string) bool {

	// Compile the regular expression
	re := regexp.MustCompile(`^998[1-9][0-9]\d{7}$`)

	// Check if the phone number matches the pattern
	return re.MatchString(phone)
}

type Validator struct {
	validator  *validator.Validate
	logger     logger.Interface
	messagesUZ map[string]string
	messagesRU map[string]string
	messagesEN map[string]string
	mutex      sync.RWMutex
}

func NewValidator(logger logger.Interface) *Validator {
	uzMessages := make(map[string]string, 10)
	ruMessages := make(map[string]string, 10)
	enMessages := make(map[string]string, 10)
	return &Validator{
		validator:  validator.New(validator.WithRequiredStructEnabled()),
		logger:     logger,
		messagesUZ: uzMessages,
		messagesRU: ruMessages,
		messagesEN: enMessages,
		mutex:      sync.RWMutex{},
	}
}

// Struct - validate struct
func (v *Validator) Struct(data any) error {
	return v.validator.Struct(data)
}

func (v *Validator) ValidationMessage(c *gin.Context, errs error) gin.H {
	var validationErrors validator.ValidationErrors
	errors.As(errs, &validationErrors)

	response := make(gin.H, len(validationErrors))

	localization := v.getLocalizationByHeader(c)

	// Lock the resource for read
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	var message string
	var ok bool

	// Loop through errors and construct proper response with validation errors
	for _, err := range validationErrors {
		message, ok = localization[strings.ToLower(err.Tag())]
		if !ok {
			message = localization[constants.DefaultValidationErrKey]
		}
		response[v.camelToSnake(err.Namespace())] = message
	}

	return response
}

func (v *Validator) CustomValidationMessage(c *gin.Context, key string, err string) gin.H {
	localization := v.getLocalizationByHeader(c)

	// Lock the resource for read
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// Get default message if key is not found
	message, ok := localization[strings.ToLower(err)]
	if !ok {
		message = localization[constants.DefaultValidationErrKey]
	}

	return gin.H{key: message}
}

func (v *Validator) getLocalizationByHeader(c *gin.Context) map[string]string {
	// Get the value of the Accept-Language header
	acceptLanguage := c.GetHeader("Accept-Language")

	// Check if the Accept-Language header contains language codes for English, Russian, or Uzbek
	isEnglish := strings.Contains(acceptLanguage, constants.LanguageEn)
	isRussian := strings.Contains(acceptLanguage, constants.LanguageRu)

	switch true {
	case isEnglish:
		return v.messagesEN
	case isRussian:
		return v.messagesRU
	default:
		return v.messagesUZ
	}
}

func (v *Validator) camelToSnake(camel string) string {
	camelLen := len(camel)
	snakeLen := camelLen
	buffer := make([]byte, 0, snakeLen)

	var prev rune
	for i, c := range camel {
		if i > 0 && unicode.IsUpper(c) && prev != '.' {
			buffer = append(buffer, '_')
		}
		buffer = append(buffer, byte(c))
		prev = c
	}

	// Based on how validator constructs namespaces, it will always have at least two elements after split
	return strings.SplitN(strings.ToLower(string(buffer)), ".", 2)[1]
}
