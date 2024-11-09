package util

import (
	"database/sql"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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

func Contains[T constraints.Ordered](s []T, e T) bool {
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

func RemoveValueFromArray[T comparable](l []T, item T) []T {
	out := make([]T, 0)

	for _, element := range l {
		if element != item {
			out = append(out, element)
		}
	}

	return out
}

type Rect[T constraints.Ordered] struct {
	Left   T
	Top    T
	Width  T
	Height T
}

func IntersectBounds[T constraints.Integer](rect1 Rect[T], rect2 Rect[T]) Rect[T] {
	rect1.Left = Max(rect2.Left, rect1.Left)
	rect1.Left = Min(rect2.Width, rect1.Left)

	rect1.Width = Max(T(0), rect1.Width)
	rect1.Width = Min(rect2.Width-rect1.Left, rect1.Width)

	rect1.Top = Max(rect2.Top, rect1.Top)
	rect1.Top = Min(rect2.Height, rect1.Top)

	rect1.Height = Max(T(0), rect1.Height)
	rect1.Height = Min(rect2.Height-rect2.Top, rect1.Height)

	return rect1
}

func NullInt64ToScalar(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}

	return 0
}

func NullInt32ToScalar(value sql.NullInt32) int32 {
	if value.Valid {
		return value.Int32
	}

	return 0
}

func NullInt16ToScalar(value sql.NullInt16) int16 {
	if value.Valid {
		return value.Int16
	}

	return 0
}

func NullStringToString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}

	return ""
}

func NullBoolToBoolPtr(value sql.NullBool) *bool {
	if value.Valid {
		return &value.Bool
	}

	return nil
}

func IsMysqlDeadlockError(err error) bool {
	var me *mysql.MySQLError

	return errors.As(err, &me) && me.Number == 1213
}
