package goautowp

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// MigrationsConfig MigrationsConfig
type MigrationsConfig struct {
	DSN string `yaml:"dsn" mapstructure:"dsn"`
	Dir string `yaml:"dir" mapstructure:"dir"`
}

// SentryConfig SentryConfig
type SentryConfig struct {
	DSN         string `yaml:"dsn"         mapstructure:"dsn"`
	Environment string `yaml:"environment" mapstructure:"environment"`
}

type FileStorageConfig struct {
	S3     S3Config `yaml:"s3"     mapstructure:"s3"`
	Bucket string   `yaml:"bucket" mapstructure:"bucket"`
}

type S3Config struct {
	Credentials      S3CredentialsConfig `yaml:"credentials"         mapstructure:"credentials"`
	Region           string              `yaml:"region"              mapstructure:"region"`
	Endpoints        []string            `yaml:"endpoints"           mapstructure:"endpoints"`
	S3ForcePathStyle bool                `yaml:"s3_force_path_style" mapstructure:"s3_force_path_style"`
}

type S3CredentialsConfig struct {
	Key    string `yaml:"key"    mapstructure:"key"`
	Secret string `yaml:"secret" mapstructure:"secret"`
}

type DuplicateFinderConfig struct {
	RabbitMQ string `yaml:"rabbitmq" mapstructure:"rabbitmq"`
	Queue    string `yaml:"queue"    mapstructure:"queue"`
}

type RestConfig struct {
	Listen string `mapstructure:"listen"`
	Mode   string `mapstructure:"mode"`
}

// OAuthConfig OAuthConfig
type OAuthConfig struct {
	Secret string `yaml:"secret" mapstructure:"secret"`
}

// Config Application config definition
type Config struct {
	Rest            RestConfig            `yaml:"rest"             mapstructure:"rest"`
	DuplicateFinder DuplicateFinderConfig `yaml:"duplicate_finder" mapstructure:"duplicate_finder"`
	DSN             string                `yaml:"dsn"              mapstructure:"dsn"`
	Migrations      MigrationsConfig      `yaml:"migrations"       mapstructure:"migrations"`
	Sentry          SentryConfig          `yaml:"sentry"           mapstructure:"sentry"`
	FileStorage     FileStorageConfig     `yaml:"file_storage"     mapstructure:"file_storage"`
	OAuth           OAuthConfig           `yaml:"oauth"            mapstructure:"oauth"`
}

// LoadConfig LoadConfig
func LoadConfig() Config {

	config := Config{}

	viper.SetConfigName("defaults")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	viper.SetConfigName("config")
	err = viper.MergeInConfig()
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("fatal error unmarshal config: %s", err))
	}

	return config
}

// ValidateConfig ValidateConfig
func ValidateConfig(config Config) {
	if config.DuplicateFinder.RabbitMQ == "" {
		log.Fatalln("Address not provided")
	}

	if config.DuplicateFinder.Queue == "" {
		log.Fatalln("DuplicateFinderQueue not provided")
	}
}
