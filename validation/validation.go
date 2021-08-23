package validation

import (
	"database/sql"
	"fmt"
	"github.com/dpapathanasiou/go-recaptcha"
	"net/mail"
	"strings"
)

const NotEmptyIsEmpty = "Value is required and can't be empty"
const EmailAddressInvalidFormat = "The input is not a valid email address"
const StringLengthTooShort = "The input is less than %d characters long"
const StringLengthTooLong = "The input is more than %d characters long"
const EmailNotExistsExists = "E-mail already registered"
const IdenticalStringsNotSame = "The two given tokens do not match"

type FilterInterface interface {
	FilterString(value string) string
}

type ValidatorInterface interface {
	IsValidString(value string) []string
}

// NotEmpty validator
type NotEmpty struct {
}

// StringLength validator
type StringLength struct {
	Min int
	Max int
}

// EmailAddress validator
type EmailAddress struct {
}

// Recaptcha validator
type Recaptcha struct {
	ClientIP string
}

// EmailNotExists validator
type EmailNotExists struct {
	DB *sql.DB
}

// IdenticalStrings validator
type IdenticalStrings struct {
	Pattern string
}

// StringTrimFilter filter
type StringTrimFilter struct {
}

// IsValidString IsValidString
func (s *NotEmpty) IsValidString(value string) []string {
	if len(value) > 0 {
		return []string{}
	}

	return []string{NotEmptyIsEmpty}
}

// IsValidString IsValidString
func (s *StringLength) IsValidString(value string) []string {
	l := len(value)
	if l < s.Min {
		return []string{fmt.Sprintf(StringLengthTooShort, s.Min)}
	}

	if l > s.Max {
		return []string{fmt.Sprintf(StringLengthTooLong, s.Max)}
	}

	return []string{}
}

// IsValidString IsValidString
func (s *EmailAddress) IsValidString(value string) []string {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return []string{EmailAddressInvalidFormat}
	}

	return []string{}
}

// IsValidString IsValidString
func (s *Recaptcha) IsValidString(value string) []string {
	_, err := recaptcha.Confirm(s.ClientIP, value)
	if err != nil {
		return []string{err.Error()}
	}

	return []string{}
}

// IsValidString IsValidString
func (s *EmailNotExists) IsValidString(value string) []string {
	var exists bool
	err := s.DB.QueryRow("SELECT 1 FROM users WHERE email = ?", value).Scan(&exists)
	if err == sql.ErrNoRows {
		return []string{}
	}

	if err != nil {
		return []string{err.Error()}
	}

	return []string{EmailNotExistsExists}
}

// IsValidString IsValidString
func (s *IdenticalStrings) IsValidString(value string) []string {
	if value != s.Pattern {
		return []string{IdenticalStringsNotSame}
	}

	return []string{}
}

// FilterString filter
func (s *StringTrimFilter) FilterString(value string) string {
	return strings.TrimSpace(value)
}

type InputFilter struct {
	Filters    []FilterInterface
	Validators []ValidatorInterface
}

// IsValidString IsValidString
func (s *InputFilter) IsValidString(value string) (string, []string) {
	value = filterString(value, s.Filters)
	errors := validateString(value, s.Validators)
	return value, errors
}

func filterString(value string, filters []FilterInterface) string {
	for _, filter := range filters {
		value = filter.FilterString(value)
	}
	return value
}

func validateString(value string, validators []ValidatorInterface) []string {
	result := make([]string, 0)
	for _, validator := range validators {
		errors := validator.IsValidString(value)
		if len(errors) > 0 {
			return errors
		}
	}

	return result
}
