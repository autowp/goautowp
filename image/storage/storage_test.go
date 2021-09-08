package storage

import (
	"database/sql"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

const TestImageFile = "./_files/Towers_Schiphol_small.jpg"
const TestImageFile2 = "./_files/mazda3_sedan_us-spec_11.jpg"

func TestS3AddImageFromFileChangeNameAndDelete2(t *testing.T) {

	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile, "brand", GenerateOptions{
		Pattern: "folder/file",
	})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	imageInfo, err := mw.GetImage(imageId)
	require.NoError(t, err)

	require.Contains(t, imageInfo.Src(), "folder/file")

	resp, err := http.Get(imageInfo.Src())
	defer util.Close(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	filesize, err := os.Stat(TestImageFile)
	require.NoError(t, err)
	require.EqualValues(t, filesize.Size(), len(body))

	err = mw.ChangeImageName(imageId, GenerateOptions{
		Pattern: "new-name/by-pattern",
	})
	require.NoError(t, err)

	err = mw.RemoveImage(imageId)
	require.NoError(t, err)

	result, err := mw.GetImage(imageId)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestAddImageFromBlobAndFormat(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	blob, err := ioutil.ReadFile(TestImageFile)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromBlob(blob, "test", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formattedImage, err := mw.GetFormattedImage(imageId, "test")
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())
}

func TestS3AddImageWithPreferredName(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile, "test", GenerateOptions{
		PreferredName: "zeliboba",
	})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	image, err := mw.GetImage(imageId)
	require.NoError(t, err)
	require.NotEmpty(t, image.src)
	require.NotEmpty(t, image.height)
	require.NotEmpty(t, image.width)

	require.Contains(t, image.Src(), "zeliboba")
}

func TestAddImageAndCrop(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	crop := sampler.Crop{Left: 1024, Top: 768, Width: 1020, Height: 500}

	err = mw.SetImageCrop(imageId, crop)
	require.NoError(t, err)

	c, err := mw.GetImageCrop(imageId)
	require.NoError(t, err)

	require.EqualValues(t, crop, *c)

	imageInfo, err := mw.GetImage(imageId)
	require.NoError(t, err)

	resp, err := http.Get(imageInfo.Src())
	defer util.Close(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	filesize, err := os.Stat(TestImageFile2)
	require.NoError(t, err)
	require.EqualValues(t, filesize.Size(), len(body))

	formattedImage, err := mw.GetFormattedImage(imageId, "picture-gallery")
	require.NoError(t, err)

	require.EqualValues(t, 1020, formattedImage.Width())
	require.EqualValues(t, 500, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())

	require.Contains(t, formattedImage.Src(), "0400030003fc01f4")
}

func TestFlopNormalizeAndMultipleRequest(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId1, err := mw.AddImageFromFile(TestImageFile, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId1)

	err = mw.Flop(imageId1)
	require.NoError(t, err)

	imageId2, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId2)

	err = mw.Normalize(imageId2)
	require.NoError(t, err)

	images, err := mw.GetImages([]int{imageId1, imageId2})
	require.NoError(t, err)

	require.EqualValues(t, 2, len(images))

	formattedImages, err := mw.GetFormattedImages([]int{imageId1, imageId2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formattedImages))

	// re-request
	formattedImages, err = mw.GetFormattedImages([]int{imageId1, imageId2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formattedImages))
}

func TestRequestFormattedImageAgain(t *testing.T) {
	cfg := config.LoadConfig("../../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, cfg.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	formatName := "test"

	formattedImage, err := mw.GetFormattedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())

	formattedImage, err = mw.GetFormattedImage(imageId, formatName)
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

	formattedImage, err := mw.GetFormattedImage(imageId, formatName)
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

	formattedImage, err := mw.GetFormattedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formattedImage.Width())
	require.EqualValues(t, 120, formattedImage.Height())
	require.True(t, formattedImage.FileSize() > 0)
	require.NotEmpty(t, formattedImage.Src())
}
*/
