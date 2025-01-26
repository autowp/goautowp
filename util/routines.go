package util

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"html/template"
	"io"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exec"
	"github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/genproto/googleapis/type/date"
)

const (
	mysqlDuplicateKeyErrorCode = 1062
	mysqlDeadlockErrorCode     = 1213
)

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func GrpcDateToTime(value *date.Date, loc *time.Location) time.Time {
	if value == nil {
		return time.Time{}
	}

	return time.Date(int(value.GetYear()), time.Month(value.GetMonth()), int(value.GetDay()), 0, 0, 0, 0, loc)
}

func TimeToDate(value time.Time) Date {
	return Date{
		Year:  value.Year(),
		Month: value.Month(),
		Day:   value.Day(),
	}
}

func TimeToGrpcDate(value time.Time) *date.Date {
	if value.IsZero() {
		return nil
	}

	return &date.Date{
		Year:  int32(value.Year()),  //nolint: gosec
		Month: int32(value.Month()), //nolint: gosec
		Day:   int32(value.Day()),   //nolint: gosec
	}
}

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

func TimePtr(v time.Time) *time.Time {
	tVar := v

	return &tVar
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

func NullByteToScalar(value sql.NullByte) byte {
	if value.Valid {
		return value.Byte
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

func isMysqlErrorCode(err error, code uint16) bool {
	var me *mysql.MySQLError

	return errors.As(err, &me) && me.Number == code
}

func IsMysqlDuplicateKeyError(err error) bool {
	return isMysqlErrorCode(err, mysqlDuplicateKeyErrorCode)
}

func IsMysqlDeadlockError(err error) bool {
	return isMysqlErrorCode(err, mysqlDeadlockErrorCode)
}

func ScanValContextAndRetryOnDeadlock(ctx context.Context, sd *goqu.SelectDataset, i interface{}) (bool, error) {
	var (
		res               bool
		err               error
		isDeadlockAvoided bool
		retriesLeft       = 10
	)

	for !isDeadlockAvoided && retriesLeft > 0 {
		res, err = sd.ScanValContext(ctx, i)
		if err != nil {
			if !IsMysqlDeadlockError(err) {
				return res, err
			}

			logrus.Warn("Deadlock detected. Retrying")
			time.Sleep(time.Millisecond)

			retriesLeft--
		} else {
			isDeadlockAvoided = true
		}
	}

	return res, err
}

func ExecAndRetryOnDeadlock(ctx context.Context, executor exec.QueryExecutor) (sql.Result, error) {
	var (
		res               sql.Result
		err               error
		isDeadlockAvoided bool
		retriesLeft       = 10
	)

	for !isDeadlockAvoided && retriesLeft > 0 {
		res, err = executor.ExecContext(ctx)
		if err != nil {
			if !IsMysqlDeadlockError(err) {
				return res, err
			}

			logrus.Warn("Deadlock detected. Retrying")
			time.Sleep(time.Millisecond)

			retriesLeft--
		} else {
			isDeadlockAvoided = true
		}
	}

	return res, err
}

func KeyOfMapMaxValue[T constraints.Integer](values map[T]int) T {
	var (
		maxCount   = 0
		selectedID T
	)

	for id, count := range values {
		if maxCount == 0 || (count > maxCount) {
			maxCount = count
			selectedID = id
		}
	}

	return selectedID
}

func RemoveDuplicate[T comparable](sliceList []T) []T {
	var (
		allKeys = make(map[T]bool)
		list    []T
	)

	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true

			list = append(list, item)
		}
	}

	return list
}

func StringDefault(value, defaultValue string) string {
	if len(value) > 0 {
		return value
	}

	return defaultValue
}

func HTMLEscapeString(value string) template.HTML {
	return template.HTML(template.HTMLEscapeString(value)) //nolint: gosec
}

func TitleCase(str string, tag language.Tag) string {
	if str == "" {
		return ""
	}

	if base, _ := tag.Base(); base.String() == "en" {
		return cases.Title(tag).String(str)
	}

	r, n := utf8.DecodeRuneInString(str)

	return string(unicode.ToUpper(r)) + str[n:]
}

var errUnsupportedIPType = errors.New("unsupported type for IP")

type IP net.IP

func (n IP) ToIP() net.IP {
	return net.IP(n)
}

// Scan implements the [Scanner] interface.
func (n *IP) Scan(value interface{}) error {
	if value == nil {
		*n = nil

		return nil
	}

	v, ok := value.([]byte)
	if !ok {
		return errUnsupportedIPType
	}

	*n = v

	return nil
}

// Value implements the [driver.Valuer] interface.
func (n IP) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil //nolint: nilnil
	}

	return goqu.Func("INET6_ATON", n.ToIP().String()), nil
}
