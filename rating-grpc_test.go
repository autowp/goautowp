package goautowp

import (
	"context"
	"testing"

	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
