package goautowp

import (
	"log"
	"os"

	"github.com/autowp/goautowp/util"
)

// Config Application config definition
type Config struct {
	RabbitMQ       string             `yaml:"rabbitmq"`
	ImageHashQueue string             `yaml:"image_hash_queue"`
	Rollbar        util.RollbarConfig `yaml:"rollbar"`
	DSN            string             `yaml:"dsn"`
}

// LoadConfig LoadConfig
func LoadConfig() Config {

	config := Config{
		RabbitMQ:       "amqp://guest:guest@" + os.Getenv("TRAFFIC_RABBITMQ_HOST") + ":" + os.Getenv("TRAFFIC_RABBITMQ_PORT") + "/",
		ImageHashQueue: os.Getenv("TRAFFIC_MONITORING_QUEUE"),
		Rollbar: util.RollbarConfig{
			Token:       os.Getenv("TRAFFIC_ROLLBAR_TOKEN"),
			Environment: os.Getenv("TRAFFIC_ROLLBAR_ENVIRONMENT"),
			Period:      os.Getenv("TRAFFIC_ROLLBAR_PERIOD"),
		},
		DSN: os.Getenv("TRAFFIC_MYSQL_USERNAME") + ":" + os.Getenv("TRAFFIC_MYSQL_PASSWORD") +
			"@tcp(" + os.Getenv("TRAFFIC_MYSQL_HOST") + ":" + os.Getenv("TRAFFIC_MYSQL_PORT") + ")/" +
			os.Getenv("TRAFFIC_MYSQL_DBNAME") + "?charset=utf8mb4&parseTime=true&loc=UTC",
	}

	return config
}

// ValidateConfig ValidateConfig
func ValidateConfig(config Config) {
	if config.RabbitMQ == "" {
		log.Fatalln("Address not provided")
	}

	if config.ImageHashQueue == "" {
		log.Fatalln("ImageHashQueue not provided")
	}
}
