package goautowp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

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

func addImage(t *testing.T, db *goqu.Database, filepath string) int {
	t.Helper()

	_, filename := path.Split(filepath)
	extension := path.Ext(filename)
	name := strings.TrimSuffix(filename, extension)

	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	require.NoError(t, err)

	newPath := name + hex.EncodeToString(randBytes) + extension
	newFullpath := os.Getenv("AUTOWP_IMAGES_DIR") + "/" + newPath

	err = os.MkdirAll(path.Dir(newFullpath), os.ModePerm)
	require.NoError(t, err)

	err = copyFile(filepath, newFullpath)
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

func addPicture(t *testing.T, db *goqu.Database, filepath string) int {
	t.Helper()

	imageID := addImage(t, db, filepath)

	randBytes := make([]byte, 3)
	_, err := rand.Read(randBytes)
	require.NoError(t, err)

	identity := hex.EncodeToString(randBytes)
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

	return int(pictureID)
}

func TestDuplicateFinder(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	df, err := NewDuplicateFinder(goquDB)
	require.NoError(t, err)

	ctx := context.Background()

	id1 := addPicture(t, goquDB, os.Getenv("AUTOWP_TEST_ASSETS_DIR")+"/large.jpg")
	err = df.Index(ctx, id1, "http://localhost:80/large.jpg")
	require.NoError(t, err)

	id2 := addPicture(t, goquDB, os.Getenv("AUTOWP_TEST_ASSETS_DIR")+"/small.jpg")
	err = df.Index(ctx, id2, "http://localhost:80/small.jpg")
	require.NoError(t, err)

	var hash1 uint64
	success, err := goquDB.Select(schema.DfHashTableHashCol).
		From(schema.DfHashTable).
		Where(schema.DfHashTablePictureIDCol.Eq(id1)).
		ScanValContext(ctx, &hash1)
	require.NoError(t, err)
	require.True(t, success)

	var hash2 uint64
	success, err = goquDB.Select(schema.DfHashTableHashCol).
		From(schema.DfHashTable).
		Where(schema.DfHashTablePictureIDCol.Eq(id2)).
		ScanValContext(ctx, &hash2)
	require.NoError(t, err)
	require.True(t, success)

	var distance int

	success, err = goquDB.Select(schema.DfDistanceTableDistanceCol).From(schema.DfDistanceTable).Where(
		schema.DfDistanceTableSrcPictureIDCol.Eq(id1),
		schema.DfDistanceTableDstPictureIDCol.Eq(id2),
	).ScanValContext(ctx, &distance)
	require.NoError(t, err)
	require.True(t, success)
	require.True(t, distance <= 2)
}
