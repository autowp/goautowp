package goautowp

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestTopBrandsList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopBrandsList(ctx, &GetTopBrandsListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r.Brands)
	require.Greater(t, r.Total, int32(0))
}

func TestTopPersonsAuthorList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopPersonsList(ctx, &GetTopPersonsListRequest{
		Language:        "ru",
		PictureItemType: PictureItemType_PICTURE_AUTHOR,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestTopPersonsContentList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopPersonsList(ctx, &GetTopPersonsListRequest{
		Language:        "ru",
		PictureItemType: PictureItemType_PICTURE_CONTENT,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestTopFactoriesList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopFactoriesList(ctx, &GetTopFactoriesListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestContentLanguages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetContentLanguages(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.Greater(t, len(r.Languages), 1)
}

func TestItemLinks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r1, err := client.CreateItemLink(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemLink{
			Name:   "Link 1 ",
			Url:    " https://example.org",
			Type:   " club",
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, r1.Id)

	r2, err := client.GetItemLink(ctx, &APIItemLinkRequest{
		Id: r1.Id,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r1.Id)

	require.Equal(t, r1.Id, r2.Id)
	require.Equal(t, "Link 1", r2.Name)
	require.Equal(t, "https://example.org", r2.Url)
	require.Equal(t, "club", r2.Type)
	require.Equal(t, int64(1), r2.ItemId)

	_, err = client.UpdateItemLink(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemLink{
			Id:     r1.Id,
			Name:   "Link 2",
			Url:    "https://example2.org",
			Type:   "default",
			ItemId: 2,
		},
	)
	require.NoError(t, err)

	r3, err := client.GetItemLink(ctx, &APIItemLinkRequest{
		Id: r1.Id,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r1.Id)

	require.Equal(t, r1.Id, r3.Id)
	require.Equal(t, "Link 2", r3.Name)
	require.Equal(t, "https://example2.org", r3.Url)
	require.Equal(t, "default", r3.Type)
	require.Equal(t, int64(2), r3.ItemId)

	r4, err := client.GetItemLinks(ctx, &APIGetItemLinksRequest{
		ItemId: r3.ItemId,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r4.Items)
}

func TestItemVehicleTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	_, err = client.CreateItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleType{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)

	r2, err := client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), r2.ItemId)
	require.Equal(t, int64(1), r2.VehicleTypeId)

	_, err = client.CreateItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleType{
			ItemId:        1,
			VehicleTypeId: 2,
		},
	)
	require.NoError(t, err)

	r4, err := client.GetItemVehicleTypes(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIGetItemVehicleTypesRequest{
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.Len(t, r4.Items, 2)

	_, err = client.DeleteItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)

	r6, err := client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 2,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), r6.ItemId)
	require.Equal(t, int64(2), r6.VehicleTypeId)

	_, err = client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.Error(t, err)

	r8, err := client.GetItemVehicleTypes(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&APIGetItemVehicleTypesRequest{
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.Len(t, r8.Items, 1)
}
