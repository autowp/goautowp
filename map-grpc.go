package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
)

type MapGRPCServer struct {
	UnimplementedMapServer
	db           *sql.DB
	imageStorage *storage.Storage
}

func NewMapGRPCServer(db *sql.DB, imageStorage *storage.Storage) *MapGRPCServer {
	return &MapGRPCServer{
		db:           db,
		imageStorage: imageStorage,
	}
}

func (s *MapGRPCServer) GetPoints(ctx context.Context, in *MapGetPointsRequest) (*MapPoints, error) {

	bounds := strings.Split(in.GetBounds(), ",")

	if len(bounds) < 4 {
		return nil, status.Error(codes.InvalidArgument, "Invalid bounds")
	}

	lngLo, err := strconv.ParseFloat(bounds[0], 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	latLo, err := strconv.ParseFloat(bounds[1], 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	lngHi, err := strconv.ParseFloat(bounds[2], 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	latHi, err := strconv.ParseFloat(bounds[3], 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pointsOnly := in.GetPointsOnly()

	// $language = $this->language();

	mapPoints := make([]*MapPoint, 0)

	polygon := fmt.Sprintf("POLYGON((%F %F, %F %F, %F %F, %F %F, %F %F))",
		lngLo,
		latLo,
		lngLo,
		latHi,
		lngHi,
		latHi,
		lngHi,
		latLo,
		lngLo,
		latLo,
	)

	if pointsOnly {
		rows, err := s.db.QueryContext(
			ctx,
			`
				SELECT ST_AsBinary(point)
				FROM item_point
				WHERE ST_Contains(ST_GeomFromText(?), point)
			`,
			polygon,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, status.Error(codes.Internal, err.Error())
		}

		for rows.Next() {
			var p orb.Point
			err = rows.Scan(wkb.Scanner(&p))
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			mapPoints = append(mapPoints, &MapPoint{
				Location: &Point{
					Lat: p.Lat(),
					Lng: p.Lon(),
				},
			})
		}
	} else {
		rows, err := s.db.QueryContext(
			ctx,
			`
				SELECT ST_AsBinary(item_point.point), item.id, item.name, item.begin_year, item.end_year, item.item_type_id, item.today
				FROM item
					INNER JOIN item_point ON item.id = item_point.item_id
				WHERE ST_Contains(ST_GeomFromText(?), item_point.point)
			`,
			polygon,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, status.Error(codes.Internal, err.Error())
		}

		nameFormatter := items.ItemNameFormatter{}

		for rows.Next() {
			var p orb.Point
			var id int64
			var name string
			var nullableBeginYear sql.NullInt32
			var nullableEndYear sql.NullInt32
			var itemTypeId items.ItemType
			var today sql.NullBool
			err = rows.Scan(wkb.Scanner(&p), &id, &name, &nullableBeginYear, &nullableEndYear, &itemTypeId, &today)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			beginYear := 0
			if nullableBeginYear.Valid {
				beginYear = int(nullableBeginYear.Int32)
			}

			endYear := 0
			if nullableEndYear.Valid {
				endYear = int(nullableEndYear.Int32)
			}

			var todayRef *bool
			if today.Valid {
				todayRef = &today.Bool
			}

			mapPoint := &MapPoint{
				Id: fmt.Sprintf("factory%d", id),
				Location: &Point{
					Lat: p.Lat(),
					Lng: p.Lon(),
				},
				Name: nameFormatter.Format(items.ItemNameFormatterOptions{
					Name:      name,
					BeginYear: beginYear,
					EndYear:   endYear,
					Today:     todayRef,
				}, in.GetLanguage()),
			}

			switch itemTypeId {
			case items.FACTORY:
				mapPoint.Url = []string{"/factories", strconv.FormatInt(id, 10)}
			case items.MUSEUM:
				mapPoint.Url = []string{"/museums", strconv.FormatInt(id, 10)}
			}

			var imageId sql.NullInt64
			err = s.db.QueryRowContext(ctx, `
				SELECT pictures.image_id
				FROM pictures 
				    INNER JOIN picture_item ON pictures.id = picture_item.picture_id
				WHERE pictures.status = ? AND picture_item.item_id = ?
			`, pictures.StatusAccepted, id).Scan(&imageId)
			if err != nil && err != sql.ErrNoRows {
				return nil, status.Error(codes.Internal, err.Error())
			}

			if err != sql.ErrNoRows && imageId.Valid {
				image, err := s.imageStorage.FormattedImage(int(imageId.Int64), "format9")
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				mapPoint.Image = ImageToAPIImage(image)
			}

			mapPoints = append(mapPoints, mapPoint)
		}
	}

	return &MapPoints{
		Points: mapPoints,
	}, nil
}