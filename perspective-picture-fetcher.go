package goautowp

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

const largeFormatThreshold = 30

type PerspectivePictureFetcher struct {
	picturesRepository *pictures.Repository
	perspectiveCache   map[int32][]int32
}

type PerspectivePictureFetcherResult struct {
	LargeFormat   bool
	Pictures      []*PerspectivePictureFetcherResultPicture
	TotalPictures int32
}

type PerspectivePictureFetcherResultPicture struct {
	IsVehicleHood bool
	Row           *schema.PictureRow
}

type PerspectivePictureFetcherOptions struct {
	PerspectivePageID     int32
	PerspectiveID         int32
	PictureItemTypeID     schema.PictureItemType
	ContainsPerspectiveID int32
	OnlyExactlyPictures   bool
}

type selectOptions struct {
	perspectiveGroup int32
	exclude          []int64
}

func NewPerspectivePictureFetcher(picturesRepository *pictures.Repository) *PerspectivePictureFetcher {
	return &PerspectivePictureFetcher{
		picturesRepository: picturesRepository,
		perspectiveCache:   make(map[int32][]int32),
	}
}

func (s *PerspectivePictureFetcher) Fetch(
	ctx context.Context, item schema.ItemRow, options PerspectivePictureFetcherOptions,
) (*PerspectivePictureFetcherResult, error) {
	totalPictures, err := s.totalPictures(ctx, item.ID, options)
	if err != nil {
		return nil, err
	}

	var (
		rows           = make([]*schema.PictureRow, 0)
		usedIDs        = make([]int64, 0)
		useLargeFormat bool
		pPageID        int32
	)

	if options.PerspectivePageID > 0 {
		pPageID = options.PerspectivePageID
	} else {
		useLargeFormat = totalPictures > largeFormatThreshold
		if useLargeFormat {
			pPageID = 5
		} else {
			pPageID = 4
		}
	}

	perspectiveGroupIDs, err := s.perspectiveGroupIDs(ctx, pPageID)
	if err != nil {
		return nil, err
	}

	for _, groupID := range perspectiveGroupIDs {
		sqSelect := s.pictureSelect(item.ID, options, selectOptions{
			perspectiveGroup: groupID,
			exclude:          usedIDs,
		})

		picture, err := s.picturesRepository.Picture(ctx, sqSelect, nil, pictures.OrderByPerspectivesGroupPerspectives)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		if picture != nil {
			rows = append(rows, picture)
			usedIDs = append(usedIDs, picture.ID)
		} else {
			rows = append(rows, nil)
		}
	}

	needMore := len(perspectiveGroupIDs) - len(usedIDs)

	if needMore > 0 {
		sqSelect := s.pictureSelect(item.ID, options, selectOptions{
			exclude: usedIDs,
		})

		sqSelect.Limit = uint32(needMore) //nolint: gosec

		morePictures, _, err := s.picturesRepository.Pictures(
			ctx, sqSelect, nil, pictures.OrderByPerspectivesGroupPerspectives, false,
		)
		if err != nil {
			return nil, err
		}

		for key, picture := range rows {
			if len(morePictures) == 0 {
				break
			}

			if picture == nil {
				rows[key] = morePictures[0]
				morePictures = morePictures[1:]
			}
		}
	}

	result := make([]*PerspectivePictureFetcherResultPicture, 0)

	var emptyPictures uint32

	for _, picture := range rows {
		if picture != nil {
			result = append(result, &PerspectivePictureFetcherResultPicture{
				Row: picture,
			})
		} else {
			result = append(result, nil)
			emptyPictures++
		}
	}

	if emptyPictures > 0 && (item.ItemTypeID == schema.ItemTableItemTypeIDEngine) {
		pictureRows, _, err := s.picturesRepository.Pictures(ctx, &query.PictureListOptions{
			Status: schema.PictureStatusAccepted,
			PictureItem: []*query.PictureItemListOptions{{
				PerspectiveID: schema.PerspectiveIDUnderTheHood,
				Item: &query.ItemListOptions{
					EngineItem: &query.ItemListOptions{
						ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
							ParentID: item.ID,
						},
					},
				},
			}},
			Limit: emptyPictures,
		}, nil, pictures.OrderByNone, false)
		if err != nil {
			return nil, err
		}

		extraPicIdx := 0

		for idx, picture := range result {
			if picture == nil {
				continue
			}

			if len(pictureRows) <= extraPicIdx {
				break
			}

			pictureRow := pictureRows[extraPicIdx]
			extraPicIdx++
			result[idx] = &PerspectivePictureFetcherResultPicture{
				Row:           pictureRow,
				IsVehicleHood: true,
			}
		}
	}

	return &PerspectivePictureFetcherResult{
		LargeFormat:   useLargeFormat,
		Pictures:      result,
		TotalPictures: int32(totalPictures), //nolint: gosec
	}, nil
}

func (s *PerspectivePictureFetcher) perspectiveGroupIDs(ctx context.Context, pageID int32) ([]int32, error) {
	if ids, ok := s.perspectiveCache[pageID]; ok {
		return ids, nil
	}

	ids, err := s.picturesRepository.PerspectivePageGroupIDs(ctx, pageID)
	if err != nil {
		return nil, err
	}

	s.perspectiveCache[pageID] = ids

	return ids, nil
}

func (s *PerspectivePictureFetcher) pictureSelect(
	itemID int64, options PerspectivePictureFetcherOptions, options2 selectOptions,
) *query.PictureListOptions {
	pictureItemType := schema.PictureItemTypeContent
	if options.PictureItemTypeID != 0 {
		pictureItemType = options.PictureItemTypeID
	}

	pictureItemOptions := query.PictureItemListOptions{
		TypeID: pictureItemType,
	}

	sqSelect := query.PictureListOptions{
		Status:      schema.PictureStatusAccepted,
		PictureItem: []*query.PictureItemListOptions{&pictureItemOptions},
		Limit:       1,
	}

	// sqSelect = sqSelect.columns(x{
	//	"id",
	//	"name",
	//	"image_id",
	//	"width",
	//	"height",
	//	"identity",
	//	"status",
	//	"owner_id",
	//	"filesize",
	//	"add_date",
	//	"dpi_x",
	//	"dpi_y",
	//	"point",
	//	"change_status_user_id",
	// })

	if options.OnlyExactlyPictures {
		pictureItemOptions.ItemID = itemID
	} else {
		pictureItemOptions.ItemParentCacheAncestor = &query.ItemParentCacheListOptions{
			ParentID:      itemID,
			ItemsByItemID: &query.ItemListOptions{}, // to order by is_concept
		}
	}

	if options.PerspectiveID != 0 {
		pictureItemOptions.PerspectiveID = options.PerspectiveID
	} else if options2.perspectiveGroup != 0 {
		pictureItemOptions.PerspectiveGroupPerspective = &query.PerspectiveGroupPerspectiveListOptions{
			GroupID: options2.perspectiveGroup,
		}
	}

	if options.ContainsPerspectiveID != 0 {
		sqSelect.PictureItem = append(sqSelect.PictureItem, &query.PictureItemListOptions{
			PerspectiveID: options.ContainsPerspectiveID,
		})
	}

	if len(options2.exclude) > 0 {
		sqSelect.ExcludeIDs = options2.exclude
	}

	return &sqSelect
}

func (s *PerspectivePictureFetcher) totalPictures(
	ctx context.Context, itemID int64, options PerspectivePictureFetcherOptions,
) (int, error) {
	if itemID == 0 {
		return 0, nil
	}

	return s.picturesRepository.Count(ctx, s.pictureSelect(itemID, options, selectOptions{}))
}
