package util

import (
	"database/sql"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
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

func SQLNullInt64ToPtr(v sql.NullInt64) *int64 {
	var r *int64

	if v.Valid {
		return &v.Int64
	}

	return r
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

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
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
