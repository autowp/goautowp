package schema

import "github.com/doug-martin/goqu/v9"

const (
	ImageTableName              = "image"
	ImageTableIDColName         = "id"
	ImageTableFilepathColName   = "filepath"
	ImageTableFilesizeColName   = "filesize"
	ImageTableWidthColName      = "width"
	ImageTableHeightColName     = "height"
	ImageTableDirColName        = "dir"
	ImageTableDateAddColName    = "date_add"
	ImageTableCropLeftColName   = "crop_left"
	ImageTableCropTopColName    = "crop_top"
	ImageTableCropWidthColName  = "crop_width"
	ImageTableCropHeightColName = "crop_height"
	ImageTableS3ColName         = "s3"
)

var (
	ImageTable              = goqu.T(ImageTableName)
	ImageTableIDCol         = ImageTable.Col(ImageTableIDColName)
	ImageTableWidthCol      = ImageTable.Col(ImageTableWidthColName)
	ImageTableHeightCol     = ImageTable.Col(ImageTableHeightColName)
	ImageTableFilesizeCol   = ImageTable.Col(ImageTableFilesizeColName)
	ImageTableFilepathCol   = ImageTable.Col(ImageTableFilepathColName)
	ImageTableDirCol        = ImageTable.Col(ImageTableDirColName)
	ImageTableCropLeftCol   = ImageTable.Col(ImageTableCropLeftColName)
	ImageTableCropTopCol    = ImageTable.Col(ImageTableCropTopColName)
	ImageTableCropWidthCol  = ImageTable.Col(ImageTableCropWidthColName)
	ImageTableCropHeightCol = ImageTable.Col(ImageTableCropHeightColName)
)

type ImageRow struct {
	ID         int    `db:"id"`
	Width      int    `db:"width"`
	Height     int    `db:"height"`
	Filesize   int    `db:"filesize"`
	Filepath   string `db:"filepath"`
	Dir        string `db:"dir"`
	CropLeft   int    `db:"crop_left"`
	CropTop    int    `db:"crop_top"`
	CropWidth  int    `db:"crop_width"`
	CropHeight int    `db:"crop_height"`
}
