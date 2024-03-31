package validation

import (
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/dpapathanasiou/go-recaptcha"
)

const (
	NotEmptyIsEmpty           = "Value is required and can't be empty"
	EmailAddressInvalidFormat = "The input is not a valid email address"
	URLInvalidFormat          = "The input is not a valid URL"
	StringLengthTooShort      = "The input is less than %d characters long"
	StringLengthTooLong       = "The input is more than %d characters long"
	EmailNotExistsExists      = "E-mail already registered"
	IdenticalStringsNotSame   = "The two given tokens do not match"
	NotInArray                = "The input was not found in the haystack"
)

type FilterInterface interface {
	FilterString(value string) string
}

type ValidatorInterface interface {
	IsValidString(value string) ([]string, error)
}

// NotEmpty validator.
type NotEmpty struct{}

// StringLength validator.
type StringLength struct {
	Min int
	Max int
}

// EmailAddress validator.
type EmailAddress struct{}

// URL validator.
type URL struct{}

// Recaptcha validator.
type Recaptcha struct {
	ClientIP string
}

// EmailNotExists validator.
type EmailNotExists struct {
	DB *sql.DB
}

// IdenticalStrings validator.
type IdenticalStrings struct {
	Pattern string
}

// InArray validator.
type InArray struct {
	Haystack []string
}

// Callback validator.
type Callback struct {
	Callback func(value string) ([]string, error)
}

// StringTrimFilter filter.
type StringTrimFilter struct{}

// StringSingleSpaces filter.
type StringSingleSpaces struct{}

// IsValidString IsValidString.
func (s *NotEmpty) IsValidString(value string) ([]string, error) {
	if len(value) > 0 {
		return []string{}, nil
	}

	return []string{NotEmptyIsEmpty}, nil
}

// IsValidString IsValidString.
func (s *StringLength) IsValidString(value string) ([]string, error) {
	l := len(value)
	if l < s.Min {
		return []string{fmt.Sprintf(StringLengthTooShort, s.Min)}, nil
	}

	if l > s.Max {
		return []string{fmt.Sprintf(StringLengthTooLong, s.Max)}, nil
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *EmailAddress) IsValidString(value string) ([]string, error) {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return []string{EmailAddressInvalidFormat}, nil //nolint:nilerr
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *URL) IsValidString(value string) ([]string, error) {
	_, err := url.ParseRequestURI(value)
	if err != nil {
		return []string{URLInvalidFormat}, nil //nolint:nilerr
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *Recaptcha) IsValidString(value string) ([]string, error) {
	_, err := recaptcha.Confirm(s.ClientIP, value)
	if err != nil {
		return []string{err.Error()}, nil //nolint:nilerr
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *EmailNotExists) IsValidString(value string) ([]string, error) {
	var exists bool
	err := s.DB.QueryRow("SELECT 1 FROM "+schema.UserTableName+" WHERE e_mail = ?", value).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return []string{}, nil
	}

	if err != nil {
		return nil, err
	}

	return []string{EmailNotExistsExists}, nil
}

// IsValidString IsValidString.
func (s *IdenticalStrings) IsValidString(value string) ([]string, error) {
	if value != s.Pattern {
		return []string{IdenticalStringsNotSame}, nil
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *InArray) IsValidString(value string) ([]string, error) {
	if !util.Contains(s.Haystack, value) {
		return []string{NotInArray}, nil
	}

	return []string{}, nil
}

// IsValidString IsValidString.
func (s *Callback) IsValidString(value string) ([]string, error) {
	return s.Callback(value)
}

// FilterString filter.
func (s *StringTrimFilter) FilterString(value string) string {
	return strings.TrimSpace(value)
}

// FilterString filter.
func (s *StringSingleSpaces) FilterString(value string) string {
	if len(value) == 0 {
		return ""
	}

	value = strings.ReplaceAll(value, "\r", "")
	lines := strings.Split(value, "\n")
	re := regexp.MustCompile("[[:space:]]+")
	out := make([]string, len(lines))

	for idx, line := range lines {
		out[idx] = re.ReplaceAllString(line, " ")
	}

	return strings.Join(out, "\n")
}

type InputFilter struct {
	Filters    []FilterInterface
	Validators []ValidatorInterface
}

// IsValidString IsValidString.
func (s *InputFilter) IsValidString(value string) (string, []string, error) {
	value = filterString(value, s.Filters)

	violations, err := validateString(value, s.Validators)
	if err != nil {
		return "", nil, err
	}

	return value, violations, nil
}

func filterString(value string, filters []FilterInterface) string {
	for _, filter := range filters {
		value = filter.FilterString(value)
	}

	return value
}

func validateString(value string, validators []ValidatorInterface) ([]string, error) {
	result := make([]string, 0)

	for _, validator := range validators {
		violations, err := validator.IsValidString(value)
		if err != nil {
			return nil, err
		}

		if len(violations) > 0 {
			return violations, nil
		}
	}

	return result, nil
}
