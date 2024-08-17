package util

import (
	"database/sql"
	"io"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
)

// Close resource and prints error.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		logrus.Error(err)
	}
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func ContainsInt32(s []int32, e int32) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func SQLNullInt64ToPtr(v sql.NullInt64) *int64 {
	var value *int64

	if v.Valid {
		return &v.Int64
	}

	return value
}

func ConnectRabbitMQ(config string) (*amqp.Connection, error) {
	const (
		connectionTimeout = 60 * time.Second
		reconnectDelay    = 100 * time.Millisecond
	)

	logrus.Info("Waiting for rabbitMQ")

	var (
		rabbitMQ *amqp.Connection
		err      error
		start    = time.Now()
	)

	for {
		rabbitMQ, err = amqp.Dial(config)
		if err == nil {
			logrus.Info("Started.")

			break
		}

		if time.Since(start) > connectionTimeout {
			return nil, err
		}

		logrus.Info(".")
		time.Sleep(reconnectDelay)
	}

	return rabbitMQ, nil
}

type TextPreviewOptions struct {
	Maxlength int
	Maxlines  int
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func GetTextPreview(text string, options TextPreviewOptions) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r", "")

	if options.Maxlines > 0 {
		lines := strings.Split(text, "\n")
		lines = lines[:options.Maxlines]
		text = strings.Join(lines, "\n")
	}

	if options.Maxlength > 0 {
		asRunes := []rune(text)
		if len(asRunes) > options.Maxlength {
			text = substr(text, 0, options.Maxlength) + "..."
		}
	}

	return text
}

func BoolPtr(b bool) *bool {
	boolVar := b

	return &boolVar
}

func Min[T constraints.Ordered](a, b T) T { //nolint: ireturn
	if a < b {
		return a
	}

	return b
}

func Max[T constraints.Ordered](a, b T) T { //nolint: ireturn
	if a > b {
		return a
	}

	return b
}
