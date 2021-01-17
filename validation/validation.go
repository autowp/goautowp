package validation

import (
	"github.com/dpapathanasiou/go-recaptcha"
	"net/mail"
	"strings"
)

const NotEmptyIsEmpty = "Value is required and can't be empty"
const EmailAddressInvalidFormat = "The input is not a valid email address"

type FilterInterface interface {
	FilterString(value string) string
}

type ValidatorInterface interface {
	IsValidString(value string) []string
}

// NotEmpty validator
type NotEmpty struct {
}

// EmailAddress validator
type EmailAddress struct {
}

// Recaptcha validator
type Recaptcha struct {
	ClientIP string
}

// StringTrimFilter filter
type StringTrimFilter struct {
}

// InputFilterString InputFilterString
func (s *NotEmpty) IsValidString(value string) []string {
	if len(value) > 0 {
		return []string{}
	}

	return []string{NotEmptyIsEmpty}
}

// InputFilterString InputFilterString
func (s *EmailAddress) IsValidString(value string) []string {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return []string{EmailAddressInvalidFormat}
	}

	return []string{}
}

// InputFilterString InputFilterString
func (s *Recaptcha) IsValidString(value string) []string {
	_, err := recaptcha.Confirm(s.ClientIP, value)
	if err != nil {
		return []string{err.Error()}
	}

	return []string{}
}

// StringTrimFilter filter
func (s *StringTrimFilter) FilterString(value string) string {
	return strings.TrimSpace(value)
}

type InputFilter struct {
	Filters    []FilterInterface
	Validators []ValidatorInterface
}

// InputFilterString InputFilterString
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
