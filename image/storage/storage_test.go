package storage

import (
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

const TestImageFile = "./_files/Towers_Schiphol_small.jpg"
const TestImageFile2 = "./_files/mazda3_sedan_us-spec_11.jpg"

// Config Application config definition
type TestConfig struct {
	AutowpDSN    string `yaml:"autowp-dsn"         mapstructure:"autowp-dsn"`
	ImageStorage Config `yaml:"image-storage"      mapstructure:"image-storage"`
}

func LoadConfig() TestConfig {

	config := TestConfig{}

	viper.SetConfigName("defaults")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./../..")

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	viper.SetConfigName("config")
	err = viper.MergeInConfig()
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("fatal error unmarshal config: %s", err))
	}

	return config
}

func TestS3AddImageFromFileChangeNameAndDelete2(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile, "brand", GenerateOptions{
		Pattern: "folder/file",
	})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	imageInfo, err := mw.GetImage(imageId)
	require.NoError(t, err)

	require.Contains(t, "folder/file", imageInfo.Src())

	blob, err := ioutil.ReadFile(imageInfo.Src())
	require.NoError(t, err)
	filesize, err := os.Stat(TestImageFile)
	require.NoError(t, err)
	require.EqualValues(t, filesize, len(blob))

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
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	blob, err := ioutil.ReadFile(TestImageFile)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromBlob(blob, "test", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formatedImage, err := mw.GetFormatedImage(imageId, "test")
	require.NoError(t, err)

	require.EqualValues(t, 160, formatedImage.Width())
	require.EqualValues(t, 120, formatedImage.Height())
	require.True(t, formatedImage.FileSize() > 0)
	require.NotEmpty(t, formatedImage.Src())
}

func TestS3AddImageWithPrefferedName(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile, "test", GenerateOptions{
		PreferredName: "zeliboba",
	})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	image, err := mw.GetImage(imageId)
	require.NoError(t, err)

	require.Contains(t, "zeliboba", image.Src())
}

func TestAddImageAndCrop(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	crop := sampler.Crop{1024, 768, 1020, 500}

	err = mw.SetImageCrop(imageId, crop)
	require.NoError(t, err)

	c, err := mw.GetImageCrop(imageId)
	require.NoError(t, err)

	require.EqualValues(t, crop, c)

	imageInfo, err := mw.GetImage(imageId)
	require.NoError(t, err)
	blob, err := ioutil.ReadFile(imageInfo.Src())
	require.NoError(t, err)
	filesize, err := os.Stat(TestImageFile2)
	require.NoError(t, err)
	require.EqualValues(t, filesize, len(blob))

	formatedImage, err := mw.GetFormatedImage(imageId, "picture-gallery")
	require.NoError(t, err)

	require.EqualValues(t, 1020, formatedImage.Width())
	require.EqualValues(t, 500, formatedImage.Height())
	require.True(t, formatedImage.FileSize() > 0)
	require.NotEmpty(t, formatedImage.Src())

	require.Contains(t, "0400030003fc01f4", formatedImage.Src())
}

func TestFlopNormalizeAndMultipleRequest(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
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

	formatedImages, err := mw.GetFormattedImages([]int{imageId1, imageId2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formatedImages))

	// re-request
	formatedImages, err = mw.GetFormattedImages([]int{imageId1, imageId2}, "test")
	require.NoError(t, err)
	require.EqualValues(t, 2, len(formatedImages))
}

func TestRequestFormatedImageAgain(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, imageId)

	formatName := "test"

	formatedImage, err := mw.GetFormatedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formatedImage.Width())
	require.EqualValues(t, 120, formatedImage.Height())
	require.True(t, formatedImage.FileSize() > 0)
	require.NotEmpty(t, formatedImage.Src())

	formatedImage, err = mw.GetFormatedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formatedImage.Width())
	require.EqualValues(t, 120, formatedImage.Height())
	require.True(t, formatedImage.FileSize() > 0)
	require.NotEmpty(t, formatedImage.Src())
}

/*func TestTimeout(t *testing.T) {
	//$app = Application::init(require __DIR__ . "/_files/config/application.config.php");

	mw := NewStorage()

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formatName := "picture-gallery"

	tables := serviceManager.get("TableManager")
	formatedImageTable := tables.get("formated_image")

	formatedImageTable.insert(Row{
		"format":            formatName,
		"image_id":          imageId,
		"status":            StatusProcessing,
		"formated_image_id": nil,
	})

	formatedImage, err := mw.GetFormatedImage(imageId, formatName)
	require.NoError(t, err)

	require.Empty(t, formatedImage)
}*/

func TestNormalizeProcessor(t *testing.T) {
	config := LoadConfig()
	db, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)
	mw, err := NewStorage(db, config.ImageStorage)
	require.NoError(t, err)

	imageId, err := mw.AddImageFromFile(TestImageFile2, "brand", GenerateOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, imageId)

	formatName := "with-processor"

	formatedImage, err := mw.GetFormatedImage(imageId, formatName)
	require.NoError(t, err)

	require.EqualValues(t, 160, formatedImage.Width())
	require.EqualValues(t, 120, formatedImage.Height())
	require.True(t, formatedImage.FileSize() > 0)
	require.NotEmpty(t, formatedImage.Src())
}
