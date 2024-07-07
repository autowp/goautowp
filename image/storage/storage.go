package storage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // GIF support
	_ "image/jpeg" // JPEG support
	_ "image/png"  // PNG support
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/doug-martin/goqu/v9"
	my "github.com/go-mysql/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const (
	StatusDefault    int = 0
	StatusProcessing int = 1
	StatusFailed     int = 2
)

const maxInsertAttempts = 15

const (
	defaultExtension = "jpg"
	pngExtension     = "png"
	jpegExtension    = "jpg"
	gifExtension     = "gif"
)

const dirNotDefinedMessage = "dir not defined"

const listBrokenImagesPerPage = 1000

var (
	ErrImageNotFound        = errors.New("image not found")
	errUnsupportedImageType = errors.New("unsupported image type")
	errDirNotFound          = errors.New(dirNotDefinedMessage)
	errFormatNotFound       = errors.New("format not found")
	errFailedToFormatImage  = errors.New("failed to format image")
	errFailedToGetImageSize = errors.New("failed to get image size")
	errSelfRename           = errors.New("trying to rename to self")
	errInvalidImageID       = errors.New("invalid image id provided")
)

var publicRead = "public-read"

var formats2ContentType = map[string]string{
	"GIF":  "image/gif",
	"PNG":  "image/png",
	"JPG":  "image/jpeg",
	"JPEG": "image/jpeg",
}

type Storage struct {
	config                config.ImageStorageConfig
	db                    *goqu.Database
	dirs                  map[string]*Dir
	formats               map[string]*sampler.Format
	formattedImageDirName string
	sampler               *sampler.Sampler
}

type imageRow struct {
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

type formattedImageRow struct {
	ImageID          int           `db:"image_id"`
	Format           string        `db:"format"`
	FormattedImageID sql.NullInt32 `db:"formated_image_id"`
	Status           int           `db:"status"`
}

type FlushOptions struct {
	Image  int
	Format string
}

func NewStorage(db *goqu.Database, config config.ImageStorageConfig) (*Storage, error) {
	dirs := make(map[string]*Dir)

	for dirName, dirConfig := range config.Dirs {
		dir, err := NewDir(dirConfig.Bucket, dirConfig.NamingStrategy)
		if err != nil {
			return nil, err
		}

		dirs[dirName] = dir
	}

	formats := make(map[string]*sampler.Format)
	for formatName, formatConfig := range config.Formats {
		formats[formatName] = sampler.NewFormat(formatConfig)
	}

	return &Storage{
		config:                config,
		db:                    db,
		dirs:                  dirs,
		formats:               formats,
		formattedImageDirName: "format",
		sampler:               sampler.NewSampler(),
	}, nil
}

func (s *Storage) Image(ctx context.Context, id int) (*Image, error) {
	var img Image

	st := struct {
		ID       int    `db:"id"`
		Width    int    `db:"width"`
		Height   int    `db:"height"`
		Filepath string `db:"filepath"`
		Filesize int    `db:"filesize"`
		Dir      string `db:"dir"`
	}{}

	success, err := s.db.Select(
		schema.ImageTableIDCol,
		schema.ImageTableWidthCol,
		schema.ImageTableHeightCol,
		schema.ImageTableFilesizeCol,
		schema.ImageTableFilepathCol,
		schema.ImageTableDirCol,
	).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.Eq(id)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrImageNotFound
	}

	img.id = st.ID
	img.width = st.Width
	img.height = st.Height
	img.filepath = st.Filepath
	img.filesize = st.Filesize
	img.dir = st.Dir

	err = s.populateSrc(&img)
	if err != nil {
		return nil, err
	}

	return &img, nil
}

func (s *Storage) populateSrc(img *Image) error {
	dir := s.dir(img.dir)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, img.dir)
	}

	bucket := dir.Bucket()

	s3Client := s.s3Client()

	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &img.filepath,
	})
	rest.Build(req)

	url := req.HTTPRequest.URL

	if len(s.config.SrcOverride.Host) > 0 {
		url.Host = s.config.SrcOverride.Host
	}

	if len(s.config.SrcOverride.Scheme) > 0 {
		url.Scheme = s.config.SrcOverride.Scheme
	}

	img.src = url.String()

	return nil
}

func (s *Storage) FormattedImage(ctx context.Context, id int, formatName string) (*Image, error) {
	var row imageRow

	success, err := s.db.Select(
		schema.ImageTableIDCol, schema.ImageTableWidthCol, schema.ImageTableHeightCol, schema.ImageTableFilesizeCol,
		schema.ImageTableFilepathCol, schema.ImageTableDirCol,
	).
		From(schema.ImageTable).
		Join(schema.FormattedImageTable, goqu.On(schema.ImageTableIDCol.Eq(schema.FormattedImageTableFormattedImageIDCol))).
		Where(
			schema.FormattedImageTableImageIDCol.Eq(id),
			schema.FormattedImageTableFormatCol.Eq(formatName),
		).ScanStructContext(ctx, &row)
	if err != nil {
		return nil, err
	}

	if success {
		var img Image

		img.id = row.ID
		img.width = row.Width
		img.height = row.Height
		img.filesize = row.Filesize
		img.filepath = row.Filepath
		img.dir = row.Dir

		err = s.populateSrc(&img)
		if err != nil {
			return nil, err
		}

		return &img, nil
	}

	formattedImageID, err := s.doFormatImage(ctx, id, formatName)
	if err != nil {
		return nil, err
	}

	return s.Image(ctx, formattedImageID)
}

func (s *Storage) dir(dirName string) *Dir {
	dir, ok := s.dirs[dirName]
	if ok {
		return dir
	}

	return nil
}

func (s *Storage) s3Client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           &s.config.S3.Region,
		Endpoint:         &s.config.S3.Endpoint,
		S3ForcePathStyle: &s.config.S3.UsePathStyleEndpoint,
		Credentials:      credentials.NewStaticCredentials(s.config.S3.Credentials.Key, s.config.S3.Credentials.Secret, ""),
	}))
	svc := s3.New(sess)

	return svc
}

func getCropSuffix(i imageRow) string {
	result := ""

	if i.CropWidth <= 0 || i.CropHeight <= 0 {
		return result
	}

	return fmt.Sprintf(
		"_%04x%04x%04x%04x",
		i.CropLeft,
		i.CropTop,
		i.CropWidth,
		i.CropHeight,
	)
}

func fileNameWithoutExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}

	return fileName
}

func (s *Storage) doFormatImage(ctx context.Context, imageID int, formatName string) (int, error) {
	// find source image
	var iRow imageRow

	success, err := s.db.Select(
		schema.ImageTableIDCol,
		schema.ImageTableWidthCol,
		schema.ImageTableHeightCol,
		schema.ImageTableFilepathCol,
		schema.ImageTableDirCol,
		schema.ImageTableCropLeftCol,
		schema.ImageTableCropTopCol,
		schema.ImageTableCropWidthCol,
		schema.ImageTableCropHeightCol,
	).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.Eq(imageID)).
		ScanStructContext(ctx, &iRow)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	dir := s.dir(iRow.Dir)
	if dir == nil {
		return 0, fmt.Errorf("%w: `%s`", errDirNotFound, iRow.Dir)
	}

	bucket := dir.Bucket()

	s3Client := s.s3Client()

	object, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &iRow.Filepath,
	})
	if err != nil {
		return 0, fmt.Errorf("s3Client.GetObject(%s, %s): %w", bucket, iRow.Filepath, err)
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	imgBytes, err := io.ReadAll(object.Body)
	if err != nil {
		return 0, err
	}

	err = mw.ReadImageBlob(imgBytes)
	if err != nil {
		return 0, err
	}

	// format
	format := s.format(formatName)
	if format == nil {
		return 0, fmt.Errorf("%w: `%s`", errFormatNotFound, formatName)
	}

	_, err = s.db.Insert(schema.FormattedImageTable).Rows(goqu.Record{
		schema.FormattedImageTableFormatColName:           formatName,
		schema.FormattedImageTableImageIDColName:          imageID,
		schema.FormattedImageTableStatusColName:           StatusProcessing,
		schema.FormattedImageTableFormattedImageIDColName: nil,
	}).Executor().ExecContext(ctx)
	if err != nil {
		ok, myerr := my.Error(err) // MySQL error
		if !ok || !errors.Is(myerr, my.ErrDupeKey) {
			return 0, err
		}

		// wait until done
		logrus.Debug("Wait until image processing done")

		var (
			done  = false
			fiRow formattedImageRow
		)

		for i := 0; i < maxInsertAttempts && !done; i++ {
			success, err = s.db.Select(schema.FormattedImageTableFormattedImageIDCol, schema.FormattedImageTableStatusCol).
				From(schema.FormattedImageTable).
				Where(schema.FormattedImageTableImageIDCol.Eq(imageID)).
				ScanStructContext(ctx, &fiRow)
			if err != nil {
				return 0, err
			}

			if !success {
				return 0, sql.ErrNoRows
			}

			done = fiRow.Status != StatusProcessing
			if !done {
				time.Sleep(time.Second)
			}
		}

		if !done {
			// mark as failed
			_, err = s.db.Update(schema.FormattedImageTable).
				Set(goqu.Record{schema.FormattedImageTableStatusColName: StatusFailed}).
				Where(
					schema.FormattedImageTableFormatCol.Eq(formatName),
					schema.FormattedImageTableImageIDCol.Eq(imageID),
					schema.FormattedImageTableStatusCol.Eq(StatusProcessing),
				).
				Executor().ExecContext(ctx)
			if err != nil {
				return 0, err
			}
		}

		if !fiRow.FormattedImageID.Valid {
			return 0, errFailedToFormatImage
		}

		return int(fiRow.FormattedImageID.Int32), nil
	}

	var formattedImageID int
	// try {
	// $crop = $this->getRowCrop(iRow);

	cropSuffix := getCropSuffix(iRow)

	crop := sampler.Crop{
		Left:   iRow.CropLeft,
		Top:    iRow.CropTop,
		Width:  iRow.CropWidth,
		Height: iRow.CropHeight,
	}

	mw, err = s.sampler.ConvertImage(mw, crop, *format)
	if err != nil {
		return 0, err
	}

	/*foreach ($cFormat->getProcessors() as $processorName) {
		$processor = $this->processors->get($processorName);
		$processor->process($imagick);
	}*/

	// store result
	newPath := strings.Join([]string{
		iRow.Dir,
		formatName,
		iRow.Filepath,
	}, "/")

	formatExt, err := format.FormatExtension()
	if err != nil {
		return 0, err
	}

	extension := formatExt
	if formatExt == "" {
		extension = strings.TrimLeft(filepath.Ext(newPath), ".")
	}

	formattedImageID, err = s.addImageFromImagick(
		ctx,
		mw,
		s.formattedImageDirName,
		GenerateOptions{
			Extension: extension,
			Pattern:   filepath.Dir(newPath) + "/" + fileNameWithoutExtension(filepath.Base(newPath)) + cropSuffix,
		},
	)
	if err != nil {
		return 0, err
	}

	_, err = s.db.Update(schema.FormattedImageTable).
		Set(goqu.Record{
			schema.FormattedImageTableFormattedImageIDColName: formattedImageID,
			schema.FormattedImageTableStatusColName:           StatusDefault,
		}).
		Where(
			schema.FormattedImageTableFormatCol.Eq(formatName),
			schema.FormattedImageTableImageIDCol.Eq(imageID),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	// } catch (Exception $e) {
	_, err = s.db.Update(schema.FormattedImageTable).
		Set(goqu.Record{
			schema.FormattedImageTableStatusColName: StatusFailed,
		}).
		Where(
			schema.FormattedImageTableFormatCol.Eq(formatName),
			schema.FormattedImageTableImageIDCol.Eq(imageID),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	// throw $e;
	// }

	return formattedImageID, nil
}

func (s *Storage) format(name string) *sampler.Format {
	format, ok := s.formats[name]
	if ok {
		return format
	}

	return nil
}

func (s *Storage) addImageFromImagick(
	ctx context.Context,
	mw *imagick.MagickWand,
	dirName string,
	options GenerateOptions,
) (int, error) {
	width := int(mw.GetImageWidth())
	height := int(mw.GetImageHeight())

	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("%w: (%v x %v)", errFailedToGetImageSize, width, height)
	}

	format := mw.GetImageFormat()

	switch strings.ToLower(format) {
	case "gif":
		options.Extension = gifExtension
	case "jpeg":
		options.Extension = jpegExtension
	case "png":
		options.Extension = pngExtension
	default:
		return 0, fmt.Errorf("%w: `%v`", errUnsupportedImageType, format)
	}

	dir := s.dir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("%w: `%s`", errDirNotFound, dirName)
	}

	blob, err := mw.GetImagesBlob()
	if err != nil {
		return 0, err
	}

	id, err := s.generateLockWrite(
		ctx,
		dirName,
		options,
		width,
		height,
		func(fileName string) error {
			s3c := s.s3Client()
			blobReader := bytes.NewReader(blob)
			bucket := dir.Bucket()

			contentType, err := imageFormatContentType(mw.GetImageFormat())
			if err != nil {
				return err
			}

			_, err = s3c.PutObject(&s3.PutObjectInput{
				Key:         &fileName,
				Body:        blobReader,
				Bucket:      &bucket,
				ACL:         &publicRead,
				ContentType: &contentType,
			})

			return err
		},
	)
	if err != nil {
		return 0, err
	}

	filesize := len(blob)
	/*exif := s.extractEXIF(id)
	if exif {
		exif = json_encode(exif, JSON_INVALID_UTF8_SUBSTITUTE|JSON_THROW_ON_ERROR)
	}*/

	_, err = s.db.Update(schema.ImageTable).
		Set(goqu.Record{schema.ImageTableFilesizeColName: filesize}).
		Where(schema.ImageTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Storage) generateLockWrite(
	ctx context.Context,
	dirName string,
	options GenerateOptions,
	width int,
	height int,
	callback func(string) error,
) (int, error) {
	var (
		insertAttemptException error
		imageID                = 0
	)

	for attemptIndex := range maxInsertAttempts {
		insertAttemptException = s.incDirCounter(ctx, dirName)

		if insertAttemptException == nil {
			opt := options
			opt.Index = indexByAttempt(attemptIndex)

			var (
				destFileName string
				res          sql.Result
			)

			destFileName, insertAttemptException = s.createImagePath(ctx, dirName, opt)
			if insertAttemptException == nil {
				// store to db
				res, insertAttemptException = s.db.Insert(schema.ImageTable).Rows(goqu.Record{
					schema.ImageTableWidthColName:      width,
					schema.ImageTableHeightColName:     height,
					schema.ImageTableDirColName:        dirName,
					schema.ImageTableFilesizeColName:   0,
					schema.ImageTableFilepathColName:   destFileName,
					schema.ImageTableDateAddColName:    goqu.Func("NOW"),
					schema.ImageTableCropLeftColName:   0,
					schema.ImageTableCropTopColName:    0,
					schema.ImageTableCropWidthColName:  0,
					schema.ImageTableCropHeightColName: 0,
					schema.ImageTableS3ColName:         1,
				}).Executor().ExecContext(ctx)

				if insertAttemptException == nil {
					var id int64

					id, insertAttemptException = res.LastInsertId()
					if insertAttemptException == nil {
						insertAttemptException = callback(destFileName)

						imageID = int(id)
					}
				}
			}
		}

		if insertAttemptException == nil {
			break
		}
	}

	return imageID, insertAttemptException
}

func (s *Storage) incDirCounter(ctx context.Context, dirName string) error {
	res, err := s.db.Update(schema.ImageDirTable).
		Set(goqu.Record{
			schema.ImageDirTableCountColName: goqu.L("? + 1", schema.ImageDirTableCountCol),
		}).
		Where(schema.ImageDirTableDirCol.Eq(dirName)).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected <= 0 {
		_, err = s.db.Insert(schema.ImageDirTable).Rows(goqu.Record{
			schema.ImageDirTableDirColName:   dirName,
			schema.ImageDirTableCountColName: 1,
		}).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) dirCounter(ctx context.Context, dirName string) (int, error) {
	var result int

	success, err := s.db.Select(schema.ImageDirTableCountCol).
		From(schema.ImageDirTable).
		Where(schema.ImageDirTableDirCol.Eq(dirName)).
		ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
}

func indexByAttempt(attempt int) int {
	const powBase = 10

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	float := float64(attempt)
	minVal := int(math.Pow(powBase, float-1))
	maxVal := int(math.Pow(powBase, float) - 1)

	return random.Intn(maxVal-minVal+1) + minVal
}

func (s *Storage) createImagePath(ctx context.Context, dirName string, options GenerateOptions) (string, error) {
	dir := s.dir(dirName)
	if dir == nil {
		return "", fmt.Errorf("%w: `%s`", errDirNotFound, dirName)
	}

	namingStrategy := dir.NamingStrategy()

	c, err := s.dirCounter(ctx, dirName)
	if err != nil {
		return "", err
	}

	options.Count = c

	if len(options.Extension) == 0 {
		options.Extension = defaultExtension
	}

	return namingStrategy.Generate(options), nil
}

func imageFormatContentType(format string) (string, error) {
	format = strings.ToUpper(format)

	result, ok := formats2ContentType[format]
	if !ok {
		return "", fmt.Errorf("%w: `%s`", errFormatNotFound, format)
	}

	return result, nil
}

func (s *Storage) RemoveImage(ctx context.Context, imageID int) error {
	var row imageRow

	success, err := s.db.Select(schema.ImageTableIDCol, schema.ImageTableDirCol, schema.ImageTableFilepathCol).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.Eq(imageID)).
		ScanStructContext(ctx, &row)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	err = s.Flush(ctx, FlushOptions{
		Image: row.ID,
	})
	if err != nil {
		return err
	}

	// to save remove formatted image
	_, err = s.db.Delete(schema.FormattedImageTable).
		Where(schema.FormattedImageTableFormattedImageIDCol.Eq(row.ID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	// important to delete row first
	_, err = s.db.Delete(schema.ImageTable).Where(schema.ImageTableIDCol.Eq(row.ID)).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	dir := s.dir(row.Dir)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, row.Dir)
	}

	s3c := s.s3Client()

	bucket := dir.Bucket()
	key := row.Filepath

	_, err = s3c.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Flush(ctx context.Context, options FlushOptions) error {
	sqSelect := s.db.Select(schema.FormattedImageTableImageIDCol, schema.FormattedImageTableFormatCol,
		schema.FormattedImageTableFormattedImageIDCol).
		From(schema.FormattedImageTable)

	if len(options.Format) > 0 {
		sqSelect = sqSelect.Where(schema.FormattedImageTableFormatCol.Eq(options.Format))
	}

	if options.Image > 0 {
		sqSelect = sqSelect.Where(schema.FormattedImageTableImageIDCol.Eq(options.Image))
	}

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	defer util.Close(rows)

	for rows.Next() {
		var (
			iID    int
			format string
			fiID   sql.NullInt32
		)

		err = rows.Scan(&iID, &format, &fiID)
		if err != nil {
			return err
		}

		if fiID.Valid && fiID.Int32 > 0 {
			err = s.RemoveImage(ctx, int(fiID.Int32))
			if err != nil {
				return err
			}
		}

		_, err = s.db.Delete(schema.FormattedImageTable).
			Where(
				schema.FormattedImageTableImageIDCol.Eq(iID),
				schema.FormattedImageTableFormatCol.Eq(format),
			).
			Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}

func (s *Storage) ChangeImageName(ctx context.Context, imageID int, options GenerateOptions) error {
	var img imageRow

	success, err := s.db.Select(schema.ImageTableIDCol, schema.ImageTableDirCol, schema.ImageTableFilepathCol).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.Eq(imageID)).
		ScanStructContext(ctx, &img)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	dir := s.dir(img.Dir)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, img.Dir)
	}

	if len(options.Extension) == 0 {
		options.Extension = strings.TrimLeft(filepath.Ext(img.Filepath), ".")
	}

	var insertAttemptException error

	s3c := s.s3Client()

	for attemptIndex := range maxInsertAttempts {
		options.Index = indexByAttempt(attemptIndex)

		destFileName, err := s.createImagePath(ctx, img.Dir, options)
		if err != nil {
			return err
		}

		if destFileName == img.Filepath {
			return errSelfRename
		}

		_, insertAttemptException = s.db.Update(schema.ImageTable).
			Set(goqu.Record{schema.ImageTableFilepathColName: destFileName}).
			Where(schema.ImageTableIDCol.Eq(img.ID)).
			Executor().ExecContext(ctx)

		if insertAttemptException == nil {
			bucket := dir.Bucket()
			copySource := dir.Bucket() + "/" + img.Filepath

			_, err = s3c.CopyObject(&s3.CopyObjectInput{
				Bucket:     &bucket,
				CopySource: &copySource,
				Key:        &destFileName,
				ACL:        &publicRead,
			})
			if err != nil {
				return err
			}

			fpath := img.Filepath

			_, err = s3c.DeleteObject(&s3.DeleteObjectInput{
				Bucket: &bucket,
				Key:    &fpath,
			})
			if err != nil {
				return err
			}

			break
		}
	}

	return insertAttemptException
}

func (s *Storage) AddImageFromFile(
	ctx context.Context,
	file string,
	dirName string,
	options GenerateOptions,
) (int, error) {
	handle, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer util.Close(handle)

	imageInfo, imageType, err := image.DecodeConfig(handle)
	if err != nil {
		return 0, err
	}

	if imageInfo.Width <= 0 || imageInfo.Height <= 0 {
		return 0, fmt.Errorf("%w: (%v x %v)", errFailedToGetImageSize, imageInfo.Width, imageInfo.Height)
	}

	if len(options.Extension) == 0 {
		var ext string

		switch imageType {
		case "gif":
			ext = gifExtension
		case "jpeg":
			ext = jpegExtension
		case "png":
			ext = pngExtension
		default:
			return 0, fmt.Errorf("%w: `%v`", errUnsupportedImageType, imageType)
		}

		options.Extension = ext
	}

	dir := s.dir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("%w: `%s`", errDirNotFound, dirName)
	}

	id, err := s.generateLockWrite(
		ctx,
		dirName,
		options,
		imageInfo.Width,
		imageInfo.Height,
		func(fileName string) error {
			bucket := dir.Bucket()

			contentType, err := imageFormatContentType(options.Extension)
			if err != nil {
				return err
			}

			handle, err := os.Open(file)
			if err != nil {
				return err
			}
			defer util.Close(handle)

			_, err = s.s3Client().PutObject(&s3.PutObjectInput{
				Key:         &fileName,
				Body:        handle,
				Bucket:      &bucket,
				ACL:         &publicRead,
				ContentType: &contentType,
			})
			if err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return 0, err
	}

	/*$exif = $this->extractEXIF($id);
	if ($exif) {
		$exif = json_encode($exif, JSON_INVALID_UTF8_SUBSTITUTE | JSON_THROW_ON_ERROR);
	}*/

	fi, err := handle.Stat()
	if err != nil {
		return 0, err
	}

	_, err = s.db.Update(schema.ImageTable).
		Set(goqu.Record{schema.ImageTableFilesizeColName: fi.Size()}).
		Where(schema.ImageTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Storage) AddImageFromBlob(
	ctx context.Context,
	blob []byte,
	dirName string,
	options GenerateOptions,
) (int, error) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	if err := mw.ReadImageBlob(blob); err != nil {
		return 0, err
	}

	id, err := s.addImageFromImagick(ctx, mw, dirName, options)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Storage) doImagickOperation(ctx context.Context, imageID int, callback func(*imagick.MagickWand) error) error {
	var img imageRow

	success, err := s.db.Select(schema.ImageTableDirCol, schema.ImageTableFilepathCol).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.Eq(imageID)).
		ScanStructContext(ctx, &img)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	dir := s.dir(img.Dir)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, img.Dir)
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	s3c := s.s3Client()

	bucket := dir.Bucket()
	fpath := img.Filepath

	object, err := s3c.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &fpath,
	})
	if err != nil {
		return err
	}

	imgBytes, err := io.ReadAll(object.Body)
	if err != nil {
		return err
	}

	err = mw.ReadImageBlob(imgBytes)
	if err != nil {
		return err
	}

	err = callback(mw)
	if err != nil {
		return err
	}

	blob, err := mw.GetImagesBlob()
	if err != nil {
		return err
	}

	blobBytes := bytes.NewReader(blob)

	contentType, err := imageFormatContentType(mw.GetImageFormat())
	if err != nil {
		return err
	}

	_, err = s3c.PutObject(&s3.PutObjectInput{
		Key:         &fpath,
		Body:        blobBytes,
		Bucket:      &bucket,
		ACL:         &publicRead,
		ContentType: &contentType,
	})
	if err != nil {
		return err
	}

	return s.Flush(ctx, FlushOptions{
		Image: imageID,
	})
}

func (s *Storage) Flop(ctx context.Context, imageID int) error {
	return s.doImagickOperation(ctx, imageID, func(mw *imagick.MagickWand) error {
		return mw.FlopImage()
	})
}

func (s *Storage) Normalize(ctx context.Context, imageID int) error {
	return s.doImagickOperation(ctx, imageID, func(mw *imagick.MagickWand) error {
		return mw.NormalizeImage()
	})
}

func (s *Storage) SetImageCrop(ctx context.Context, imageID int, crop sampler.Crop) error {
	if imageID <= 0 {
		return fmt.Errorf("%w: `%v`", errInvalidImageID, imageID)
	}

	if crop.Left < 0 || crop.Top < 0 || crop.Width <= 0 || crop.Height <= 0 {
		crop.Left = 0
		crop.Top = 0
		crop.Width = 0
		crop.Height = 0
	}

	_, err := s.db.Update(schema.ImageTable).
		Set(goqu.Record{
			schema.ImageTableCropLeftColName:   crop.Left,
			schema.ImageTableCropTopColName:    crop.Top,
			schema.ImageTableCropWidthColName:  crop.Width,
			schema.ImageTableCropHeightColName: crop.Height,
		}).
		Where(schema.ImageTableIDCol.Eq(imageID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	for formatName, format := range s.formats {
		if !format.IsIgnoreCrop() {
			err = s.Flush(ctx, FlushOptions{
				Format: formatName,
				Image:  imageID,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Storage) imageCrop(ctx context.Context, imageID int) (*sampler.Crop, error) {
	var crop sampler.Crop

	success, err := s.db.Select(
		schema.ImageTableCropLeftCol,
		schema.ImageTableCropTopCol,
		schema.ImageTableCropWidthCol,
		schema.ImageTableCropHeightCol,
	).
		From(schema.ImageTable).
		Where(
			schema.ImageTableIDCol.Eq(imageID),
			schema.ImageTableCropWidthCol.Gt(0),
			schema.ImageTableCropHeightCol.Gt(0),
		).ScanStructContext(ctx, &crop)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, sql.ErrNoRows
	}

	return &crop, nil
}

func (s *Storage) images(ctx context.Context, imageIDs []int) (map[int]Image, error) {
	sqSelect := s.db.Select(schema.ImageTableIDCol, schema.ImageTableWidthCol, schema.ImageTableHeightCol,
		schema.ImageTableFilesizeCol, schema.ImageTableFilepathCol, schema.ImageTableDirCol).
		From(schema.ImageTable).
		Where(schema.ImageTableIDCol.In(imageIDs))

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if errors.Is(err, sql.ErrNoRows) {
		return make(map[int]Image), nil
	}

	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	result := make(map[int]Image)

	for rows.Next() {
		var img Image

		err = rows.Scan(&img.id, &img.width, &img.height, &img.filesize, &img.filepath, &img.dir)
		if err != nil {
			return nil, err
		}

		err = s.populateSrc(&img)
		if err != nil {
			return nil, err
		}

		result[img.id] = img
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Storage) FormattedImages(ctx context.Context, imageIDs []int, formatName string) (map[int]Image, error) {
	sqSelect := s.db.Select(
		schema.ImageTableIDCol, schema.ImageTableWidthCol, schema.ImageTableHeightCol, schema.ImageTableFilesizeCol,
		schema.ImageTableFilepathCol, schema.ImageTableDirCol, schema.FormattedImageTableImageIDCol,
	).
		From(schema.ImageTable).
		Join(
			schema.FormattedImageTable,
			goqu.On(schema.ImageTableIDCol.Eq(schema.FormattedImageTableFormattedImageIDCol)),
		).
		Where(
			schema.FormattedImageTableImageIDCol.In(imageIDs),
			schema.FormattedImageTableFormatCol.Eq(formatName),
		)

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if errors.Is(err, sql.ErrNoRows) {
		return make(map[int]Image), nil
	}

	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	result := make(map[int]Image)

	for rows.Next() {
		var (
			img        Image
			srcImageID int
		)

		err = rows.Scan(&img.id, &img.width, &img.height, &img.filesize, &img.filepath, &img.dir, &srcImageID)
		if err != nil {
			return nil, err
		}

		err = s.populateSrc(&img)
		if err != nil {
			return nil, err
		}

		result[srcImageID] = img
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, imageID := range imageIDs {
		_, ok := result[imageID]
		if !ok {
			formattedImageID, err := s.doFormatImage(ctx, imageID, formatName)
			if err != nil {
				return nil, err
			}

			img, err := s.Image(ctx, formattedImageID)
			if err != nil {
				return nil, err
			}

			result[imageID] = *img
		}
	}

	return result, nil
}

func (s *Storage) ListBrokenImages(ctx context.Context, dirName string) error {
	dir := s.dir(dirName)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, dirName)
	}

	var sts []struct {
		Filepath string `db:"filepath"`
	}

	var (
		isLastPage bool
		page       uint
	)

	for !isLastPage {
		err := s.db.Select(schema.ImageTableFilepathCol).
			From(schema.ImageTable).
			Where(schema.ImageTableDirCol.Eq(dirName)).
			Order(schema.ImageTableFilepathCol.Asc()).
			Limit(listBrokenImagesPerPage).
			Offset(page*listBrokenImagesPerPage).
			ScanStructsContext(ctx, &sts)
		if err != nil {
			return err
		}

		s3Client := s.s3Client()
		bucket := dir.Bucket()

		isLastPage = len(sts) < listBrokenImagesPerPage
		page++

		for _, st := range sts {
			_, err := s3Client.HeadObject(&s3.HeadObjectInput{
				Bucket: &bucket,
				Key:    &st.Filepath,
			})
			if err != nil {
				fmt.Println(st.Filepath) //nolint:forbidigo
			}
		}
	}

	return nil
}

func (s *Storage) ListUnlinkedObjects(ctx context.Context, dirName string) error {
	dir := s.dir(dirName)
	if dir == nil {
		return fmt.Errorf("%w: `%s`", errDirNotFound, dirName)
	}

	s3Client := s.s3Client()
	bucket := dir.Bucket()

	err := s3Client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: &bucket,
	}, func(list *s3.ListObjectsOutput, _ bool) bool {
		var id int64

		for _, item := range list.Contents {
			success, err := s.db.Select(schema.ImageTableIDCol).
				From(schema.ImageTable).
				Where(
					schema.ImageTableDirCol.Eq(dirName),
					schema.ImageTableFilepathCol.Eq(*item.Key),
				).
				ScanValContext(ctx, &id)
			if err != nil {
				logrus.Errorf(err.Error())

				return false
			}

			if !success {
				fmt.Println(*item.Key) //nolint:forbidigo
			}
		}

		return true
	})

	return err
}
