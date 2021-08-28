package goautowp

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/Nerzal/gocloak/v8"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"math"
	"math/rand"
	"net/url"
	"time"
)

const KeyCloakExternalAccountID = "keycloak"

type GetUsersOptions struct {
	ID         int
	InContacts int
	Order      []string
	Fields     map[string]bool
	Deleted    *bool
}

// APIUser APIUser
type APIUser struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Deleted    bool       `json:"deleted"`
	LongAway   bool       `json:"long_away"`
	Green      bool       `json:"green"`
	Route      []string   `json:"route"`
	Identity   *string    `json:"identity"`
	Avatar     *string    `json:"avatar,omitempty"`
	Gravatar   *string    `json:"gravatar,omitempty"`
	LastOnline *time.Time `json:"last_online,omitempty"`
}

// DBUser DBUser
type DBUser struct {
	ID         int
	Name       string
	Deleted    bool
	Identity   *string
	LastOnline *time.Time
	Role       string
	EMail      *string
	Img        *int
}

// CreateUserOptions CreateUserOptions
type CreateUserOptions struct {
	UserName        string `json:"user_name"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Timezone        string `json:"timezone"`
	Language        string `json:"language"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	Captcha         string `json:"captcha"`
}

// UserRepository Main Object
type UserRepository struct {
	autowpDB       *sql.DB
	usersSalt      string
	emailSalt      string
	languages      map[string]LanguageConfig
	emailSender    EmailSender
	keyCloak       gocloak.GoCloak
	keyCloakConfig KeyCloakConfig
}

// NewUserRepository constructor
func NewUserRepository(
	autowpDB *sql.DB,
	usersSalt string,
	emailSalt string,
	languages map[string]LanguageConfig,
	emailSender EmailSender,
	keyCloak gocloak.GoCloak,
	keyCloakConfig KeyCloakConfig,
) (*UserRepository, error) {

	if autowpDB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	s := &UserRepository{
		autowpDB:       autowpDB,
		usersSalt:      usersSalt,
		emailSalt:      emailSalt,
		languages:      languages,
		emailSender:    emailSender,
		keyCloak:       keyCloak,
		keyCloakConfig: keyCloakConfig,
	}

	return s, nil
}

func (s *UserRepository) GetUser(options GetUsersOptions) (*DBUser, error) {

	users, err := s.GetUsers(options)
	if err != nil {
		return nil, err
	}

	if len(users) <= 0 {
		return nil, nil
	}

	return &users[0], nil
}

func (s *UserRepository) GetUsers(options GetUsersOptions) ([]DBUser, error) {

	result := make([]DBUser, 0)

	var r DBUser
	valuePtrs := []interface{}{&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role}

	sqSelect := sq.Select("users.id, users.name, users.deleted, users.identity, users.last_online, users.role").From("users")

	if options.ID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"users.id": options.ID})
	}

	if options.InContacts != 0 {
		sqSelect = sqSelect.Join("contact ON users.id = contact.contact_user_id").Where(sq.Eq{"contact.user_id": options.InContacts})
	}

	if options.Deleted != nil {
		if *options.Deleted {
			sqSelect = sqSelect.Where("users.deleted")
		} else {
			sqSelect = sqSelect.Where("not users.deleted")
		}
	}

	if len(options.Order) > 0 {
		sqSelect = sqSelect.OrderBy(options.Order...)
	}

	if len(options.Fields) > 0 {
		for field := range options.Fields {
			switch field {
			case "avatar":
				sqSelect = sqSelect.Columns("users.img")
				valuePtrs = append(valuePtrs, &r.Img)
			case "gravatar":
				sqSelect = sqSelect.Columns("users.e_mail")
				valuePtrs = append(valuePtrs, &r.EMail)
			}
		}
	}

	rows, err := sqSelect.RunWith(s.autowpDB).Query()
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	return result, nil
}

func (s *UserRepository) ValidateCreateUser(options CreateUserOptions, captchaEnabled bool, ip string) ([]*errdetails.BadRequest_FieldViolation, error) {
	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Min: 2, Max: 50},
		},
	}
	options.Name, problems = nameInputFilter.IsValidString(options.Name)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	emailInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.EmailAddress{},
			&validation.StringLength{Max: 50},
			&validation.EmailNotExists{DB: s.autowpDB},
		},
	}
	options.Email, problems = emailInputFilter.IsValidString(options.Email)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "email",
			Description: fv,
		})
	}

	passwordInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
		},
	}
	options.Password, problems = passwordInputFilter.IsValidString(options.Password)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "password",
			Description: fv,
		})
	}

	passwordConfirmInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
			&validation.IdenticalStrings{Pattern: options.Password},
		},
	}
	options.PasswordConfirm, problems = passwordConfirmInputFilter.IsValidString(options.PasswordConfirm)
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "password_confirm",
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
		options.Captcha, problems = captchaInputFilter.IsValidString(options.Captcha)
		for _, fv := range problems {
			result = append(result, &errdetails.BadRequest_FieldViolation{
				Field:       "captcha",
				Description: fv,
			})
		}
	}

	return result, nil
}

func (s *UserRepository) CreateUser(options CreateUserOptions) (int64, error) {

	ctx := context.Background()
	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return 0, err
	}

	credentialsType := "PASSWORD"
	credentials := []gocloak.CredentialRepresentation{
		{
			Type:  &credentialsType,
			Value: &options.Password,
		},
	}
	f := false
	t := true
	userGuid, err := s.keyCloak.CreateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		Enabled:       &t,
		Totp:          &f,
		EmailVerified: &f,
		Username:      &options.Email,
		FirstName:     &options.Name,
		Email:         &options.Email,
		Credentials:   &credentials,
	})
	if err != nil {
		return 0, err
	}

	md5Bytes := md5.Sum([]byte(fmt.Sprintf("%s%s%d", s.emailSalt, options.Email, rand.Int())))
	emailCheckCode := hex.EncodeToString(md5Bytes[:])

	username := &options.UserName
	if len(options.UserName) <= 0 {
		username = nil
	}

	r, err := s.autowpDB.Exec(`
		INSERT INTO users (login, e_mail, password, email_to_check, hide_e_mail, email_check_code, name, reg_date, 
		                   last_online, timezone, last_ip, language)
		VALUES (?, NULL, MD5(CONCAT(?, ?)), ?, 1, ?, ?, NOW(), NOW(), ?, INET6_ATON(?), ?)
	`, username, s.usersSalt, options.Password, options.Email, emailCheckCode, options.Name, options.Timezone, "127.0.0.1", options.Language)
	if err != nil {
		return 0, err
	}

	userID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = s.autowpDB.Exec(`
		INSERT INTO user_account (service_id, external_id, user_id, used_for_reg, name, link)
		VALUES (?, ?, ?, 0, ?, "")
	`, KeyCloakExternalAccountID, userGuid, userID, options.Name)
	if err != nil {
		return 0, err
	}

	language, ok := s.languages[options.Language]
	if !ok {
		return 0, fmt.Errorf("language `%s` is not defined", options.Language)
	}

	err = s.sendRegistrationConfirmEmail(options.Email, emailCheckCode, options.Name, language.Hostname)
	if err != nil {
		return 0, err
	}

	err = s.RefreshUserConflicts(userID)
	if err != nil {
		return 0, err
	}

	err = s.UpdateUserVoteLimit(userID)
	if err != nil {
		return 0, err
	}

	_, err = s.autowpDB.Exec("UPDATE users SET votes_left = votes_per_day WHERE id = ?", userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *UserRepository) sendRegistrationConfirmEmail(email string, code string, name string, hostname string) error {
	if len(email) <= 0 || len(code) <= 0 {
		return nil
	}

	fromStr := "Robot " + hostname
	subject := fmt.Sprintf("Registration on %s", hostname)
	message := fmt.Sprintf(
		"Hello.\n"+
			"You are registered on website %s\n"+
			"Your registration details:\n"+
			"E-mail: %s\n"+
			"To confirm registration, and your e-mail address, you will need to click on the link %s\n\n"+
			"If you are not registered on the site, simply remove this message\n\n"+
			"Sincerely, %s",
		"https://"+hostname+"/",
		email,
		"https://"+hostname+"/account/emailcheck/"+url.QueryEscape(code),
		fromStr,
	)

	return s.emailSender.Send(fromStr+" <no-reply@autowp.ru>", []string{name + " <" + email + ">"}, subject, message, "")
}

func (s *UserRepository) UpdateUserVoteLimit(userId int64) error {

	var age int
	err := s.autowpDB.QueryRow("SELECT TIMESTAMPDIFF(YEAR, reg_date, NOW()) FROM users WHERE id = ?", userId).Scan(&age)
	if err != nil {
		return err
	}

	def := 10

	avgVote, err := s.GetUserAvgVote(userId)
	if err != nil {
		return err
	}

	var picturesExists int
	err = s.autowpDB.QueryRow("SELECT 1 FROM pictures WHERE owner_id = ? AND status = ? LIMIT 1", userId, "accepted").Scan(&picturesExists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	value := math.Round((avgVote + float64(def+age+picturesExists)) / 100)
	if value < 0 {
		value = 0
	}

	_, err = s.autowpDB.Exec("UPDATE users SET votes_per_day = ? WHERE id = ?", value, userId)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserRepository) GetUserAvgVote(userId int64) (float64, error) {
	var result float64
	err := s.autowpDB.QueryRow("SELECT IFNULL(avg(vote), 0) FROM comment_message WHERE author_id = ? AND vote <> 0", userId).Scan(&result)
	return result, err
}

func (s *UserRepository) RefreshUserConflicts(userId int64) error {
	_, err := s.autowpDB.Exec(`
		UPDATE users 
		SET users.specs_weight = (1.5 * ((1 + IFNULL((
		    SELECT sum(weight) FROM attrs_user_values WHERE user_id = users.id AND weight > 0
		), 0)) / (1 + IFNULL((
			SELECT abs(sum(weight)) FROM attrs_user_values WHERE user_id = users.id AND weight < 0
		), 0))))
		WHERE users.id = ?
	`, userId)
	return err
}

func (s *UserRepository) ensureUserExportedToKeyCloak(userID int64) (string, error) {
	var userGuid string
	err := s.autowpDB.QueryRow(`
		SELECT external_id FROM user_account WHERE service_id = ? AND user_id = ?
	`, KeyCloakExternalAccountID, userID).Scan(&userGuid)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if err == nil {
		return userGuid, nil
	}

	var deleted bool
	var email string
	var emailToCheck string
	var login string
	var name string
	err = s.autowpDB.QueryRow(`
			SELECT deleted, e_mail, e_mail_to_check, login, name FROM users WHERE id = ?
		`, userID).Scan(&deleted, &email, &emailToCheck, &login, &name)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return "", err
	}

	emailVerified := true
	if len(email) <= 0 {
		email = emailToCheck
		emailVerified = false
	}
	username := login
	if len(login) <= 0 {
		login = email
	}
	f := false
	return s.keyCloak.CreateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		Enabled:       &deleted,
		Totp:          &f,
		EmailVerified: &emailVerified,
		Username:      &username,
		FirstName:     &name,
		Email:         &email,
	})
}

func (s *UserRepository) SetPassword(userID int64, password string) error {

	userGuid, err := s.ensureUserExportedToKeyCloak(userID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return err
	}

	credentialsType := "PASSWORD"
	credentials := []gocloak.CredentialRepresentation{
		{
			Type:  &credentialsType,
			Value: &password,
		},
	}
	err = s.keyCloak.UpdateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		ID:          &userGuid,
		Credentials: &credentials,
	})
	if err != nil {
		return err
	}

	_, err = s.autowpDB.Exec(`
		UPDATE users SET password = MD5(CONCAT(?, ?)) WHERE id = ?
	`, s.usersSalt, password, userID)
	if err != nil {
		return err
	}

	return err
}

func (s *UserRepository) GetLogin(userID int64) (string, error) {
	var login string
	var email string
	err := s.autowpDB.QueryRow("SELECT login, e_mail FROM users WHERE id = ?", userID).Scan(&login, &email)
	if err != nil {
		return "", err
	}

	if len(email) > 0 {
		return email, nil
	}

	return login, nil
}

func (s *UserRepository) EmailChangeFinish(code string) error {
	if len(code) <= 0 {
		return fmt.Errorf("token is invalid")
	}

	var id int64
	var email string
	err := s.autowpDB.QueryRow(`
		SELECT id, email_to_check FROM users
		WHERE not deleted AND
		      email_check_code = ? AND
		      LENGTH(email_check_code) > 0 AND
		      LENGTH(email_to_check) > 0
	`, code).Scan(&id, &email)
	if err != nil {
		return err
	}

	_, err = s.autowpDB.Exec(`
		UPDATE users SET e_mail = email_to_check, email_check_code = NULL, email_to_check = NULL
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	return nil
}
