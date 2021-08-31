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
	InContacts int64
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
) *UserRepository {

	return &UserRepository{
		autowpDB:       autowpDB,
		usersSalt:      usersSalt,
		emailSalt:      emailSalt,
		languages:      languages,
		emailSender:    emailSender,
		keyCloak:       keyCloak,
		keyCloakConfig: keyCloakConfig,
	}
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
	var err error

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Min: 2, Max: 50},
		},
	}
	options.Name, problems, err = nameInputFilter.IsValidString(options.Name)
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
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.EmailAddress{},
			&validation.StringLength{Max: 50},
			&validation.EmailNotExists{DB: s.autowpDB},
		},
	}
	options.Email, problems, err = emailInputFilter.IsValidString(options.Email)
	if err != nil {
		return nil, err
	}
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
	options.Password, problems, err = passwordInputFilter.IsValidString(options.Password)
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
	options.PasswordConfirm, problems, err = passwordConfirmInputFilter.IsValidString(options.PasswordConfirm)
	if err != nil {
		return nil, err
	}
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
		options.Captcha, problems, err = captchaInputFilter.IsValidString(options.Captcha)
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

func (s *UserRepository) emailChangeCode(email string) string {
	md5Bytes := md5.Sum([]byte(fmt.Sprintf("%s%s%d", s.emailSalt, email, rand.Int())))
	return hex.EncodeToString(md5Bytes[:])
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

	emailCheckCode := s.emailChangeCode(options.Email)

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
	var email sql.NullString
	var emailToCheck sql.NullString
	var login string
	var name string
	err = s.autowpDB.QueryRow(`
			SELECT deleted, e_mail, email_to_check, login, name FROM users WHERE id = ?
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

	var keyCloakEmail = &email.String
	emailVerified := true
	if !email.Valid || len(email.String) <= 0 {
		keyCloakEmail = &emailToCheck.String
		emailVerified = false
	}
	username := login
	if len(login) <= 0 && keyCloakEmail != nil {
		login = *keyCloakEmail
	}
	f := false
	enabled := !deleted
	userGuid, err = s.keyCloak.CreateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		Enabled:       &enabled,
		Totp:          &f,
		EmailVerified: &emailVerified,
		Username:      &username,
		FirstName:     &name,
		Email:         keyCloakEmail,
	})
	if err != nil {
		return "", err
	}

	_, err = s.autowpDB.Exec(`
		INSERT INTO user_account (service_id, external_id, user_id, used_for_reg, name, link)
		VALUES (?, ?, ?, 0, ?, "")
		ON DUPLICATE KEY UPDATE user_id=VALUES(user_id), name=VALUES(name);
	`, KeyCloakExternalAccountID, userGuid, userID, name)
	if err != nil {
		return "", err
	}

	return userGuid, err
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

func (s *UserRepository) EmailChangeStart(userID int64, email string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string
	var err error

	emailInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.EmailAddress{},
			&validation.StringLength{Max: 50},
			&validation.EmailNotExists{DB: s.autowpDB},
		},
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

	var name string
	var languageCode string
	err = s.autowpDB.QueryRow(`
		SELECT name, language FROM users
		WHERE id = ?
	`, userID).Scan(&name, &languageCode)
	if err != nil {
		return nil, err
	}

	language, ok := s.languages[languageCode]
	if !ok {
		return nil, fmt.Errorf("language `%s` is not defined", languageCode)
	}

	emailCheckCode := s.emailChangeCode(email)

	_, err = s.autowpDB.Exec(`
		UPDATE users SET email_to_check = ?, email_check_code = ?
		WHERE id = ?
	`, email, emailCheckCode, userID)

	return nil, s.sendChangeConfirmEmail(email, emailCheckCode, name, language.Hostname)
}

func (s *UserRepository) EmailChangeFinish(ctx context.Context, code string) error {
	if len(code) <= 0 {
		return fmt.Errorf("token is invalid")
	}

	var userID int64
	var email string
	err := s.autowpDB.QueryRow(`
		SELECT id, email_to_check FROM users
		WHERE not deleted AND
		      email_check_code = ? AND
		      LENGTH(email_check_code) > 0 AND
		      LENGTH(email_to_check) > 0
	`, code).Scan(&userID, &email)
	if err != nil {
		return err
	}

	_, err = s.autowpDB.Exec(`
		UPDATE users SET e_mail = email_to_check, email_check_code = NULL, email_to_check = NULL
		WHERE id = ?
	`, userID)
	if err != nil {
		return err
	}

	userGuid, err := s.ensureUserExportedToKeyCloak(userID)
	if err != nil {
		return err
	}

	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return err
	}

	return s.keyCloak.UpdateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		ID:       &userGuid,
		Email:    &email,
		Username: &email,
	})
}

func (s *UserRepository) sendChangeConfirmEmail(email string, code string, name string, hostname string) error {
	if len(email) <= 0 || len(code) <= 0 {
		return nil
	}

	fromStr := "Robot " + hostname
	subject := fmt.Sprintf("E-mail confirm on %s", hostname)
	message := fmt.Sprintf(
		"Hello.\n\n"+
			"On the %s you or someone else asked to change contact address of account to %s\n"+
			"For confirmation of this action, you must click on the link %s\n\n"+
			"If the message has got to you by mistake - just delete it\n\n"+
			"Sincerely, %s",
		"https://"+hostname+"/",
		email,
		"https://"+hostname+"/account/emailcheck/"+url.QueryEscape(code),
		fromStr,
	)

	return s.emailSender.Send(fromStr+" <no-reply@autowp.ru>", []string{name + " <" + email + ">"}, subject, message, "")
}

func (s *UserRepository) ValidateChangePassword(userID int64, oldPassword, newPassword, newPasswordConfirm string) ([]*errdetails.BadRequest_FieldViolation, error) {

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string
	var err error

	oldPasswordInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.Callback{
				Callback: func(value string) ([]string, error) {
					match, err := s.PasswordMatch(userID, oldPassword)
					if err != nil {
						return nil, err
					}
					if !match {
						return []string{"Current password is incorrect"}, nil
					}
					return []string{}, nil
				},
			},
		},
	}
	oldPassword, problems, err = oldPasswordInputFilter.IsValidString(oldPassword)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "oldPassword",
			Description: fv,
		})
	}

	newPasswordInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
		},
	}
	newPassword, problems, err = newPasswordInputFilter.IsValidString(newPassword)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "newPassword",
			Description: fv,
		})
	}

	newPasswordConfirmInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{
				Min: 6,
				Max: 50,
			},
			&validation.IdenticalStrings{Pattern: newPassword},
		},
	}
	newPasswordConfirm, problems, err = newPasswordConfirmInputFilter.IsValidString(newPasswordConfirm)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "newPasswordConfirm",
			Description: fv,
		})
	}

	return result, nil
}

func (s *UserRepository) PasswordMatch(userID int64, password string) (bool, error) {
	var exists bool
	err := s.autowpDB.QueryRow(`
		SELECT 1 FROM users 
		WHERE password = MD5(CONCAT(?, ?)) AND id = ? AND NOT deleted
	`, s.usersSalt, password, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *UserRepository) DeleteUser(userID int64) (bool, error) {

	userGuid, err := s.ensureUserExportedToKeyCloak(userID)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return false, err
	}

	f := false
	err = s.keyCloak.UpdateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		ID:      &userGuid,
		Enabled: &f,
	})
	if err != nil {
		return false, err
	}

	var val int
	err = s.autowpDB.QueryRow("SELECT 1 FROM users WHERE id = ? AND NOT deleted", userID).Scan(&val)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// $oldImageId = $row['img'];

	_, err = s.autowpDB.Exec(`
		UPDATE users SET deleted = 1 WHERE id = ?
	`, userID)
	// 'img'     => null,
	if err != nil {
		return false, err
	}

	/*if ($oldImageId) {
		$this->imageStorage->removeImage($oldImageId);
	}*/

	_, err = s.autowpDB.Exec("DELETE FROM telegram_chat WHERE user_id = ?", userID)
	if err != nil {
		return false, err
	}

	// delete linked profiles
	_, err = s.autowpDB.Exec(`
		DELETE FROM user_account WHERE user_id = ? AND service_id != ?
	`, userID, KeyCloakExternalAccountID)
	if err != nil {
		return false, err
	}

	// unsubscribe from items
	_, err = s.autowpDB.Exec(`
		DELETE FROM user_item_subscribe WHERE user_id = ?
	`, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *UserRepository) UpdateUser(ctx context.Context, userID int64, name string) ([]*errdetails.BadRequest_FieldViolation, error) {

	userGuid, err := s.ensureUserExportedToKeyCloak(userID)
	if err != nil {
		return nil, err
	}

	result := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{
			&validation.StringTrimFilter{},
			&validation.StringSingleSpaces{},
		},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Min: 2, Max: 50},
		},
	}
	name, problems, err = nameInputFilter.IsValidString(name)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	if len(result) > 0 {
		return result, nil
	}

	oldName := ""
	err = s.autowpDB.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&oldName)
	if err != nil {
		return nil, err
	}

	_, err = s.autowpDB.Exec("UPDATE users SET name = ? WHERE id = ?", name, userID)
	if err != nil {
		return nil, err
	}

	_, err = s.autowpDB.Exec("INSERT INTO user_renames (user_id, old_name, new_name, date) VALUES (?, ?, ?, NOW())", userID, oldName, name)
	if err != nil {
		return nil, err
	}

	token, err := s.keyCloak.LoginClient(
		ctx,
		s.keyCloakConfig.ClientID,
		s.keyCloakConfig.ClientSecret,
		s.keyCloakConfig.Realm,
	)
	if err != nil {
		return nil, err
	}

	err = s.keyCloak.UpdateUser(ctx, token.AccessToken, s.keyCloakConfig.Realm, gocloak.User{
		ID:        &userGuid,
		FirstName: &name,
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
