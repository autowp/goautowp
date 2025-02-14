package goautowp

import (
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"github.com/stretchr/testify/require"
)

func TestDuplicateFinder(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	df, err := NewDuplicateFinder(goquDB)
	require.NoError(t, err)

	ctx := t.Context()

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	id1, _ := addPicture(t, imageStorage, goquDB, "./test/large.jpg", schema.PictureStatusInbox)
	err = df.Index(ctx, id1, "http://localhost:80/large.jpg")
	require.NoError(t, err)

	id2, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
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
