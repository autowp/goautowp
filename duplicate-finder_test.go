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

	stmt, err := db.Prepare(`
		INSERT INTO ` + schema.TableImage + ` (filepath, filesize, width, height, dir)
		VALUES (?, 1, 1, 1, "picture")
	`)
	require.NoError(t, err)

	defer util.Close(stmt)

	res, err := stmt.Exec(newPath)
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

	res, err := db.Insert(schema.PictureTableName).Rows(goqu.Record{
		"image_id": imageID,
		"identity": identity,
		"ip":       goqu.Func("INET6_ATON", "127.0.0.1"),
		"owner_id": nil,
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
	err = db.QueryRow("SELECT hash FROM "+schema.DfHashTableName+" WHERE picture_id = ?", id1).Scan(&hash1)
	require.NoError(t, err)

	var hash2 uint64
	err = db.QueryRow("SELECT hash FROM "+schema.DfHashTableName+" WHERE picture_id = ?", id2).Scan(&hash2)
	require.NoError(t, err)

	var distance int
	err = db.QueryRow(`
		SELECT distance FROM `+schema.TableDfDistance+` 
		WHERE src_picture_id = ? AND dst_picture_id = ?
	`, id1, id2).Scan(&distance)
	require.NoError(t, err)

	require.True(t, distance <= 2)
}
