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

func TestGetVODData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewDonationsClient(conn)

	r, err := client.GetVODData(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r.Dates)
	require.NotEmpty(t, r.Sum)
}
