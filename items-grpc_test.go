package goautowp

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestTopCategoriesList(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := NewItemsClient(conn)

	_, err := client.GetTopCategoriesList(ctx, &GetTopCategoriesListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
}

func TestGetTwinsBrandsList(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("brand1-%d", random.Int()),
		IsGroup:         true,
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
		IsGroup:         true,
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
		IsGroup:         true,
		ItemTypeID:      4,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("twins-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	twins, err := r5.LastInsertId()
	require.NoError(t, err)

	client := NewItemsClient(conn)

	kc := cnt.Keycloak()

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: vehicle1, ParentId: brand1, Catname: "vehicle1",
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: vehicle2, ParentId: brand2, Catname: "vehicle2",
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: vehicle1, ParentId: twins, Catname: "vehicle1",
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: vehicle2, ParentId: twins, Catname: "vehicle2",
		},
	)
	require.NoError(t, err)

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

	ctx := t.Context()

	client := NewItemsClient(conn)

	res, err := client.GetTopBrandsList(ctx, &GetTopBrandsListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestTopPersonsAuthorList(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

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

	ctx := t.Context()

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

	ctx := t.Context()

	client := NewItemsClient(conn)

	r, err := client.GetTopFactoriesList(ctx, &GetTopFactoriesListRequest{
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r)
}

func TestContentLanguages(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := NewItemsClient(conn)

	r, err := client.GetContentLanguages(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.Greater(t, len(r.GetLanguages()), 1)
}

func TestItemLinks(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

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

	r2, err := client.GetItemLink(ctx, &ItemLinksRequest{
		Options: &ItemLinkListOptions{
			Id: r1.GetId(),
		},
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

	r3, err := client.GetItemLink(ctx, &ItemLinksRequest{
		Options: &ItemLinkListOptions{
			Id: r1.GetId(),
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, r1.GetId())

	require.Equal(t, r1.GetId(), r3.GetId())
	require.Equal(t, "Link 2", r3.GetName())
	require.Equal(t, "https://example2.org", r3.GetUrl())
	require.Equal(t, "default", r3.GetType())
	require.Equal(t, int64(2), r3.GetItemId())

	r4, err := client.GetItemLinks(ctx, &ItemLinksRequest{
		Options: &ItemLinkListOptions{
			ItemId: r3.GetItemId(),
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, r4.GetItems())
}

func TestItemVehicleTypes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

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

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

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

	client := NewItemsClient(conn)

	res, err := client.List(ctx, &ItemsRequest{
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

	ctx := t.Context()

	client := NewItemsClient(conn)

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

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

	ctx := t.Context()

	client := NewItemsClient(conn)

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

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

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

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

		_, err = client.UpdateItemLanguage(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
			&ItemLanguage{
				ItemId:   itemID,
				Language: items.DefaultLanguageCode,
				Name:     childName,
			},
		)
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

		_, err = client.UpdateItemLanguage(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
			&ItemLanguage{
				ItemId:   parentID,
				Language: items.DefaultLanguageCode,
				Name:     parentName,
			},
		)
		require.NoError(t, err)

		_, err = client.CreateItemParent(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
			&ItemParent{
				ItemId: itemID, ParentId: parentID, Catname: "child-item", Type: ItemParentType_ITEM_TYPE_DEFAULT,
			},
		)
		require.NoError(t, err)

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

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

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

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         true,
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

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID, ParentId: itemID, Catname: "child-item", Type: ItemParentType_ITEM_TYPE_DEFAULT,
		},
	)
	require.NoError(t, err)

	_, err = client.GetNewItems(ctx, &NewItemsRequest{
		ItemId:   itemID,
		Language: "en",
	})
	require.NoError(t, err)
}

func TestInboxPicturesCount(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	repository := items.NewRepository(goquDB, 200, cfg.ContentLanguages, textstorage.New(goquDB))
	kc := cnt.Keycloak()

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

	client := NewItemsClient(conn)
	picturesClient := NewPicturesClient(conn)

	// login with admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: childID, ParentId: brandID, Catname: "child-item", Type: ItemParentType_ITEM_TYPE_DEFAULT,
		},
	)
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
			IP:       util.IP(net.IPv4zero),
			AddDate:  time.Now(),
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		pictureID, err := res.LastInsertId()
		require.NoError(t, err)

		_, err = picturesClient.CreatePictureItem(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
			&CreatePictureItemRequest{
				PictureId: pictureID,
				ItemId:    childID,
				Type:      PictureItemType_PICTURE_ITEM_CONTENT,
			},
		)
		require.NoError(t, err)
	}

	_, err = client.List(ctx, &ItemsRequest{
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

	list, err := client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemsRequest{
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

func TestCreateMoveDeleteItemParent(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("category-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentID1, err := r1.LastInsertId()
	require.NoError(t, err)

	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("category-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentID2, err := r2.LastInsertId()
	require.NoError(t, err)

	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	childID, err := r3.LastInsertId()
	require.NoError(t, err)

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	// attach to first parent
	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID, ParentId: parentID1, Catname: "child-item", Type: ItemParentType_ITEM_TYPE_DEFAULT,
		},
	)
	require.NoError(t, err)

	// check child in parent 1
	res, err := client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID1,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 1)

	resItem := res.GetItems()[0]
	require.Equal(t, childID, resItem.GetId())

	// move to second parent
	_, err = client.MoveItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&MoveItemParentRequest{
			ItemId: childID, ParentId: parentID1, DestParentId: parentID2,
		},
	)
	require.NoError(t, err)

	// check no childs in parent 1
	res, err = client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID1,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Empty(t, res.GetItems())

	// check child in parent 2
	res, err = client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID2,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 1)

	resItem = res.GetItems()[0]
	require.Equal(t, childID, resItem.GetId())

	// delete
	_, err = client.DeleteItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&DeleteItemParentRequest{
			ItemId:   childID,
			ParentId: parentID2,
		},
	)
	require.NoError(t, err)

	// check no childs in parent 2
	res, err = client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID2,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Empty(t, res.GetItems())
}

func TestDeleteItemParentNotDeletesSecondChild(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("category-%d", random.Int()),
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("category-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentID, err := r1.LastInsertId()
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

	childID1, err := r2.LastInsertId()
	require.NoError(t, err)

	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", random.Int())},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	childID2, err := r3.LastInsertId()
	require.NoError(t, err)

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	// attach first to parent
	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID1, ParentId: parentID, Catname: "child-item-1", Type: ItemParentType_ITEM_TYPE_DEFAULT,
		},
	)
	require.NoError(t, err)

	// attach second to parent
	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID2, ParentId: parentID, Catname: "child-item-2", Type: ItemParentType_ITEM_TYPE_DEFAULT,
		},
	)
	require.NoError(t, err)

	// check childs in parent
	res, err := client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 2)

	// delete
	_, err = client.DeleteItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&DeleteItemParentRequest{
			ItemId:   childID1,
			ParentId: parentID,
		},
	)
	require.NoError(t, err)

	// check child 2 in parent
	res, err = client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 1)

	resItem := res.GetItems()[0]
	require.Equal(t, childID2, resItem.GetId())
}

func TestUpdateItemParent(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	parentName := fmt.Sprintf("Peugeot-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            parentName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   parentID,
			Language: items.DefaultLanguageCode,
			Name:     parentName,
		},
	)
	require.NoError(t, err)

	childName := fmt.Sprintf("Peugeot-%d 407", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            childName,
		IsGroup:         false,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d 407", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	childID, err := r2.LastInsertId()
	require.NoError(t, err)

	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   childID,
			Language: items.DefaultLanguageCode,
			Name:     childName,
		},
	)
	require.NoError(t, err)

	// attach first to parent
	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID, ParentId: parentID,
		},
	)
	require.NoError(t, err)

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	row, err := rep.ItemParent(ctx, &query.ItemParentListOptions{
		ItemID:   childID,
		ParentID: parentID,
	}, items.ItemParentFields{})
	require.NoError(t, err)
	require.Equal(t, "407", row.Catname)
	require.Equal(t, schema.ItemParentTypeDefault, row.Type)
	require.False(t, row.ManualCatname)

	// update
	_, err = client.UpdateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID, ParentId: parentID, Type: ItemParentType_ITEM_TYPE_DESIGN, Catname: "custom",
		},
	)
	require.NoError(t, err)

	// check child in parent
	res, err := client.List(ctx, &ItemsRequest{
		Options: &ItemListOptions{
			Parent: &ItemParentListOptions{
				ParentId: parentID,
			},
		},
		Language: "en",
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 1)

	resItem := res.GetItems()[0]
	require.Equal(t, childID, resItem.GetId())

	// check row
	row, err = rep.ItemParent(ctx, &query.ItemParentListOptions{
		ItemID:   childID,
		ParentID: parentID,
	}, items.ItemParentFields{})
	require.NoError(t, err)
	require.Equal(t, "custom", row.Catname)
	require.Equal(t, schema.ItemParentTypeDesign, row.Type)
	require.True(t, row.ManualCatname)

	// update
	_, err = client.UpdateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: childID, ParentId: parentID, Type: ItemParentType_ITEM_TYPE_TUNING, Catname: "",
		},
	)
	require.NoError(t, err)

	// check row
	row, err = rep.ItemParent(ctx, &query.ItemParentListOptions{
		ItemID:   childID,
		ParentID: parentID,
	}, items.ItemParentFields{})
	require.NoError(t, err)
	require.Equal(t, "407", row.Catname)
	require.Equal(t, schema.ItemParentTypeTuning, row.Type)
	require.False(t, row.ManualCatname)
}

func TestUpdateItemLanguage(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	itemName := fmt.Sprintf("Peugeot-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            itemName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   itemID,
			Language: "fr",
			Name:     itemName,
		},
	)
	require.NoError(t, err)

	res, err := client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&APIGetItemLanguagesRequest{
			ItemId: itemID,
		},
	)
	require.NoError(t, err)

	rows := res.GetItems()
	require.Len(t, rows, 1)

	row := rows[0]
	require.Equal(t, itemID, row.GetItemId())
	require.Equal(t, "fr", row.GetLanguage())
	require.Equal(t, itemName, row.GetName())
	require.Zero(t, row.GetTextId())
	require.Zero(t, row.GetFullTextId())

	// setup text
	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   itemID,
			Language: "fr",
			Name:     itemName,
			Text:     "a text",
		},
	)
	require.NoError(t, err)

	res, err = client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&APIGetItemLanguagesRequest{
			ItemId: itemID,
		},
	)
	require.NoError(t, err)

	rows = res.GetItems()
	require.Len(t, rows, 1)

	row = rows[0]
	require.Equal(t, itemID, row.GetItemId())
	require.Equal(t, "fr", row.GetLanguage())
	require.Equal(t, itemName, row.GetName())
	require.NotZero(t, row.GetTextId())
	require.Equal(t, "a text", row.GetText())
	require.Zero(t, row.GetFullTextId())

	lastTextID := row.GetTextId()

	// update text
	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   itemID,
			Language: "fr",
			Name:     itemName,
			Text:     "a second text",
		},
	)
	require.NoError(t, err)

	res, err = client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&APIGetItemLanguagesRequest{
			ItemId: itemID,
		},
	)
	require.NoError(t, err)

	rows = res.GetItems()
	require.Len(t, rows, 1)

	row = rows[0]
	require.Equal(t, itemID, row.GetItemId())
	require.Equal(t, "fr", row.GetLanguage())
	require.Equal(t, itemName, row.GetName())
	require.Equal(t, lastTextID, row.GetTextId())
	require.Equal(t, "a second text", row.GetText())
	require.Zero(t, row.GetFullTextId())

	// setup full text
	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   itemID,
			Language: "fr",
			Name:     itemName,
			Text:     "a second text",
			FullText: "a full text",
		},
	)
	require.NoError(t, err)

	res, err = client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&APIGetItemLanguagesRequest{
			ItemId: itemID,
		},
	)
	require.NoError(t, err)

	rows = res.GetItems()
	require.Len(t, rows, 1)

	row = rows[0]
	require.Equal(t, itemID, row.GetItemId())
	require.Equal(t, "fr", row.GetLanguage())
	require.Equal(t, itemName, row.GetName())
	require.Equal(t, lastTextID, row.GetTextId())
	require.Equal(t, "a second text", row.GetText())
	require.Equal(t, "a full text", row.GetFullText())
	require.NotZero(t, row.GetFullTextId())

	lastFullTextID := row.GetFullTextId()

	// clear texts
	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   itemID,
			Language: "fr",
			Name:     itemName,
		},
	)
	require.NoError(t, err)

	res, err = client.GetItemLanguages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&APIGetItemLanguagesRequest{
			ItemId: itemID,
		},
	)
	require.NoError(t, err)

	rows = res.GetItems()
	require.Len(t, rows, 1)

	row = rows[0]
	require.Equal(t, itemID, row.GetItemId())
	require.Equal(t, "fr", row.GetLanguage())
	require.Equal(t, itemName, row.GetName())
	require.Equal(t, lastTextID, row.GetTextId())
	require.Equal(t, "", row.GetText())
	require.Equal(t, "", row.GetFullText())
	require.Equal(t, lastFullTextID, row.GetFullTextId())
}

func TestSetUserItemSubscription(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	itemName := fmt.Sprintf("Peugeot-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            itemName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDCategory,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = client.SetUserItemSubscription(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetUserItemSubscriptionRequest{
			ItemId:     itemID,
			Subscribed: true,
		},
	)
	require.NoError(t, err)

	_, err = client.SetUserItemSubscription(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetUserItemSubscriptionRequest{
			ItemId:     itemID,
			Subscribed: true,
		},
	)
	require.NoError(t, err)

	_, err = client.SetUserItemSubscription(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetUserItemSubscriptionRequest{
			ItemId:     itemID,
			Subscribed: false,
		},
	)
	require.NoError(t, err)
}

func TestSetItemEngine(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	itemName := fmt.Sprintf("Peugeot-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            itemName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	engineName := fmt.Sprintf("Peugeot-%d-Engine", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            engineName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDEngine,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d-engine", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	engineID, err := r2.LastInsertId()
	require.NoError(t, err)

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineItemId:    engineID,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err := client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Equal(t, engineID, res.GetEngineItemId())

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineItemId:    engineID,
			EngineInherited: true,
		},
	)
	require.NoError(t, err)

	res, err = client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Zero(t, res.GetEngineItemId())

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineItemId:    engineID,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err = client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Equal(t, engineID, res.GetEngineItemId())

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineItemId:    0,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err = client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Zero(t, res.GetEngineItemId())
}

func TestSetItemEngineInheritance(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	itemName := fmt.Sprintf("Peugeot-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            itemName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	engineName := fmt.Sprintf("Peugeot-%d-Engine", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            engineName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDEngine,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d-engine", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	engineID, err := r2.LastInsertId()
	require.NoError(t, err)

	// test inheritance
	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            fmt.Sprintf("Peugeot-%d-Parent", randomInt),
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDVehicle,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d-parent", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentItemID, err := r3.LastInsertId()
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: itemID, ParentId: parentItemID, Catname: "vehicle1",
		},
	)
	require.NoError(t, err)

	// set parent engine
	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          parentItemID,
			EngineItemId:    engineID,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err := client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Zero(t, res.GetEngineItemId())

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineInherited: true,
		},
	)
	require.NoError(t, err)

	res, err = client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Equal(t, engineID, res.GetEngineItemId())

	// reset parent engine
	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&SetItemEngineRequest{
			ItemId:          parentItemID,
			EngineItemId:    0,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err = client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Zero(t, res.GetEngineItemId())
}

func TestGetBrands(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	testCases := []struct {
		Name      string
		Catname   string
		Language  string
		Category  APIBrandsListLine_Category
		Character string
	}{
		{
			Name:      "123",
			Catname:   "numeric",
			Language:  "en",
			Category:  APIBrandsListLine_NUMBER,
			Character: "1",
		},
		{
			Name:      "Бренд",
			Catname:   "cyrillic",
			Language:  "en",
			Category:  APIBrandsListLine_CYRILLIC,
			Character: "Б",
		},
		{
			Name:      "Latin Brand",
			Catname:   "latin",
			Language:  "en",
			Category:  APIBrandsListLine_LATIN,
			Character: "L",
		},
		{
			Name:      "所有",
			Catname:   "han",
			Language:  "en",
			Category:  APIBrandsListLine_LATIN,
			Character: "S",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Character, func(t *testing.T) {
			t.Parallel()

			brandName := fmt.Sprintf("%s-%d", testCase.Name, randomInt)
			r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
				Name:       brandName,
				IsGroup:    true,
				ItemTypeID: schema.ItemTableItemTypeIDBrand,
				Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("%s-%d", testCase.Catname, randomInt)},
				Body:       "",
			}).Executor().ExecContext(ctx)
			require.NoError(t, err)

			brandID, err := r1.LastInsertId()
			require.NoError(t, err)

			// setup text
			_, err = client.UpdateItemLanguage(
				metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
				&ItemLanguage{
					ItemId:   brandID,
					Language: testCase.Language,
					Name:     brandName,
				},
			)
			require.NoError(t, err)

			idx, err := cnt.Index()
			require.NoError(t, err)

			err = idx.GenerateBrandsCache(ctx, testCase.Language)
			require.NoError(t, err)

			res, err := client.GetBrands(ctx, &GetBrandsRequest{Language: testCase.Language})
			require.NoError(t, err)

			lineIndex := slices.IndexFunc(res.GetLines(), func(line *APIBrandsListLine) bool {
				return testCase.Category == line.GetCategory()
			})
			require.GreaterOrEqual(t, lineIndex, 0)

			line := res.GetLines()[lineIndex]

			characterIndex := slices.IndexFunc(line.GetCharacters(), func(character *APIBrandsListCharacter) bool {
				return testCase.Character == character.GetCharacter()
			})
			require.GreaterOrEqual(t, characterIndex, 0)

			character := line.GetCharacters()[characterIndex]

			require.True(t,
				slices.ContainsFunc(character.GetItems(), func(item *APIBrandsListItem) bool {
					return item.GetId() == brandID
				}),
			)
		})
	}
}

func TestBrandSections(t *testing.T) {
	t.Parallel()

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	var ids []int64
	err = goquDB.Select(schema.ItemTableIDCol).
		From(schema.ItemTable).
		Where(schema.ItemTableItemTypeIDCol.Eq(schema.ItemTableItemTypeIDBrand)).
		ScanValsContext(ctx, &ids)
	require.NoError(t, err)

	for _, id := range ids {
		_, err = client.GetBrandSections(ctx, &GetBrandSectionsRequest{Language: "en", ItemId: id})
		require.NoError(t, err)
	}
}

func TestBrandSections2(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	// create brand
	brandName := fmt.Sprintf("Opel-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       brandName,
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDBrand,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("opel-%d", randomInt)},
		Body:       "",
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brandID, err := r1.LastInsertId()
	require.NoError(t, err)

	// setup text
	_, err = client.UpdateItemLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemLanguage{
			ItemId:   brandID,
			Language: "en",
			Name:     brandName,
		},
	)
	require.NoError(t, err)

	testCases := []struct {
		ItemTypeID    schema.ItemTableItemTypeID
		Name          string
		Catname       string
		SectionNames  []string
		VehicleTypeID []int64
	}{
		{
			ItemTypeID:   schema.ItemTableItemTypeIDVehicle,
			Name:         "Calibra",
			Catname:      "calibra",
			SectionNames: []string{""},
		},
		{
			ItemTypeID:    schema.ItemTableItemTypeIDVehicle,
			Name:          "Insignia",
			Catname:       "Insignia",
			SectionNames:  []string{""},
			VehicleTypeID: []int64{items.VehicleTypeIDCar},
		},
		{
			ItemTypeID:    schema.ItemTableItemTypeIDVehicle,
			Name:          "Motoclub",
			Catname:       "motoclub",
			SectionNames:  []string{"catalogue/section/moto"},
			VehicleTypeID: []int64{items.VehicleTypeIDMoto},
		},
		{
			ItemTypeID:    schema.ItemTableItemTypeIDVehicle,
			Name:          "Blitz",
			Catname:       "blitz",
			SectionNames:  []string{"catalogue/section/trucks", "catalogue/section/buses"},
			VehicleTypeID: []int64{items.VehicleTypeIDTruck, items.VehicleTypeIDBus},
		},
		{
			ItemTypeID:   schema.ItemTableItemTypeIDEngine,
			Name:         "Engine",
			Catname:      "engine",
			SectionNames: []string{"catalogue/section/engines"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()

			// create child vehicle
			childName := fmt.Sprintf("Opel-%d %s", randomInt, testCase.Name)
			r1, err = goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
				Name:       childName,
				IsGroup:    true,
				ItemTypeID: testCase.ItemTypeID,
				Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("opel-%d-%s", randomInt, testCase.Catname)},
				Body:       "",
			}).Executor().ExecContext(ctx)
			require.NoError(t, err)

			childID, err := r1.LastInsertId()
			require.NoError(t, err)

			// setup text
			_, err = client.UpdateItemLanguage(
				metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
				&ItemLanguage{
					ItemId:   childID,
					Language: "en",
					Name:     childName,
				},
			)
			require.NoError(t, err)

			for _, vehicleTypeID := range testCase.VehicleTypeID {
				_, err = client.CreateItemVehicleType(
					metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
					&APIItemVehicleType{
						ItemId:        childID,
						VehicleTypeId: vehicleTypeID,
					},
				)
				require.NoError(t, err)
			}

			_, err = client.CreateItemParent(
				metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
				&ItemParent{
					ItemId: childID, ParentId: brandID, Type: ItemParentType_ITEM_TYPE_DEFAULT,
				},
			)
			require.NoError(t, err)

			res, err := client.GetBrandSections(ctx, &GetBrandSectionsRequest{Language: "en", ItemId: brandID})
			require.NoError(t, err)

			for _, sectionName := range testCase.SectionNames {
				idx := slices.IndexFunc(res.GetSections(), func(section *APIBrandSection) bool {
					return section.GetName() == sectionName
				})
				require.GreaterOrEqual(t, idx, 0, "section `%s` not present in results", sectionName)

				section := res.GetSections()[idx]

				require.True(t, slices.ContainsFunc(section.GetGroups(), func(a *APIBrandSection) bool {
					return a.GetName() == testCase.Name
				}), "item `%s` not found in section `%s`", testCase.Name, sectionName)
			}
		})
	}
}

func TestTwinsGroupBrands(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	groupName := fmt.Sprintf("Twins-Group-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       groupName,
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDTwins,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("twins-group-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	groupID, err := r1.LastInsertId()
	require.NoError(t, err)

	brandName := fmt.Sprintf("Brand-%d", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       brandName,
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDBrand,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("brand-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brandID, err := r2.LastInsertId()
	require.NoError(t, err)

	vehicleName := fmt.Sprintf("Vehicle-%d", randomInt)
	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       vehicleName,
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	vehicleID, err := r3.LastInsertId()
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: groupID,
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: brandID,
		},
	)
	require.NoError(t, err)

	res, err := client.List(
		ctx,
		&ItemsRequest{
			Options: &ItemListOptions{
				TypeId: ItemType_ITEM_TYPE_BRAND,
				Child: &ItemParentListOptions{
					ItemParentParentByChild: &ItemParentListOptions{
						ParentId: groupID,
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, res)

	list := res.GetItems()

	require.Len(t, list, 1)

	var found bool

	for _, item := range list {
		if item.GetId() == brandID {
			found = true

			break
		}
	}

	require.True(t, found)
}

func TestAutocomplete(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	client := NewItemsClient(conn)

	_, err = client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemsRequest{
			Options: &ItemListOptions{
				ParentTypesOf:        ItemType_ITEM_TYPE_BRAND,
				IsGroup:              true,
				ExcludeSelfAndChilds: 3,
				Autocomplete:         "Test",
			},
		},
	)
	require.NoError(t, err)
}

func TestTooBig(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	client := NewItemsClient(conn)

	_, err = client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemsRequest{
			Order: ItemsRequest_CHILDS_COUNT,
			Fields: &ItemFields{
				ChildsCount: true,
			},
			Limit: 100,
		},
	)
	require.NoError(t, err)
}

func TestSuggestionsTo(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	client := NewItemsClient(conn)

	_, err = client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemsRequest{
			Options: &ItemListOptions{
				SuggestionsTo: 1,
			},
			Limit: 1,
		},
	)
	require.NoError(t, err)
}

func TestGetItemParents(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	parentName := fmt.Sprintf("Parent-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       parentName,
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDCategory,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("parent-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	parentID, err := r1.LastInsertId()
	require.NoError(t, err)

	child1Name := fmt.Sprintf("Child1-%d", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       child1Name,
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("child1-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	child1ID, err := r2.LastInsertId()
	require.NoError(t, err)

	child2Name := fmt.Sprintf("Child2-%d", randomInt)
	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       child2Name,
		ItemTypeID: schema.ItemTableItemTypeIDCategory,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("child2-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	child2ID, err := r3.LastInsertId()
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: child1ID, ParentId: parentID,
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: child2ID, ParentId: parentID,
		},
	)
	require.NoError(t, err)

	res, err := client.GetItemParents(t.Context(), &ItemParentsRequest{
		Options: &ItemParentListOptions{
			ParentId: parentID,
		},
		Order: ItemParentsRequest_AUTO,
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 2)

	for _, row := range res.GetItems() {
		require.Equal(t, parentID, row.GetParentId())
		require.Contains(t, []int64{child1ID, child2ID}, row.GetItemId())
	}

	res, err = client.GetItemParents(t.Context(), &ItemParentsRequest{
		Options: &ItemParentListOptions{
			ParentId: parentID,
		},
		Order: ItemParentsRequest_CATEGORIES_FIRST,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, res.GetItems(), 1)
	require.Equal(t, int32(2), res.GetPaginator().GetTotalItemCount())
	require.Equal(t, int32(2), res.GetPaginator().GetPageCount())
	require.Equal(t, int32(1), res.GetPaginator().GetCurrent())
	require.Equal(t, parentID, res.GetItems()[0].GetParentId())
	require.Equal(t, child2ID, res.GetItems()[0].GetItemId())
}

func TestItemFields(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewItemsClient(conn)
	kc := cnt.Keycloak()
	cfg := config.LoadConfig(".")

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	res, err := client.List(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemsRequest{
			Language: "ru",
			Fields: &ItemFields{
				AltNames:                   true,
				AcceptedPicturesCount:      true,
				AttrZoneId:                 true,
				Brandicon:                  true,
				ChildsCounts:               true,
				ChildsCount:                true,
				CommentsAttentionsCount:    true,
				HasChildSpecs:              true,
				HasSpecs:                   true,
				InboxPicturesCount:         true,
				IsCompilesItemOfDay:        true,
				Location:                   true,
				Links:                      &ItemLinksRequest{},
				Logo120:                    true,
				MostsActive:                true,
				OtherNames:                 true,
				PictureItems:               &PictureItemsRequest{},
				PreviewPictures:            &PreviewPicturesRequest{},
				PublicRoutes:               true,
				Route:                      true,
				SpecsRoute:                 true,
				NameHtml:                   true,
				NameDefault:                true,
				NameOnly:                   true,
				NameText:                   true,
				DescendantsCount:           true,
				Description:                true,
				HasText:                    true,
				DescendantTwinsGroupsCount: true,
				Design:                     true,
				RelatedGroupPictures:       true,
				ItemOfDayPictures:          true,
				ParentsCount:               true,
				ExactPicturesCount:         true,
				Logo:                       true,
				EngineVehiclesCount:        true,
				ItemLanguageCount:          true,
				LinksCount:                 true,
				SpecificationsCount:        true,
				Subscription:               true,
			},
			Limit: 200,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestItemParentFields(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewItemsClient(conn)
	kc := cnt.Keycloak()
	cfg := config.LoadConfig(".")

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	res, err := client.GetItemParents(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParentsRequest{
			Language: "ru",
			Fields: &ItemParentFields{
				Item: &ItemFields{
					AcceptedPicturesCount:   true,
					ChildsCounts:            true,
					CommentsAttentionsCount: true,
					Description:             true,
					Design:                  true,
					FullText:                true,
					InboxPicturesCount:      true,
					NameDefault:             true,
					NameHtml:                true,
					NameText:                true,
					OtherNames:              true,
					SpecsRoute:              true,
					EngineVehicles:          &ItemsRequest{},
				},
				Parent:          &ItemFields{},
				DuplicateParent: &ItemFields{},
				DuplicateChild:  &ItemFields{},
			},
			Limit: 200,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestTwinsGroupPictures(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)
	picturesClient := NewPicturesClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	groupName := fmt.Sprintf("Twins-Group-%d", randomInt)
	r1, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       groupName,
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDTwins,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("twins-group-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	groupID, err := r1.LastInsertId()
	require.NoError(t, err)

	vehicleName := fmt.Sprintf("Vehicle-%d", randomInt)
	r3, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:       vehicleName,
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", randomInt)},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	vehicleID, err := r3.LastInsertId()
	require.NoError(t, err)

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: groupID,
		},
	)
	require.NoError(t, err)

	identity := "t" + strconv.Itoa(randomInt%100000)

	res, err := goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusAccepted,
		IP:       util.IP(net.IPv4zero),
		AddDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = picturesClient.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    vehicleID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	res2, err := client.GetItemParents(ctx, &ItemParentsRequest{
		Options: &ItemParentListOptions{
			ParentId: groupID,
			ItemId:   vehicleID,
		},
		Fields: &ItemParentFields{
			ChildDescendantPictures: &PicturesRequest{
				Limit: 1,
				Options: &PictureListOptions{
					Status: PictureStatus_PICTURE_STATUS_ACCEPTED,
				},
				Order: PicturesRequest_ORDER_FRONT_PERSPECTIVES,
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res2.GetItems())
	require.NotEmpty(t, res2.GetItems()[0].GetChildDescendantPictures())
	require.NotEmpty(t, res2.GetItems()[0].GetChildDescendantPictures().GetItems()[0])
	require.EqualValues(t, pictureID, res2.GetItems()[0].GetChildDescendantPictures().GetItems()[0].GetId())
}

func TestPersonPreviewPictures(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)
	picturesClient := NewPicturesClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	personName := fmt.Sprintf("Person-%d", randomInt)
	personID := createItem(t, goquDB, schema.ItemRow{
		Name:       personName,
		IsGroup:    false,
		ItemTypeID: schema.ItemTableItemTypeIDPerson,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("person-%d", randomInt)},
	})

	identity := "t" + strconv.Itoa(randomInt%100000)

	res, err := goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusAccepted,
		IP:       util.IP(net.IPv4zero),
		AddDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = picturesClient.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    personID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	res2, err := client.List(ctx, &ItemsRequest{
		Fields: &ItemFields{
			Description: true,
			HasText:     true,
			NameDefault: true,
			NameHtml:    true,
			PreviewPictures: &PreviewPicturesRequest{
				Pictures: &PicturesRequest{
					Options: &PictureListOptions{
						PictureItem: &PictureItemListOptions{
							TypeId: PictureItemType_PICTURE_ITEM_CONTENT,
						},
					},
				},
			},
		},
		Language: "en",
		Limit:    10,
		Options: &ItemListOptions{
			Id: personID,
			Descendant: &ItemParentCacheListOptions{
				PictureItemsByItemId: &PictureItemListOptions{
					Pictures: &PictureListOptions{Status: PictureStatus_PICTURE_STATUS_ACCEPTED},
					TypeId:   PictureItemType_PICTURE_ITEM_CONTENT,
				},
			},
			TypeId: ItemType_ITEM_TYPE_PERSON,
		},
		Order: ItemsRequest_NAME,
		Page:  1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, res2.GetItems())
	require.NotEmpty(t, res2.GetItems()[0].GetPreviewPictures())
	require.EqualValues(t, 1, res2.GetItems()[0].GetPreviewPictures().GetTotalPictures())
	require.EqualValues(t, pictureID, res2.GetItems()[0].GetPreviewPictures().GetPictures()[0].GetPicture().GetId())
}

func TestCutawayAuthorsWithPreviewPictures(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()

	client := NewItemsClient(conn)
	picturesClient := NewPicturesClient(conn)

	// admin
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	personName := fmt.Sprintf("Person-%d", randomInt)
	personID := createItem(t, goquDB, schema.ItemRow{
		Name:       personName,
		IsGroup:    false,
		ItemTypeID: schema.ItemTableItemTypeIDPerson,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("person-%d", randomInt)},
	})

	vehicleName := fmt.Sprintf("Vehicle-%d", randomInt)
	vehicleID := createItem(t, goquDB, schema.ItemRow{
		Name:       vehicleName,
		IsGroup:    false,
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", randomInt)},
	})

	identity := "t" + strconv.Itoa(randomInt%100000)

	res, err := goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusAccepted,
		IP:       util.IP(net.IPv4zero),
		AddDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = picturesClient.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CreatePictureItemRequest{
			PictureId:     pictureID,
			ItemId:        vehicleID,
			Type:          PictureItemType_PICTURE_ITEM_CONTENT,
			PerspectiveId: schema.PerspectiveCutaway,
		},
	)
	require.NoError(t, err)

	_, err = picturesClient.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    personID,
			Type:      PictureItemType_PICTURE_ITEM_AUTHOR,
		},
	)
	require.NoError(t, err)

	res2, err := client.List(ctx, &ItemsRequest{
		Fields: &ItemFields{
			Description: true,
			HasText:     true,
			NameDefault: true,
			NameHtml:    true,
			PreviewPictures: &PreviewPicturesRequest{
				OnlyExactlyPictures: true,
				Pictures: &PicturesRequest{
					Options: &PictureListOptions{
						PictureItem: &PictureItemListOptions{
							PictureItemByPictureId: &PictureItemListOptions{PerspectiveId: schema.PerspectiveCutaway},
							TypeId:                 PictureItemType_PICTURE_ITEM_AUTHOR,
						},
						Status: PictureStatus_PICTURE_STATUS_ACCEPTED,
					},
				},
			},
		},
		Language: "en",
		Limit:    12,
		Options: &ItemListOptions{
			Id: personID,
			PictureItems: &PictureItemListOptions{
				Pictures: &PictureListOptions{
					PictureItem: &PictureItemListOptions{PerspectiveId: schema.PerspectiveCutaway},
					Status:      PictureStatus_PICTURE_STATUS_ACCEPTED,
				},
				TypeId: PictureItemType_PICTURE_ITEM_AUTHOR,
			},
			TypeId: ItemType_ITEM_TYPE_PERSON,
		},
		Order: ItemsRequest_AGE,
	})
	require.NoError(t, err)
	require.NotEmpty(t, res2.GetItems())
	require.NotEmpty(t, res2.GetItems()[0].GetPreviewPictures())
	require.EqualValues(t, 1, res2.GetItems()[0].GetPreviewPictures().GetTotalPictures())
	require.EqualValues(t, pictureID, res2.GetItems()[0].GetPreviewPictures().GetPictures()[0].GetPicture().GetId())
}

func TestItemOfDayPicture(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewItemsClient(conn)
	kc := cnt.Keycloak()
	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	vehicleName := fmt.Sprintf("Vehicle-%d", randomInt)
	vehicleID := createItem(t, goquDB, schema.ItemRow{
		Name:       vehicleName,
		IsGroup:    false,
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", randomInt)},
	})

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, vehicleID)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusAccepted)

	picturesClient := NewPicturesClient(conn)

	_, err = picturesClient.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&CreatePictureItemRequest{
			PictureId:     pictureID,
			ItemId:        vehicleID,
			Type:          PictureItemType_PICTURE_ITEM_CONTENT,
			PerspectiveId: schema.PerspectiveFrontStrict,
		},
	)
	require.NoError(t, err)

	res, err := client.Item(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemRequest{
			Language: "ru",
			Fields:   &ItemFields{ItemOfDayPictures: true},
			Id:       vehicleID,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.NotEmpty(t, res.GetItemOfDayPictures())
	require.NotEmpty(t, res.GetItemOfDayPictures()[0].GetThumb().GetSrc())
}

func TestGetTopSpecsContributions(t *testing.T) {
	t.Parallel()

	client := NewItemsClient(conn)
	attsClient := NewAttrsClient(conn)

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	itemID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Name:       "Test",
		Body:       "E31",
		IsGroup:    true,
	})

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, itemID)
	require.NoError(t, err)

	_, err = attsClient.SetUserValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrSetUserValuesRequest{
			Items: []*AttrUserValue{
				{
					AttributeId: schema.FuelSupplySystemAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:      AttrAttributeType_LIST,
						Valid:     true,
						ListValue: []int64{25},
					},
				},
				{
					AttributeId: schema.EngineCylinderDiameter,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:    AttrAttributeType_FLOAT,
						Valid:   true,
						IsEmpty: true,
					},
				},
				{
					AttributeId: schema.WidthAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:    AttrAttributeType_INTEGER,
						Valid:   true,
						IsEmpty: true,
					},
				},
				{
					AttributeId: schema.ABSAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:    AttrAttributeType_BOOLEAN,
						Valid:   true,
						IsEmpty: true,
					},
				},
				{
					AttributeId: schema.EngineTypeAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:      AttrAttributeType_TREE,
						Valid:     true,
						IsEmpty:   false,
						ListValue: []int64{105},
					},
				},
				{
					AttributeId: schema.EnginePlacementOrientationAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:      AttrAttributeType_LIST,
						Valid:     true,
						IsEmpty:   true,
						ListValue: []int64{},
					},
				},
				{
					AttributeId: schema.DriveUnitAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:      AttrAttributeType_LIST,
						Valid:     true,
						IsEmpty:   true,
						ListValue: []int64{},
					},
				},
				{
					AttributeId: schema.TurningDiameterAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:       AttrAttributeType_FLOAT,
						Valid:      true,
						FloatValue: 7.091,
					},
				},
				{
					AttributeId: schema.LengthAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:     AttrAttributeType_INTEGER,
						Valid:    true,
						IntValue: 6,
					},
				},
				{
					AttributeId: schema.FrontSuspensionTypeAttr,
					ItemId:      itemID,
					Value: &AttrValueValue{
						Type:        AttrAttributeType_STRING,
						Valid:       true,
						StringValue: "suspension test",
					},
				},
			},
		},
	)
	require.NoError(t, err)

	res, err := client.GetTopSpecsContributions(ctx, &TopSpecsContributionsRequest{Language: "ru"})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetItems())
}

func TestVehiclesOnEnginesMerge(t *testing.T) {
	t.Parallel()

	client := NewItemsClient(conn)

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	itemID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Name:       "5 Series",
		Body:       "",
		IsGroup:    true,
	})

	childID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Name:       "5 Series",
		Body:       "E31",
		IsGroup:    true,
	})

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemParent{
			ItemId: childID, ParentId: itemID, Catname: "vehicle1",
		},
	)
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	engineName := fmt.Sprintf("Peugeot-%d-Engine", randomInt)
	r2, err := goquDB.Insert(schema.ItemTable).Rows(schema.ItemRow{
		Name:            engineName,
		IsGroup:         true,
		ItemTypeID:      schema.ItemTableItemTypeIDEngine,
		Catname:         sql.NullString{Valid: true, String: fmt.Sprintf("peugeot-%d-engine", randomInt)},
		Body:            "",
		ProducedExactly: false,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	engineID, err := r2.LastInsertId()
	require.NoError(t, err)

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetItemEngineRequest{
			ItemId:          itemID,
			EngineItemId:    engineID,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	res, err := client.Item(ctx, &ItemRequest{
		Id: itemID,
	})
	require.NoError(t, err)
	require.Equal(t, engineID, res.GetEngineItemId())

	_, err = client.SetItemEngine(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetItemEngineRequest{
			ItemId:          childID,
			EngineItemId:    engineID,
			EngineInherited: false,
		},
	)
	require.NoError(t, err)

	rep, err := cnt.ItemsRepository()
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, itemID)
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, childID)
	require.NoError(t, err)

	_, err = rep.RebuildCache(ctx, engineID)
	require.NoError(t, err)

	res, err = client.Item(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemRequest{
			Id: engineID,
			Fields: &ItemFields{
				EngineVehicles:      &ItemsRequest{},
				EngineVehiclesCount: true,
			},
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Len(t, res.GetEngineVehicles(), 1)
	require.EqualValues(t, 2, res.GetEngineVehiclesCount())

	for i := range 10 {
		childChildID := createItem(t, goquDB, schema.ItemRow{
			ItemTypeID: schema.ItemTableItemTypeIDVehicle,
			Name:       "5 Series",
			Body:       fmt.Sprintf("E31-%d", i),
			IsGroup:    false,
		})

		_, err = client.CreateItemParent(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
			&ItemParent{
				ItemId: childChildID, ParentId: childID, Catname: "vehicle1",
			},
		)
		require.NoError(t, err)

		_, err = rep.RebuildCache(ctx, childChildID)
		require.NoError(t, err)

		_, err = client.SetItemEngine(
			metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
			&SetItemEngineRequest{
				ItemId:          childChildID,
				EngineItemId:    engineID,
				EngineInherited: false,
			},
		)
		require.NoError(t, err)
	}

	res, err = client.Item(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemRequest{
			Id: engineID,
			Fields: &ItemFields{
				EngineVehicles:      &ItemsRequest{},
				EngineVehiclesCount: true,
			},
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Len(t, res.GetEngineVehicles(), 1)
	require.EqualValues(t, 12, res.GetEngineVehiclesCount())
}

func TestBrandSectionLanguageName(t *testing.T) {
	t.Parallel()

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	cfg := config.LoadConfig(".")
	client := NewItemsClient(conn)

	// admin
	kc := cnt.Keycloak()
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	brandID := createItem(t, goquDB, schema.ItemRow{
		Name:       fmt.Sprintf("brand-%d", random.Int()),
		IsGroup:    true,
		ItemTypeID: schema.ItemTableItemTypeIDBrand,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("brand-%d", randomInt)},
	})

	vehicleID := createItem(t, goquDB, schema.ItemRow{
		Name:       fmt.Sprintf("vehicle-%d", random.Int()),
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
		Catname:    sql.NullString{Valid: true, String: fmt.Sprintf("vehicle-%d", randomInt)},
	})

	_, err = client.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: brandID, Catname: "vehicle1",
		},
	)
	require.NoError(t, err)

	_, err = client.SetItemParentLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParentLanguage{
			ItemId:   vehicleID,
			ParentId: brandID,
			Language: "ru",
			Name:     "Azazaza",
		},
	)
	require.NoError(t, err)

	_, err = client.SetItemParentLanguage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&ItemParentLanguage{
			ItemId:   vehicleID,
			ParentId: brandID,
			Language: "en",
			Name:     "Custom name",
		},
	)
	require.NoError(t, err)

	_, err = client.CreateItemVehicleType(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&APIItemVehicleType{
			ItemId:        vehicleID,
			VehicleTypeId: 19,
		},
	)
	require.NoError(t, err)

	res, err := client.GetBrandSections(ctx, &GetBrandSectionsRequest{Language: "en", ItemId: brandID})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetSections()[2].GetGroups())
	require.Equal(t, "Custom name", res.GetSections()[2].GetGroups()[0].GetName())
}

func TestAlpha(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := config.LoadConfig(".")
	client := NewItemsClient(conn)

	// admin
	kc := cnt.Keycloak()
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	_, err = client.GetAlpha(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}
