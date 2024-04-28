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

func TestLikesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	client := NewRatingClient(conn)

	r, err := client.GetUserCommentsRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.Users {
		_, err = client.GetUserCommentsRatingFans(ctx, &UserRatingDetailsRequest{UserId: item.UserId})
		require.NoError(t, err)
	}
}

func TestPictureLikesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig("..")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

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

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint: gosec
	identity := "p" + strconv.Itoa(random.Int())[:6]

	res, err := goquDB.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   "accepted",
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  tester.Id,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	picturesClient := NewPicturesClient(conn)

	_, err = picturesClient.Vote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&PicturesVoteRequest{PictureId: pictureID, Value: 1},
	)
	require.NoError(t, err)

	client := NewRatingClient(conn)

	r, err := client.GetUserPictureLikesRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.Users {
		_, err = client.GetUserPictureLikesRatingFans(ctx, &UserRatingDetailsRequest{UserId: item.UserId})
		require.NoError(t, err)
	}
}

func TestPicturesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	client := NewRatingClient(conn)

	r, err := client.GetUserPicturesRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.Users {
		_, err = client.GetUserPicturesRatingBrands(ctx, &UserRatingDetailsRequest{UserId: item.UserId})
		require.NoError(t, err)
	}
}

func TestSpecsRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)

	defer util.Close(conn)

	client := NewRatingClient(conn)

	r, err := client.GetUserSpecsRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.Users {
		_, err = client.GetUserSpecsRatingBrands(ctx, &UserRatingDetailsRequest{UserId: item.UserId})
		require.NoError(t, err)
	}
}
