package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	Localizer *i18n.Localizer
	Bundle    *i18n.Bundle // Global bundle variable
)

func InitI18n() error {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	localesPath := "./locales"
	files, err := os.ReadDir(localesPath)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			_, err := Bundle.LoadMessageFile(filepath.Join(localesPath, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to load locale file %s: %w", file.Name(), err)
			}
		}
	}

	Localizer = i18n.NewLocalizer(Bundle, language.English.String())
	return nil
}

func Translate(lang, messageID string) string {
	localizer := i18n.NewLocalizer(Bundle, lang) // Use the global Bundle
	res := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	return res
}
