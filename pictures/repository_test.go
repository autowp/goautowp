package pictures

import (
	"context"
	"database/sql"
	"io"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gopkg.in/gographics/imagick.v3/imagick"
)

func createRandomUser(t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	emailAddr := "test" + strconv.Itoa(random.Int()) + "@example.com"
	name := "ivan"
	res, err := db.Insert(schema.UserTable).
		Rows(goqu.Record{
			schema.UserTableLoginColName:          nil,
			schema.UserTableEmailColName:          emailAddr,
			schema.UserTablePasswordColName:       nil,
			schema.UserTableEmailToCheckColName:   nil,
			schema.UserTableHideEmailColName:      1,
			schema.UserTableEmailCheckCodeColName: nil,
			schema.UserTableNameColName:           name,
			schema.UserTableRegDateColName:        goqu.Func("NOW"),
			schema.UserTableLastOnlineColName:     goqu.Func("NOW"),
			schema.UserTableTimezoneColName:       "Europe/Moscow",
			schema.UserTableLastIPColName:         goqu.Func("INET6_ATON", "127.0.0.1"),
			schema.UserTableLanguageColName:       "en",
			schema.UserTableUUIDColName:           goqu.Func("UUID_TO_BIN", uuid.New().String()),
		}).
		Executor().ExecContext(t.Context())
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return id
}

func repository(t *testing.T) (*goqu.Database, *Repository) {
	t.Helper()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	pgDB, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	pgGoquDB := goqu.New("postgres", pgDB)

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	textStorage := textstorage.New(goquDB)
	itemsRepo := items.NewRepository(
		goquDB,
		cfg.MostsMinCarsCount,
		cfg.ContentLanguages,
		textStorage,
		imageStorage,
	)
	keycloak := gocloak.NewClient(cfg.Keycloak.URL)
	userRepo := users.NewRepository(
		goquDB,
		pgGoquDB,
		cfg.UsersSalt,
		cfg.Languages,
		keycloak,
		cfg.Keycloak,
		cfg.MessageInterval,
		imageStorage,
	)
	i, err := i18nbundle.New()
	require.NoError(t, err)

	msgRepo := messaging.NewRepository(
		goquDB,
		func(_ context.Context, _ int64, _ int64, _ string) error {
			return nil
		},
		i,
	)
	hostsManager := hosts.NewManager(cfg.Languages)
	commentsRepo := comments.NewRepository(goquDB, userRepo, msgRepo, hostsManager)

	return goquDB, NewRepository(
		goquDB,
		imageStorage,
		textStorage,
		itemsRepo,
		cfg.DuplicateFinder,
		commentsRepo,
	)
}

func TestImageExif(t *testing.T) {
	t.Parallel()

	goquDB, repo := repository(t)
	textStorage := textstorage.New(goquDB)
	ctx := t.Context()

	userID := createRandomUser(t, goquDB)

	handle, err := os.OpenFile("../test/test_exif.jpeg", os.O_RDONLY, 0)
	require.NoError(t, err)
	defer util.Close(handle)

	pictureID, err := repo.AddPictureFromReader(ctx, handle, userID, "127.0.0.1", 0, 0, 0)
	require.NoError(t, err)

	picture, err := repo.Picture(
		ctx,
		&query.PictureListOptions{ID: pictureID},
		&PictureFields{},
		OrderByNone,
	)
	require.NoError(t, err)

	require.EqualValues(t, 2022, picture.TakenYear.Int16)
	require.EqualValues(t, 11, picture.TakenMonth.Byte)
	require.EqualValues(t, 15, picture.TakenDay.Byte)
	require.NotEmpty(t, picture.CopyrightsTextID.Int32)

	text, err := textStorage.Text(ctx, picture.CopyrightsTextID.Int32)
	require.NoError(t, err)
	require.Equal(t, "Corey Escobar ©2021 Courtesy of RM Sotheby's", text)

	require.False(t, picture.Point.Valid)
}

func TestImageExifGPS(t *testing.T) {
	t.Parallel()

	goquDB, repo := repository(t)
	ctx := t.Context()

	userID := createRandomUser(t, goquDB)

	handle, err := os.OpenFile("../test/test_exif_gps.jpeg", os.O_RDONLY, 0)
	require.NoError(t, err)
	defer util.Close(handle)

	pictureID, err := repo.AddPictureFromReader(ctx, handle, userID, "127.0.0.1", 0, 0, 0)
	require.NoError(t, err)

	picture, err := repo.Picture(
		ctx,
		&query.PictureListOptions{ID: pictureID},
		&PictureFields{},
		OrderByNone,
	)
	require.NoError(t, err)

	require.EqualValues(t, 2008, picture.TakenYear.Int16)
	require.EqualValues(t, 10, picture.TakenMonth.Byte)
	require.EqualValues(t, 22, picture.TakenDay.Byte)
	require.True(t, picture.Point.Valid)
	require.InDelta(t, 43.464455, picture.Point.Point.Lat(), 0.001)
	require.InDelta(t, 11.881478333333334, picture.Point.Point.Lng(), 0.001)
}

func TestImageBlackEdgeCrop(t *testing.T) {
	t.Parallel()

	goquDB, repo := repository(t)
	ctx := t.Context()

	userID := createRandomUser(t, goquDB)

	handle, err := os.OpenFile("../test/black-edge.jpeg", os.O_RDONLY, 0)
	require.NoError(t, err)
	defer util.Close(handle)

	pictureID, err := repo.AddPictureFromReader(ctx, handle, userID, "127.0.0.1", 0, 0, 0)
	require.NoError(t, err)

	cfg := config.LoadConfig("../")

	picture, err := repo.Picture(
		ctx,
		&query.PictureListOptions{ID: pictureID},
		&PictureFields{},
		OrderByNone,
	)
	require.NoError(t, err)

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	image, err := imageStorage.FormattedImage(ctx, int(picture.ImageID.Int64), "picture-thumb-large")
	require.NoError(t, err)

	imageBlob, err := imageStorage.ImageBlob(ctx, image.ID())
	require.NoError(t, err)

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	imgBytes, err := io.ReadAll(imageBlob)
	require.NoError(t, err)

	err = mw.ReadImageBlob(imgBytes)
	require.NoError(t, err)

	color, err := mw.GetImagePixelColor(0, 2730)
	require.NoError(t, err)

	defer color.Destroy()

	require.InEpsilon(t, 0, color.GetRed(), 0.01)
	require.InEpsilon(t, 0, color.GetBlue(), 0.01)
	require.InEpsilon(t, 0, color.GetGreen(), 0.01)
}
