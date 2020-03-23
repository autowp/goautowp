package goautowp

import (
	"log"
	"os"
)

// MigrationsConfig MigrationsConfig
type MigrationsConfig struct {
	DSN string `yaml:"dsn"`
	Dir string `yaml:"dir"`
}

// SentryConfig SentryConfig
type SentryConfig struct {
	DSN         string `yaml:"dsn"`
	Environment string `yaml:"environment"`
}

// Config Application config definition
type Config struct {
	RabbitMQ             string           `yaml:"rabbitmq"`
	DuplicateFinderQueue string           `yaml:"duplicate_finder_queue"`
	DSN                  string           `yaml:"dsn"`
	Migrations           MigrationsConfig `yaml:"migrations"`
	Sentry               SentryConfig     `yaml:"sentry"`
}

// LoadConfig LoadConfig
func LoadConfig() Config {

	rabbitMQ := "amqp://guest:guest@" + os.Getenv("AUTOWP_RABBITMQ_HOST") + ":" + os.Getenv("AUTOWP_RABBITMQ_PORT") + "/"

	config := Config{
		RabbitMQ:             rabbitMQ,
		DuplicateFinderQueue: os.Getenv("AUTOWP_DUPLICATE_FINDER_QUEUE"),
		DSN:                  os.Getenv("AUTOWP_MYSQL_DSN"),
		Migrations: MigrationsConfig{
			DSN: os.Getenv("AUTOWP_MIGRATIONS_DSN"),
			Dir: os.Getenv("AUTOWP_MIGRATIONS_DIR"),
		},
		Sentry: SentryConfig{
			DSN:         os.Getenv("AUTOWP_SENTRY_DSN"),
			Environment: os.Getenv("AUTOWP_SENTRY_ENVIRONMENT"),
		},
	}

	return config
}

// ValidateConfig ValidateConfig
func ValidateConfig(config Config) {
	if config.RabbitMQ == "" {
		log.Fatalln("Address not provided")
	}

	if config.DuplicateFinderQueue == "" {
		log.Fatalln("DuplicateFinderQueue not provided")
	}
}
