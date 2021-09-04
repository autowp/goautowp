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
