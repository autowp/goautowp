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
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getPictureID(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint: gosec
	identity := "p" + strconv.Itoa(random.Int())[:6]

	res, err := db.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   "accepted",
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  1,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	return pictureID
}

func TestView(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewPicturesClient(conn)

	_, err = client.View(ctx, &PicturesViewRequest{PictureId: getPictureID(ctx, t, goquDB)})
	require.NoError(t, err)
}

func TestVote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.Vote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PicturesVoteRequest{PictureId: getPictureID(ctx, t, goquDB), Value: 1},
	)
	require.NoError(t, err)
}

func TestModerVoteTemplate(t *testing.T) {
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

	client := NewPicturesClient(conn)

	template, err := client.CreateModerVoteTemplate(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ModerVoteTemplate{Message: "test", Vote: 1},
	)
	require.NoError(t, err)

	_, err = client.GetModerVoteTemplates(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)

	_, err = client.DeleteModerVoteTemplate(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeleteModerVoteTemplateRequest{Id: template.GetId()},
	)
	require.NoError(t, err)
}

func TestModerVote(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: 1, Save: true},
	)
	require.NoError(t, err)

	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: 1, Save: true},
	)
	require.NoError(t, err)

	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: -1, Save: true},
	)
	require.NoError(t, err)

	_, err = client.DeleteModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeleteModerVoteRequest{PictureId: pictureID},
	)
	require.NoError(t, err)

	secondUserID, _ := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	var picStatus schema.PictureStatus

	// test unaccepting
	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableStatusColName:             schema.PictureStatusAccepted,
		schema.PictureTableChangeStatusUserIDColName: secondUserID,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: -1, Save: false},
	)
	require.NoError(t, err)

	success, err := goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusInbox, picStatus)

	// test restore from removing
	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableStatusColName:             schema.PictureStatusRemoving,
		schema.PictureTableChangeStatusUserIDColName: secondUserID,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: 1, Save: false},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusInbox, picStatus)

	_, err = client.DeleteModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeleteModerVoteRequest{PictureId: pictureID},
	)
	require.NoError(t, err)
}

func TestUserSummary(t *testing.T) {
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

	client := NewPicturesClient(conn)

	_, err = client.GetUserSummary(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestFlopNormalizeAndRepair(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableStatusColName: schema.PictureStatusInbox,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.Flop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	_, err = client.Normalize(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	_, err = client.Repair(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)
}

func TestDeleteSimilar(t *testing.T) {
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

	client := NewPicturesClient(conn)

	_, err = client.DeleteSimilar(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeleteSimilarRequest{Id: 1, SimilarPictureId: 2},
	)
	require.NoError(t, err)
}

func TestPictureItemAreaAndPerspective(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	res, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableNameColName:            fmt.Sprintf("vehicle-%d", random.Int()),
		schema.ItemTableIsGroupColName:         0,
		schema.ItemTableItemTypeIDColName:      ItemType_ITEM_TYPE_VEHICLE,
		schema.ItemTableCatnameColName:         fmt.Sprintf("vehicle-%d", random.Int()),
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := res.LastInsertId()
	require.NoError(t, err)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    itemID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureItemArea(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureItemAreaRequest{
			PictureId:  pictureID,
			ItemId:     itemID,
			Type:       PictureItemType_PICTURE_ITEM_CONTENT,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  10,
			CropHeight: 10,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureItemArea(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureItemAreaRequest{
			PictureId:  pictureID,
			ItemId:     itemID,
			Type:       PictureItemType_PICTURE_ITEM_CONTENT,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  0,
			CropHeight: 10,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureItemPerspective(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureItemPerspectiveRequest{
			PictureId:     pictureID,
			ItemId:        itemID,
			Type:          PictureItemType_PICTURE_ITEM_CONTENT,
			PerspectiveId: 1,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureItemPerspective(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureItemPerspectiveRequest{
			PictureId:     pictureID,
			ItemId:        itemID,
			Type:          PictureItemType_PICTURE_ITEM_CONTENT,
			PerspectiveId: 0,
		},
	)
	require.NoError(t, err)
}

func TestPictureItemSetPictureItemItemID(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	res, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableNameColName:            fmt.Sprintf("vehicle-1-%d", random.Int()),
		schema.ItemTableIsGroupColName:         0,
		schema.ItemTableItemTypeIDColName:      ItemType_ITEM_TYPE_VEHICLE,
		schema.ItemTableCatnameColName:         fmt.Sprintf("vehicle-1-%d", random.Int()),
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID1, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableNameColName:            fmt.Sprintf("vehicle-2-%d", random.Int()),
		schema.ItemTableIsGroupColName:         0,
		schema.ItemTableItemTypeIDColName:      ItemType_ITEM_TYPE_VEHICLE,
		schema.ItemTableCatnameColName:         fmt.Sprintf("vehicle-2-%d", random.Int()),
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID2, err := res.LastInsertId()
	require.NoError(t, err)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    itemID1,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureItemItemID(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureItemItemIDRequest{
			PictureId: pictureID,
			ItemId:    itemID1,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
			NewItemId: itemID2,
		},
	)
	require.NoError(t, err)

	_, err = client.DeletePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeletePictureItemRequest{
			PictureId: pictureID,
			ItemId:    itemID1,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.Error(t, err)

	_, err = client.DeletePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&DeletePictureItemRequest{
			PictureId: pictureID,
			ItemId:    itemID2,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)
}

func TestPictureCrop(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  0,
			CropHeight: 10,
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  10,
			CropHeight: 10,
		},
	)
	require.NoError(t, err)

	fmtImg, err := imageStorage.FormattedImage(ctx, imageID, "picture-gallery")
	require.NoError(t, err)

	require.Equal(t, 10, fmtImg.Width())
	require.Equal(t, 10, fmtImg.Height())
}

func TestPictureCropByOneAxis(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  200,
			CropHeight: 130,
		},
	)
	require.NoError(t, err)

	crop, err := imageStorage.ImageCrop(ctx, imageID)
	require.NoError(t, err)
	require.Equal(t, 0, crop.Left)
	require.Equal(t, 0, crop.Top)
	require.Equal(t, 200, crop.Width)
	require.Equal(t, 130, crop.Height)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  180,
			CropHeight: 143,
		},
	)
	require.NoError(t, err)

	crop, err = imageStorage.ImageCrop(ctx, imageID)
	require.NoError(t, err)
	require.Equal(t, 0, crop.Left)
	require.Equal(t, 0, crop.Top)
	require.Equal(t, 180, crop.Width)
	require.Equal(t, 143, crop.Height)
}

func TestInvalidPictureCrop(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  202,
			CropHeight: 140,
		},
	)
	require.NoError(t, err)

	crop, err := imageStorage.ImageCrop(ctx, imageID)
	require.NoError(t, err)
	require.Equal(t, 0, crop.Left)
	require.Equal(t, 0, crop.Top)
	require.Equal(t, 200, crop.Width)
	require.Equal(t, 140, crop.Height)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   0,
			CropTop:    0,
			CropWidth:  190,
			CropHeight: 145,
		},
	)
	require.NoError(t, err)

	crop, err = imageStorage.ImageCrop(ctx, imageID)
	require.NoError(t, err)
	require.Equal(t, 0, crop.Left)
	require.Equal(t, 0, crop.Top)
	require.Equal(t, 190, crop.Width)
	require.Equal(t, 143, crop.Height)

	_, err = client.SetPictureCrop(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCropRequest{
			PictureId:  pictureID,
			CropLeft:   30,
			CropTop:    0,
			CropWidth:  190,
			CropHeight: 143,
		},
	)
	require.NoError(t, err)

	crop, err = imageStorage.ImageCrop(ctx, imageID)
	require.NoError(t, err)
	require.Equal(t, 30, crop.Left)
	require.Equal(t, 0, crop.Top)
	require.Equal(t, 170, crop.Width)
	require.Equal(t, 143, crop.Height)
}

func TestClearReplacePicture(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")
	replacePictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableReplacePictureIDColName: replacePictureID,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.ClearReplacePicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	var value sql.NullInt64

	success, err := goquDB.Select(schema.PictureTableReplacePictureIDCol).From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &value)
	require.NoError(t, err)
	require.True(t, success)
	require.False(t, value.Valid)
}

func TestSetPicturePoint(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	textStorageRepository := textstorage.New(goquDB)
	repo := pictures.NewRepository(goquDB, imageStorage, textStorageRepository)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
			Point: &latlng.LatLng{
				Latitude:  0,
				Longitude: 0,
			},
		},
	)
	require.NoError(t, err)

	pic, err := repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.Nil(t, pic.Point)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
			Point: &latlng.LatLng{
				Latitude:  10,
				Longitude: 0,
			},
		},
	)
	require.NoError(t, err)

	pic, err = repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.True(t, pic.Point.Valid)
	require.InDelta(t, float64(10), pic.Point.Point.Lat(), 0.001)
	require.InDelta(t, float64(0), pic.Point.Point.Lng(), 0.001)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
			Point: &latlng.LatLng{
				Latitude:  0,
				Longitude: 10,
			},
		},
	)
	require.NoError(t, err)

	pic, err = repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.True(t, pic.Point.Valid)
	require.InDelta(t, float64(0), pic.Point.Point.Lat(), 0.001)
	require.InDelta(t, float64(10), pic.Point.Point.Lng(), 0.001)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
			Point: &latlng.LatLng{
				Latitude:  -10,
				Longitude: 10,
			},
		},
	)
	require.NoError(t, err)

	pic, err = repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.True(t, pic.Point.Valid)
	require.InDelta(t, float64(-10), pic.Point.Point.Lat(), 0.001)
	require.InDelta(t, float64(10), pic.Point.Point.Lng(), 0.001)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
		},
	)
	require.NoError(t, err)

	pic, err = repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.Nil(t, pic.Point)
}

func TestUpdatePicture(t *testing.T) {
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

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.UpdatePicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdatePictureRequest{
			Id:   pictureID,
			Name: "Foo",
			TakenDate: &date.Date{
				Year:  2020,
				Month: 2,
				Day:   1,
			},
		},
	)
	require.NoError(t, err)

	var pic schema.PictureRow

	success, err := goquDB.Select(
		schema.PictureTableTakenYearColName, schema.PictureTableTakenMonthColName, schema.PictureTableTakenDayColName,
	).
		From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(pictureID)).ScanStructContext(ctx, &pic)
	require.NoError(t, err)
	require.True(t, success)

	require.Equal(t, int16(2020), pic.TakenYear.Int16)
	require.True(t, pic.TakenYear.Valid)
	require.Equal(t, byte(2), pic.TakenMonth.Byte)
	require.True(t, pic.TakenMonth.Valid)
	require.Equal(t, byte(1), pic.TakenDay.Byte)
	require.True(t, pic.TakenDay.Valid)

	_, err = client.UpdatePicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdatePictureRequest{
			Id:   pictureID,
			Name: "Foo",
			TakenDate: &date.Date{
				Year:  2020,
				Month: 2,
			},
		},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(
		schema.PictureTableTakenYearColName, schema.PictureTableTakenMonthColName, schema.PictureTableTakenDayColName,
	).
		From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(pictureID)).ScanStructContext(ctx, &pic)
	require.NoError(t, err)
	require.True(t, success)

	require.Equal(t, int16(2020), pic.TakenYear.Int16)
	require.True(t, pic.TakenYear.Valid)
	require.Equal(t, byte(2), pic.TakenMonth.Byte)
	require.True(t, pic.TakenMonth.Valid)
	require.False(t, pic.TakenDay.Valid)
}

func TestSetPictureCopyrights(t *testing.T) {
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

	textStorageRepository := textstorage.New(goquDB)

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	repo := pictures.NewRepository(goquDB, imageStorage, textStorageRepository)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")
	pictureID2, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.SetPictureCopyrights(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCopyrightsRequest{
			Id:         pictureID,
			Copyrights: "First",
		},
	)
	require.NoError(t, err)

	pic, err := repo.Picture(ctx, pictureID)
	require.NoError(t, err)
	require.True(t, pic.CopyrightsTextID.Valid)
	require.NotEmpty(t, pic.CopyrightsTextID.Int32)

	text, err := textStorageRepository.Text(ctx, pic.CopyrightsTextID.Int32)
	require.NoError(t, err)
	require.Equal(t, "First", text)

	_, err = client.SetPictureCopyrights(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCopyrightsRequest{
			Id:         pictureID,
			Copyrights: "Second",
		},
	)
	require.NoError(t, err)

	_, err = client.SetPictureCopyrights(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureCopyrightsRequest{
			Id:         pictureID2,
			Copyrights: "Third",
		},
	)
	require.NoError(t, err)

	text, err = textStorageRepository.Text(ctx, pic.CopyrightsTextID.Int32)
	require.NoError(t, err)
	require.Equal(t, "Second", text)

	pic2, err := repo.Picture(ctx, pictureID2)
	require.NoError(t, err)
	require.True(t, pic2.CopyrightsTextID.Valid)
	require.NotEmpty(t, pic2.CopyrightsTextID.Int32)
	require.NotEqual(t, pic.CopyrightsTextID.Int32, pic2.CopyrightsTextID.Int32)

	text, err = textStorageRepository.Text(ctx, pic2.CopyrightsTextID.Int32)
	require.NoError(t, err)
	require.Equal(t, "Third", text)
}
