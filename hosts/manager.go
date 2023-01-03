package hosts

import (
	"errors"
	"net/url"

	"github.com/autowp/goautowp/config"
)

type Manager struct {
	languages map[string]config.LanguageConfig
}

func NewManager(languages map[string]config.LanguageConfig) *Manager {
	return &Manager{
		languages: languages,
	}
}

func (s *Manager) URIByLanguage(language string) (*url.URL, error) {
	langConfig, ok := s.languages[language]

	if !ok {
		return nil, errors.New("host for language `$language` not found")
	}

	return url.Parse("https://" + langConfig.Hostname)
}
