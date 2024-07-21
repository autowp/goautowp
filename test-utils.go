package goautowp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer util.Close(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer util.Close(out)

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func randomHexString(t *testing.T, length int) string {
	t.Helper()

	randBytes := make([]byte, length)
	_, err := rand.Read(randBytes)
	require.NoError(t, err)

	return hex.EncodeToString(randBytes)
}

func addImage(t *testing.T, db *goqu.Database, imageFilepath string) int {
	t.Helper()

	_, filename := path.Split(imageFilepath)
	extension := path.Ext(filename)
	name := strings.TrimSuffix(filename, extension)

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	exPath := filepath.Dir(ex)

	const hexPathLength = 16

	newPath := name + randomHexString(t, hexPathLength) + extension
	newFullpath := exPath + "/images/" + newPath

	err = os.MkdirAll(path.Dir(newFullpath), os.ModePerm)
	require.NoError(t, err)

	err = copyFile(imageFilepath, newFullpath)
	require.NoError(t, err)

	res, err := db.Insert(schema.ImageTable).Rows(goqu.Record{
		schema.ImageTableFilepathColName: newPath,
		schema.ImageTableFilesizeColName: 1,
		schema.ImageTableWidthColName:    1,
		schema.ImageTableHeightColName:   1,
		schema.ImageTableDirColName:      "picture",
	}).Executor().ExecContext(context.Background())
	require.NoError(t, err)

	imageID, err := res.LastInsertId()
	require.NoError(t, err)

	return int(imageID)
}

func addPicture(t *testing.T, db *goqu.Database, filepath string) int64 {
	t.Helper()

	imageID := addImage(t, db, filepath)
	identity := randomHexString(t, pictures.IdentityLength/2)
	ctx := context.Background()

	res, err := db.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableImageIDColName:  imageID,
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableIPColName:       goqu.Func("INET6_ATON", "127.0.0.1"),
		schema.PictureTableOwnerIDColName:  nil,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	return pictureID
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
