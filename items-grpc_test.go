package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestTopCategoriesList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	_, err = client.GetTopCategoriesList(ctx, &GetTopCategoriesListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
}

func TestGetTwinsBrandsList(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	cnt := NewContainer(cfg)
	defer util.Close(cnt)

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	ctx := context.Background()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("brand1-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      5,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("brand1-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brand1, err := r1.LastInsertId()
	require.NoError(t, err)

	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("brand2-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      5,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("brand2-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brand2, err := r2.LastInsertId()
	require.NoError(t, err)

	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle1-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      1,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle1-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	vehicle1, err := r3.LastInsertId()
	require.NoError(t, err)

	r4, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle2-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      1,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle2-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	vehicle2, err := r4.LastInsertId()
	require.NoError(t, err)

	r5, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("twins-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      4,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("twins-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	twins, err := r5.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentTable).
		Cols(schema.ItemParentTableItemIDColName, schema.ItemParentTableParentIDColName,
			schema.ItemParentTableCatnameColName, schema.ItemParentTableTypeColName).
		Vals(
			goqu.Vals{vehicle1, brand1, "vehicle1", 0},
			goqu.Vals{vehicle2, brand2, "vehicle2", 0},
			goqu.Vals{vehicle1, twins, "vehicle1", 0},
			goqu.Vals{vehicle2, twins, "vehicle2", 0},
		).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	toRebuild := []int64{brand1, brand2, vehicle1, vehicle2, twins}
	for _, id := range toRebuild {
		_, err := rep.RebuildCache(ctx, id)
		require.NoError(t, err)
	}

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	res, err := client.GetTopTwinsBrandsList(ctx, &GetTopTwinsBrandsListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res)

	r6, err := client.GetTwinsBrandsList(ctx, &GetTwinsBrandsListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r6)
	require.NotEmpty(t, r6.GetItems())
}

func TestTopBrandsList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	res, err := client.GetTopBrandsList(ctx, &GetTopBrandsListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestTopPersonsAuthorList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopPersonsList(ctx, &GetTopPersonsListRequest{
		Language:        "ru",
		PictureItemType: PictureItemType_PICTURE_ITEM_AUTHOR,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestTopPersonsContentList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetTopPersonsList(ctx, &GetTopPersonsListRequest{
		Language:        "ru",
		PictureItemType: PictureItemType_PICTURE_ITEM_CONTENT,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestTopFactoriesList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	r, err := client.GetContentLanguages(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.Greater(t, len(r.GetLanguages()), 1)
}

func TestItemLinks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemLink{
			Name:   "Link 1 ",
			Url:    " https://example.org",
			Type:   " club",
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, r1.GetId())

	r2, err := client.GetItemLink(ctx, &APIItemLinkRequest{
		Id: r1.GetId(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, r1.GetId())

	require.Equal(t, r1.GetId(), r2.GetId())
	require.Equal(t, "Link 1", r2.GetName())
	require.Equal(t, "https://example.org", r2.GetUrl())
	require.Equal(t, "club", r2.GetType())
	require.Equal(t, int64(1), r2.GetItemId())

	_, err = client.UpdateItemLink(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemLink{
			Id:     r1.GetId(),
			Name:   "Link 2",
			Url:    "https://example2.org",
			Type:   "default",
			ItemId: 2,
		},
	)
	require.NoError(t, err)

	r3, err := client.GetItemLink(ctx, &APIItemLinkRequest{
		Id: r1.GetId(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, r1.GetId())

	require.Equal(t, r1.GetId(), r3.GetId())
	require.Equal(t, "Link 2", r3.GetName())
	require.Equal(t, "https://example2.org", r3.GetUrl())
	require.Equal(t, "default", r3.GetType())
	require.Equal(t, int64(2), r3.GetItemId())

	r4, err := client.GetItemLinks(ctx, &APIGetItemLinksRequest{
		ItemId: r3.GetItemId(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, r4.GetItems())
}

func TestItemVehicleTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleType{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)

	r2, err := client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), r2.GetItemId())
	require.Equal(t, int64(1), r2.GetVehicleTypeId())

	_, err = client.CreateItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleType{
			ItemId:        1,
			VehicleTypeId: 2,
		},
	)
	require.NoError(t, err)

	r4, err := client.GetItemVehicleTypes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIGetItemVehicleTypesRequest{
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.Len(t, r4.GetItems(), 2)

	_, err = client.DeleteItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.NoError(t, err)

	r6, err := client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 2,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), r6.GetItemId())
	require.Equal(t, int64(2), r6.GetVehicleTypeId())

	_, err = client.GetItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleTypeRequest{
			ItemId:        1,
			VehicleTypeId: 1,
		},
	)
	require.Error(t, err)

	r8, err := client.GetItemVehicleTypes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIGetItemVehicleTypesRequest{
			ItemId: 1,
		},
	)
	require.NoError(t, err)
	require.Len(t, r8.GetItems(), 1)
}

func TestItemParentLanguages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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

	_, err = client.GetItemParentLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIGetItemParentLanguagesRequest{
			ItemId:   1,
			ParentId: 1,
		},
	)
	require.NoError(t, err)
}

func TestItemLanguages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
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

	_, err = client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIGetItemLanguagesRequest{
			ItemId: 1,
		},
	)
	require.NoError(t, err)
}

func TestCatalogueMenuList(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	ctx := context.Background()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      3,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("category-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	category, err := r1.LastInsertId()
	require.NoError(t, err)

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, category)
	require.NoError(t, err)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	res, err := client.List(ctx, &ListItemsRequest{
		Language: "ru",
		Fields: &ItemFields{
			NameText:         true,
			DescendantsCount: true,
		},
		Limit: 20,
		Options: &ItemListOptions{
			NoParent: true,
			TypeId:   ItemType_ITEM_TYPE_CATEGORY,
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.NotEmpty(t, res.GetItems())
}

func TestStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	r, err := client.GetStats(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
	require.NotEmpty(t, r.GetValues())
}

func TestSetItemParentLanguage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	cases := []struct {
		ParentName           string
		ParentBeginYear      sql.NullInt32
		ParentEndYear        sql.NullInt32
		ParentBeginModelYear sql.NullInt32
		ParentEndModelYear   sql.NullInt32
		ParentSpecID         sql.NullInt32
		ChildName            string
		ChildBeginYear       sql.NullInt32
		ChildEndYear         sql.NullInt32
		ChildBeginModelYear  sql.NullInt32
		ChildEndModelYear    sql.NullInt32
		ChildSpecID          sql.NullInt32
		Result               string
	}{
		{
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{},
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2005},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{},
			"2000–05",
		},
		{
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{},
			"Peugeot %d Coupe",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{},
			"Coupe",
		},
		{
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{},
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 0},
			sql.NullInt32{Valid: true, Int32: 29},
			"Worldwide",
		},
		{
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 2001},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{},
			"Peugeot %d",
			sql.NullInt32{Valid: true, Int32: 2000},
			sql.NullInt32{Valid: true, Int32: 2010},
			sql.NullInt32{Valid: true, Int32: 2001},
			sql.NullInt32{Valid: true, Int32: 2005},
			sql.NullInt32{},
			"2001–05",
		},
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	for _, testCase := range cases {
		randomInt := random.Int()
		childName := fmt.Sprintf(testCase.ChildName, randomInt)
		parentName := fmt.Sprintf(testCase.ParentName, randomInt)

		r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
			Name:            childName,
			IsGroup:         false,
			ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
			Catname:         sql.NullString{Valid: false},
			Body:            "",
			ProducedExactly: false,
			BeginYear:       testCase.ChildBeginYear,
			EndYear:         testCase.ChildEndYear,
			BeginModelYear:  testCase.ChildBeginModelYear,
			EndModelYear:    testCase.ChildEndModelYear,
			SpecID:          testCase.ChildSpecID,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		itemID, err := r1.LastInsertId()
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.ItemLanguageTable).Rows(schema.ItemLanguageRow{
			ItemID:   itemID,
			Language: "xx",
			Name:     sql.NullString{Valid: true, String: childName},
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
			Name:            parentName,
			IsGroup:         true,
			ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
			Catname:         sql.NullString{Valid: false},
			Body:            "",
			ProducedExactly: false,
			BeginYear:       testCase.ParentBeginYear,
			EndYear:         testCase.ParentEndYear,
			BeginModelYear:  testCase.ParentBeginModelYear,
			EndModelYear:    testCase.ParentEndModelYear,
			SpecID:          testCase.ParentSpecID,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		parentID, err := r2.LastInsertId()
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.ItemLanguageTable).Rows(schema.ItemLanguageRow{
			ItemID:   parentID,
			Language: "xx",
			Name:     sql.NullString{Valid: true, String: parentName},
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.ItemParentTable).Rows(schema.ItemParentRow{
			ItemID:   itemID,
			ParentID: parentID,
			Catname:  "child-item",
			Type:     schema.ItemParentTypeDefault,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

		_, err = client.SetItemParentLanguage(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
			&ItemParentLanguage{
				ItemId:   itemID,
				ParentId: parentID,
				Language: "en",
				Name:     "",
			},
		)
		require.NoError(t, err)

		r3, err := client.GetItemParentLanguages(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
			&APIGetItemParentLanguagesRequest{
				ItemId:   itemID,
				ParentId: parentID,
			},
		)
		require.NoError(t, err)

		var itemParentLanguageRow *ItemParentLanguage

		for _, row := range r3.GetItems() {
			if row.GetLanguage() == "en" {
				itemParentLanguageRow = row

				break
			}
		}

		require.NotNil(t, itemParentLanguageRow)
		require.Equal(t, testCase.Result, itemParentLanguageRow.GetName())
	}
}

func TestBrandNewItems(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	ctx := context.Background()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("brand-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDBrand,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("brand-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	_, err = client.GetBrandNewItems(ctx, &NewItemsRequest{
		ItemId:   itemID,
		Language: "ru",
	})
	require.NoError(t, err)
}

func TestNewItems(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	ctx := context.Background()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("category-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	childID, err := r2.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentTable).Rows(schema.ItemParentRow{
		ItemID:   childID,
		ParentID: itemID,
		Catname:  "child-item",
		Type:     schema.ItemParentTypeDefault,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	_, err = client.GetNewItems(ctx, &NewItemsRequest{
		ItemId:   itemID,
		Language: "en",
	})
	require.NoError(t, err)
}

func TestInboxPicturesCount(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	ctx := context.Background()
	repository := items.NewRepository(goquDB, 200)
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	// create brand
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("brand-%d", random.Int()),
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDBrand,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("brand-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brandID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = repository.RebuildCache(ctx, brandID)
	require.NoError(t, err)

	// create vehicle
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	childID, err := r2.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentTable).Rows(schema.ItemParentRow{
		ItemID:   childID,
		ParentID: brandID,
		Catname:  "child-item",
		Type:     schema.ItemParentTypeDefault,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = repository.RebuildCache(ctx, childID)
	require.NoError(t, err)

	// create inbox pictures
	for i := range 10 {
		identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

		status := schema.PictureStatusInbox
		if i >= 5 {
			status = schema.PictureStatusAccepted
		}

		res, err := goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
			Identity: identity,
			Status:   status,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		pictureID, err := res.LastInsertId()
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.PictureItemTable).Rows(schema.PictureItemRow{
			PictureID: pictureID,
			ItemID:    childID,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)
	}

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	client := NewItemsClient(conn)

	_, err = client.List(ctx, &ListItemsRequest{
		Fields: &ItemFields{
			InboxPicturesCount: true,
		},
		Options: &ItemListOptions{
			Id: brandID,
		},
		Language: "en",
	})
	require.ErrorContains(t, err, "PermissionDenied")

	_, err = client.Item(ctx, &ItemRequest{
		Id: brandID,
		Fields: &ItemFields{
			InboxPicturesCount: true,
		},
		Language: "en",
	})
	require.ErrorContains(t, err, "PermissionDenied")

	// login with admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	list, err := client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ListItemsRequest{
			Fields: &ItemFields{
				InboxPicturesCount: true,
			},
			Options: &ItemListOptions{
				Id: brandID,
			},
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Len(t, list.GetItems(), 1)
	require.Equal(t, int32(5), list.GetItems()[0].GetInboxPicturesCount())

	item, err := client.Item(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemRequest{
			Id: brandID,
			Fields: &ItemFields{
				InboxPicturesCount: true,
			},
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Equal(t, int32(5), item.GetInboxPicturesCount())
}
