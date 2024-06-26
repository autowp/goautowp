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

	"github.com/Nerzal/gocloak/v13"
	"github.com/Nerzal/gocloak/v13/pkg/jwx"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const lastOnlineUpdateThreshold = 5 * time.Second

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
	ID          int64
	InContacts  int64
	Order       []exp.OrderedExpression
	Deleted     *bool
	HasSpecs    *bool
	IsOnline    bool
	HasPictures *bool
	Limit       uint64
	Page        uint64
}

// DBUser DBUser.
type DBUser struct {
	ID            int64
	Name          string
	Deleted       bool
	Identity      *string
	LastOnline    *time.Time
	Role          string
	EMail         *string
	Img           *int
	SpecsWeight   float64
	SpecsVolume   int64
	PicturesTotal int64
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
	keycloak        *gocloak.GoCloak
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
	keyCloak *gocloak.GoCloak,
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

func (s *Repository) User(ctx context.Context, options GetUsersOptions) (*DBUser, error) {
	users, _, err := s.Users(ctx, options)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	return &users[0], nil
}

func (s *Repository) UserIDByIdentity(ctx context.Context, identity string) (int64, error) {
	var userID int64

	success, err := s.autowpDB.From(schema.UserTable).
		Where(schema.UserTableIdentityCol.Eq(identity)).
		ScanValContext(ctx, &userID)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return userID, nil
}

func (s *Repository) Users(ctx context.Context, options GetUsersOptions) ([]DBUser, *util.Pages, error) {
	var err error

	result := make([]DBUser, 0)

	var r DBUser
	valuePtrs := []interface{}{
		&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role,
		&r.SpecsWeight, &r.Img, &r.EMail, &r.PicturesTotal, &r.SpecsVolume,
	}

	columns := []interface{}{
		schema.UserTableIDCol, schema.UserTableNameCol, schema.UserTableDeletedCol, schema.UserTableIdentityCol,
		schema.UserTableLastOnlineCol, schema.UserTableRoleCol, schema.UserTableSpecsWeightCol, schema.UserTableImgCol,
		schema.UserTableEmailCol, schema.UserTablePicturesTotalCol, schema.UserTableSpecsVolumeCol,
	}

	sqSelect := s.autowpDB.From(schema.UserTable)

	if options.ID != 0 {
		sqSelect = sqSelect.Where(schema.UserTableIDCol.Eq(options.ID))
	}

	if options.InContacts != 0 {
		sqSelect = sqSelect.Join(
			schema.ContactTable,
			goqu.On(schema.UserTableIDCol.Eq(schema.ContactTable.Col("contact_user_id")))).
			Where(schema.ContactTable.Col("user_id").Eq(options.InContacts))
	}

	if options.Deleted != nil {
		if *options.Deleted {
			sqSelect = sqSelect.Where(schema.UserTableDeletedCol.IsTrue())
		} else {
			sqSelect = sqSelect.Where(schema.UserTableDeletedCol.IsFalse())
		}
	}

	if options.HasSpecs != nil {
		if *options.HasSpecs {
			sqSelect = sqSelect.Where(schema.UserTableSpecsVolumeCol.Gt(0))
		} else {
			sqSelect = sqSelect.Where(schema.UserTableSpecsVolumeCol.Eq(0))
		}
	}

	if options.IsOnline {
		sqSelect = sqSelect.Where(schema.UserTableLastOnlineCol.Gte(goqu.L("DATE_SUB(NOW(), INTERVAL 5 MINUTE)")))
	}

	if len(options.Order) > 0 {
		sqSelect = sqSelect.Order(options.Order...)
	}

	sqSelect = sqSelect.Select(columns...)

	var pages *util.Pages

	if options.Page > 0 {
		paginator := util.Paginator{
			SQLSelect:        sqSelect,
			ItemCountPerPage: int32(options.Limit),
		}

		pages, err = paginator.GetPages(ctx)
		if err != nil {
			return nil, nil, err
		}

		sqSelect, err = paginator.GetItemsByPage(ctx, int32(options.Page))
		if err != nil {
			return nil, nil, err
		}
	} else if options.Limit > 0 {
		sqSelect = sqSelect.Limit(uint(options.Limit))
	}

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return result, pages, nil
	}

	if err != nil {
		return nil, nil, err
	}

	defer util.Close(rows)

	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, r)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return result, pages, nil
}

func (s *Repository) GetVotesLeft(ctx context.Context, userID int64) (int, error) {
	var votesLeft int

	success, err := s.autowpDB.Select(schema.UserTableVotesLeftCol).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanValContext(ctx, &votesLeft)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return votesLeft, nil
}

func (s *Repository) DecVotes(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableVotesLeftColName: goqu.L(schema.UserTableVotesLeftColName + " - 1"),
	}).Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)

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

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableVotesLeftColName: schema.UserTableVotesPerDayCol,
	}).Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UpdateUserVoteLimit(ctx context.Context, userID int64) error {
	var age int

	success, err := s.autowpDB.Select(goqu.L("TIMESTAMPDIFF(YEAR, "+schema.UserTableRegDateColName+", NOW())")).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanValContext(ctx, &age)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	def := 10

	avgVote, err := s.UserAvgVote(ctx, userID)
	if err != nil {
		return err
	}

	var picturesExists int

	success, err = s.autowpDB.Select(goqu.COUNT(goqu.Star())).From(schema.PictureTable).Where(
		schema.PictureTableOwnerIDCol.Eq(userID),
		schema.PictureTableStatusCol.Eq("accepted"),
	).ScanValContext(ctx, &picturesExists)
	if err != nil {
		return err
	}

	if !success {
		picturesExists = 0
	}

	value := math.Round(avgVote + float64(def+age+picturesExists/100))
	if value < 0 {
		value = 0
	}

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableVotesPerDayColName: value,
	}).Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Repository) UserAvgVote(ctx context.Context, userID int64) (float64, error) {
	var result float64
	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT IFNULL(avg(vote), 0) FROM "+schema.CommentMessageTableName+" WHERE author_id = ? AND vote <> 0",
		userID,
	).Scan(&result)

	return result, err
}

func (s *Repository) RefreshUserConflicts(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.ExecContext(ctx, `
		UPDATE `+schema.UserTableName+` 
		SET `+schema.UserTableName+`.`+schema.UserTableSpecsWeightColName+` = (1.5 * ((1 + IFNULL((
		    SELECT sum(weight) FROM `+schema.AttrsUserValuesTableName+` 
			WHERE user_id = `+schema.UserTableName+`.id AND weight > 0
		), 0)) / (1 + IFNULL((
			SELECT abs(sum(weight)) FROM `+schema.AttrsUserValuesTableName+` 
			WHERE user_id = `+schema.UserTableName+`.id AND weight < 0
		), 0))))
		WHERE `+schema.UserTableName+`.id = ?
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

	r, err := s.autowpDB.Insert(schema.UserTable).
		Rows(goqu.Record{
			schema.UserTableLoginColName:          nil,
			schema.UserTableEmailColName:          emailAddr,
			schema.UserTablePasswordColName:       nil,
			schema.UserTableEmailToCheckColName:   nil,
			schema.UserTableHideEmailColName:      1,
			schema.UserTableEmailCheckCodeColName: nil,
			schema.UserTableNameColName:           name,
			schema.UserTableRegDateColName:        goqu.Func("NOW"),
			schema.UserTableLastOnlineColName:     goqu.Func("NOW"),
			schema.UserTableTimezoneColName:       language.Timezone,
			schema.UserTableLastIPColName:         goqu.Func("INET6_ATON", remoteAddr),
			schema.UserTableLanguageColName:       locale,
			schema.UserTableRoleColName:           role,
			schema.UserTableUUIDColName:           goqu.Func("UUID_TO_BIN", guid),
		}).
		OnConflict(goqu.DoUpdate(schema.UserTableUUIDColName, goqu.Record{
			schema.UserTableEmailColName:  goqu.Func("values", goqu.C(schema.UserTableEmailColName)),
			schema.UserTableNameColName:   goqu.Func("values", goqu.C(schema.UserTableNameColName)),
			schema.UserTableLastIPColName: goqu.Func("values", goqu.C(schema.UserTableLastIPColName)),
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

	success, err := s.autowpDB.Select(schema.UserTableIDCol, schema.UserTableRoleCol).
		From(schema.UserTable).
		Where(schema.UserTableUUIDCol.Eq(goqu.Func("UUID_TO_BIN", guid))).
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
			`
				SELECT `+schema.UserTableDeletedColName+`, `+schema.UserTableEmailColName+`, `+
				schema.UserTableEmailToCheckColName+`, `+schema.UserTableLoginColName+`, `+
				schema.UserTableNameColName+`, IFNULL(BIN_TO_UUID(`+schema.UserTableUUIDColName+`), '')
				FROM `+schema.UserTableName+` WHERE `+schema.UserTableIDColName+` = ?
			`,
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

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableUUIDColName: goqu.Func("UUID_TO_BIN", userGUID),
	}).Where(goqu.C("user_id").Eq(userID)).Executor().ExecContext(ctx)
	if err != nil {
		return "", err
	}

	return userGUID, err
}

func (s *Repository) PasswordMatch(ctx context.Context, userID int64, password string) (bool, error) {
	var exists bool
	err := s.autowpDB.QueryRowContext(ctx, `
		SELECT 1 FROM `+schema.UserTableName+` 
		WHERE `+schema.UserTablePasswordColName+` = MD5(CONCAT(?, ?)) AND id = ? AND NOT `+schema.UserTableDeletedColName+`
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
	err = s.autowpDB.QueryRowContext(
		ctx,
		"SELECT 1 FROM "+schema.UserTableName+" WHERE id = ? AND NOT deleted",
		userID,
	).Scan(&val)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	// $oldImageId = $row['img'];

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		"deleted": 1,
	}).Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)
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
	_, err := s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableVotesLeftColName: schema.UserTableVotesPerDayCol,
	}).Where(
		schema.UserTableVotesLeftCol.Lt(schema.UserTableVotesPerDayCol),
		schema.UserTableDeletedCol.IsFalse(),
	).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UpdateVotesLimits(ctx context.Context) (int, error) {
	rows, err := s.autowpDB.QueryContext(
		ctx,
		"SELECT "+schema.UserTableIDColName+" FROM "+schema.UserTableName+
			" WHERE NOT "+schema.UserTableDeletedColName+" AND "+
			schema.UserTableLastOnlineColName+" > DATE_SUB(NOW(), INTERVAL 3 MONTH)",
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

	if err = rows.Err(); err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *Repository) UpdateSpecsVolumes(ctx context.Context) error {
	var sts []struct {
		UserID int64 `db:"id"`
		Count  int64 `db:"count"`
	}

	err := s.autowpDB.Select(schema.UserTableIDCol, goqu.COUNT(schema.AttrsUserValuesTableUserIDCol).As("count")).
		From(schema.UserTable).
		LeftJoin(schema.AttrsUserValuesTable, goqu.On(schema.UserTableIDCol.Eq(schema.AttrsUserValuesTableUserIDCol))).
		Where(
			schema.UserTableSpecsVolumeValidCol.IsFalse(),
			schema.UserTableDeletedCol.IsFalse(),
		).
		GroupBy(schema.UserTableIDCol).
		ScanStructsContext(ctx, &sts)
	if err != nil {
		return err
	}

	for _, st := range sts {
		_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
			schema.UserTableSpecsVolumeColName:      st.Count,
			schema.UserTableSpecsVolumeValidColName: 1,
		}).Where(schema.UserTableIDCol.Eq(st.UserID)).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) ExportUsersToKeycloak(ctx context.Context) error {
	var ids []int64

	err := s.autowpDB.Select(schema.UserTableIDCol).From(schema.UserTable).Where(
		goqu.Or(
			goqu.Func("LENGTH", schema.UserTableLoginCol).Gt(0),
			goqu.Func("LENGTH", schema.UserTableEmailCol).Gt(0),
			goqu.Func("LENGTH", schema.UserTableEmailToCheckCol).Gt(0),
		),
	).Order(schema.UserTableIDCol.Desc()).ScanValsContext(ctx, &ids)
	if err != nil {
		return err
	}

	for _, userID := range ids {
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
	query := s.db.Insert(schema.UserUserPreferencesTable).
		Rows(goqu.Record{
			schema.UserUserPreferencesTableUserIDColName:   userID,
			schema.UserUserPreferencesTableToUserIDColName: toUserID,
			schema.UserUserPreferencesTableDCNColName:      disabled,
		}).
		OnConflict(
			goqu.DoUpdate(
				schema.UserUserPreferencesTableUserIDColName+", "+schema.UserUserPreferencesTableToUserIDColName,
				goqu.Record{
					schema.UserUserPreferencesTableDCNColName: goqu.L("EXCLUDED." + schema.UserUserPreferencesTableDCNColName),
				},
			),
		)

	_, err := query.Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UserPreferences(ctx context.Context, userID int64, toUserID int64) (*UserPreferences, error) {
	var row UserPreferences

	_, err := s.db.Select(schema.UserUserPreferencesTableDCNCol).
		From(schema.UserUserPreferencesTable).
		Where(
			schema.UserUserPreferencesTableUserIDCol.Eq(userID),
			schema.UserUserPreferencesTableToUserIDCol.Eq(toUserID),
		).ScanStructContext(ctx, &row)

	return &row, err
}

func (s *Repository) SetupPrivateRouter(_ context.Context, r *gin.Engine) {
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

func (s *Repository) incForumTopicsRecord() goqu.Record {
	r := s.incForumMessagesRecord()
	r[schema.UserTableForumsTopicsColName] = goqu.L(schema.UserTableForumsTopicsColName + " + 1")

	return r
}

func (s *Repository) IncForumTopics(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).
		Set(s.incForumTopicsRecord()).
		Where(schema.UserTableIDCol.Eq(userID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) incForumMessagesRecord() goqu.Record {
	r := s.touchLastMessageRecord()
	r[schema.UserTableForumsMessagesColName] = goqu.L(schema.UserTableForumsMessagesColName + " + 1")

	return r
}

func (s *Repository) IncForumMessages(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).
		Set(s.incForumMessagesRecord()).
		Where(schema.UserTableIDCol.Eq(userID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) touchLastMessageRecord() goqu.Record {
	return goqu.Record{schema.UserTableLastMessageTimeColName: goqu.Func("NOW")}
}

func (s *Repository) TouchLastMessage(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).
		Set(s.touchLastMessageRecord()).
		Where(schema.UserTableIDCol.Eq(userID)).
		Executor().ExecContext(ctx)

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

	st := struct {
		LastMessageTime   sql.NullTime `db:"last_message_time"`
		RegDate           sql.NullTime `db:"reg_date"`
		MessagingInterval int64        `db:"messaging_interval"`
	}{}

	success, err := s.autowpDB.Select(
		schema.UserTableLastMessageTimeCol, schema.UserTableRegDateCol, schema.UserTableMessagingIntervalCol,
	).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return time.Time{}, err
	}

	if !success {
		return time.Time{}, nil
	}

	if st.LastMessageTime.Valid {
		st.MessagingInterval = s.messagingInterval(st.RegDate.Time, st.MessagingInterval)
		if st.MessagingInterval > 0 {
			interval := time.Second * time.Duration(st.MessagingInterval)

			return st.LastMessageTime.Time.Add(interval), nil
		}
	}

	return time.Time{}, nil
}

func (s *Repository) RegisterVisit(ctx context.Context, userID int64) error {
	var (
		lastOnline sql.NullTime
		lastIP     *net.IP
	)

	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT "+schema.UserTableLastOnlineColName+", "+schema.UserTableLastIPColName+
			" FROM "+schema.UserTableName+" WHERE id = ?",
		userID,
	).
		Scan(&lastOnline, &lastIP)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	set := goqu.Record{}

	if !lastOnline.Valid || lastOnline.Time.Add(lastOnlineUpdateThreshold).Before(time.Now()) {
		set[schema.UserTableLastOnlineColName] = goqu.Func("NOW")
	}

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

	ip := net.ParseIP(remoteAddr)

	if ip != nil && (lastIP == nil || !lastIP.Equal(ip)) {
		set[schema.UserTableLastIPColName] = goqu.Func("INET6_ATON", remoteAddr)
	}

	if len(set) > 0 {
		_, err = s.autowpDB.Update(schema.UserTable).Set(set).Where(schema.UserTableIDCol.Eq(userID)).
			Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
