package goautowp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v3/imagick"
)

const (
	iconFormat                = "brandicon"
	brandsSpriteImageFilename = "brands.avif"
	brandsSpriteCSSFilename   = "brands.css"
)

var (
	errBadStatus       = errors.New("bad status")
	errEmptyMagickWand = errors.New("empty magickWand returned")
	errNoBrands        = errors.New("no brands found to generate sprite")
)

func downloadFile(ctx context.Context, filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer util.Close(out)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req) //nolint: bodyclose
	if err != nil {
		return err
	}

	defer util.Close(resp.Body)

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s", errBadStatus, resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}

func createIconsSprite(
	ctx context.Context, repository *items.Repository, imageStorage *storage.Storage,
	fileStorageConfig config.FileStorageConfig,
) error {
	list, _, err := repository.List(ctx, &query.ItemListOptions{
		TypeID:   []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		HasLogo:  true,
		Language: "en",
	}, nil, items.OrderByName, false)
	if err != nil {
		return err
	}

	count := len(list)

	if count == 0 {
		return errNoBrands
	}

	format := imageStorage.Format(iconFormat)

	tmpDir, err := os.MkdirTemp("", "brands-sprite")
	if err != nil {
		return err
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	var pw *imagick.PixelWand

	if format.Background() != "" {
		pw = imagick.NewPixelWand()
		defer pw.Destroy()

		pw.SetColor(format.Background())

		err = mw.SetBackgroundColor(pw)
		if err != nil {
			return err
		}
	}

	index := 0
	css := make([]string, 0, len(list))

	width := int(math.Ceil(math.Sqrt(float64(count))))
	if width <= 0 {
		width = 1
	}

	for _, brand := range list {
		logrus.Infof("Processing `%s`", brand.Catname.String)

		if !brand.LogoID.Valid {
			continue
		}

		img, err := imageStorage.FormattedImage(ctx, int(brand.LogoID.Int64), iconFormat)
		if err != nil {
			return err
		}

		catname := strings.ReplaceAll(brand.Catname.String, ".", "_")
		path := tmpDir + "/" + catname + ".png"

		err = downloadFile(ctx, path, img.Src())
		if err != nil {
			return err
		}

		mwi := imagick.NewMagickWand()

		err = mwi.ReadImage(path)
		if err != nil {
			mwi.Destroy()

			return err
		}

		err = mw.AddImage(mwi)
		if err != nil {
			mwi.Destroy()

			return err
		}

		mwi.Destroy()

		top := index / width
		left := index - top*width
		css = append(css, fmt.Sprintf(
			".brandicon.brandicon-%s {background-position: -%dpx -%dpx}",
			catname,
			1+(format.Width()+1+1)*left,
			1+(format.Height()+1+1)*top,
		))
		index++
	}

	destImg := tmpDir + "/" + brandsSpriteImageFilename

	logrus.Info("Montage ...")

	dw := imagick.NewDrawingWand()
	defer dw.Destroy()

	mwr := mw.MontageImage(dw, fmt.Sprintf("%dx", width), "+1+1", imagick.MONTAGE_MODE_UNDEFINED, "0x0+0+0")
	if mwr == nil {
		return errEmptyMagickWand
	}

	err = mwr.WriteImage(destImg)
	if err != nil {
		return err
	}

	logrus.Info("Upload results ...")

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           &fileStorageConfig.S3.Region,
		Endpoint:         &fileStorageConfig.S3.Endpoint,
		S3ForcePathStyle: &fileStorageConfig.S3.UsePathStyleEndpoint,
		Credentials: credentials.NewStaticCredentials(
			fileStorageConfig.S3.Credentials.Key, fileStorageConfig.S3.Credentials.Secret, "",
		),
	}))
	svc := s3.New(sess)

	var (
		publicRead     = "public-read"
		imageKey       = brandsSpriteImageFilename
		cssKey         = brandsSpriteCSSFilename
		cssContentType = "text/css"
	)

	imageContentType, err := sampler.ContentTypeByFilepath(brandsSpriteImageFilename)
	if err != nil {
		return err
	}

	handle, err := os.Open(destImg)
	if err != nil {
		return err
	}

	defer util.Close(handle)

	_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Key:         &imageKey,
		Body:        handle,
		Bucket:      &fileStorageConfig.Bucket,
		ACL:         &publicRead,
		ContentType: &imageContentType,
	})
	if err != nil {
		return err
	}

	_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Key:         &cssKey,
		Body:        bytes.NewReader([]byte(strings.Join(css, "\n"))),
		Bucket:      &fileStorageConfig.Bucket,
		ACL:         &publicRead,
		ContentType: &cssContentType,
	})
	if err != nil {
		return err
	}

	logrus.Info("Brands sprite uploaded")

	return nil
}
