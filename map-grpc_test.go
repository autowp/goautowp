package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func createItemWithPoint(ctx context.Context, t *testing.T) {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	res, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableNameColName:            fmt.Sprintf("vehicle-%d", random.Int()),
		schema.ItemTableIsGroupColName:         0,
		schema.ItemTableItemTypeIDColName:      ItemType_ITEM_TYPE_VEHICLE,
		schema.ItemTableCatnameColName:         fmt.Sprintf("vehicle-%d", random.Int()),
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemPointTable).Rows(goqu.Record{
		schema.ItemPointTableItemIDColName: itemID,
		schema.ItemPointTablePointColName:  goqu.Func("point", 30, 30),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func TestGetPoints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createItemWithPoint(ctx, t)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewMapClient(conn)

	_, err = client.GetPoints(
		ctx,
		&MapGetPointsRequest{
			Bounds:   "0,0,60,60",
			Language: "en",
		},
	)
	require.NoError(t, err)
}

func TestGetPointsOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	createItemWithPoint(ctx, t)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewMapClient(conn)

	_, err = client.GetPoints(
		ctx,
		&MapGetPointsRequest{
			Bounds:     "0,0,60,60",
			Language:   "en",
			PointsOnly: true,
		},
	)
	require.NoError(t, err)
}
