package storage

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/util"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/gographics/imagick.v2/imagick"
	"io"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusDefault    int = 0
	StatusProcessing int = 1
	StatusFailed     int = 2
)

const maxInsertAttempts = 15

const defaultExtension = "jpg"

var formats2ContentType = map[string]string{
	"GIF": "image/gif",
	"PNG": "image/png",
	"JPG": "image/jpeg",
}

type Storage struct {
	config               StorageConfig
	db                   *sql.DB
	dirs                 map[string]*Dir
	formats              map[string]*sampler.Format
	formatedImageDirName string
	sampler              *sampler.Sampler
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

type formatedImageRow struct {
	ID              int
	Format          string
	FormatedImageID int
	Status          int
}

func NewStorage(db *sql.DB, config StorageConfig) (*Storage, error) {
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
		config:               config,
		db:                   db,
		dirs:                 dirs,
		formats:              formats,
		formatedImageDirName: "format",
		sampler:              sampler.NewSampler(),
	}, nil
}

func (s *Storage) GetImage(id int) (*Image, error) {
	row := s.db.QueryRow(`
		SELECT id, width, height, filesize, filepath
		FROM image
		WHERE id = ?
	`, id)

	var r Image
	err := row.Scan(&r.id, &r.width, &r.height, &r.filesize)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (s *Storage) GetFormatedImage(id int, formatName string) (*Image, error) {

	rows, err := s.db.Query(`
		SELECT image.id, image.width, image.height, image.filesize, image.filepath
		FROM image
			INNER JOIN formated_image ON image.id = formated_image.formated_image_id
		WHERE formated_image.image_id = ? AND formated_image.format = ?
	`, id, formatName)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	for rows.Next() {
		var r Image
		err = rows.Scan(&r.id, &r.width, &r.height, &r.filesize)
		if err != nil {
			return nil, err
		}

		return &r, nil
	}

	formatedImageId, err := s.doFormatImage(id, formatName)
	if err != nil {
		return nil, err
	}
	return s.GetImage(formatedImageId)
}

func (s *Storage) getDir(dirName string) *Dir {
	dir, ok := s.dirs[dirName]
	if ok {
		return dir
	}
	return nil
}

func (s *Storage) getS3Client() (*s3.S3, error) {
	sess := session.Must(session.NewSession())
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
	// find source image
	row := s.db.QueryRow(`
		SELECT id, width, height, filepath, dir, crop_left, crop_top, crop_width, crop_height
		FROM image
		WHERE id = ?
	`, imageId)

	var imageRow imageRow
	err := row.Scan(&imageRow.ID, &imageRow.Width, &imageRow.Height, &imageRow.Filepath, &imageRow.Dir, &imageRow.CropLeft, &imageRow.CropTop, &imageRow.CropWidth, &imageRow.CropHeight)
	if err != nil {
		return 0, err
	}

	dir := s.getDir(imageRow.Dir)
	if dir == nil {
		return 0, fmt.Errorf("dir '%s' not defined", imageRow.Dir)
	}

	bucket := dir.Bucket()
	s3Client, err := s.getS3Client()
	if err != nil {
		return 0, err
	}
	object, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &imageRow.Filepath,
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
		done := false
		var formatedImageRow formatedImageRow

		for i := 0; i < maxInsertAttempts && !done; i++ {
			row := s.db.QueryRow("SELECT id, status FROM formated_image WHERE id = ?", imageId)
			err := row.Scan(&formatedImageRow.ID, &formatedImageRow.Status)
			if err != nil {
				return 0, err
			}

			done = formatedImageRow.Status != StatusProcessing
			if !done {
				time.Sleep(time.Second)
			}
		}

		if !done {
			// mark as failed
			_, err := s.db.Exec(
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

		if formatedImageRow.ID == 0 {
			return 0, fmt.Errorf("failed to format image")
		}

		return formatedImageRow.FormatedImageID, nil
	}

	var formatedImageId int
	// try {
	// $crop = $this->getRowCrop($imageRow);

	cropSuffix := getCropSuffix(imageRow)

	crop := sampler.Crop{
		Left:   imageRow.CropLeft,
		Top:    imageRow.CropTop,
		Width:  imageRow.CropWidth,
		Height: imageRow.CropHeight,
	}

	mw, err = s.sampler.ConvertImage(mw, crop, *format)

	/*foreach ($cFormat->getProcessors() as $processorName) {
		$processor = $this->processors->get($processorName);
		$processor->process($imagick);
	}*/

	// store result
	newPath := strings.Join([]string{
		imageRow.Dir,
		formatName,
		imageRow.Filepath,
	}, "/")
	formatExt, err := format.FormatExtension()
	if err != nil {
		return 0, err
	}
	extension := formatExt
	if formatExt == "" {
		extension = strings.TrimLeft(filepath.Ext(newPath), ".")
	}
	formatedImageId, err = s.addImageFromImagick(
		mw,
		s.formatedImageDirName,
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
		formatedImageId,
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

	return formatedImageId, nil
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
			acl := "public-read"

			contentType, err := imageFormatContentType(mw.GetImageFormat())
			if err != nil {
				return err
			}

			_, err = s3c.PutObject(&s3.PutObjectInput{
				Key:         &fileName,
				Body:        r,
				Bucket:      &bucket,
				ACL:         &acl,
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
	for attemptIndex := 0; attemptIndex >= maxInsertAttempts; attemptIndex++ {
		insertAttemptException = s.incDirCounter(dirName)

		if insertAttemptException == nil {
			opt := options
			opt.Index = indexByAttempt(attemptIndex)
			var destFileName string
			var res sql.Result

			destFileName, insertAttemptException = s.createImagePath(dirName, opt)

			// store to db
			res, insertAttemptException = s.db.Exec(
				`
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

	row := s.db.QueryRow("SELECT count FROM image_dir WHERE dir = ?", dirName)

	var r int
	err := row.Scan(&r)

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
	result, ok := formats2ContentType[format]
	if !ok {
		return "", fmt.Errorf("unknown format `%s`", format)
	}

	return result, nil
}
