package config

// MigrationsConfig MigrationsConfig
type MigrationsConfig struct {
	DSN string `yaml:"dsn" mapstructure:"dsn"`
	Dir string `yaml:"dir" mapstructure:"dir"`
}

// LanguageConfig LanguageConfig
type LanguageConfig struct {
	Hostname string   `yaml:"hostname" mapstructure:"hostname"`
	Timezone string   `yaml:"timezone" mapstructure:"timezone"`
	Name     string   `yaml:"name"     mapstructure:"name"`
	Flag     string   `yaml:"flag"     mapstructure:"flag"`
	Aliases  []string `yaml:"aliases"  mapstructure:"aliases"`
}

// KeyCloakConfig KeyCloakConfig
type KeyCloakConfig struct {
	URL          string `yaml:"url"           mapstructure:"url"`
	ClientID     string `yaml:"client-id"     mapstructure:"client-id"`
	ClientSecret string `yaml:"client-secret" mapstructure:"client-secret"`
	Realm        string `yaml:"realm"         mapstructure:"realm"`
}

// SMTPConfig SMTPConfig
type SMTPConfig struct {
	Hostname string `yaml:"hostname" mapstructure:"hostname"`
	Port     int    `yaml:"port"     mapstructure:"port"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
}
