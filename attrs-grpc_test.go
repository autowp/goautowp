package goautowp

import (
	"context"
	"database/sql"
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

const (
	floatAttributeID        int64 = 11
	intAttributeID          int64 = 1
	stringAttributeID       int64 = 8
	boolAttributeID         int64 = 53
	listAttributeID         int64 = 20
	treeAttributeID         int64 = 23
	treeMultipleAttributeID int64 = 98
)

func TestGetUnits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetUnits(
		ctx,
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetZones(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetZones(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func TestGetAttributeTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetAttributeTypes(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func TestGetAttributes(t *testing.T) {
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

	client := NewAttrsClient(conn)

	_, err = client.GetAttributes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrAttributesRequest{ZoneId: 1},
	)
	require.NoError(t, err)
}

func TestGetZoneAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetZoneAttributes(ctx, &AttrZoneAttributesRequest{ZoneId: 1})
	require.NoError(t, err)
}

func TestGetListOptions(t *testing.T) {
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

	client := NewAttrsClient(conn)

	_, err = client.GetListOptions(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrListOptionsRequest{AttributeId: 1},
	)
	require.NoError(t, err)
}

func requireUserValues(t *testing.T, userValues *AttrUserValuesResponse) {
	t.Helper()

	var (
		floatFound        bool
		intFound          bool
		stringFound       bool
		boolFound         bool
		listFound         bool
		treeFound         bool
		treeMultipleFound bool
	)

	require.NotEmpty(t, userValues.GetItems())

	for _, val := range userValues.GetItems() { //nolint:dupl
		require.True(t, val.GetValue().GetValid())
		require.False(t, val.GetValue().GetIsEmpty())

		switch val.GetAttributeId() {
		case floatAttributeID:
			floatFound = true

			require.InDelta(t, 7.091, val.GetValue().GetFloatValue(), 0.01)
			require.Equal(t, "7.1", val.GetValueText())
		case intAttributeID:
			intFound = true

			require.Equal(t, int32(6), val.GetValue().GetIntValue())
		case stringAttributeID:
			stringFound = true

			require.Equal(t, "test", val.GetValue().GetStringValue())
		case boolAttributeID:
			boolFound = true

			require.True(t, val.GetValue().GetBoolValue())
		case listAttributeID:
			listFound = true

			require.Equal(t, []int64{1}, val.GetValue().GetListValue())
		case treeAttributeID:
			treeFound = true

			require.Equal(t, []int64{25}, val.GetValue().GetListValue())
		case treeMultipleAttributeID:
			treeMultipleFound = true

			require.Equal(t, []int64{28, 29}, val.GetValue().GetListValue())
		}
	}

	require.True(t, floatFound)
	require.True(t, intFound)
	require.True(t, stringFound)
	require.True(t, boolFound)
	require.True(t, listFound)
	require.True(t, treeFound)
	require.True(t, treeMultipleFound)
}

func requireValues(t *testing.T, values *AttrValuesResponse) {
	t.Helper()

	require.NotEmpty(t, values.GetItems())

	var (
		floatFound        bool
		intFound          bool
		stringFound       bool
		boolFound         bool
		listFound         bool
		treeFound         bool
		treeMultipleFound bool
	)

	for _, val := range values.GetItems() { //nolint:dupl
		require.True(t, val.GetValue().GetValid())
		require.False(t, val.GetValue().GetIsEmpty())

		switch val.GetAttributeId() {
		case floatAttributeID:
			floatFound = true

			require.InDelta(t, 7.091, val.GetValue().GetFloatValue(), 0.01)
			require.Equal(t, "7.1", val.GetValueText())
		case intAttributeID:
			intFound = true

			require.Equal(t, int32(6), val.GetValue().GetIntValue())
		case stringAttributeID:
			stringFound = true

			require.Equal(t, "test", val.GetValue().GetStringValue())
		case boolAttributeID:
			boolFound = true

			require.True(t, val.GetValue().GetBoolValue())
		case listAttributeID:
			listFound = true

			require.Equal(t, []int64{1}, val.GetValue().GetListValue())
		case treeAttributeID:
			treeFound = true

			require.Equal(t, []int64{25}, val.GetValue().GetListValue())
		case treeMultipleAttributeID:
			treeMultipleFound = true

			require.Equal(t, []int64{28, 29}, val.GetValue().GetListValue())
		}
	}

	require.True(t, floatFound)
	require.True(t, intFound)
	require.True(t, stringFound)
	require.True(t, boolFound)
	require.True(t, listFound)
	require.True(t, treeFound)
	require.True(t, treeMultipleFound)
}

func insertAttrsUserValue(t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64) {
	t.Helper()

	ctx := context.Background()

	_, err := goquDB.Insert(schema.AttrsUserValuesTable).Rows(schema.AttrsUserValueRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UserID:      userID,
		UpdateDate:  time.Now(),
		AddDate:     time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func insertAttrsFloatValue(
	t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64, value sql.NullFloat64,
) {
	t.Helper()

	ctx := context.Background()

	insertAttrsUserValue(t, goquDB, attributeID, itemID, userID)

	_, err := goquDB.Insert(schema.AttrsUserValuesFloatTable).Rows(schema.AttrsUserValuesFloatRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UserID:      userID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesFloatTable).Rows(schema.AttrsValuesFloatRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func insertAttrsIntValue(
	t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64, value sql.NullInt32,
) {
	t.Helper()

	ctx := context.Background()

	insertAttrsUserValue(t, goquDB, attributeID, itemID, userID)

	_, err := goquDB.Insert(schema.AttrsUserValuesIntTable).Rows(schema.AttrsUserValuesIntRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UserID:      userID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func insertAttrsStringValue(
	t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64, value sql.NullString,
) {
	t.Helper()

	ctx := context.Background()

	insertAttrsUserValue(t, goquDB, attributeID, itemID, userID)

	_, err := goquDB.Insert(schema.AttrsUserValuesStringTable).Rows(schema.AttrsUserValuesStringRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UserID:      userID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesStringTable).Rows(schema.AttrsValuesStringRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func insertAttrsBoolValue(
	t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64, value sql.NullInt32,
) {
	t.Helper()

	ctx := context.Background()

	insertAttrsUserValue(t, goquDB, attributeID, itemID, userID)

	_, err := goquDB.Insert(schema.AttrsUserValuesIntTable).Rows(schema.AttrsUserValuesIntRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		UserID:      userID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: attributeID,
		ItemID:      itemID,
		Value:       value,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)
}

func insertAttrsListValue(
	t *testing.T, goquDB *goqu.Database, attributeID int64, itemID int64, userID int64, values []sql.NullInt64,
) {
	t.Helper()

	ctx := context.Background()

	insertAttrsUserValue(t, goquDB, attributeID, itemID, userID)

	for idx, val := range values {
		_, err := goquDB.Insert(schema.AttrsUserValuesListTable).Rows(schema.AttrsUserValuesListRow{
			AttributeID: attributeID,
			ItemID:      itemID,
			UserID:      userID,
			Value:       val,
			Ordering:    int64(idx + 1),
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
			AttributeID: attributeID,
			ItemID:      itemID,
			Value:       val,
			Ordering:    int64(idx + 1),
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)
	}
}

func insertAttrsTestData(t *testing.T, goquDB *goqu.Database, itemID int64, userID int64) {
	t.Helper()

	// insert float value
	insertAttrsFloatValue(t, goquDB, floatAttributeID, itemID, userID, sql.NullFloat64{Float64: 7.091, Valid: true})

	// insert int value
	insertAttrsIntValue(t, goquDB, intAttributeID, itemID, userID, sql.NullInt32{Int32: 6, Valid: true})

	// insert string value
	insertAttrsStringValue(t, goquDB, stringAttributeID, itemID, userID, sql.NullString{String: "test", Valid: true})

	// insert bool value
	insertAttrsBoolValue(t, goquDB, boolAttributeID, itemID, userID, sql.NullInt32{Int32: 1, Valid: true})

	// insert list value
	insertAttrsListValue(t, goquDB, listAttributeID, itemID, userID, []sql.NullInt64{{Int64: 1, Valid: true}})

	// insert tree value
	insertAttrsListValue(t, goquDB, treeAttributeID, itemID, userID, []sql.NullInt64{{Int64: 25, Valid: true}})

	// insert tree multiple value
	insertAttrsListValue(t, goquDB, treeMultipleAttributeID, itemID, userID,
		[]sql.NullInt64{{Int64: 28, Valid: true}, {Int64: 29, Valid: true}})
}

func TestGetValues(t *testing.T) {
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

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	usersClient := NewUsersClient(conn)

	user, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	client := NewAttrsClient(conn)

	itemID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
	})

	values, err := client.GetValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrValuesRequest{
			ItemId:   itemID,
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Empty(t, values.GetItems())

	insertAttrsTestData(t, goquDB, itemID, user.GetId())

	// check values
	values, err = client.GetValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrValuesRequest{
			ItemId:   itemID,
			Language: "en",
		},
	)
	require.NoError(t, err)
	requireValues(t, values)

	// check user values by item_id
	userValues, err := client.GetUserValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrUserValuesRequest{
			ItemId:   itemID,
			Language: "en",
			Fields: &AttrUserValuesFields{
				ValueText: true,
			},
		},
	)
	require.NoError(t, err)
	requireUserValues(t, userValues)
}

func TestGetEmptyValues(t *testing.T) {
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

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	usersClient := NewUsersClient(conn)

	user, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	client := NewAttrsClient(conn)

	itemID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
	})

	const (
		floatEmptyAttributeID  int64 = 28
		intEmptyAttributeID    int64 = 2
		stringEmptyAttributeID int64 = 9
		boolEmptyAttributeID   int64 = 77
		listEmptyAttributeID   int64 = 21
		treeEmptyAttributeID   int64 = 41
	)

	values, err := client.GetValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrValuesRequest{
			ItemId:   itemID,
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.Empty(t, values.GetItems())

	userID := user.GetId()

	// insert float empty value
	insertAttrsFloatValue(t, goquDB, floatEmptyAttributeID, itemID, userID, sql.NullFloat64{Valid: false})

	// insert int empty value
	insertAttrsIntValue(t, goquDB, intEmptyAttributeID, itemID, userID, sql.NullInt32{Valid: false})

	// insert string empty value
	insertAttrsStringValue(t, goquDB, stringEmptyAttributeID, itemID, userID, sql.NullString{Valid: false})

	// insert bool empty value
	insertAttrsBoolValue(t, goquDB, boolEmptyAttributeID, itemID, userID, sql.NullInt32{Valid: false})

	// insert list empty value
	insertAttrsListValue(t, goquDB, listEmptyAttributeID, itemID, userID, []sql.NullInt64{{Valid: false}})

	// insert tree empty value
	insertAttrsListValue(t, goquDB, treeEmptyAttributeID, itemID, userID, []sql.NullInt64{{Valid: false}})

	// insert tree multiple value
	insertAttrsListValue(t, goquDB, treeMultipleAttributeID, itemID, userID,
		[]sql.NullInt64{{Int64: 28, Valid: true}, {Int64: 29, Valid: true}})

	// check values
	values, err = client.GetValues(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrValuesRequest{
			ItemId:   itemID,
			Language: "en",
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, values.GetItems())

	var (
		floatEmptyFound  bool
		intEmptyFound    bool
		stringEmptyFound bool
		boolEmptyFound   bool
		listEmptyFound   bool
		treeEmptyFound   bool
	)

	for _, val := range values.GetItems() {
		switch val.GetAttributeId() {
		case floatEmptyAttributeID:
			floatEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())
		case intEmptyAttributeID:
			intEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())

		case stringEmptyAttributeID:
			stringEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())
		case boolEmptyAttributeID:
			boolEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())
		case listEmptyAttributeID:
			listEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())
		case treeEmptyAttributeID:
			treeEmptyFound = true

			require.True(t, val.GetValue().GetIsEmpty())
			require.True(t, val.GetValue().GetValid())
		}
	}

	require.True(t, floatEmptyFound)
	require.True(t, intEmptyFound)
	require.True(t, stringEmptyFound)
	require.True(t, boolEmptyFound)
	require.True(t, listEmptyFound)
	require.True(t, treeEmptyFound)
}
