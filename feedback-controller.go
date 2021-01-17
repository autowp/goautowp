package goautowp

import (
	"fmt"
	"github.com/autowp/goautowp/validation"
	"github.com/gin-gonic/gin"
	"gopkg.in/gomail.v2"
	"log"
	"net"
	"net/http"
	"strings"
)

// FeedbackController Main Object
type FeedbackController struct {
	config          FeedbackConfig
	recaptchaConfig RecaptchaConfig
	smtpConfig      SMTPConfig
}

// FeedbackRequestPostBody FeedbackRequestPostBody
type FeedbackRequestPostBody struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
	Captcha string `json:"captcha"`
}

// NewFeedbackController constructor
func NewFeedbackController(config FeedbackConfig, recaptchaConfig RecaptchaConfig, smtpConfig SMTPConfig) (*FeedbackController, error) {

	s := &FeedbackController{
		config:          config,
		recaptchaConfig: recaptchaConfig,
		smtpConfig:      smtpConfig,
	}

	return s, nil
}

func (s *FeedbackController) SetupRouter(apiGroup *gin.RouterGroup) {
	apiGroup.POST("/feedback", func(c *gin.Context) {

		request := FeedbackRequestPostBody{}
		err := c.BindJSON(&request)

		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		InvalidParams, err := request.Validate(c, s.config.Captcha)
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		if len(InvalidParams) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"type":           "http://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html",
				"title":          "Validation error",
				"status":         400,
				"detail":         "Data is invalid. Check `invalid_params`.",
				"invalid_params": InvalidParams,
			})
			return
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
			log.Println(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusCreated)
	})
}

func (s *FeedbackRequestPostBody) Validate(c *gin.Context, captchaEnabled bool) (map[string][]string, error) {

	result := make(map[string][]string)
	var problems []string

	nameInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	s.Name, problems = nameInputFilter.IsValidString(s.Name)
	if len(problems) > 0 {
		result["name"] = problems
	}

	emailInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}, &validation.EmailAddress{}},
	}
	s.Email, problems = emailInputFilter.IsValidString(s.Email)
	if len(problems) > 0 {
		result["email"] = problems
	}

	messageInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	s.Message, problems = messageInputFilter.IsValidString(s.Message)
	if len(problems) > 0 {
		result["message"] = problems
	}

	if captchaEnabled {
		ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusBadRequest, err.Error())
			return nil, err
		}

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
		if len(problems) > 0 {
			result["captcha"] = problems
		}
	}

	return result, nil
}
