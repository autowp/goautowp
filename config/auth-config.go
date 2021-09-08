package config

import (
	"github.com/autowp/goautowp/auth/oauth2server/models"
)

// OAuthConfig OAuthConfig
type OAuthConfig struct {
	Driver                string          `yaml:"driver"                     mapstructure:"driver"`
	DSN                   string          `yaml:"dsn"                        mapstructure:"dsn"`
	Secret                string          `yaml:"secret"                     mapstructure:"secret"`
	Clients               []models.Client `yaml:"clients"                    mapstructure:"clients"`
	AccessTokenExpiresIn  uint            `yaml:"access_token_expires_in"    mapstructure:"access_token_expires_in"`
	RefreshTokenExpiresIn uint            `yaml:"refresh_token_expires_in"   mapstructure:"refresh_token_expires_in"`
}

// ServiceConfig ServiceConfig
type ServiceConfig struct {
	ClientID     string   `yaml:"client_id"     mapstructure:"client_id"`
	ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret"`
	Scopes       []string `yaml:"scopes"        mapstructure:"scopes"`
}

// ServicesConfig ...
type ServicesConfig struct {
	RedirectURI string        `yaml:"redirect_uri" mapstructure:"redirect_uri"`
	Google      ServiceConfig `yaml:"google"       mapstructure:"google"`
	Facebook    ServiceConfig `yaml:"facebook"     mapstructure:"facebook"`
	VK          ServiceConfig `yaml:"vk"           mapstructure:"vk"`
}

// AuthConfig Application config definition
type AuthConfig struct {
	Listen     string           `yaml:"listen"`
	Migrations MigrationsConfig `yaml:"migrations"`
	OAuth      OAuthConfig      `yaml:"oauth"`
	Services   ServicesConfig   `yaml:"services"`
}
