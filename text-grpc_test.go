package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestGetText(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	cnt := NewContainer(cfg)
	defer util.Close(cnt)

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewTextClient(conn)
	kc := gocloak.NewClient(cfg.Keycloak.URL)
	usersClient := NewUsersClient(conn)

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	// tester (me)
	tester, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	res, err := goquDB.Insert(schema.TextstorageTextTable).Rows(goqu.Record{
		schema.TextstorageTextTableTextColName:        "Text 2",
		schema.TextstorageTextTableLastUpdatedColName: goqu.Func("NOW"),
		schema.TextstorageTextTableRevisionColName:    2,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.TextstorageRevisionTable).Rows(goqu.Record{
		schema.TextstorageRevisionTableTextIDColName:    id,
		schema.TextstorageRevisionTableRevisionColName:  1,
		schema.TextstorageRevisionTableTextColName:      "Text 1",
		schema.TextstorageRevisionTableTimestampColName: goqu.Func("NOW"),
		schema.TextstorageRevisionTableUserIDColName:    tester.Id,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.TextstorageRevisionTable).Rows(goqu.Record{
		schema.TextstorageRevisionTableTextIDColName:    id,
		schema.TextstorageRevisionTableRevisionColName:  2,
		schema.TextstorageRevisionTableTextColName:      "Text 2",
		schema.TextstorageRevisionTableTimestampColName: goqu.Func("NOW"),
		schema.TextstorageRevisionTableUserIDColName:    tester.Id,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	r, err := client.GetText(ctx, &APIGetTextRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, "Text 2", r.Current.Text)
	require.Equal(t, tester.Id, r.Current.UserId)
	require.Equal(t, int64(2), r.Current.Revision)
	require.Equal(t, "Text 1", r.Prev.Text)
	require.Equal(t, tester.Id, r.Prev.UserId)
	require.Equal(t, int64(1), r.Prev.Revision)
	require.Equal(t, int64(0), r.Next.Revision)
}
