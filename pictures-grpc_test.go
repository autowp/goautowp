package goautowp

import (
	"context"
	"database/sql"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getPictureID(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint: gosec
	identity := "p" + strconv.Itoa(random.Int())[:6]

	res, err := db.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   "accepted",
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	return pictureID
}

func TestView(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewPicturesClient(conn)

	_, err = client.View(ctx, &PicturesViewRequest{PictureId: getPictureID(ctx, t, goquDB)})
	require.NoError(t, err)
}

func TestVote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.Vote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PicturesVoteRequest{PictureId: getPictureID(ctx, t, goquDB), Value: 1},
	)
	require.NoError(t, err)
}

func TestModerVoteTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	template, err := client.CreateModerVoteTemplate(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ModerVoteTemplate{Message: "test", Vote: 1},
	)
	require.NoError(t, err)

	_, err = client.GetModerVoteTemplates(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)

	_, err = client.DeleteModerVoteTemplate(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeleteModerVoteTemplateRequest{Id: template.GetId()},
	)
	require.NoError(t, err)
}
