package i18nbundle

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed *.json
var LocaleFS embed.FS

type I18n struct {
	bundle *i18n.Bundle
}

func New() (*I18n, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := LocaleFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		_, err = bundle.LoadMessageFileFS(LocaleFS, file.Name())
		if err != nil {
			return nil, err
		}
	}

	return &I18n{
		bundle: bundle,
	}, nil
}

func (s *I18n) Localizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(s.bundle, lang)
}
