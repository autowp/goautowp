package goautowp

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/autowp/goautowp/frontend"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type PicturesREST struct {
	picturesRepository   *pictures.Repository
	pictureNameFormatter *pictures.PictureNameFormatter
	hostManager          *hosts.Manager
	imageStorage         *storage.Storage
	itemOfDay            *itemofday.Repository
	itemsRepository      *items.Repository
}

func NewPicturesREST(picturesRepository *pictures.Repository, pictureNameFormatter *pictures.PictureNameFormatter,
	hostManager *hosts.Manager, imageStorage *storage.Storage, itemOfDay *itemofday.Repository,
	itemsRepository *items.Repository,
) *PicturesREST {
	return &PicturesREST{
		picturesRepository:   picturesRepository,
		pictureNameFormatter: pictureNameFormatter,
		hostManager:          hostManager,
		imageStorage:         imageStorage,
		itemOfDay:            itemOfDay,
		itemsRepository:      itemsRepository,
	}
}

func (s *PicturesREST) detectLanguage(ctx *gin.Context) (string, error) {
	tags, _, err := language.ParseAcceptLanguage(ctx.Request.Header.Get("Accept-Language"))
	if err != nil {
		return "", err
	}

	lang := "en"

	if len(tags) > 0 {
		base, _ := tags[0].Base()
		lang = base.String()
	}

	return lang, nil
}

func (s *PicturesREST) handlePicture(ctx *gin.Context, orderBy pictures.OrderBy) {
	lang, err := s.detectLanguage(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	row, err := s.picturesRepository.Picture(ctx, &query.PictureListOptions{
		Status: schema.PictureStatusAccepted,
	}, &pictures.PictureFields{NameText: true}, orderBy)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"status": false,
		})

		return
	}

	s.populatePicture(ctx, row, lang)
}

func (s *PicturesREST) handleItemOfDayPicture(ctx *gin.Context) {
	lang, err := s.detectLanguage(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	itemOfDay, err := s.itemOfDay.Current(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	var row *schema.PictureRow

	if itemOfDay.ItemID > 0 {
		carRow, err := s.itemsRepository.Item(ctx, &query.ItemListOptions{
			ItemID: itemOfDay.ItemID,
		}, &items.ListFields{NameText: true})
		if err != nil && !errors.Is(err, items.ErrItemNotFound) {
			ctx.String(http.StatusInternalServerError, err.Error())

			return
		}

		if err == nil {
			for _, groupID := range []int32{31, 0} {
				filter := query.PictureListOptions{
					Status: schema.PictureStatusAccepted,
					PictureItem: &query.PictureItemListOptions{
						ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
							ParentID: carRow.ID,
						},
					},
				}
				order := pictures.OrderByResolutionDesc

				if groupID > 0 {
					filter.PictureItem.PerspectiveGroupPerspective = &query.PerspectiveGroupPerspectiveListOptions{
						GroupID: groupID,
					}
					order = pictures.OrderByPerspectivesGroupPerspectives
				}

				row, err = s.picturesRepository.Picture(ctx, &filter, &pictures.PictureFields{NameText: true}, order)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					ctx.String(http.StatusInternalServerError, err.Error())

					return
				}

				if err == nil {
					break
				}
			}
		}
	}

	s.populatePicture(ctx, row, lang)
}

func (s *PicturesREST) populatePicture(ctx *gin.Context, row *schema.PictureRow, lang string) {
	if row == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"status": false,
		})

		return
	}

	if !row.ImageID.Valid {
		ctx.Status(http.StatusNotFound)

		return
	}

	imageInfo, err := s.imageStorage.Image(ctx, int(row.ImageID.Int64))
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	uri, err := s.hostManager.URIByLanguage(lang)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	namesData, err := s.picturesRepository.NameData(ctx, []*schema.PictureRow{row}, pictures.NameDataOptions{
		Language: lang,
	})
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	nameText, err := s.pictureNameFormatter.FormatText(namesData[row.ID], lang)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": true,
		"url":    imageInfo.Src(),
		"name":   nameText,
		"page":   frontend.PictureURL(uri, row.Identity),
	})
}

func (s *PicturesREST) SetupRouter(router *gin.Engine) {
	router.GET("/api/picture/random-picture", func(ctx *gin.Context) {
		s.handlePicture(ctx, pictures.OrderByRandom)
	})

	router.GET("/api/picture/new-picture", func(ctx *gin.Context) {
		s.handlePicture(ctx, pictures.OrderByAcceptDatetimeDesc)
	})

	router.GET("/api/picture/car-of-day-picture", func(ctx *gin.Context) {
		s.handleItemOfDayPicture(ctx)
	})
}
