package goautowp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	testUsername  = "tester"
	testPassword  = "123123"
	adminUsername = "admin"
	adminPassword = "123123"
)

const bearerPrefix = "Bearer "

const authorizationHeader = "authorization"

func randomHexString(t *testing.T, length int) string {
	t.Helper()

	randBytes := make([]byte, length)
	_, err := rand.Read(randBytes)
	require.NoError(t, err)

	return hex.EncodeToString(randBytes)
}

func addPicture(t *testing.T, imageStorage *storage.Storage, db *goqu.Database, filepath string) (int64, int) {
	t.Helper()

	ctx := context.Background()

	imageID, err := imageStorage.AddImageFromFile(ctx, filepath, "picture", storage.GenerateOptions{})
	require.NoError(t, err)

	img, err := imageStorage.Image(ctx, imageID)
	require.NoError(t, err)

	identity := randomHexString(t, schema.PicturesTableIdentityLength/2)

	res, err := db.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableImageIDColName:  imageID,
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableIPColName:       goqu.Func("INET6_ATON", "127.0.0.1"),
		schema.PictureTableOwnerIDColName:  nil,
		schema.PictureTableWidthColName:    img.Width(),
		schema.PictureTableHeightColName:   img.Height(),
		schema.PictureTableStatusColName:   schema.PictureStatusInbox,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	return pictureID, imageID
}

//nolint:unparam
func getUserWithCleanHistory(
	t *testing.T,
	conn *grpc.ClientConn,
	cfg config.Config,
	db *goqu.Database,
	username string,
	password string,
) (int64, string) {
	t.Helper()

	ctx := context.Background()
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, username, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	user, err := NewUsersClient(conn).Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	const exampleVotesLeftColName = 100

	_, err = db.Update(schema.UserTable).
		Set(goqu.Record{
			schema.UserTableLastMessageTimeColName: "2000-01-01",
			schema.UserTableVotesLeftColName:       exampleVotesLeftColName,
		}).
		Where(schema.UserTableIDCol.Eq(user.GetId())).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	return user.GetId(), token.AccessToken
}

func createItem(t *testing.T, goquDB *goqu.Database, row schema.ItemRow) int64 {
	t.Helper()

	ctx := context.Background()

	res, err := goquDB.Insert(schema.ItemTable).Rows(row).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemLanguageTable).Rows(schema.ItemLanguageRow{
		ItemID:   itemID,
		Language: "en",
		Name:     sql.NullString{Valid: true, String: row.Name},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	cfg := config.LoadConfig(".")

	repository := items.NewRepository(goquDB, 0, cfg.ContentLanguages)
	_, err = repository.RebuildCache(ctx, itemID)
	require.NoError(t, err)

	return itemID
}
