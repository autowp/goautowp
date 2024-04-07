package textstorage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql" // enable mysql dialect
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestGetText(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	r, err := goquDB.Insert(schema.TextstorageTextTable).Rows(goqu.Record{
		schema.TextstorageTextTableTextColName:        "test",
		schema.TextstorageTextTableLastUpdatedColName: goqu.Func("NOW"),
		schema.TextstorageTextTableRevisionColName:    1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	id, err := r.LastInsertId()
	require.NoError(t, err)

	repository := New(goquDB)

	_, err = repository.Text(ctx, id)
	require.NoError(t, err)

	_, err = repository.FirstText(ctx, []int64{id})
	require.NoError(t, err)
}
