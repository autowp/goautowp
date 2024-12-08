package goautowp

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestLikesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := NewRatingClient(conn)

	r, err := client.GetUserCommentsRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.GetUsers() {
		_, err = client.GetUserCommentsRatingFans(ctx, &UserRatingDetailsRequest{UserId: item.GetUserId()})
		require.NoError(t, err)
	}
}

func TestPictureLikesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("..")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	kc := cnt.Keycloak()
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
		schema.PictureTableStatusColName:   schema.PictureStatusAccepted,
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  tester.GetId(),
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

	for _, item := range r.GetUsers() {
		_, err = client.GetUserPictureLikesRatingFans(ctx, &UserRatingDetailsRequest{UserId: item.GetUserId()})
		require.NoError(t, err)
	}
}

func TestPicturesRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := NewRatingClient(conn)

	r, err := client.GetUserPicturesRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.GetUsers() {
		_, err = client.GetUserPicturesRatingBrands(ctx, &UserRatingDetailsRequest{
			UserId:   item.GetUserId(),
			Language: "en",
		})
		require.NoError(t, err)
	}
}

func TestSpecsRating(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := NewRatingClient(conn)

	r, err := client.GetUserSpecsRating(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	for _, item := range r.GetUsers() {
		_, err = client.GetUserSpecsRatingBrands(ctx, &UserRatingDetailsRequest{UserId: item.GetUserId()})
		require.NoError(t, err)
	}
}
