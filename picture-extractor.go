package goautowp

import (
	"context"

	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"google.golang.org/genproto/googleapis/type/latlng"
)

type PictureExtractor struct {
	imageStorage *storage.Storage
}

func NewPictureExtractor(imageStorage *storage.Storage) *PictureExtractor {
	return &PictureExtractor{
		imageStorage: imageStorage,
	}
}

func (s *PictureExtractor) Extract(
	ctx context.Context, row *schema.PictureRow, fields *PictureFields,
) (*Picture, error) {
	result := &Picture{
		Id:     row.ID,
		Width:  uint32(row.Width),
		Height: uint32(row.Height),
	}

	if row.Point.Valid {
		result.Point = &latlng.LatLng{
			Latitude:  row.Point.Point.Lat(),
			Longitude: row.Point.Point.Lng(),
		}
	}

	if fields.GetImage() {
		if row.ImageID.Valid {
			image, err := s.imageStorage.Image(ctx, int(row.ImageID.Int64))
			if err != nil {
				return nil, err
			}

			result.Image = APIImageToGRPC(image)
		}
	}

	return result, nil
}
