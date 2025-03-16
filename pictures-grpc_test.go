package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func getPictureID(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint: gosec
	identity := "p" + strconv.Itoa(random.Int())[:6]

	res, err := db.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   schema.PictureStatusAccepted,
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

	ctx := t.Context()

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	client := NewPicturesClient(conn)

	_, err = client.View(ctx, &PicturesViewRequest{PictureId: getPictureID(ctx, t, goquDB)})
	require.NoError(t, err)
}

func TestVote(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableStatusColName: schema.PictureStatusInbox,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	itemID := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	itemID1 := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("vehicle-1-%d", random.Int()),
		IsGroup:    false,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	itemID2 := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("vehicle-2-%d", random.Int()),
		IsGroup:    false,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, imageID := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	replacePictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableReplacePictureIDColName: replacePictureID,
	}).Where(schema.PictureTableIDCol.Eq(pictureID)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	pic, err := client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.Nil(t, pic.GetPoint())

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

	pic, err = client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.NotNil(t, pic.GetPoint())
	require.InDelta(t, float64(10), pic.GetPoint().GetLatitude(), 0.001)
	require.InDelta(t, float64(0), pic.GetPoint().GetLongitude(), 0.001)

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

	pic, err = client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.NotNil(t, pic.GetPoint())
	require.InDelta(t, float64(0), pic.GetPoint().GetLatitude(), 0.001)
	require.InDelta(t, float64(10), pic.GetPoint().GetLongitude(), 0.001)

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

	pic, err = client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.NotNil(t, pic.GetPoint())
	require.InDelta(t, float64(-10), pic.GetPoint().GetLatitude(), 0.001)
	require.InDelta(t, float64(10), pic.GetPoint().GetLongitude(), 0.001)

	_, err = client.SetPicturePoint(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPicturePointRequest{
			PictureId: pictureID,
		},
	)
	require.NoError(t, err)

	pic, err = client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.Nil(t, pic.GetPoint())
}

func TestUpdatePicture(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	textStorageRepository := textstorage.New(goquDB)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	pictureID2, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
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

	pic, err := client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID}})
	require.NoError(t, err)
	require.NotZero(t, pic.GetCopyrightsTextId())
	require.NotEmpty(t, pic.GetCopyrightsTextId())

	text, err := textStorageRepository.Text(ctx, pic.GetCopyrightsTextId())
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

	text, err = textStorageRepository.Text(ctx, pic.GetCopyrightsTextId())
	require.NoError(t, err)
	require.Equal(t, "Second", text)

	pic2, err := client.GetPicture(ctx, &PicturesRequest{Options: &PictureListOptions{Id: pictureID2}})
	require.NoError(t, err)
	require.NotZero(t, pic2.GetCopyrightsTextId())
	require.NotEmpty(t, pic2.GetCopyrightsTextId())
	require.NotEqual(t, pic.GetCopyrightsTextId(), pic2.GetCopyrightsTextId())

	text, err = textStorageRepository.Text(ctx, pic2.GetCopyrightsTextId())
	require.NoError(t, err)
	require.Equal(t, "Third", text)
}

func TestSetPictureStatus(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	var picStatus schema.PictureStatus

	// accept
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_ACCEPTED,
		},
	)
	require.NoError(t, err)

	success, err := goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusAccepted, picStatus)

	// unaccept
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_INBOX,
		},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusInbox, picStatus)

	// remove without vote
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_REMOVING,
		},
	)
	require.ErrorContains(t, err, "PermissionDenied")

	// vote for remove
	_, err = client.UpdateModerVote(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&UpdateModerVoteRequest{PictureId: pictureID, Reason: "test", Vote: -1},
	)
	require.NoError(t, err)

	// remove with vote
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_REMOVING,
		},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusRemoving, picStatus)

	// restore
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_INBOX,
		},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusInbox, picStatus)

	// accept
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_ACCEPTED,
		},
	)
	require.Error(t, err)
}

func TestReplacePicture(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	pictureID2, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	// tester (me)
	usersClient := NewUsersClient(conn)
	tester, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	client := NewPicturesClient(conn)

	var picStatus schema.PictureStatus

	// accept
	_, err = client.SetPictureStatus(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&SetPictureStatusRequest{
			Id:     pictureID,
			Status: PictureStatus_PICTURE_STATUS_ACCEPTED,
		},
	)
	require.NoError(t, err)

	success, err := goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusAccepted, picStatus)

	// set replace
	_, err = goquDB.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableReplacePictureIDColName: pictureID,
		schema.PictureTableOwnerIDColName:          tester.GetId(),
	}).Where(schema.PictureTableIDCol.Eq(pictureID2)).Executor().ExecContext(ctx)
	require.NoError(t, err)

	// accept replace
	_, err = client.AcceptReplacePicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{
			Id: pictureID,
		},
	)
	require.Error(t, err)

	_, err = client.AcceptReplacePicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{
			Id: pictureID2,
		},
	)
	require.NoError(t, err)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusRemoving, picStatus)

	success, err = goquDB.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).Where(schema.PictureTableIDCol.Eq(pictureID2)).
		ScanValContext(ctx, &picStatus)
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, schema.PictureStatusAccepted, picStatus)
}

func TestGetPictures(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := NewPicturesClient(conn)

	_, err := client.GetPictures(ctx, &PicturesRequest{Fields: &PictureFields{
		NameText:    true,
		Image:       true,
		ThumbMedium: true,
	}, Limit: 100})
	require.ErrorContains(t, err, "PictureItem.ItemParentCacheAncestor.ItemID or OwnerID is required")

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	_, err = client.GetPictures(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PicturesRequest{Fields: &PictureFields{
			NameText:         true,
			NameHtml:         true,
			Image:            true,
			ThumbMedium:      true,
			Views:            true,
			Votes:            true,
			CommentsCount:    true,
			ModerVote:        true,
			PictureItem:      &PictureItemsRequest{},
			DfDistance:       &DfDistanceRequest{},
			ImageGalleryFull: true,
			Path:             &PicturePathRequest{},
			Thumb:            true,
		}, Limit: 100},
	)
	require.NoError(t, err)
}

func TestGetPictureWithPerspectivePrefix(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewPicturesClient(conn)
	itemsClient := NewItemsClient(conn)
	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	itemName := fmt.Sprintf("vehicle-%d", random.Int())

	itemID := createItem(t, conn, cnt, &APIItem{
		Name:            itemName,
		IsGroup:         false,
		ItemTypeId:      ItemType_ITEM_TYPE_VEHICLE,
		Produced:        &wrapperspb.Int32Value{Value: 777},
		ProducedExactly: true,
		BeginYear:       1999,
		EndYear:         2001,
		BeginModelYear:  2000,
		EndModelYear:    2001,
		SpecId:          schema.SpecIDWorldwide,
	})

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId:     pictureID,
			ItemId:        itemID,
			Type:          PictureItemType_PICTURE_ITEM_CONTENT,
			PerspectiveId: schema.PerspectiveIDUnderTheHood,
		},
	)
	require.NoError(t, err)

	picture, err := client.GetPicture(
		ctx,
		&PicturesRequest{
			Language: "en",
			Options:  &PictureListOptions{Id: pictureID},
			Fields:   &PictureFields{NameText: true, NameHtml: true},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, picture.GetNameText())
	require.NotEmpty(t, picture.GetNameHtml())

	item, err := itemsClient.Item(ctx, &ItemRequest{
		Id:       itemID,
		Fields:   &ItemFields{NameText: true, NameHtml: true},
		Language: "en",
	})
	require.NoError(t, err)
	require.NotEmpty(t, item.GetNameText())
	require.NotEmpty(t, item.GetNameHtml())

	require.Equal(t, picture.GetNameText(), "Under The Hood "+item.GetNameText())
	require.Equal(t, picture.GetNameHtml(), "Under The Hood "+item.GetNameHtml())

	picture, err = client.GetPicture(
		ctx,
		&PicturesRequest{
			Language: "ru",
			Options:  &PictureListOptions{Id: pictureID},
			Fields:   &PictureFields{NameText: true, NameHtml: true},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, picture.GetNameText())
	require.NotEmpty(t, picture.GetNameHtml())

	item, err = itemsClient.Item(ctx, &ItemRequest{
		Id:       itemID,
		Fields:   &ItemFields{NameText: true, NameHtml: true},
		Language: "ru",
	})
	require.NoError(t, err)
	require.NotEmpty(t, item.GetNameText())
	require.NotEmpty(t, item.GetNameHtml())

	require.Equal(t, picture.GetNameText(), "Под капотом "+item.GetNameText())
	require.Equal(t, picture.GetNameHtml(), "Под капотом "+item.GetNameHtml())
}

func TestGetPicturePath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewPicturesClient(conn)
	itemsClient := NewItemsClient(conn)
	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	randomInt := random.Int()

	// create brand
	brandName := fmt.Sprintf("Opel-%d", randomInt)
	brandID := createItem(t, conn, cnt, &APIItem{
		Name:       brandName,
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_BRAND,
		Catname:    fmt.Sprintf("opel-%d", randomInt),
		Body:       "",
	})

	itemName := fmt.Sprintf("vehicle-%d", randomInt)
	itemID := createItem(t, conn, cnt, &APIItem{
		Name:       itemName,
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	childName := fmt.Sprintf("child-%d", randomInt)
	childID := createItem(t, conn, cnt, &APIItem{
		Name:       childName,
		IsGroup:    false,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	_, err = itemsClient.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemParent{
			ItemId: itemID, ParentId: brandID, Type: ItemParentType_ITEM_TYPE_DEFAULT, Catname: "item",
		},
	)
	require.NoError(t, err)

	_, err = itemsClient.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemParent{
			ItemId: childID, ParentId: itemID, Type: ItemParentType_ITEM_TYPE_DEFAULT, Catname: "child",
		},
	)
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    childID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	picture, err := client.GetPicture(
		ctx,
		&PicturesRequest{
			Options: &PictureListOptions{Id: pictureID},
			Fields: &PictureFields{Path: &PicturePathRequest{
				ParentId: brandID,
			}},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, picture.GetPath())
	require.Equal(t, "child", picture.GetPath()[0].GetItem().GetParents()[0].GetCatname())
	require.Equal(t, "item", picture.GetPath()[0].GetItem().GetParents()[0].GetItem().GetParents()[0].GetCatname())
}

func TestGetPicturesOrders(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := NewPicturesClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	testCases := []PicturesRequest_Order{
		PicturesRequest_ORDER_NONE,
		PicturesRequest_ORDER_ADD_DATE_DESC,
		PicturesRequest_ORDER_ADD_DATE_ASC,
		PicturesRequest_ORDER_RESOLUTION_DESC,
		PicturesRequest_ORDER_RESOLUTION_ASC,
		PicturesRequest_ORDER_FILESIZE_DESC,
		PicturesRequest_ORDER_FILESIZE_ASC,
		PicturesRequest_ORDER_COMMENTS,
		PicturesRequest_ORDER_VIEWS,
		PicturesRequest_ORDER_MODER_VOTES,
		PicturesRequest_ORDER_DF_DISTANCE_SIMILARITY,
		PicturesRequest_ORDER_REMOVING_DATE,
		PicturesRequest_ORDER_LIKES,
		PicturesRequest_ORDER_DISLIKES,
		PicturesRequest_ORDER_ACCEPT_DATETIME_DESC,
		PicturesRequest_ORDER_PERSPECTIVES,
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase), func(t *testing.T) {
			t.Parallel()

			request := PicturesRequest{
				Fields: &PictureFields{
					NameText:      true,
					NameHtml:      true,
					Image:         true,
					ThumbMedium:   true,
					Views:         true,
					Votes:         true,
					CommentsCount: true,
					ModerVote:     true,
					PictureItem:   &PictureItemsRequest{},
				},
				Limit: 100,
				Order: testCase,
				Options: &PictureListOptions{
					PictureModerVote: &PictureModerVoteListOptions{},
					PictureItem:      &PictureItemListOptions{},
					DfDistance:       &DfDistanceListOptions{},
				},
			}

			_, err = client.GetPictures(
				metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
				&request,
			)
			require.NoError(t, err)
		})
	}
}

func TestGetPicturesFilters(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := NewPicturesClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	request := PicturesRequest{
		Fields: &PictureFields{
			NameText:      true,
			NameHtml:      true,
			Image:         true,
			ThumbMedium:   true,
			Views:         true,
			Votes:         true,
			CommentsCount: true,
			ModerVote:     true,
			PictureItem:   &PictureItemsRequest{},
		},
		Limit: 100,
		Options: &PictureListOptions{
			PictureModerVote: &PictureModerVoteListOptions{},
			PictureItem:      &PictureItemListOptions{},
			DfDistance:       &DfDistanceListOptions{},
			Statuses: []PictureStatus{
				PictureStatus_PICTURE_STATUS_ACCEPTED,
				PictureStatus_PICTURE_STATUS_INBOX,
				PictureStatus_PICTURE_STATUS_ACCEPTED,
			},
			OwnerId:               123,
			AcceptedInDays:        3,
			AddDate:               &date.Date{Year: 2025, Month: 1, Day: 1},
			AcceptDate:            &date.Date{Year: 2025, Month: 1, Day: 1},
			AddedFrom:             &date.Date{Year: 2025, Month: 1, Day: 1},
			CommentTopic:          &CommentTopicListOptions{MessagesGtZero: true},
			HasNoComments:         true,
			HasPoint:              true,
			HasNoPoint:            true,
			HasNoPictureItem:      true,
			ReplacePicture:        &PictureListOptions{},
			HasNoReplacePicture:   true,
			HasNoPictureModerVote: true,
			HasSpecialName:        true,
		},
	}

	_, err = client.GetPictures(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&request,
	)
	require.NoError(t, err)
}

func TestGetPictureIP(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	kc := cnt.Keycloak()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	res, err := goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusAccepted,
		IP:       util.IP(net.IPv4allrouter),
		AddDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	picture, err := client.GetPicture(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PicturesRequest{
			Options: &PictureListOptions{Id: pictureID},
		},
	)
	require.NoError(t, err)
	require.Equal(t, "224.0.0.2", picture.GetIp())
}

func TestInbox(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	kc := cnt.Keycloak()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	_, err = goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusInbox,
		IP:       util.IP(net.IPv4allrouter),
		AddDate:  time.Now(),
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.GetInbox(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&InboxRequest{
			Language: "en",
		},
	)
	require.NoError(t, err)

	_, err = client.GetInbox(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&InboxRequest{
			BrandId:  1,
			Language: "en",
			Date: &date.Date{
				Year:  2005,
				Month: 1,
				Day:   1,
			},
		},
	)
	require.NoError(t, err)

	_, err = client.GetInbox(
		ctx,
		&InboxRequest{
			BrandId:  1,
			Language: "en",
			Date: &date.Date{
				Year:  2005,
				Month: 1,
				Day:   1,
			},
		},
	)
	require.ErrorContains(t, err, "Unauthenticated")
}

func TestNewbox(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	ctx := t.Context()
	kc := cnt.Keycloak()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	_, err = goquDB.Insert(schema.PictureTable).Rows(schema.PictureRow{
		Identity: identity,
		Status:   schema.PictureStatusAccepted,
		IP:       util.IP(net.IPv4allrouter),
		AddDate:  time.Now(),
		AcceptDatetime: sql.NullTime{
			Valid: true,
			Time:  time.Now().AddDate(-1, 0, 0),
		},
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	_, err = client.GetNewbox(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&NewboxRequest{
			Language: "en",
		},
	)
	require.NoError(t, err)
}

func TestInboxCount(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	ctx := t.Context()
	kc := cnt.Keycloak()

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)

	res, err := client.GetPicturesPaginator(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PicturesRequest{
			Options: &PictureListOptions{
				Status: PictureStatus_PICTURE_STATUS_INBOX,
			},
			Paginator: true,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestCorrectFileNamesVote(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := config.LoadConfig(".")

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewPicturesClient(conn)
	itemsClient := NewItemsClient(conn)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	randomInt := random.Int()

	vehicleName := fmt.Sprintf("Toyota %d Corolla", randomInt)
	vehicleID := createItem(t, conn, cnt, &APIItem{
		Name:       vehicleName,
		IsGroup:    false,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    vehicleID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	_, err = client.CorrectFileNames(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	picture, err := client.GetPicture(ctx, &PicturesRequest{
		Options: &PictureListOptions{Id: pictureID},
		Fields:  &PictureFields{Image: true},
	})
	require.NoError(t, err)
	require.Contains(t,
		picture.GetImage().GetSrc(),
		fmt.Sprintf("t/toyota_%d_corolla/toyota_%d_corolla", randomInt, randomInt),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodHead, picture.GetImage().GetSrc(), nil)
	require.NoError(t, err)

	httpResponse, err := http.DefaultClient.Do(request) //nolint: bodyclose
	require.NoError(t, err)

	defer util.Close(httpResponse.Body)

	require.EqualValues(t, 33914, httpResponse.ContentLength)

	// add brand
	brandName := fmt.Sprintf("Toyota %d", randomInt)
	brandID := createItem(t, conn, cnt, &APIItem{
		Name:       brandName,
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_BRAND,
		Catname:    fmt.Sprintf("toyota-%d", randomInt),
	})

	_, err = itemsClient.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: brandID, Type: ItemParentType_ITEM_TYPE_DEFAULT, Catname: "corolla",
		},
	)
	require.NoError(t, err)

	_, err = client.CorrectFileNames(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	picture, err = client.GetPicture(ctx, &PicturesRequest{
		Options: &PictureListOptions{Id: pictureID},
		Fields:  &PictureFields{Image: true},
	})
	require.NoError(t, err)
	require.Contains(t,
		picture.GetImage().GetSrc(),
		fmt.Sprintf("t/toyota-%d/corolla/toyota_%d_corolla", randomInt, randomInt),
	)

	request, err = http.NewRequestWithContext(ctx, http.MethodHead, picture.GetImage().GetSrc(), nil)
	require.NoError(t, err)

	httpResponse, err = http.DefaultClient.Do(request) //nolint: bodyclose
	require.NoError(t, err)

	defer util.Close(httpResponse.Body)

	require.EqualValues(t, 33914, httpResponse.ContentLength)

	// add second brand
	brand2Name := fmt.Sprintf("Peugeot %d", randomInt)
	brand2ID := createItem(t, conn, cnt, &APIItem{
		Name:       brand2Name,
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_BRAND,
		Catname:    fmt.Sprintf("peugeot-%d", randomInt),
	})

	_, err = itemsClient.CreateItemParent(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&ItemParent{
			ItemId: vehicleID, ParentId: brand2ID, Type: ItemParentType_ITEM_TYPE_DEFAULT, Catname: "corolla",
		},
	)
	require.NoError(t, err)

	_, err = client.CorrectFileNames(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&PictureIDRequest{Id: pictureID},
	)
	require.NoError(t, err)

	picture, err = client.GetPicture(ctx, &PicturesRequest{
		Options: &PictureListOptions{Id: pictureID},
		Fields:  &PictureFields{Image: true},
	})
	require.NoError(t, err)
	require.Contains(t,
		picture.GetImage().GetSrc(),
		fmt.Sprintf("p/peugeot-%d/toyota-%d/corolla/toyota_%d_corolla", randomInt, randomInt, randomInt),
	)

	request, err = http.NewRequestWithContext(ctx, http.MethodHead, picture.GetImage().GetSrc(), nil)
	require.NoError(t, err)

	httpResponse, err = http.DefaultClient.Do(request) //nolint: bodyclose
	require.NoError(t, err)

	defer util.Close(httpResponse.Body)

	require.EqualValues(t, 33914, httpResponse.ContentLength)
}

func TestGetGallery(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := NewPicturesClient(conn)
	cfg := config.LoadConfig(".")
	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	imageStorage, err := cnt.ImageStorage()
	require.NoError(t, err)

	kc := cnt.Keycloak()
	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	pictureID, _ := addPicture(t, imageStorage, goquDB, "./test/small.jpg", schema.PictureStatusInbox)
	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	itemID := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("vehicle-%d", random.Int()),
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_VEHICLE,
	})

	_, err = client.CreatePictureItem(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&CreatePictureItemRequest{
			PictureId: pictureID,
			ItemId:    itemID,
			Type:      PictureItemType_PICTURE_ITEM_CONTENT,
		},
	)
	require.NoError(t, err)

	_, err = client.GetGallery(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&GalleryRequest{
			Request: &PicturesRequest{
				Options: &PictureListOptions{
					PictureItem: &PictureItemListOptions{
						ItemId: itemID,
					},
				},
				Fields: &PictureFields{
					NameText:         true,
					NameHtml:         true,
					Image:            true,
					ThumbMedium:      true,
					Views:            true,
					Votes:            true,
					CommentsCount:    true,
					ModerVote:        true,
					PictureItem:      &PictureItemsRequest{},
					DfDistance:       &DfDistanceRequest{},
					ImageGalleryFull: true,
					Path:             &PicturePathRequest{},
					Thumb:            true,
				},
				Limit: 100,
			},
		},
	)
	require.NoError(t, err)
}
