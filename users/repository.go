package users

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/Nerzal/gocloak/v11"
	"github.com/Nerzal/gocloak/v11/pkg/jwx"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"math"
	"net"
	"strings"
	"time"
)

type Claims struct {
	jwx.Claims
	Audience       []string       `json:"aud,omitempty"`
	Locale         string         `json:"locale,omitempty"`
	ResourceAccess ResourceAccess `json:"resource_access,omitempty"`
}

type ResourceAccess struct {
	Autowp AutowpResourceAccess `json:"autowp,omitempty"`
}

type AutowpResourceAccess struct {
	Roles []string `json:"roles,omitempty"`
}

const KeycloakExternalAccountID = "keycloak"

type GetUsersOptions struct {
	ID         int64
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
	ID          int64
	Name        string
	Deleted     bool
	Identity    *string
	LastOnline  *time.Time
	Role        string
	EMail       *string
	Img         *int
	SpecsWeight float64
}

// CreateUserOptions CreateUserOptions
type CreateUserOptions struct {
	UserName        string `json:"user_name"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	Timezone        string `json:"timezone"`
	Language        string `json:"language"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	Captcha         string `json:"captcha"`
}

// Repository Main Object
type Repository struct {
	autowpDB       *sql.DB
	usersSalt      string
	languages      map[string]config.LanguageConfig
	keycloak       gocloak.GoCloak
	keycloakConfig config.KeycloakConfig
}

// NewRepository constructor
func NewRepository(
	autowpDB *sql.DB,
	usersSalt string,
	languages map[string]config.LanguageConfig,
	keyCloak gocloak.GoCloak,
	keyCloakConfig config.KeycloakConfig,
) *Repository {

	return &Repository{
		autowpDB:       autowpDB,
		usersSalt:      usersSalt,
		languages:      languages,
		keycloak:       keyCloak,
		keycloakConfig: keyCloakConfig,
	}
}

func (s *Repository) User(options GetUsersOptions) (*DBUser, error) {

	users, err := s.Users(options)
	if err != nil {
		return nil, err
	}

	if len(users) <= 0 {
		return nil, nil
	}

	return &users[0], nil
}

func (s *Repository) Users(options GetUsersOptions) ([]DBUser, error) {

	result := make([]DBUser, 0)

	var r DBUser
	valuePtrs := []interface{}{&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight}

	sqSelect := sq.Select("users.id, users.name, users.deleted, users.identity, users.last_online, users.role, users.specs_weight").From("users")

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

func (s *Repository) GetVotesLeft(ctx context.Context, userID int64) (int, error) {
	var votesLeft int
	err := s.autowpDB.QueryRowContext(ctx, "SELECT votes_left FROM users WHERE id = ?", userID).Scan(&votesLeft)
	return votesLeft, err
}

func (s *Repository) DecVotes(ctx context.Context, userId int64) error {
	_, err := s.autowpDB.ExecContext(ctx, "UPDATE users SET votes_left = votes_left - 1 WHERE id = ?", userId)
	return err
}

func (s *Repository) AfterUserCreated(userID int64) error {
	err := s.RefreshUserConflicts(userID)
	if err != nil {
		return err
	}

	err = s.UpdateUserVoteLimit(userID)
	if err != nil {
		return err
	}

	_, err = s.autowpDB.Exec("UPDATE users SET votes_left = votes_per_day WHERE id = ?", userID)

	return err
}

func (s *Repository) UpdateUserVoteLimit(userId int64) error {

	var age int
	err := s.autowpDB.QueryRow("SELECT TIMESTAMPDIFF(YEAR, reg_date, NOW()) FROM users WHERE id = ?", userId).Scan(&age)
	if err != nil {
		return err
	}

	def := 10

	avgVote, err := s.UserAvgVote(userId)
	if err != nil {
		return err
	}

	var picturesExists int
	err = s.autowpDB.QueryRow("SELECT count(1) FROM pictures WHERE owner_id = ? AND status = ? LIMIT 1", userId, "accepted").Scan(&picturesExists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	value := math.Round(avgVote + float64(def+age+picturesExists/100))
	if value < 0 {
		value = 0
	}

	_, err = s.autowpDB.Exec("UPDATE users SET votes_per_day = ? WHERE id = ?", value, userId)
	if err != nil {
		return err
	}

	return nil
}

func (s *Repository) UserAvgVote(userId int64) (float64, error) {
	var result float64
	err := s.autowpDB.QueryRow("SELECT IFNULL(avg(vote), 0) FROM comment_message WHERE author_id = ? AND vote <> 0", userId).Scan(&result)
	return result, err
}

func (s *Repository) RefreshUserConflicts(userId int64) error {
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

func (s *Repository) EnsureUserImported(ctx context.Context, claims Claims) (int64, string, error) {

	remoteAddr := "127.0.0.1"
	p, ok := peer.FromContext(ctx)
	if ok {
		nw := p.Addr.String()
		if nw != "bufconn" {
			ip, _, err := net.SplitHostPort(nw)
			if err != nil {
				logrus.Errorf("userip: %q is not IP:port", nw)
			} else {
				remoteAddr = ip
			}
		}
	}

	locale := strings.ToLower(claims.Locale)

	language, ok := s.languages[locale]
	if !ok {
		locale = "en"
		language, ok = s.languages["en"]
	}
	if !ok {
		return 0, "", status.Errorf(codes.InvalidArgument, "language `%s` is not defined", locale)
	}

	guid := claims.Subject
	emailAddr := claims.Email
	name := fullName(claims.GivenName, claims.FamilyName, claims.PreferredUsername)
	role := "user"

	if util.Contains(claims.ResourceAccess.Autowp.Roles, "admin") {
		role = "admin"
	}

	logrus.Debugf("Ensure user `%s` imported", guid)

	var userID int64
	err := s.autowpDB.
		QueryRow("SELECT user_id FROM user_account WHERE service_id = ? AND external_id = ?", KeycloakExternalAccountID, guid).
		Scan(&userID)
	if err != nil && err != sql.ErrNoRows {
		return 0, "", err
	}

	if err == sql.ErrNoRows {
		var r sql.Result
		r, err = s.autowpDB.Exec(`
			INSERT INTO users (login, e_mail, password, email_to_check, hide_e_mail, email_check_code, name, reg_date, 
							   last_online, timezone, last_ip, language, role)
			VALUES (NULL, ?, NULL, NULL, 1, NULL, ?, NOW(), NOW(), ?, INET6_ATON(?), ?, ?)
		`, emailAddr, name, language.Timezone, remoteAddr, locale, role)
		if err != nil {
			return 0, "", err
		}

		userID, err = r.LastInsertId()
		if err != nil {
			return 0, "", err
		}

		err = s.AfterUserCreated(userID)
		if err != nil {
			return 0, "", err
		}

	} else {
		_, err = s.autowpDB.Exec(`
			UPDATE users SET e_mail = ?, name = ?, last_ip = ?
			WHERE id = ?
		`, emailAddr, name, remoteAddr, userID)
		if err != nil {
			return 0, "", err
		}

		err = s.autowpDB.
			QueryRow("SELECT role FROM users WHERE id = ?", userID).
			Scan(&role)
		if err != nil {
			return 0, "", err
		}
	}

	_, err = s.autowpDB.Exec(`
		INSERT INTO user_account (user_id, service_id, external_id, used_for_reg, name, link)
		VALUES (?, ?, ?, 0, ?, "")
		ON DUPLICATE KEY UPDATE user_id=VALUES(user_id), name=VALUES(name)`,
		userID,
		KeycloakExternalAccountID,
		guid,
		name,
	)
	if err != nil {
		return 0, "", err
	}

	return userID, role, nil
}

func (s *Repository) ensureUserExportedToKeycloak(userID int64) (string, error) {
	logrus.Debugf("Ensure user `%d` exported to Keycloak", userID)
	var userGuid string
	err := s.autowpDB.
		QueryRow("SELECT external_id FROM user_account WHERE service_id = ? AND user_id = ?", KeycloakExternalAccountID, userID).
		Scan(&userGuid)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if err == nil {
		return userGuid, nil
	}

	var deleted bool
	var userEmail sql.NullString
	var emailToCheck sql.NullString
	var login sql.NullString
	var name string
	err = s.autowpDB.
		QueryRow("SELECT deleted, e_mail, email_to_check, login, name FROM users WHERE id = ?", userID).
		Scan(&deleted, &userEmail, &emailToCheck, &login, &name)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	token, err := s.keycloak.LoginClient(
		ctx,
		s.keycloakConfig.ClientID,
		s.keycloakConfig.ClientSecret,
		s.keycloakConfig.Realm,
	)
	if err != nil {
		return "", err
	}

	var keyCloakEmail = &userEmail.String
	emailVerified := true
	if !userEmail.Valid || len(userEmail.String) <= 0 {
		keyCloakEmail = &emailToCheck.String
		emailVerified = false
	}
	username := login.String
	if (!login.Valid || len(login.String) <= 0) && keyCloakEmail != nil && len(*keyCloakEmail) > 0 {
		username = *keyCloakEmail
	}
	f := false
	enabled := !deleted
	userGuid, err = s.keycloak.CreateUser(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.User{
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
	`, KeycloakExternalAccountID, userGuid, userID, name)
	if err != nil {
		return "", err
	}

	return userGuid, err
}

func (s *Repository) PasswordMatch(userID int64, password string) (bool, error) {
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

func (s *Repository) DeleteUser(userID int64) (bool, error) {

	userGuid, err := s.ensureUserExportedToKeycloak(userID)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	token, err := s.keycloak.LoginClient(
		ctx,
		s.keycloakConfig.ClientID,
		s.keycloakConfig.ClientSecret,
		s.keycloakConfig.Realm,
	)
	if err != nil {
		return false, err
	}

	f := false
	err = s.keycloak.UpdateUser(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.User{
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
	`, userID, KeycloakExternalAccountID)
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

func (s *Repository) RestoreVotes() error {
	_, err := s.autowpDB.Exec(
		"UPDATE users SET votes_left = votes_per_day WHERE votes_left < votes_per_day AND not deleted",
	)
	return err
}

func (s *Repository) UpdateVotesLimits() (int, error) {

	rows, err := s.autowpDB.Query(
		"SELECT id FROM users WHERE NOT deleted AND last_online > DATE_SUB(NOW(), INTERVAL 3 MONTH)",
	)
	if err != nil {
		return 0, err
	}
	defer util.Close(rows)

	affected := 0

	for rows.Next() {
		var userID int64
		err = rows.Scan(&userID)
		if err != nil {
			return 0, err
		}
		err = s.UpdateUserVoteLimit(userID)
		if err != nil {
			return 0, err
		}
		affected++
	}

	return affected, nil
}

func (s *Repository) UpdateSpecsVolumes() error {

	rows, err := s.autowpDB.Query(`
		SELECT id, count(attrs_user_values.user_id)
		FROM users
			LEFT JOIN attrs_user_values ON attrs_user_values.user_id = users.id
		WHERE NOT not users.specs_volume_valid AND NOT users.deleted
		GROUP BY users.id
	`)
	if err != nil {
		return err
	}
	defer util.Close(rows)

	for rows.Next() {
		var userID int64
		var count int
		err = rows.Scan(&userID, &count)
		if err != nil {
			return err
		}
		_, err = s.autowpDB.Exec(
			"UPDATE users SET specs_volume = ?, specs_volume_valid = 1 WHERE id = ?",
			userID, count,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Repository) ExportUsersToKeycloak() error {
	rows, err := s.autowpDB.Query(`
		SELECT id 
		FROM users 
		WHERE LENGTH(login) > 0 OR LENGTH(e_mail) > 0 OR LENGTH(email_to_check) > 0 
		ORDER BY id DESC
	`)
	if err != nil {
		return err
	}
	defer util.Close(rows)

	for rows.Next() {
		var userID int64
		err = rows.Scan(&userID)
		if err != nil {
			return err
		}

		guid, err := s.ensureUserExportedToKeycloak(userID)
		if err != nil {
			logrus.Debugf("Error exporting user %d", userID)
			return err
		}

		logrus.Debugf("User %d exported to keycloak as %s", userID, guid)
	}
	return nil
}

func fullName(firstName, lastName, username string) string {
	result := strings.TrimSpace(firstName + " " + lastName)
	if len(result) <= 0 {
		result = username
	}
	return result
}
