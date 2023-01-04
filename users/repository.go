package users

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	sq "github.com/Masterminds/squirrel"
	"github.com/Nerzal/gocloak/v11"
	"github.com/Nerzal/gocloak/v11/pkg/jwx"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	Decimal   = 10
	BitSize64 = 64
)

var ErrUserNotFound = errors.New("user not found")

type Claims struct {
	jwx.Claims
	Audience       interface{}    `json:"aud,omitempty"`
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

// APIUser APIUser.
type APIUser struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Deleted     bool       `json:"deleted"`
	LongAway    bool       `json:"long_away"`
	Green       bool       `json:"green"`
	Route       []string   `json:"route"`
	Identity    string     `json:"identity"`
	Avatar      *APIImage  `json:"avatar,omitempty"`
	Gravatar    string     `json:"gravatar,omitempty"`
	LastOnline  *time.Time `json:"last_online,omitempty"`
	SpecsWeight float64    `json:"specs_weight,omitempty"`
}

// DBUser DBUser.
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

// CreateUserOptions CreateUserOptions.
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

// Repository Main Object.
type Repository struct {
	autowpDB        *goqu.Database
	db              *goqu.Database
	usersSalt       string
	languages       map[string]config.LanguageConfig
	keycloak        gocloak.GoCloak
	keycloakConfig  config.KeycloakConfig
	messageInterval int64
}

// UserPreferences object.
type UserPreferences struct {
	DisableCommentsNotifications bool `db:"disable_comments_notifications" json:"disable_comments_notifications"`
}

// NewRepository constructor.
func NewRepository(
	autowpDB *goqu.Database,
	db *goqu.Database,
	usersSalt string,
	languages map[string]config.LanguageConfig,
	keyCloak gocloak.GoCloak,
	keyCloakConfig config.KeycloakConfig,
	messageInterval int64,
) *Repository {
	return &Repository{
		autowpDB:        autowpDB,
		db:              db,
		usersSalt:       usersSalt,
		languages:       languages,
		keycloak:        keyCloak,
		keycloakConfig:  keyCloakConfig,
		messageInterval: messageInterval,
	}
}

func (s *Repository) User(options GetUsersOptions) (*DBUser, error) {
	users, err := s.Users(options)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	return &users[0], nil
}

func (s *Repository) Users(options GetUsersOptions) ([]DBUser, error) {
	result := make([]DBUser, 0)

	var r DBUser
	valuePtrs := []interface{}{&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight}

	sqSelect := sq.Select(
		"users.id, users.name, users.deleted, users.identity, users.last_online, users.role, users.specs_weight",
	).From("users")

	if options.ID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"users.id": options.ID})
	}

	if options.InContacts != 0 {
		sqSelect = sqSelect.Join("contact ON users.id = contact.contact_user_id").
			Where(sq.Eq{"contact.user_id": options.InContacts})
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
	if errors.Is(err, sql.ErrNoRows) {
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

func (s *Repository) DecVotes(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.ExecContext(ctx, "UPDATE users SET votes_left = votes_left - 1 WHERE id = ?", userID)

	return err
}

func (s *Repository) AfterUserCreated(ctx context.Context, userID int64) error {
	err := s.RefreshUserConflicts(ctx, userID)
	if err != nil {
		return err
	}

	err = s.UpdateUserVoteLimit(ctx, userID)
	if err != nil {
		return err
	}

	_, err = s.autowpDB.ExecContext(ctx, "UPDATE users SET votes_left = votes_per_day WHERE id = ?", userID)

	return err
}

func (s *Repository) UpdateUserVoteLimit(ctx context.Context, userID int64) error {
	var age int

	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT TIMESTAMPDIFF(YEAR, reg_date, NOW()) FROM users WHERE id = ?",
		userID,
	).Scan(&age)
	if err != nil {
		return err
	}

	def := 10

	avgVote, err := s.UserAvgVote(ctx, userID)
	if err != nil {
		return err
	}

	var picturesExists int
	err = s.autowpDB.QueryRowContext(
		ctx,
		"SELECT count(1) FROM pictures WHERE owner_id = ? AND status = ? LIMIT 1",
		userID,
		"accepted",
	).Scan(&picturesExists)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	value := math.Round(avgVote + float64(def+age+picturesExists/100))
	if value < 0 {
		value = 0
	}

	_, err = s.autowpDB.ExecContext(ctx, "UPDATE users SET votes_per_day = ? WHERE id = ?", value, userID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Repository) UserAvgVote(ctx context.Context, userID int64) (float64, error) {
	var result float64
	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT IFNULL(avg(vote), 0) FROM comment_message WHERE author_id = ? AND vote <> 0",
		userID,
	).Scan(&result)

	return result, err
}

func (s *Repository) RefreshUserConflicts(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.ExecContext(ctx, `
		UPDATE users 
		SET users.specs_weight = (1.5 * ((1 + IFNULL((
		    SELECT sum(weight) FROM attrs_user_values WHERE user_id = users.id AND weight > 0
		), 0)) / (1 + IFNULL((
			SELECT abs(sum(weight)) FROM attrs_user_values WHERE user_id = users.id AND weight < 0
		), 0))))
		WHERE users.id = ?
	`, userID)

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

	var r sql.Result

	r, err := s.autowpDB.Insert("users").
		Rows(goqu.Record{
			"login":            nil,
			"e_mail":           emailAddr,
			"password":         nil,
			"email_to_check":   nil,
			"hide_e_mail":      1,
			"email_check_code": nil,
			"name":             name,
			"reg_date":         goqu.L("NOW()"),
			"last_online":      goqu.L("NOW()"),
			"timezone":         language.Timezone,
			"last_ip":          goqu.L("INET6_ATON(?)", remoteAddr),
			"language":         locale,
			"role":             role,
			"uuid":             goqu.L("UUID_TO_BIN(?)", guid),
		}).
		OnConflict(goqu.DoUpdate("uuid", goqu.Record{
			"e_mail":  goqu.L("values(e_mail)"),
			"name":    goqu.L("values(name)"),
			"last_ip": goqu.L("values(last_ip)"),
		})).
		Executor().Exec()
	if err != nil {
		return 0, "", err
	}

	affected, err := r.RowsAffected()
	if err != nil {
		return 0, "", err
	}

	if affected == 1 { // row just inserted
		userID, err := r.LastInsertId()
		if err != nil {
			return 0, "", err
		}

		err = s.AfterUserCreated(ctx, userID)
		if err != nil {
			return 0, "", err
		}
	}

	row := struct {
		ID   int64  `db:"id"`
		Role string `db:"role"`
	}{}

	success, err := s.autowpDB.Select("id", "role").
		From("users").
		Where(goqu.L("uuid = UUID_TO_BIN(?)", guid)).
		ScanStructContext(ctx, &row)
	if err != nil {
		return 0, "", err
	}

	if !success {
		return 0, "", ErrUserNotFound
	}

	return row.ID, row.Role, nil
}

func (s *Repository) ensureUserExportedToKeycloak(ctx context.Context, userID int64) (string, error) {
	logrus.Debugf("Ensure user `%d` exported to Keycloak", userID)

	var (
		userGUID     string
		deleted      bool
		userEmail    sql.NullString
		emailToCheck sql.NullString
		login        sql.NullString
		name         string
	)

	err := s.autowpDB.
		QueryRowContext(
			ctx,
			"SELECT deleted, e_mail, email_to_check, login, name, IFNULL(BIN_TO_UUID(uuid), '') FROM users WHERE id = ?",
			userID,
		).Scan(&deleted, &userEmail, &emailToCheck, &login, &name, &userGUID)
	if err != nil {
		return "", err
	}

	if len(userGUID) > 0 {
		return userGUID, nil
	}

	token, err := s.keycloak.LoginClient(
		ctx,
		s.keycloakConfig.ClientID,
		s.keycloakConfig.ClientSecret,
		s.keycloakConfig.Realm,
	)
	if err != nil {
		return "", err
	}

	var (
		keyCloakEmail = &userEmail.String
		emailVerified = true
	)

	if !userEmail.Valid || len(userEmail.String) == 0 {
		keyCloakEmail = &emailToCheck.String
		emailVerified = false
	}

	username := login.String
	if (!login.Valid || len(login.String) == 0) && keyCloakEmail != nil && len(*keyCloakEmail) > 0 {
		username = *keyCloakEmail
	}

	f := false
	enabled := !deleted
	userGUID, err = s.keycloak.CreateUser(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.User{
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

	_, err = s.autowpDB.ExecContext(ctx, "UPDATE users SET uuid = UUID_TO_BIN(?) WHERE user_id = ?", userGUID, userID)
	if err != nil {
		return "", err
	}

	return userGUID, err
}

func (s *Repository) PasswordMatch(ctx context.Context, userID int64, password string) (bool, error) {
	var exists bool
	err := s.autowpDB.QueryRowContext(ctx, `
		SELECT 1 FROM users 
		WHERE password = MD5(CONCAT(?, ?)) AND id = ? AND NOT deleted
	`, s.usersSalt, password, userID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Repository) DeleteUser(ctx context.Context, userID int64) (bool, error) {
	userGUID, err := s.ensureUserExportedToKeycloak(ctx, userID)
	if err != nil {
		return false, err
	}

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
		ID:      &userGUID,
		Enabled: &f,
	})

	if err != nil {
		return false, err
	}

	var val int
	err = s.autowpDB.QueryRowContext(ctx, "SELECT 1 FROM users WHERE id = ? AND NOT deleted", userID).Scan(&val)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	// $oldImageId = $row['img'];

	_, err = s.autowpDB.ExecContext(ctx, `
		UPDATE users SET deleted = 1 WHERE id = ?
	`, userID)
	// 'img'     => null,
	if err != nil {
		return false, err
	}

	/*if ($oldImageId) {
		$this->imageStorage->removeImage($oldImageId);
	}*/

	_, err = s.autowpDB.ExecContext(ctx, "DELETE FROM telegram_chat WHERE user_id = ?", userID)
	if err != nil {
		return false, err
	}

	// delete linked profiles
	_, err = s.autowpDB.ExecContext(ctx, `
		DELETE FROM user_account WHERE user_id = ? AND service_id != ?
	`, userID, KeycloakExternalAccountID)
	if err != nil {
		return false, err
	}

	// unsubscribe from items
	_, err = s.autowpDB.ExecContext(ctx, `
		DELETE FROM user_item_subscribe WHERE user_id = ?
	`, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Repository) RestoreVotes(ctx context.Context) error {
	_, err := s.autowpDB.ExecContext(
		ctx,
		"UPDATE users SET votes_left = votes_per_day WHERE votes_left < votes_per_day AND not deleted",
	)

	return err
}

func (s *Repository) UpdateVotesLimits(ctx context.Context) (int, error) {
	rows, err := s.autowpDB.QueryContext(
		ctx,
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

		err = s.UpdateUserVoteLimit(ctx, userID)

		if err != nil {
			return 0, err
		}
		affected++
	}

	return affected, nil
}

func (s *Repository) UpdateSpecsVolumes(ctx context.Context) error {
	rows, err := s.autowpDB.QueryContext(ctx, `
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
		var (
			userID int64
			count  int
		)

		err = rows.Scan(&userID, &count)

		if err != nil {
			return err
		}

		_, err = s.autowpDB.ExecContext(
			ctx,
			"UPDATE users SET specs_volume = ?, specs_volume_valid = 1 WHERE id = ?",
			userID, count,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) ExportUsersToKeycloak(ctx context.Context) error {
	rows, err := s.autowpDB.QueryContext(ctx, `
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

		guid, err := s.ensureUserExportedToKeycloak(ctx, userID)
		if err != nil {
			logrus.Debugf("Error exporting user %d", userID)

			return err
		}

		logrus.Debugf("User %d exported to keycloak as %s", userID, guid)
	}

	return nil
}

func (s *Repository) SetDisableUserCommentsNotifications(
	ctx context.Context,
	userID int64,
	toUserID int64,
	disabled bool,
) error {
	query := s.db.Insert("user_user_preferences").
		Rows(goqu.Record{
			"user_id":                        userID,
			"to_user_id":                     toUserID,
			"disable_comments_notifications": disabled,
		}).
		OnConflict(goqu.DoUpdate("user_id, to_user_id", goqu.Record{
			"disable_comments_notifications": goqu.L("EXCLUDED.disable_comments_notifications"),
		}))

	_, err := query.Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UserPreferences(ctx context.Context, userID int64, toUserID int64) (*UserPreferences, error) {
	var row UserPreferences

	_, err := s.db.Select("disable_comments_notifications").
		From("user_user_preferences").
		Where(
			goqu.I("user_id").Eq(userID),
			goqu.I("to_user_id").Eq(toUserID),
		).ScanStructContext(ctx, &row)

	return &row, err
}

func (s *Repository) SetupPrivateRouter(r *gin.Engine) {
	r.GET("/user-user-preferences/:user_id/:to_user_id", func(c *gin.Context) {
		userID, err := strconv.ParseInt(c.Param("user_id"), Decimal, BitSize64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid user_id")

			return
		}

		toUserID, err := strconv.ParseInt(c.Param("to_user_id"), Decimal, BitSize64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid to_user_id")

			return
		}

		prefs, err := s.UserPreferences(c, userID, toUserID)
		if err != nil {
			c.String(http.StatusInternalServerError, "InternalServerError")

			return
		}

		c.JSON(http.StatusOK, prefs)
	})
}

func (s *Repository) IncForumMessages(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.ExecContext(
		ctx,
		"UPDATE users SET forums_messages = forums_messages + 1, last_message_time = NOW() WHERE id = ?",
		userID,
	)

	return err
}

func (s *Repository) TouchLastMessage(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.ExecContext(
		ctx,
		"UPDATE users SET last_message_time = NOW() WHERE id = ?",
		userID,
	)

	return err
}

func fullName(firstName, lastName, username string) string {
	result := strings.TrimSpace(firstName + " " + lastName)
	if len(result) == 0 {
		result = username
	}

	return result
}

func (s *Repository) messagingInterval(regDate time.Time, messagingInterval int64) int64 {
	if regDate.IsZero() {
		return s.messageInterval
	}

	tenDaysBefore := time.Now().AddDate(0, 0, -10)
	if tenDaysBefore.After(regDate) {
		return messagingInterval
	}

	return util.MaxInt64(messagingInterval, s.messageInterval)
}

func (s *Repository) NextMessageTime(ctx context.Context, userID int64) (time.Time, error) {
	if s.messageInterval <= 0 {
		return time.Time{}, nil
	}

	var (
		lastMessageTime   sql.NullTime
		regDate           sql.NullTime
		messagingInterval int64
	)

	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT last_message_time, reg_date, messaging_interval FROM users WHERE id = ?",
		userID,
	).Scan(&lastMessageTime, &regDate, &messagingInterval)

	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, err
	}

	if lastMessageTime.Valid {
		messagingInterval = s.messagingInterval(regDate.Time, messagingInterval)
		if messagingInterval > 0 {
			interval := time.Second * time.Duration(messagingInterval)

			return lastMessageTime.Time.Add(interval), nil
		}
	}

	return time.Time{}, nil
}
