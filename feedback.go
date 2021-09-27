package goautowp

import (
	"fmt"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// Feedback Main Object
type Feedback struct {
	captchaEnabled  bool
	config          config.FeedbackConfig
	recaptchaConfig config.RecaptchaConfig
	emailSender     email.Sender
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
func NewFeedback(config config.FeedbackConfig, recaptchaConfig config.RecaptchaConfig, captchaEnabled bool, emailSender email.Sender) (*Feedback, error) {

	s := &Feedback{
		config:          config,
		recaptchaConfig: recaptchaConfig,
		captchaEnabled:  captchaEnabled,
		emailSender:     emailSender,
	}

	return s, nil
}

func (s *Feedback) Create(request CreateFeedbackRequest) ([]*errdetails.BadRequest_FieldViolation, error) {

	InvalidParams, err := request.Validate(s.captchaEnabled, request.IP)
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return InvalidParams, nil
	}

	message := fmt.Sprintf("Имя: %s\nE-mail: %s\nСообщение:\n%s", request.Name, request.Email, request.Message)

	err = s.emailSender.Send(s.config.From, s.config.To, s.config.Subject, message, request.Email)

	return nil, err
}

func (s *CreateFeedbackRequest) Validate(captchaEnabled bool, ip string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string
	var err error

	nameInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	s.Name, problems, err = nameInputFilter.IsValidString(s.Name)
	if err != nil {
		return nil, err
	}
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
	s.Email, problems, err = emailInputFilter.IsValidString(s.Email)
	if err != nil {
		return nil, err
	}
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
	s.Message, problems, err = messageInputFilter.IsValidString(s.Message)
	if err != nil {
		return nil, err
	}
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
		s.Captcha, problems, err = captchaInputFilter.IsValidString(s.Captcha)
		if err != nil {
			return nil, err
		}
		for _, fv := range problems {
			result = append(result, &errdetails.BadRequest_FieldViolation{
				Field:       "captcha",
				Description: fv,
			})
		}
	}

	return result, nil
}
