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

// FileStorageConfig FileStorageConfig
type FileStorageConfig struct {
	S3     S3Config `yaml:"s3"     mapstructure:"s3"`
	Bucket string   `yaml:"bucket" mapstructure:"bucket"`
}

// S3Config S3Config
type S3Config struct {
	Credentials      S3CredentialsConfig `yaml:"credentials"         mapstructure:"credentials"`
	Region           string              `yaml:"region"              mapstructure:"region"`
	Endpoints        []string            `yaml:"endpoints"           mapstructure:"endpoints"`
	S3ForcePathStyle bool                `yaml:"s3_force_path_style" mapstructure:"s3_force_path_style"`
}

// S3CredentialsConfig S3CredentialsConfig
type S3CredentialsConfig struct {
	Key    string `yaml:"key"    mapstructure:"key"`
	Secret string `yaml:"secret" mapstructure:"secret"`
}

// DuplicateFinderConfig DuplicateFinderConfig
type DuplicateFinderConfig struct {
	RabbitMQ string `yaml:"rabbitmq" mapstructure:"rabbitmq"`
	Queue    string `yaml:"queue"    mapstructure:"queue"`
}

// RestConfig RestConfig
type RestConfig struct {
	Listen string `mapstructure:"listen"`
}

// OAuthConfig OAuthConfig
type OAuthConfig struct {
	Secret string `yaml:"secret" mapstructure:"secret"`
}

// Config Application config definition
type Config struct {
	GinMode           string                `yaml:"gin-mode" mapstructure:"gin-mode"`
	PublicRest        RestConfig            `yaml:"public-rest"        mapstructure:"public-rest"`
	DuplicateFinder   DuplicateFinderConfig `yaml:"duplicate_finder"   mapstructure:"duplicate_finder"`
	AutowpDSN         string                `yaml:"autowp-dsn"         mapstructure:"autowp-dsn"`
	AutowpMigrations  MigrationsConfig      `yaml:"autowp-migrations"  mapstructure:"autowp-migrations"`
	Sentry            SentryConfig          `yaml:"sentry"             mapstructure:"sentry"`
	FileStorage       FileStorageConfig     `yaml:"file_storage"       mapstructure:"file_storage"`
	OAuth             OAuthConfig           `yaml:"oauth"              mapstructure:"oauth"`
	RabbitMQ          string                `yaml:"rabbitmq"           mapstructure:"rabbitmq"`
	MonitoringQueue   string                `yaml:"monitoring_queue"   mapstructure:"monitoring_queue"`
	PrivateRest       RestConfig            `yaml:"private-rest"       mapstructure:"private-rest"`
	TrafficDSN        string                `yaml:"traffic-dsn"        mapstructure:"traffic-dsn"`
	TrafficMigrations MigrationsConfig      `yaml:"traffic-migrations" mapstructure:"traffic-migrations"`
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

	if config.RabbitMQ == "" {
		log.Fatalln("Address not provided")
	}

	if config.MonitoringQueue == "" {
		log.Fatalln("MonitoringQueue not provided")
	}
}
