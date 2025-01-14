package goautowp

import (
	"context"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

type DfDistanceExtractor struct {
	picturesRepository *pictures.Repository
	pictureExtractor   *PictureExtractor
}

func NewDfDistanceExtractor(
	picturesRepository *pictures.Repository, pictureExtractor *PictureExtractor,
) *DfDistanceExtractor {
	return &DfDistanceExtractor{
		picturesRepository: picturesRepository,
		pictureExtractor:   pictureExtractor,
	}
}

func (s *DfDistanceExtractor) ExtractRows(
	ctx context.Context, rows []*schema.DfDistanceRow, fields *DfDistanceFields, lang string, isModer bool, userID int64,
	role string,
) ([]*DfDistance, error) {
	result := make([]*DfDistance, 0, len(rows))

	for _, row := range rows {
		var dstPicture *Picture

		dstPictureRequest := fields.GetDstPicture()
		if row.DstPictureID != 0 && dstPictureRequest != nil {
			dstPictureFields := dstPictureRequest.GetFields()

			dstPictureOptions, err := convertPictureListOptions(dstPictureRequest.GetOptions())
			if err != nil {
				return nil, err
			}

			if dstPictureOptions == nil {
				dstPictureOptions = &query.PictureListOptions{}
			}

			dstPictureOptions.ID = row.DstPictureID

			picRow, err := s.picturesRepository.Picture(
				ctx,
				dstPictureOptions,
				convertPictureFields(dstPictureFields),
				convertPicturesOrder(dstPictureRequest.GetOrder()),
			)
			if err != nil {
				return nil, err
			}

			dstPicture, err = s.pictureExtractor.Extract(ctx, picRow, dstPictureFields, lang, isModer, userID, role)
			if err != nil {
				return nil, err
			}
		}

		result = append(result, &DfDistance{
			SrcPictureId: row.SrcPictureID,
			DstPictureId: row.DstPictureID,
			Distance:     int32(row.Distance), //nolint: gosec
			DstPicture:   dstPicture,
		})
	}

	return result, nil
}
