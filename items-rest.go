package goautowp

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // GIF support
	_ "image/jpeg" // JPEG support
	_ "image/png"  // PNG support
	"net/http"
	"strconv"

	"github.com/autowp/goautowp/items"
	"github.com/casbin/casbin"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp" // WEBP support
)

const itemLogoFileField = "file"

type ItemsREST struct {
	auth       *Auth
	enforcer   *casbin.Enforcer
	repository *items.Repository
	events     *Events
}

func NewItemsREST(
	auth *Auth, enforcer *casbin.Enforcer, repository *items.Repository, events *Events,
) *ItemsREST {
	return &ItemsREST{
		auth:       auth,
		enforcer:   enforcer,
		repository: repository,
		events:     events,
	}
}

func (s *ItemsREST) postLogoAction(ctx *gin.Context) {
	userID, role, err := s.auth.ValidateREST(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	if !s.enforcer.Enforce(role, "brand", "logo") {
		ctx.Status(http.StatusForbidden)

		return
	}

	itemIDStr := ctx.Param("id")

	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())

		return
	}

	file, err := ctx.FormFile(itemLogoFileField)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	if file.Size > items.ItemLogoMaxFileSize {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"invalid_params": gin.H{itemLogoFileField: map[string]string{
				"fileFilesSizeTooBig": fmt.Sprintf(
					"All files in sum should have a maximum size of '%d' but '%d' were detected",
					items.ItemLogoMaxFileSize, file.Size,
				),
			}},
		})

		return
	}

	handle, err := file.Open()
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	mime, err := mimetype.DetectReader(handle)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	if !mime.Is("image/png") {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"invalid_params": gin.H{itemLogoFileField: map[string]string{
				"fileIsImageFalseType": fmt.Sprintf(
					"File is no image, '%s' detected",
					mime.String(),
				),
			}},
		})

		return
	}

	_, err = handle.Seek(0, 0)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	imageConfig, _, err := image.DecodeConfig(handle)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	if imageConfig.Width < items.ItemLogoMinWidth || imageConfig.Height < items.ItemLogoMinHeight {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"invalid_params": gin.H{itemLogoFileField: map[string]string{
				"fileImageSizeTooSmall": fmt.Sprintf(
					"Minimum expected size for image should be '%dx%d' but '%dx%d' detected",
					items.ItemLogoMinWidth, items.ItemLogoMinHeight, imageConfig.Width, imageConfig.Height,
				),
			}},
		})

		return
	}

	_, err = handle.Seek(0, 0)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	ctxWithoutCancel := context.WithoutCancel(ctx)

	err = s.repository.SetItemLogo(ctxWithoutCancel, itemID, handle)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			ctx.Status(http.StatusNotFound)

			return
		}

		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	err = s.events.Add(ctxWithoutCancel, Event{
		UserID:  userID,
		Message: "Закачен логотип",
		Items:   []int64{itemID},
	})
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())

		return
	}

	ctx.Status(http.StatusOK)
}

func (s *ItemsREST) SetupRouter(router *gin.Engine) {
	router.POST("/api/item/:id/logo", func(ctx *gin.Context) {
		s.postLogoAction(ctx)
	})
}
