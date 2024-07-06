package goautowp

import (
	"context"
	"testing"

	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGetArticles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewArticlesClient(conn)

	response1, err := client.GetList(
		ctx,
		&ArticlesRequest{},
	)
	require.NoError(t, err)
	require.NotNil(t, response1)
	require.Len(t, response1.GetItems(), 1)
	require.Equal(t, int64(1), response1.GetItems()[0].GetId())

	response2, err := client.GetItemByCatname(
		ctx,
		&ArticleByCatnameRequest{Catname: "test-article"},
	)
	require.NoError(t, err)
	require.NotNil(t, response2)
	require.Equal(t, int64(1), response2.GetId())
	require.Equal(t, "Test html", response2.GetHtml())
}
