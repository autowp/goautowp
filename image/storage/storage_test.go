package storage

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

const (
	TestImageFile  = "./_files/Towers_Schiphol_small.jpg"
	TestImageFile2 = "./_files/mazda3_sedan_us-spec_11.jpg"
)

func TestS3AddImageFromFileChangeNameAndDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	imageStorage, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	imageID, err := imageStorage.AddImageFromFile(ctx, TestImageFile, "brand", GenerateOptions{
		Pattern: "folder/file",
	})
	require.NoError(t, err)
	require.NotEmpty(t, imageID)

	imageInfo, err := imageStorage.Image(ctx, imageID)
	require.NoError(t, err)

	require.Contains(t, imageInfo.Src(), "folder/file")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageInfo.Src(), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	require.NoError(t, err)

	defer util.Close(resp.Body)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	filesize, err := os.Stat(TestImageFile)
	require.NoError(t, err)
	require.EqualValues(t, filesize.Size(), len(body))

	err = imageStorage.ChangeImageName(ctx, imageID, GenerateOptions{
		Pattern: "new-name/by-pattern",
	})
	require.NoError(t, err)

	err = imageStorage.RemoveImage(imageID)
	require.NoError(t, err)

	_, err = imageStorage.Image(ctx, imageID)
	require.ErrorIs(t, ErrImageNotFound, err)
}

func TestAddImageFromBlobAndFormat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	mw, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	blob, err := os.ReadFile(TestImageFile)
	require.NoError(t, err)

	imageID, err := mw.AddImageFromBlob(ctx, blob, "test", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageID)

	formattedImage, err := mw.FormattedImage(ctx, imageID, "test")
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())
}

func TestS3AddImageWithPreferredName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	mw, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	imageID, err := mw.AddImageFromFile(ctx, TestImageFile, "test", GenerateOptions{
		PreferredName: "zeliboba",
	})
	require.NoError(t, err)

	require.NotEmpty(t, imageID)

	image, err := mw.Image(ctx, imageID)
	require.NoError(t, err)
	require.NotEmpty(t, image.src)
	require.NotEmpty(t, image.height)
	require.NotEmpty(t, image.width)

	require.Contains(t, image.Src(), "zeliboba")
}

func TestAddImageAndCrop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	mw, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	imageID, err := mw.AddImageFromFile(ctx, TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageID)

	crop := sampler.Crop{Left: 1024, Top: 768, Width: 1020, Height: 500}

	err = mw.SetImageCrop(imageID, crop)
	require.NoError(t, err)

	c, err := mw.imageCrop(imageID)
	require.NoError(t, err)

	require.EqualValues(t, crop, *c)

	imageInfo, err := mw.Image(ctx, imageID)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imageInfo.Src(), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	require.NoError(t, err)

	defer util.Close(resp.Body)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	filesize, err := os.Stat(TestImageFile2)
	require.NoError(t, err)
	require.EqualValues(t, filesize.Size(), len(body))

	formattedImage, err := mw.FormattedImage(ctx, imageID, "picture-gallery")
	require.NoError(t, err)

	require.EqualValues(t, 1020, formattedImage.Width())
	require.EqualValues(t, 500, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())

	require.Contains(t, formattedImage.Src(), "0400030003fc01f4")
}

func TestFlopNormalizeAndMultipleRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	mw, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	imageID1, err := mw.AddImageFromFile(ctx, TestImageFile, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageID1)

	err = mw.Flop(imageID1)
	require.NoError(t, err)

	imageID2, err := mw.AddImageFromFile(ctx, TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageID2)

	err = mw.Normalize(imageID2)
	require.NoError(t, err)

	images, err := mw.images([]int{imageID1, imageID2})
	require.NoError(t, err)

	require.EqualValues(t, 2, len(images))

	formattedImages, err := mw.FormattedImages(ctx, []int{imageID1, imageID2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formattedImages))

	// re-request
	formattedImages, err = mw.FormattedImages(ctx, []int{imageID1, imageID2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formattedImages))
}

func TestRequestFormattedImageAgain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	mw, err := NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	imageID, err := mw.AddImageFromFile(ctx, TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageID)

	formatName := "test"

	formattedImage, err := mw.FormattedImage(ctx, imageID, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())

	formattedImage, err = mw.FormattedImage(ctx, imageID, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())
}

/*func TestTimeout(t *testing.T) {
	//$app = Application::init(require __DIR__ . "/_files/config/application.config.php");

	mw := NewStorage()

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formatName := "picture-gallery"

	tables := serviceManager.get("TableManager")
	formattedImageTable := tables.get("formated_image")

	formattedImageTable.insert(Row{
		"format":            formatName,
		"image_id":          imageId,
		"status":            StatusProcessing,
		"formated_image_id": nil,
	})

	formattedImage, err := mw.FormattedImage(imageId, formatName)
	require.NoError(t, err)

	require.Empty(t, formattedImage)
}*/

/*func TestNormalizeProcessor(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formatName := "with-processor"

	formattedImage, err := mw.FormattedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())
}
*/
