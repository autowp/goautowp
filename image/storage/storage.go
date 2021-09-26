package storage

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/config"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
	"github.com/sirupsen/logrus"
	"image"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/gographics/imagick.v2/imagick"
)

import _ "image/png"
import _ "image/jpeg"
import _ "image/gif"

const (
	StatusDefault    int = 0
	StatusProcessing int = 1
	StatusFailed     int = 2
)

const maxInsertAttempts = 15

const defaultExtension = "jpg"

var publicRead = "public-read"

var formats2ContentType = map[string]string{
	"GIF":  "image/gif",
	"PNG":  "image/png",
	"JPG":  "image/jpeg",
	"JPEG": "image/jpeg",
}

type Storage struct {
	config                config.ImageStorageConfig
	db                    *sql.DB
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

func NewStorage(db *sql.DB, config config.ImageStorageConfig) (*Storage, error) {
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

func (s *Storage) GetImage(id int) (*Image, error) {
	logrus.Debugf("GetImage %d", id)
	var r Image
	err := s.db.QueryRow(`
		SELECT id, width, height, filesize, filepath, dir
		FROM image
		WHERE id = ?
	`, id).Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir)
	if err == sql.ErrNoRows {
		return nil, nil
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

	dir := s.getDir(r.dir)
	if dir == nil {
		return fmt.Errorf("dir '%s' not defined", r.dir)
	}

	bucket := dir.Bucket()

	s3Client, err := s.getS3Client()
	if err != nil {
		return err
	}

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

func (s *Storage) GetFormattedImage(id int, formatName string) (*Image, error) {
	logrus.Debugf("GetFormattedImage %d, %s", id, formatName)
	var r Image
	err := s.db.QueryRow(`
		SELECT image.id, image.width, image.height, image.filesize, image.filepath, image.dir
		FROM image
			INNER JOIN formated_image ON image.id = formated_image.formated_image_id
		WHERE formated_image.image_id = ? AND formated_image.format = ?
	`, id, formatName).Scan(&r.id, &r.width, &r.height, &r.filesize, &r.filepath, &r.dir)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == nil {

		err = s.populateSrc(&r)
		if err != nil {
			return nil, err
		}

		return &r, nil
	}

	formattedImageId, err := s.doFormatImage(id, formatName)
	if err != nil {
		return nil, err
	}
	return s.GetImage(formattedImageId)
}

func (s *Storage) getDir(dirName string) *Dir {
	dir, ok := s.dirs[dirName]
	if ok {
		return dir
	}
	return nil
}

func (s *Storage) getS3Client() (*s3.S3, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           &s.config.S3.Region,
		Endpoint:         &s.config.S3.Endpoint,
		S3ForcePathStyle: &s.config.S3.UsePathStyleEndpoint,
		Credentials:      credentials.NewStaticCredentials(s.config.S3.Credentials.Key, s.config.S3.Credentials.Secret, ""),
	}))
	svc := s3.New(sess)

	return svc, nil
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

func (s *Storage) doFormatImage(imageId int, formatName string) (int, error) {
	logrus.Debugf("doFormatImage %d, %s", imageId, formatName)
	// find source image
	row := s.db.QueryRow(`
		SELECT id, width, height, filepath, dir, crop_left, crop_top, crop_width, crop_height
		FROM image
		WHERE id = ?
	`, imageId)

	var iRow imageRow
	err := row.Scan(&iRow.ID, &iRow.Width, &iRow.Height, &iRow.Filepath, &iRow.Dir, &iRow.CropLeft, &iRow.CropTop, &iRow.CropWidth, &iRow.CropHeight)
	if err != nil {
		return 0, err
	}

	dir := s.getDir(iRow.Dir)
	if dir == nil {
		return 0, fmt.Errorf("dir '%s' not defined", iRow.Dir)
	}

	bucket := dir.Bucket()
	s3Client, err := s.getS3Client()
	if err != nil {
		return 0, err
	}
	object, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &iRow.Filepath,
	})

	if err != nil {
		return 0, err
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
	format := s.getFormat(formatName)
	if format == nil {
		return 0, fmt.Errorf("format `%s` not found", formatName)
	}

	_, err = s.db.Exec(
		"INSERT INTO formated_image (format, image_id, status, formated_image_id) VALUES (?, ?, ?, ?)",
		formatName,
		imageId,
		StatusProcessing,
		nil,
	)

	if err != nil {
		mysqlError, ok := err.(*mysql.MySQLError)
		if !ok || mysqlError.Number != 1062 {
			return 0, err
		}

		// wait until done
		logrus.Debug("Wait until image processing done")
		done := false
		var fiRow formattedImageRow

		for i := 0; i < maxInsertAttempts && !done; i++ {
			var id sql.NullInt32
			err = s.db.QueryRow("SELECT formated_image_id, status FROM formated_image WHERE image_id = ?", imageId).
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
			_, err = s.db.Exec(
				"UPDATE formated_image SET status = ? WHERE format = ? AND image_id = ? AND status = ?",
				StatusFailed,
				formatName,
				imageId,
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

	var formattedImageId int
	// try {
	// $crop = $this->getRowCrop(iRow);

	cropSuffix := getCropSuffix(iRow)

	crop := sampler.Crop{
		Left:   iRow.CropLeft,
		Top:    iRow.CropTop,
		Width:  iRow.CropWidth,
		Height: iRow.CropHeight,
	}

	b := mw.GetImagesBlob()
	fmt.Printf("BLOB IS %v\n", len(b))
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
	b2 := mw.GetImagesBlob()
	fmt.Printf("BLOB2 IS %v\n", len(b2))
	formattedImageId, err = s.addImageFromImagick(
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

	_, err = s.db.Exec(
		"UPDATE formated_image SET formated_image_id = ?, status = ? WHERE format = ? AND image_id = ?",
		formattedImageId,
		StatusDefault,
		formatName,
		imageId,
	)
	if err != nil {
		return 0, err
	}

	// } catch (Exception $e) {
	_, err = s.db.Exec(
		"UPDATE formated_image SET status = ? WHERE format = ? AND image_id = ?",
		StatusFailed,
		formatName,
		imageId,
	)
	if err != nil {
		return 0, err
	}

	// throw $e;
	// }

	return formattedImageId, nil
}

func (s *Storage) getFormat(name string) *sampler.Format {
	format, ok := s.formats[name]
	if ok {
		return format
	}
	return nil
}

func (s *Storage) addImageFromImagick(mw *imagick.MagickWand, dirName string, options GenerateOptions) (int, error) {
	width := int(mw.GetImageWidth())
	height := int(mw.GetImageHeight())

	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("failed to get image size (%v x %v)", width, height)
	}

	format := mw.GetImageFormat()

	switch strings.ToLower(format) {
	case "gif":
		options.Extension = "gif"
	case "jpeg":
		options.Extension = "jpg"
	case "png":
		options.Extension = "png"
	default:
		return 0, fmt.Errorf("unsupported image type `%v`", format)
	}

	dir := s.getDir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("dir '%v' not defined", dirName)
	}

	blob := mw.GetImagesBlob()
	id, err := s.generateLockWrite(
		dirName,
		options,
		width,
		height,
		func(fileName string) error {
			s3c, err := s.getS3Client()
			if err != nil {
				return err
			}
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

	_, err = s.db.Exec(
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

func (s *Storage) generateLockWrite(dirName string, options GenerateOptions, width int, height int, callback func(string) error) (int, error) {
	var insertAttemptException error
	imageId := 0
	for attemptIndex := 0; attemptIndex < maxInsertAttempts; attemptIndex++ {
		insertAttemptException = s.incDirCounter(dirName)

		if insertAttemptException == nil {
			opt := options
			opt.Index = indexByAttempt(attemptIndex)
			var destFileName string
			var res sql.Result

			destFileName, insertAttemptException = s.createImagePath(dirName, opt)
			if insertAttemptException == nil {
				// store to db
				res, insertAttemptException = s.db.Exec(`
				INSERT INTO image (width, height, dir, filesize, filepath, date_add, crop_left, crop_top, crop_width, crop_height, s3)
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

						imageId = int(id)
					}
				}
			}
		}

		if insertAttemptException == nil {
			break
		}
	}

	return imageId, insertAttemptException
}

func (s *Storage) incDirCounter(dirName string) error {
	res, err := s.db.Exec("UPDATE image_dir SET count = count + 1 WHERE dir = ?", dirName)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected <= 0 {
		_, err = s.db.Exec("INSERT INTO image_dir (dir, count) VALUES (?, 1)", dirName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) getDirCounter(dirName string) (int, error) {
	var r int
	err := s.db.QueryRow("SELECT count FROM image_dir WHERE dir = ?", dirName).Scan(&r)

	return r, err
}

func indexByAttempt(attempt int) int {
	rand.Seed(time.Now().UnixNano())
	float := float64(attempt)
	min := int(math.Pow(10, float-1))
	max := int(math.Pow(10, float) - 1)

	return rand.Intn(max-min+1) + min
}

func (s *Storage) createImagePath(dirName string, options GenerateOptions) (string, error) {
	dir := s.getDir(dirName)
	if dir == nil {
		return "", fmt.Errorf("dir '%v' not defined", dirName)
	}

	namingStrategy := dir.NamingStrategy()

	c, err := s.getDirCounter(dirName)
	if err != nil {
		return "", err
	}
	options.Count = c

	if len(options.Extension) <= 0 {
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

func (s *Storage) RemoveImage(imageId int) error {
	var r Image
	err := s.db.QueryRow(`
		SELECT id, dir, filepath
		FROM image
		WHERE id = ?
	`, imageId).Scan(&r.id, &r.dir, &r.filepath)

	if err != nil {
		return err
	}

	err = s.Flush(FlushOptions{
		Image: r.Id(),
	})
	if err != nil {
		return err
	}

	// to save remove formatted image
	_, err = s.db.Exec("DELETE FROM formated_image WHERE formated_image_id = ?", r.Id())
	if err != nil {
		return err
	}

	// important to delete row first
	_, err = s.db.Exec("DELETE FROM image WHERE id = ?", r.Id())
	if err != nil {
		return err
	}

	dir := s.getDir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%s' not defined", r.Dir())
	}

	s3c, err := s.getS3Client()
	if err != nil {
		return err
	}

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

func (s *Storage) Flush(options FlushOptions) error {

	sqSelect := sq.Select("image_id, format, formated_image_id").From("formated_image")

	if len(options.Format) > 0 {
		sqSelect = sqSelect.Where(sq.Eq{"formated_image.format": options.Format})
	}

	if options.Image > 0 {
		sqSelect = sqSelect.Where(sq.Eq{"formated_image.image_id": options.Image})
	}

	rows, err := sqSelect.RunWith(s.db).Query()
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	defer util.Close(rows)

	for rows.Next() {
		var iID int
		var f string
		var fiID sql.NullInt32
		err = rows.Scan(&iID, &f, &fiID)
		if err != nil {
			return err
		}

		if fiID.Valid && fiID.Int32 > 0 {
			err = s.RemoveImage(int(fiID.Int32))
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

func (s *Storage) ChangeImageName(imageId int, options GenerateOptions) error {
	var r Image
	err := s.db.QueryRow(`
		SELECT id, dir, filepath
		FROM image
		WHERE id = ?
	`, imageId).Scan(&r.id, &r.dir, &r.filepath)

	if err != nil {
		return err
	}

	dir := s.getDir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%v' not defined", r.Dir())
	}

	if len(options.Extension) <= 0 {
		options.Extension = strings.TrimLeft(filepath.Ext(r.Filepath()), ".")
	}

	var insertAttemptException error

	s3c, err := s.getS3Client()
	if err != nil {
		return err
	}

	for attemptIndex := 0; attemptIndex < maxInsertAttempts; attemptIndex++ {
		options.Index = indexByAttempt(attemptIndex)
		destFileName, err := s.createImagePath(r.Dir(), options)
		if err != nil {
			return nil
		}

		if destFileName == r.Filepath() {
			return fmt.Errorf("trying to rename to self")
		}

		_, insertAttemptException = s.db.Exec("UPDATE image SET filepath = ? WHERE id = ?", destFileName, r.id)

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

func (s *Storage) AddImageFromFile(file string, dirName string, options GenerateOptions) (int, error) {
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

	if len(options.Extension) <= 0 {
		var ext string
		switch imageType {
		case "gif":
			ext = "gif"
		case "jpeg":
			ext = "jpg"
		case "png":
			ext = "png"
		default:
			return 0, fmt.Errorf("unsupported image type `%v`", imageType)
		}
		options.Extension = ext
	}

	dir := s.getDir(dirName)
	if dir == nil {
		return 0, fmt.Errorf("dir '%v' not defined", dirName)
	}

	id, err := s.generateLockWrite(
		dirName,
		options,
		imageInfo.Width,
		imageInfo.Height,
		func(fileName string) error {
			s3c, err := s.getS3Client()
			if err != nil {
				return err
			}
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

			_, err = s3c.PutObject(&s3.PutObjectInput{
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

	_, err = s.db.Exec(
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

func (s *Storage) AddImageFromBlob(blob []byte, dirName string, options GenerateOptions) (int, error) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	err := mw.ReadImageBlob(blob)
	if err != nil {
		return 0, err
	}

	id, err := s.addImageFromImagick(mw, dirName, options)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Storage) Flop(imageId int) error {
	var r Image
	err := s.db.QueryRow(`
		SELECT dir, filepath
		FROM image
		WHERE id = ?
	`, imageId).Scan(&r.dir, &r.filepath)

	if err != nil {
		return err
	}

	dir := s.getDir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%v' not defined", r.Dir())
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	s3c, err := s.getS3Client()
	if err != nil {
		return err
	}
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

	// format
	err = mw.FlopImage()
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

	return s.Flush(FlushOptions{
		Image: imageId,
	})
}

func (s *Storage) Normalize(imageId int) error {
	var r Image
	err := s.db.QueryRow(`
		SELECT dir, filepath
		FROM image
		WHERE id = ?
	`, imageId).Scan(&r.dir, &r.filepath)

	if err != nil {
		return err
	}

	dir := s.getDir(r.Dir())
	if dir == nil {
		return fmt.Errorf("dir '%v' not defined", r.Dir())
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	s3c, err := s.getS3Client()
	if err != nil {
		return err
	}
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

	// format
	err = mw.NormalizeImage()
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

	return s.Flush(FlushOptions{
		Image: imageId,
	})
}

func (s *Storage) SetImageCrop(imageId int, crop sampler.Crop) error {
	if imageId <= 0 {
		return fmt.Errorf("invalid image id provided `%v`", imageId)
	}

	if crop.Left < 0 || crop.Top < 0 || crop.Width <= 0 || crop.Height <= 0 {
		crop.Left = 0
		crop.Top = 0
		crop.Width = 0
		crop.Height = 0
	}

	_, err := s.db.Exec(
		"UPDATE image SET crop_left = ?, crop_top = ?, crop_width = ?, crop_height = ? WHERE id = ?",
		crop.Left, crop.Top, crop.Width, crop.Height, imageId,
	)
	if err != nil {
		return err
	}

	for formatName, format := range s.formats {
		if !format.IsIgnoreCrop() {
			err = s.Flush(FlushOptions{
				Format: formatName,
				Image:  imageId,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Storage) GetImageCrop(imageId int) (*sampler.Crop, error) {
	var crop sampler.Crop
	err := s.db.QueryRow(`
		SELECT crop_left, crop_top, crop_width, crop_height
		FROM image
		WHERE id = ?
	`, imageId).Scan(&crop.Left, &crop.Top, &crop.Width, &crop.Height)

	if err != nil {
		return nil, err
	}

	if crop.Width <= 0 || crop.Height <= 0 {
		return nil, nil
	}

	return &crop, nil
}

func (s *Storage) GetImages(imageIds []int) (map[int]Image, error) {
	sqSelect := sq.Select("id, width, height, filesize, filepath, dir").
		From("image").
		Where(sq.Eq{"id": imageIds})

	rows, err := sqSelect.RunWith(s.db).Query()
	if err == sql.ErrNoRows {
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

func (s *Storage) GetFormattedImages(imageIds []int, formatName string) (map[int]Image, error) {

	sqSelect := sq.Select("image.id, image.width, image.height, image.filesize, image.filepath, image.dir, formated_image.image_id").
		From("image").
		Join("formated_image ON image.id = formated_image.formated_image_id").
		Where(sq.Eq{"formated_image.image_id": imageIds}).
		Where(sq.Eq{"formated_image.format": formatName})

	rows, err := sqSelect.RunWith(s.db).Query()
	if err == sql.ErrNoRows {
		return make(map[int]Image), nil
	}
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	result := make(map[int]Image)

	for rows.Next() {
		var r Image
		var srcImageID int
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

	for _, imageId := range imageIds {
		_, ok := result[imageId]
		if !ok {
			formattedImageId, err := s.doFormatImage(imageId, formatName)
			if err != nil {
				return nil, err
			}
			img, err := s.GetImage(formattedImageId)
			if err != nil {
				return nil, err
			}
			result[imageId] = *img
		}
	}

	return result, nil
}
