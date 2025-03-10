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
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
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

type UserFields struct {
	Email         bool
	Timezone      bool
	Language      bool
	VotesPerDay   bool
	VotesLeft     bool
	RegDate       bool
	LastOnline    bool
	Accounts      bool
	PicturesAdded bool
	LastIP        bool
	Login         bool
}

type OrderBy int

const (
	OrderByNone OrderBy = iota
	OrderByPicturesTotalDesc
	OrderBySpecsVolumeDesc
	OrderByDeletedName
)

// Repository Main Object.
type Repository struct {
	autowpDB        *goqu.Database
	db              *goqu.Database
	usersSalt       string
	languages       map[string]config.LanguageConfig
	keycloak        *gocloak.GoCloak
	keycloakConfig  config.KeycloakConfig
	messageInterval int64
	imageStorage    *storage.Storage
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
	imageStorage *storage.Storage,
) *Repository {
	return &Repository{
		autowpDB:        autowpDB,
		db:              db,
		usersSalt:       usersSalt,
		languages:       languages,
		keycloak:        keyCloak,
		keycloakConfig:  keyCloakConfig,
		messageInterval: messageInterval,
		imageStorage:    imageStorage,
	}
}

func (s *Repository) User(
	ctx context.Context, options *query.UserListOptions, fields UserFields, orderBy OrderBy,
) (*schema.UsersRow, error) {
	users, _, err := s.Users(ctx, options, fields, orderBy)
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

func (s *Repository) Users(
	ctx context.Context, options *query.UserListOptions, fields UserFields, orderBy OrderBy,
) ([]schema.UsersRow, *util.Pages, error) {
	var err error

	result := make([]schema.UsersRow, 0)

	var row schema.UsersRow
	valuePtrs := []interface{}{
		&row.ID, &row.Name, &row.Deleted, &row.Identity, &row.LastOnline, &row.Role,
		&row.SpecsWeight, &row.Img, &row.EMail, &row.PicturesTotal, &row.SpecsVolume, &row.Language,
	}

	alias := query.UserTableAlias
	aliasTable := goqu.T(alias)

	columns := []interface{}{
		aliasTable.Col(schema.UserTableIDColName), aliasTable.Col(schema.UserTableNameColName),
		aliasTable.Col(schema.UserTableDeletedColName), aliasTable.Col(schema.UserTableIdentityColName),
		aliasTable.Col(schema.UserTableLastOnlineColName), aliasTable.Col(schema.UserTableRoleColName),
		aliasTable.Col(schema.UserTableSpecsWeightColName), aliasTable.Col(schema.UserTableImgColName),
		aliasTable.Col(schema.UserTableEmailColName), aliasTable.Col(schema.UserTablePicturesTotalColName),
		aliasTable.Col(schema.UserTableSpecsVolumeColName), aliasTable.Col(schema.UserTableLanguageColName),
	}

	if fields.VotesLeft {
		valuePtrs = append(valuePtrs, &row.VotesLeft)
		columns = append(columns, aliasTable.Col(schema.UserTableVotesLeftColName))
	}

	if fields.VotesPerDay {
		valuePtrs = append(valuePtrs, &row.VotesPerDay)
		columns = append(columns, aliasTable.Col(schema.UserTableVotesPerDayColName))
	}

	if fields.Language {
		valuePtrs = append(valuePtrs, &row.Language)
		columns = append(columns, aliasTable.Col(schema.UserTableLanguageColName))
	}

	if fields.Timezone {
		valuePtrs = append(valuePtrs, &row.Timezone)
		columns = append(columns, aliasTable.Col(schema.UserTableTimezoneColName))
	}

	if fields.RegDate {
		valuePtrs = append(valuePtrs, &row.RegDate)
		columns = append(columns, aliasTable.Col(schema.UserTableRegDateColName))
	}

	if fields.PicturesAdded {
		valuePtrs = append(valuePtrs, &row.PicturesAdded)
		columns = append(columns, aliasTable.Col(schema.UserTablePicturesAddedColName))
	}

	if fields.LastIP {
		valuePtrs = append(valuePtrs, &row.LastIP)
		columns = append(columns, goqu.Func("INET6_NTOA", aliasTable.Col(schema.UserTableLastIPColName)))
	}

	if fields.Login {
		valuePtrs = append(valuePtrs, &row.Login)
		columns = append(columns, aliasTable.Col(schema.UserTableLoginColName))
	}

	sqSelect := options.Select(s.autowpDB, alias).Select(columns...)

	switch orderBy {
	case OrderByNone:
	case OrderByPicturesTotalDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.UserTablePicturesTotalColName).Desc())
	case OrderBySpecsVolumeDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.UserTableSpecsVolumeColName).Desc())
	case OrderByDeletedName:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.UserTableDeletedColName).Asc(),
			aliasTable.Col(schema.UserTableNameColName).Asc(),
		)
	}

	var pages *util.Pages

	if options.Limit > 0 {
		paginator := util.Paginator{
			SQLSelect:         sqSelect,
			ItemCountPerPage:  int32(options.Limit), //nolint: gosec
			CurrentPageNumber: int32(options.Page),  //nolint: gosec
		}

		pages, err = paginator.GetPages(ctx)
		if err != nil {
			return nil, nil, err
		}

		sqSelect, err = paginator.GetItemsByPage(ctx, int32(options.Page)) //nolint: gosec
		if err != nil {
			return nil, nil, err
		}
	}

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
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

		result = append(result, row)
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
	ctx = context.WithoutCancel(ctx)

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

	picturesExists, err := s.autowpDB.From(schema.PictureTable).Where(
		schema.PictureTableOwnerIDCol.Eq(userID),
		schema.PictureTableStatusCol.Eq(schema.PictureStatusAccepted),
	).CountContext(ctx)
	if err != nil {
		return err
	}

	value := math.Round(avgVote + float64(def+age) + float64(picturesExists)/100)
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

	success, err := s.autowpDB.Select(goqu.Func("IFNULL", goqu.AVG(schema.CommentMessageTableVoteCol), 0)).
		From(schema.CommentMessageTable).
		Where(
			schema.CommentMessageTableAuthorIDCol.Eq(userID),
			schema.CommentMessageTableVoteCol.Neq(0),
		).
		ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
}

func (s *Repository) RefreshUserConflicts(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableSpecsWeightColName: goqu.L(
			"1.5 * (1 + ?) / (1 + ?)",
			goqu.Func(
				"IFNULL",
				s.autowpDB.Select(goqu.SUM(schema.AttrsUserValuesTableWeightCol)).
					From(schema.AttrsUserValuesTable).
					Where(
						schema.AttrsUserValuesTableUserIDCol.Eq(schema.UserTableIDCol),
						schema.AttrsUserValuesTableWeightCol.Gt(0),
					),
				0,
			),
			goqu.Func(
				"IFNULL",
				s.autowpDB.Select(goqu.Func("ABS", goqu.SUM(schema.AttrsUserValuesTableWeightCol))).
					From(schema.AttrsUserValuesTable).
					Where(
						schema.AttrsUserValuesTableUserIDCol.Eq(schema.UserTableIDCol),
						schema.AttrsUserValuesTableWeightCol.Lt(0),
					),
				0,
			),
		),
	}).
		Where(schema.UserTableIDCol.Eq(userID)).
		Executor().ExecContext(ctx)

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

	var res sql.Result

	ctx = context.WithoutCancel(ctx)

	res, err := s.autowpDB.Insert(schema.UserTable).
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
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, "", err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, "", err
	}

	if affected == 1 { // row just inserted
		userID, err := res.LastInsertId()
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

	st := struct {
		Deleted      bool           `db:"deleted"`
		Email        sql.NullString `db:"e_mail"`
		EmailToCheck sql.NullString `db:"email_to_check"`
		Login        sql.NullString `db:"login"`
		Name         string         `db:"name"`
		GUID         string         `db:"guid"`
	}{}

	success, err := s.autowpDB.Select(
		schema.UserTableDeletedCol, schema.UserTableEmailCol, schema.UserTableEmailToCheckCol, schema.UserTableLoginCol,
		schema.UserTableNameCol, goqu.Func("IFNULL", goqu.Func("BIN_TO_UUID", schema.UserTableUUIDCol), "").As("guid"),
	).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return "", err
	}

	if !success {
		return "", sql.ErrNoRows
	}

	if len(st.GUID) > 0 {
		return st.GUID, nil
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
		keyCloakEmail = &st.Email.String
		emailVerified = true
	)

	if !st.Email.Valid || len(st.Email.String) == 0 {
		keyCloakEmail = &st.EmailToCheck.String
		emailVerified = false
	}

	username := st.Login.String
	if (!st.Login.Valid || len(st.Login.String) == 0) && keyCloakEmail != nil && len(*keyCloakEmail) > 0 {
		username = *keyCloakEmail
	}

	falseRef := false
	enabled := !st.Deleted
	ctx = context.WithoutCancel(ctx)

	st.GUID, err = s.keycloak.CreateUser(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.User{
		Enabled:       &enabled,
		Totp:          &falseRef,
		EmailVerified: &emailVerified,
		Username:      &username,
		FirstName:     &st.Name,
		Email:         keyCloakEmail,
	})
	if err != nil {
		return "", err
	}

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableUUIDColName: goqu.Func("UUID_TO_BIN", st.GUID),
	}).Where(goqu.C("user_id").Eq(userID)).Executor().ExecContext(ctx)
	if err != nil {
		return "", err
	}

	return st.GUID, err
}

func (s *Repository) PasswordMatch(ctx context.Context, userID int64, password string) (bool, error) {
	var exists bool

	succcess, err := s.autowpDB.Select(goqu.V(1)).
		From(schema.UserTable).
		Where(
			schema.UserTablePasswordCol.Eq(goqu.Func("MD5", goqu.Func("CONCAT", s.usersSalt, password))),
			schema.UserTableIDCol.Eq(userID),
			schema.UserTableDeletedCol.IsFalse(),
		).ScanValContext(ctx, &exists)
	if err != nil {
		return false, err
	}

	return succcess && exists, nil
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
	ctx = context.WithoutCancel(ctx)

	err = s.keycloak.UpdateUser(ctx, token.AccessToken, s.keycloakConfig.Realm, gocloak.User{
		ID:      &userGUID,
		Enabled: &f,
	})
	if err != nil {
		return false, err
	}

	var val int

	success, err := s.autowpDB.Select(goqu.V(1)).
		From(schema.UserTable).
		Where(
			schema.UserTableIDCol.Eq(userID),
			schema.UserTableDeletedCol.IsFalse(),
		).ScanValContext(ctx, &val)
	if err != nil {
		return false, err
	}

	if !success {
		return false, nil
	}

	// $oldImageId = $row['img'];

	_, err = s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableDeletedColName: 1,
	}).Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)
	// 'img'     => null,
	if err != nil {
		return false, err
	}

	/*if ($oldImageId) {
		$this->imageStorage->removeImage($oldImageId);
	}*/

	_, err = s.autowpDB.Delete(schema.TelegramChatTable).
		Where(schema.TelegramChatTableUserIDCol.Eq(userID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	// delete linked profiles
	_, err = s.autowpDB.Delete(schema.UserAccountTable).
		Where(
			schema.UserAccountTableUserIDCol.Eq(userID),
			schema.UserAccountTableServiceIDCol.Eq(KeycloakExternalAccountID),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	// unsubscribe from items
	_, err = s.autowpDB.Delete(schema.UserItemSubscribeTable).
		Where(schema.UserItemSubscribeTableUserIDCol.Eq(userID)).
		Executor().ExecContext(ctx)
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
	var ids []int64

	ctx = context.WithoutCancel(ctx)

	err := s.autowpDB.Select(schema.UserTableIDCol).
		From(schema.UserTable).
		Where(
			schema.UserTableDeletedCol.IsFalse(),
			schema.UserTableLastOnlineCol.Gt(goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL 3 MONTH"))),
		).
		ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	affected := 0

	for _, userID := range ids {
		err = s.UpdateUserVoteLimit(ctx, userID)
		if err != nil {
			return 0, err
		}

		affected++
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

	ctx = context.WithoutCancel(ctx)

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
	_, err := s.db.Insert(schema.UserUserPreferencesTable).
		Rows(goqu.Record{
			schema.UserUserPreferencesTableUserIDColName:   userID,
			schema.UserUserPreferencesTableToUserIDColName: toUserID,
			schema.UserUserPreferencesTableDCNColName:      disabled,
		}).
		OnConflict(
			goqu.DoUpdate(
				schema.UserUserPreferencesTableUserIDColName+", "+schema.UserUserPreferencesTableToUserIDColName,
				goqu.Record{
					schema.UserUserPreferencesTableDCNColName: schema.Excluded(schema.UserUserPreferencesTableDCNColName),
				},
			),
		).Executor().ExecContext(ctx)

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
	r.GET("/user-user-preferences/:user_id/:to_user_id", func(ctx *gin.Context) { //nolint: contextcheck
		userID, err := strconv.ParseInt(ctx.Param("user_id"), Decimal, BitSize64)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Invalid user_id")

			return
		}

		toUserID, err := strconv.ParseInt(ctx.Param("to_user_id"), Decimal, BitSize64)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Invalid to_user_id")

			return
		}

		prefs, err := s.UserPreferences(ctx, userID, toUserID)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "InternalServerError")

			return
		}

		ctx.JSON(http.StatusOK, prefs)
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

	return util.Max(messagingInterval, s.messageInterval)
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
	st := struct {
		LastOnline sql.NullTime `db:"last_online"`
		LastIP     *net.IP      `db:"last_ip"`
	}{}

	success, err := s.autowpDB.Select(schema.UserTableLastOnlineCol, schema.UserTableLastIPCol).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return err
	}

	if !success {
		return nil
	}

	set := goqu.Record{}

	if !st.LastOnline.Valid || st.LastOnline.Time.Add(lastOnlineUpdateThreshold).Before(time.Now()) {
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

	if ip != nil && (st.LastIP == nil || !st.LastIP.Equal(ip)) {
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

func (s *Repository) UserLanguage(ctx context.Context, userID int64) (string, error) {
	language := ""

	success, err := s.autowpDB.Select(schema.UserTableLanguageCol).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanValContext(ctx, &language)
	if err != nil {
		return "", err
	}

	if !success {
		return "", ErrUserNotFound
	}

	return language, nil
}

func (s *Repository) RefreshPicturesCount(ctx context.Context, userID int64) error {
	_, err := s.autowpDB.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTablePicturesTotalColName: s.autowpDB.Select(goqu.COUNT(goqu.Star())).
			From(schema.PictureTable).
			Where(
				schema.PictureTableOwnerIDCol.Eq(userID),
				schema.PictureTableStatusCol.Eq(schema.PictureStatusAccepted),
			),
	}).
		Where(schema.UserTableIDCol.Eq(userID)).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UserAccounts(ctx context.Context, userID int64) ([]*schema.UserAccountRow, error) {
	var rows []*schema.UserAccountRow
	err := s.db.Select(goqu.Star()).From(schema.UserAccountTable).Where(schema.UserAccountTableUserIDCol.Eq(userID)).
		ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) HaveAccountsForOtherServices(ctx context.Context, userID int64, id int64) (bool, error) {
	var found bool

	success, err := s.db.Select(goqu.V(true)).
		From(schema.UserAccountTable).
		Where(
			schema.UserAccountTableIDCol.Neq(id),
			schema.UserAccountTableUserIDCol.Eq(userID),
		).
		ScanValContext(ctx, &found)

	return success && found, err
}

func (s *Repository) RemoveUserAccount(ctx context.Context, id int64) error {
	_, err := s.db.Delete(schema.UserAccountTable).
		Where(schema.UserAccountTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) DeletePhoto(ctx context.Context, userID int64) (bool, error) {
	row, err := s.User(ctx, &query.UserListOptions{ID: userID}, UserFields{}, OrderByNone)
	if err != nil {
		return false, err
	}

	ctx = context.WithoutCancel(ctx)

	if row.Img != nil {
		_, err = s.autowpDB.Update(schema.UserTable).
			Set(goqu.Record{schema.UserTableImgColName: nil}).
			Where(schema.UserTableIDCol.Eq(userID)).
			Executor().ExecContext(ctx)
		if err != nil {
			return false, err
		}

		err = s.imageStorage.RemoveImage(ctx, *row.Img)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}
