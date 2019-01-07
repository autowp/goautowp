package goautowp

import (
	"log"
	"os"

	"github.com/autowp/goautowp/util"
)

// MigrationsConfig MigrationsConfig
type MigrationsConfig struct {
	DSN string `yaml:"dsn"`
	Dir string `yaml:"dir"`
}

// Config Application config definition
type Config struct {
	RabbitMQ             string             `yaml:"rabbitmq"`
	DuplicateFinderQueue string             `yaml:"duplicate_finder_queue"`
	Rollbar              util.RollbarConfig `yaml:"rollbar"`
	DSN                  string             `yaml:"dsn"`
	Migrations           MigrationsConfig   `yaml:"migrations"`
	ImagesDir            string             `yaml:"images_dir"`
}

// LoadConfig LoadConfig
func LoadConfig() Config {

	rabbitMQ := "amqp://guest:guest@" + os.Getenv("AUTOWP_RABBITMQ_HOST") + ":" + os.Getenv("AUTOWP_RABBITMQ_PORT") + "/"

	config := Config{
		RabbitMQ:             rabbitMQ,
		DuplicateFinderQueue: os.Getenv("AUTOWP_DUPLICATE_FINDER_QUEUE"),
		Rollbar: util.RollbarConfig{
			Token:       os.Getenv("AUTOWP_ROLLBAR_TOKEN"),
			Environment: os.Getenv("AUTOWP_ROLLBAR_ENVIRONMENT"),
			Period:      os.Getenv("AUTOWP_ROLLBAR_PERIOD"),
		},
		DSN: os.Getenv("AUTOWP_MYSQL_USERNAME") + ":" + os.Getenv("AUTOWP_MYSQL_PASSWORD") +
			"@tcp(" + os.Getenv("AUTOWP_MYSQL_HOST") + ":" + os.Getenv("AUTOWP_MYSQL_PORT") + ")/" +
			os.Getenv("AUTOWP_MYSQL_DBNAME") + "?charset=utf8mb4&parseTime=true&loc=UTC",
		ImagesDir: os.Getenv("AUTOWP_IMAGES_DIR"),
		Migrations: MigrationsConfig{
			DSN: "mysql://" + os.Getenv("AUTOWP_MYSQL_USERNAME") + ":" + os.Getenv("AUTOWP_MYSQL_PASSWORD") +
				"@tcp(" + os.Getenv("AUTOWP_MYSQL_HOST") + ":" + os.Getenv("AUTOWP_MYSQL_PORT") + ")/" +
				os.Getenv("AUTOWP_MYSQL_DBNAME") + "?charset=utf8mb4&parseTime=true&loc=UTC",
			Dir: os.Getenv("AUTOWP_MIGRATIONS_DIR"),
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
