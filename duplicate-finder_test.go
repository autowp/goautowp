package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

func TestDuplicateFinder(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	df, err := NewDuplicateFinder(goquDB)
	require.NoError(t, err)

	ctx := context.Background()

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	id1 := addPicture(t, imageStorage, goquDB, "./test/large.jpg")
	err = df.Index(ctx, id1, "http://localhost:80/large.jpg")
	require.NoError(t, err)

	id2 := addPicture(t, imageStorage, goquDB, "./test/small.jpg")
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
	require.LessOrEqual(t, distance, 2)
}
