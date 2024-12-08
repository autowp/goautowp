package goautowp

import (
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
)

type PictureItemExtractor struct {
	enforcer *casbin.Enforcer
}

func NewPictureItemExtractor(
	enforcer *casbin.Enforcer,
) *PictureItemExtractor {
	return &PictureItemExtractor{
		enforcer: enforcer,
	}
}

func (s *PictureItemExtractor) Extract(row schema.PictureItemRow) *PictureItem {
	result := &PictureItem{
		PictureId:  row.PictureID,
		ItemId:     row.ItemID,
		Type:       reverseConvertPictureItemType(row.Type),
		CropLeft:   uint32(util.NullInt32ToScalar(row.CropLeft)),   //nolint:gosec
		CropTop:    uint32(util.NullInt32ToScalar(row.CropTop)),    //nolint:gosec
		CropWidth:  uint32(util.NullInt32ToScalar(row.CropWidth)),  //nolint:gosec
		CropHeight: uint32(util.NullInt32ToScalar(row.CropHeight)), //nolint:gosec
	}

	return result
}
