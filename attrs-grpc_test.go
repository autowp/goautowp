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

	client := NewAttrsClient(conn)

	itemID := createItem(t, goquDB, schema.ItemRow{
		ItemTypeID: schema.ItemTableItemTypeIDVehicle,
	})

	const (
		floatAttributeID        int64 = 11
		intAttributeID          int64 = 1
		stringAttributeID       int64 = 8
		boolAttributeID         int64 = 53
		listAttributeID         int64 = 20
		treeAttributeID         int64 = 23
		treeMultipleAttributeID int64 = 98
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

	// insert float value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: floatAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesFloatTable).Rows(schema.AttrsValuesFloatRow{
		AttributeID: floatAttributeID,
		ItemID:      itemID,
		Value:       sql.NullFloat64{Float64: 7.0, Valid: true},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert int value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: intAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: intAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt32{Int32: 6, Valid: true},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert string value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: stringAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesStringTable).Rows(schema.AttrsValuesStringRow{
		AttributeID: stringAttributeID,
		ItemID:      itemID,
		Value:       sql.NullString{String: "test", Valid: true},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert bool value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: boolAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: boolAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt32{Int32: 1, Valid: true},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert list value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: listAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
		AttributeID: listAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Int64: 1, Valid: true},
		Ordering:    1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert tree value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: treeAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
		AttributeID: treeAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Int64: 25, Valid: true},
		Ordering:    1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert tree multiple value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: treeMultipleAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
		AttributeID: treeMultipleAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Int64: 28, Valid: true},
		Ordering:    1,
	}, schema.AttrsValuesListRow{
		AttributeID: treeMultipleAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Int64: 29, Valid: true},
		Ordering:    2,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

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
		floatFound        bool
		intFound          bool
		stringFound       bool
		boolFound         bool
		listFound         bool
		treeFound         bool
		treeMultipleFound bool
	)

	for _, val := range values.GetItems() {
		require.True(t, val.GetValue().GetValid())
		require.False(t, val.GetValue().GetIsEmpty())

		switch val.GetAttributeId() {
		case floatAttributeID:
			floatFound = true

			require.InDelta(t, 7.0, val.GetValue().GetFloatValue(), 0.01)
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

	// insert float empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: floatEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesFloatTable).Rows(schema.AttrsValuesFloatRow{
		AttributeID: floatEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullFloat64{Valid: false},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert int empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: intEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: intEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt32{Valid: false},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert string empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: stringEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesStringTable).Rows(schema.AttrsValuesStringRow{
		AttributeID: stringEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullString{Valid: false},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert bool empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: boolEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesIntTable).Rows(schema.AttrsValuesIntRow{
		AttributeID: boolEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt32{Valid: false},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert list empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: listEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
		AttributeID: listEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Valid: false},
		Ordering:    1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// insert tree empty value
	_, err = goquDB.Insert(schema.AttrsValuesTable).Rows(schema.AttrsValueRow{
		AttributeID: treeEmptyAttributeID,
		ItemID:      itemID,
		UpdateDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.AttrsValuesListTable).Rows(schema.AttrsValuesListRow{
		AttributeID: treeEmptyAttributeID,
		ItemID:      itemID,
		Value:       sql.NullInt64{Valid: false},
		Ordering:    1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

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
