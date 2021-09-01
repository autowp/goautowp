package goautowp

import (
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"math/rand"
	"net/url"
)

type PasswordRecovery struct {
	captchaEnabled bool
	db             *sql.DB
	languages      map[string]LanguageConfig
	emailSender    EmailSender
}

func NewPasswordRecovery(db *sql.DB, captchaEnabled bool, languages map[string]LanguageConfig, emailSender EmailSender) *PasswordRecovery {
	return &PasswordRecovery{
		captchaEnabled: captchaEnabled,
		db:             db,
		languages:      languages,
		emailSender:    emailSender,
	}
}

func (s *PasswordRecovery) Start(email string, captcha string, ip string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string
	var err error

	if s.captchaEnabled {
		captchaInputFilter := validation.InputFilter{
			Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
			Validators: []validation.ValidatorInterface{
				&validation.NotEmpty{},
				&validation.Recaptcha{
					ClientIP: ip,
				},
			},
		}
		_, problems, err = captchaInputFilter.IsValidString(captcha)
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

	emailInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}, &validation.EmailAddress{}},
	}
	email, problems, err = emailInputFilter.IsValidString(email)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "email",
			Description: fv,
		})
	}

	if len(result) > 0 {
		return result, nil
	}

	var id int
	var languageCode string
	if len(email) > 0 {
		err := s.db.QueryRow(`
		SELECT id, language FROM users WHERE e_mail = ? AND NOT deleted LIMIT 1
	`, email).Scan(&id, &languageCode)
		if err == sql.ErrNoRows {
			err = nil
			result = append(result, &errdetails.BadRequest_FieldViolation{
				Field:       "email",
				Description: "User with that e-mail not found",
			})
		}
		if err != nil {
			return nil, err
		}
	}

	if len(result) > 0 {
		return result, nil
	}

	code, err := s.createToken(id)
	if err != nil {
		return nil, err
	}

	language, ok := s.languages[languageCode]
	if !ok {
		return nil, fmt.Errorf("language `%s` is not defined", languageCode)
	}

	uri := "https://" + language.Hostname + "/restore-password/new?code=" + url.QueryEscape(code)

	fromStr := "Robot www.wheelsage.org"
	message := fmt.Sprintf(
		"Follow link to enter new password: %s\n\nSincerely, %s",
		uri,
		fromStr,
	)

	err = s.emailSender.Send(fromStr+" <no-reply@autowp.ru>", []string{email}, "Password recovery", message, "")

	return nil, err
}

func (s *PasswordRecovery) createToken(userID int) (string, error) {
	exists := true
	token := ""
	for exists {
		token = RandStringBytesRmndr(32)

		err := s.db.QueryRow("SELECT 1 FROM user_password_remind WHERE hash = ?", token).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		exists = err == nil
	}

	_, err := s.db.Exec(`
		INSERT INTO user_password_remind (user_id, hash, created)
		VALUES (?, ?, NOW())
	`, userID, token)
	if err != nil {
		return "", err
	}

	return token, nil
}

const letterBytes = "0123456789abcdefABCDEF"

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (s *PasswordRecovery) GetUserID(token string) (int64, error) {

	var userID int64
	err := s.db.QueryRow(`
		SELECT user_id FROM user_password_remind
		WHERE hash = ? AND created > DATE_SUB(NOW(), INTERVAL 10 DAY)
	`, token).Scan(&userID)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return userID, err
}

func (s *PasswordRecovery) DeleteToken(token string) error {
	_, err := s.db.Exec("DELETE FROM user_password_remind WHERE hash = ?", token)
	return err
}

func (s *PasswordRecovery) ValidateNewPassword(password string, passwordConfirm string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string
	var err error

	passwordInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
		},
	}
	password, problems, err = passwordInputFilter.IsValidString(password)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "password",
			Description: fv,
		})
	}

	passwordConfirmInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
			&validation.IdenticalStrings{Pattern: password},
		},
	}
	_, problems, err = passwordConfirmInputFilter.IsValidString(passwordConfirm)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "passwordConfirm",
			Description: fv,
		})
	}

	return result, nil
}

func (s *PasswordRecovery) Finish(token string, password string, passwordConfirm string) ([]*errdetails.BadRequest_FieldViolation, int64, error) {
	userId, err := s.GetUserID(token)
	if err != nil {
		return nil, 0, err
	}

	if userId == 0 {
		return nil, 0, fmt.Errorf("token not found")
	}

	fv, err := s.ValidateNewPassword(password, passwordConfirm)
	if err != nil {
		return nil, 0, err
	}

	if len(fv) > 0 {
		return fv, 0, nil
	}

	err = s.DeleteToken(token)
	if err != nil {
		return nil, 0, err
	}

	return nil, userId, nil
}

func (s *PasswordRecovery) GC() (int64, error) {
	r, err := s.db.Exec("DELETE FROM user_password_remind WHERE created < DATE_SUB(NOW(), INTERVAL 10 DAY)")
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}
