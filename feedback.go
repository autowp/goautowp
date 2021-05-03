package goautowp

import (
	"fmt"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"gopkg.in/gomail.v2"
)

// Feedback Main Object
type Feedback struct {
	config          FeedbackConfig
	recaptchaConfig RecaptchaConfig
	smtpConfig      SMTPConfig
}

// CreateFeedbackRequest CreateFeedbackRequest
type CreateFeedbackRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
	Captcha string `json:"captcha"`
	IP      string
}

// NewFeedback constructor
func NewFeedback(config FeedbackConfig, recaptchaConfig RecaptchaConfig, smtpConfig SMTPConfig) (*Feedback, error) {

	s := &Feedback{
		config:          config,
		recaptchaConfig: recaptchaConfig,
		smtpConfig:      smtpConfig,
	}

	return s, nil
}

func (s *Feedback) Create(request CreateFeedbackRequest) ([]*errdetails.BadRequest_FieldViolation, error) {

	InvalidParams, err := request.Validate(s.config.Captcha, request.IP)
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return InvalidParams, nil
	}

	message := fmt.Sprintf("Имя: %s\nE-mail: %s\nСообщение:\n%s", request.Name, request.Email, request.Message)

	m := gomail.NewMessage()
	m.SetHeader("From", s.config.From)
	m.SetHeader("To", s.config.To...)
	m.SetHeader("Subject", s.config.Subject)
	m.SetBody("text/plain", message)
	m.SetHeader("Reply-To", request.Email)

	d := gomail.NewDialer(s.smtpConfig.Hostname, s.smtpConfig.Port, s.smtpConfig.Username, s.smtpConfig.Password)

	if err := d.DialAndSend(m); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *CreateFeedbackRequest) Validate(captchaEnabled bool, ip string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string

	nameInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	s.Name, problems = nameInputFilter.IsValidString(s.Name)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	emailInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}, &validation.EmailAddress{}},
	}
	s.Email, problems = emailInputFilter.IsValidString(s.Email)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "email",
			Description: fv,
		})
	}

	messageInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	s.Message, problems = messageInputFilter.IsValidString(s.Message)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: fv,
		})
	}

	if captchaEnabled {
		captchaInputFilter := validation.InputFilter{
			Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
			Validators: []validation.ValidatorInterface{
				&validation.NotEmpty{},
				&validation.Recaptcha{
					ClientIP: ip,
				},
			},
		}
		s.Captcha, problems = captchaInputFilter.IsValidString(s.Captcha)
		for _, fv := range problems {
			result = append(result, &errdetails.BadRequest_FieldViolation{
				Field:       "captcha",
				Description: fv,
			})
		}
	}

	return result, nil
}
