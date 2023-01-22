package storage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/doug-martin/goqu/v9"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v2/imagick"

	_ "image/gif"  // GIF support
	_ "image/jpeg" // JPEG support
	_ "image/png"  // PNG support
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

var ErrImageNotFound = errors.New("image not found")

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
	ID         int
	Width      int
	Height     int
	Filesize   int
	Filepath   string
	Dir        string
	CropLeft   int
	CropTop    int
	CropWidth  int
	CropHeight int
}

type formattedImageRow struct {
	ImageID          int
	Format           string
	FormattedImageID int
	Status           int
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
	var r Image
	err := s.db.QueryRowContext(ctx, `
		SELECT id, width, height, filesize, filepath, dir
		FROM image
		WHERE id = ?
	`, id).Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrImageNotFound
	}

	if err != nil {
		return nil, err
	}

	err = s.populateSrc(&r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (s *Storage) populateSrc(r *Image) error {
	dir := s.dir(r.dir)
	if dir == nil {
		return fmt.Errorf("dir '%s' not defined", r.dir)
	}

	bucket := dir.Bucket()

	s3Client := s.s3Client()

	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &r.filepath,
	})
	rest.Build(req)

	url := req.HTTPRequest.URL

	if len(s.config.SrcOverride.Host) > 0 {
		url.Host = s.config.SrcOverride.Host
	}

	if len(s.config.SrcOverride.Scheme) > 0 {
		url.Scheme = s.config.SrcOverride.Scheme
	}

	r.src = url.String()

	return nil
}

func (s *Storage) FormattedImage(ctx context.Context, id int, formatName string) (*Image, error) {
	var r Image

	err := s.db.QueryRowContext(ctx, `
		SELECT image.id, image.width, image.height, image.filesize, image.filepath, image.dir
		FROM image
			INNER JOIN formated_image ON image.id = formated_image.formated_image_id
		WHERE formated_image.image_id = ? AND formated_image.format = ?
	`, id, formatName).Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		err = s.populateSrc(&r)
		if err != nil {
			return nil, err
		}

		return &r, nil
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
	row := s.db.QueryRowContext(ctx, `
		SELECT id, width, height, filepath, dir, crop_left, crop_top, crop_width, crop_height
		FROM image
		WHERE id = ?
	`, imageID)

	var iRow imageRow

	err := row.Scan(
		&iRow.ID, &iRow.Width, &iRow.Height, &iRow.Filepath, &iRow.Dir,
		&iRow.CropLeft, &iRow.CropTop, &iRow.CropWidth, &iRow.CropHeight,
	)
	if err != nil {
		return 0, err
	}

	dir := s.dir(iRow.Dir)
	if dir == nil {
		return 0, fmt.Errorf("dir '%s' not defined", iRow.Dir)
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
		return 0, fmt.Errorf("format `%s` not found", formatName)
	}

	_, err = s.db.ExecContext(
		ctx,
		"INSERT INTO formated_image (format, image_id, status, formated_image_id) VALUES (?, ?, ?, ?)",
		formatName,
		imageID,
		StatusProcessing,
		nil,
	)

	if err != nil {
		var mysqlError *mysql.MySQLError
		ok := errors.Is(err, mysqlError)

		if !ok || mysqlError.Number != 1062 {
			return 0, err
		}

		// wait until done
		logrus.Debug("Wait until image processing done")

		var (
			done  = false
			fiRow formattedImageRow
		)

		for i := 0; i < maxInsertAttempts && !done; i++ {
			var id sql.NullInt32

			err = s.db.QueryRowContext(ctx, "SELECT formated_image_id, status FROM formated_image WHERE image_id = ?", imageID).
				Scan(&id, &fiRow.Status)
			if err != nil {
				return 0, err
			}

			fiRow.FormattedImageID = 0
			if id.Valid {
				fiRow.FormattedImageID = int(id.Int32)
			}

			done = fiRow.Status != StatusProcessing
			if !done {
				time.Sleep(time.Second)
			}
		}

		if !done {
			// mark as failed
			_, err = s.db.ExecContext(
				ctx,
				"UPDATE formated_image SET status = ? WHERE format = ? AND image_id = ? AND status = ?",
				StatusFailed,
				formatName,
				imageID,
				StatusProcessing,
			)
			if err != nil {
				return 0, err
			}
		}

		if fiRow.FormattedImageID == 0 {
			return 0, fmt.Errorf("failed to format image")
		}

		return fiRow.FormattedImageID, nil
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

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE formated_image SET formated_image_id = ?, status = ? WHERE format = ? AND image_id = ?",
		formattedImageID,
		StatusDefault,
		formatName,
		imageID,
	)
	if err != nil {
		return 0, err
	}

	// } catch (Exception $e) {
	_, err = s.db.ExecContext(
		ctx,
		"UPDATE formated_image SET status = ? WHERE format = ? AND image_id = ?",
		StatusFailed,
		formatName,
		imageID,
	)
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
		return 0, fmt.Errorf("failed to get image size (%v x %v)", width, height)
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
		return 0, fmt.Errorf("unsupported image type `%v`", format)
	}

	dir := s.dir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("dir '%v' not defined", dirName)
	}

	blob := mw.GetImagesBlob()

	id, err := s.generateLockWrite(
		ctx,
		dirName,
		options,
		width,
		height,
		func(fileName string) error {
			s3c := s.s3Client()
			r := bytes.NewReader(blob)
			bucket := dir.Bucket()

			contentType, err := imageFormatContentType(mw.GetImageFormat())
			if err != nil {
				return err
			}

			_, err = s3c.PutObject(&s3.PutObjectInput{
				Key:         &fileName,
				Body:        r,
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

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE image SET filesize = ? WHERE id = ?",
		filesize,
		// exif,
		id,
	)
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

	for attemptIndex := 0; attemptIndex < maxInsertAttempts; attemptIndex++ {
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
				res, insertAttemptException = s.db.ExecContext(ctx, `
				    INSERT INTO image (width, height, dir, filesize, filepath, date_add, 
                        crop_left, crop_top, crop_width, crop_height, s3)
				    VALUES (?, ?, ?, 0, ?, NOW(), 0, 0, 0, 0, 1)
			    `,
					width,
					height,
					dirName,
					destFileName,
				)

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
	res, err := s.db.ExecContext(ctx, "UPDATE image_dir SET count = count + 1 WHERE dir = ?", dirName)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected <= 0 {
		_, err = s.db.ExecContext(ctx, "INSERT INTO image_dir (dir, count) VALUES (?, 1)", dirName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) dirCounter(ctx context.Context, dirName string) (int, error) {
	var r int
	err := s.db.QueryRowContext(ctx, "SELECT count FROM image_dir WHERE dir = ?", dirName).Scan(&r)

	return r, err
}

func indexByAttempt(attempt int) int {
	const powBase = 10

	rand.Seed(time.Now().UnixNano())

	float := float64(attempt)
	min := int(math.Pow(powBase, float-1))
	max := int(math.Pow(powBase, float) - 1)

	return rand.Intn(max-min+1) + min //nolint: gosec
}

func (s *Storage) createImagePath(ctx context.Context, dirName string, options GenerateOptions) (string, error) {
	dir := s.dir(dirName)
	if dir == nil {
		return "", fmt.Errorf("dir '%v' not defined", dirName)
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
		return "", fmt.Errorf("unknown format `%s`", format)
	}

	return result, nil
}

func (s *Storage) RemoveImage(ctx context.Context, imageID int) error {
	var r Image

	err := s.db.QueryRowContext(ctx, `
		SELECT id, dir, filepath
		FROM image
		WHERE id = ?
	`, imageID).Scan(&r.id, &r.dir, &r.filepath)
	if err != nil {
		return err
	}

	err = s.Flush(ctx, FlushOptions{
		Image: r.ID(),
	})
	if err != nil {
		return err
	}

	// to save remove formatted image
	_, err = s.db.Exec("DELETE FROM formated_image WHERE formated_image_id = ?", r.ID())
	if err != nil {
		return err
	}

	// important to delete row first
	_, err = s.db.Exec("DELETE FROM image WHERE id = ?", r.ID())
	if err != nil {
		return err
	}

	dir := s.dir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%s' not defined", r.Dir())
	}

	s3c := s.s3Client()

	bucket := dir.Bucket()
	key := r.Filepath()
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
	sqSelect := s.db.Select("image_id, format, formated_image_id").From("formated_image")

	if len(options.Format) > 0 {
		sqSelect = sqSelect.Where(goqu.Ex{"formated_image.format": options.Format})
	}

	if options.Image > 0 {
		sqSelect = sqSelect.Where(goqu.Ex{"formated_image.image_id": options.Image})
	}

	rows, err := sqSelect.Executor().QueryContext(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	defer util.Close(rows)

	for rows.Next() {
		var (
			iID  int
			f    string
			fiID sql.NullInt32
		)

		err = rows.Scan(&iID, &f, &fiID)
		if err != nil {
			return err
		}

		if fiID.Valid && fiID.Int32 > 0 {
			err = s.RemoveImage(ctx, int(fiID.Int32))
			if err != nil {
				return err
			}
		}

		_, err = s.db.Exec("DELETE FROM formated_image WHERE image_id = ? AND format = ?", iID, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) ChangeImageName(ctx context.Context, imageID int, options GenerateOptions) error {
	var r Image

	err := s.db.QueryRowContext(ctx, `
		SELECT id, dir, filepath
		FROM image
		WHERE id = ?
	`, imageID).Scan(&r.id, &r.dir, &r.filepath)
	if err != nil {
		return err
	}

	dir := s.dir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%v' not defined", r.Dir())
	}

	if len(options.Extension) == 0 {
		options.Extension = strings.TrimLeft(filepath.Ext(r.Filepath()), ".")
	}

	var insertAttemptException error

	s3c := s.s3Client()

	for attemptIndex := 0; attemptIndex < maxInsertAttempts; attemptIndex++ {
		options.Index = indexByAttempt(attemptIndex)

		destFileName, err := s.createImagePath(ctx, r.Dir(), options)
		if err != nil {
			return err
		}

		if destFileName == r.Filepath() {
			return fmt.Errorf("trying to rename to self")
		}

		_, insertAttemptException = s.db.ExecContext(
			ctx,
			"UPDATE image SET filepath = ? WHERE id = ?",
			destFileName, r.id,
		)

		if insertAttemptException == nil {
			bucket := dir.Bucket()
			copySource := dir.Bucket() + "/" + r.Filepath()

			_, err = s3c.CopyObject(&s3.CopyObjectInput{
				Bucket:     &bucket,
				CopySource: &copySource,
				Key:        &destFileName,
				ACL:        &publicRead,
			})
			if err != nil {
				return err
			}

			fpath := r.Filepath()

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
		return 0, fmt.Errorf("failed to get image size of '$file' (%v x %v)", imageInfo.Width, imageInfo.Height)
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
			return 0, fmt.Errorf("unsupported image type `%v`", imageType)
		}

		options.Extension = ext
	}

	dir := s.dir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("dir '%v' not defined", dirName)
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

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE image SET filesize = ? WHERE id = ?",
		fi.Size(),
		// exif,
		id,
	)
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
	var r Image

	err := s.db.QueryRowContext(ctx, `
		SELECT dir, filepath
		FROM image
		WHERE id = ?
	`, imageID).Scan(&r.dir, &r.filepath)
	if err != nil {
		return err
	}

	dir := s.dir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%v' not defined", r.Dir())
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	s3c := s.s3Client()

	bucket := dir.Bucket()
	fpath := r.Filepath()

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

	b := bytes.NewReader(mw.GetImagesBlob())

	contentType, err := imageFormatContentType(mw.GetImageFormat())
	if err != nil {
		return err
	}

	_, err = s3c.PutObject(&s3.PutObjectInput{
		Key:         &fpath,
		Body:        b,
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
		return fmt.Errorf("invalid image id provided `%v`", imageID)
	}

	if crop.Left < 0 || crop.Top < 0 || crop.Width <= 0 || crop.Height <= 0 {
		crop.Left = 0
		crop.Top = 0
		crop.Width = 0
		crop.Height = 0
	}

	_, err := s.db.Exec(
		"UPDATE image SET crop_left = ?, crop_top = ?, crop_width = ?, crop_height = ? WHERE id = ?",
		crop.Left, crop.Top, crop.Width, crop.Height, imageID,
	)
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

	err := s.db.QueryRowContext(ctx, `
		SELECT crop_left, crop_top, crop_width, crop_height
		FROM image
		WHERE id = ? AND crop_width > 0 and crop_height > 0
	`, imageID).Scan(&crop.Left, &crop.Top, &crop.Width, &crop.Height)
	if err != nil {
		return nil, err
	}

	return &crop, nil
}

func (s *Storage) images(ctx context.Context, imageIds []int) (map[int]Image, error) {
	sqSelect := s.db.Select("id, width, height, filesize, filepath, dir").
		From("image").
		Where(goqu.Ex{"id": imageIds})

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return make(map[int]Image), nil
	}

	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	result := make(map[int]Image)

	for rows.Next() {
		var r Image

		err = rows.Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir)
		if err != nil {
			return nil, err
		}

		err = s.populateSrc(&r)
		if err != nil {
			return nil, err
		}

		result[r.id] = r
	}

	return result, nil
}

func (s *Storage) FormattedImages(ctx context.Context, imageIds []int, formatName string) (map[int]Image, error) {
	sqSelect := s.db.Select(
		"image.id, image.width, image.height, image.filesize, image.filepath, image.dir, formated_image.image_id",
	).
		From("image").
		Join(goqu.T("formated_image"), goqu.On(goqu.Ex{"image.id": "formated_image.formated_image_id"})).
		Where(goqu.Ex{
			"formated_image.image_id": imageIds,
			"formated_image.format":   formatName,
		})

	rows, err := sqSelect.Executor().QueryContext(ctx)
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
			r          Image
			srcImageID int
		)

		err = rows.Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir, &srcImageID)
		if err != nil {
			return nil, err
		}

		err = s.populateSrc(&r)
		if err != nil {
			return nil, err
		}

		result[srcImageID] = r
	}

	for _, imageID := range imageIds {
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
