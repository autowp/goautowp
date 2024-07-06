package goautowp

import (
	"context"

	"github.com/autowp/goautowp/attrs"
	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AttrsGRPCServer struct {
	UnimplementedAttrsServer
	repository *attrs.Repository
	enforcer   *casbin.Enforcer
	auth       *Auth
}

func NewAttrsGRPCServer(repository *attrs.Repository, enforcer *casbin.Enforcer, auth *Auth) *AttrsGRPCServer {
	return &AttrsGRPCServer{
		repository: repository,
		enforcer:   enforcer,
		auth:       auth,
	}
}

func convertNullTypeID(typeID attrs.NullAttributeTypeID) AttrAttributeType_ID {
	if !typeID.Valid {
		return AttrAttributeType_UNKNOWN
	}

	return convertTypeID(typeID.AttributeTypeID)
}

func convertTypeID(typeID attrs.AttributeTypeID) AttrAttributeType_ID {
	switch typeID {
	case attrs.TypeUnknown:
		return AttrAttributeType_UNKNOWN
	case attrs.TypeString:
		return AttrAttributeType_STRING
	case attrs.TypeInteger:
		return AttrAttributeType_INTEGER
	case attrs.TypeFloat:
		return AttrAttributeType_FLOAT
	case attrs.TypeText:
		return AttrAttributeType_TEXT
	case attrs.TypeBoolean:
		return AttrAttributeType_BOOLEAN
	case attrs.TypeList:
		return AttrAttributeType_LIST
	case attrs.TypeTree:
		return AttrAttributeType_TREE
	}

	return AttrAttributeType_UNKNOWN
}

func convertAttribute(row attrs.Attribute) *AttrAttribute {
	var parentID int64
	if row.ParentID.Valid {
		parentID = row.ParentID.Int64
	}

	var unitID int64
	if row.UnitID.Valid {
		unitID = row.UnitID.Int64
	}

	var description string
	if row.Description.Valid {
		description = row.Description.String
	}

	var precision int32
	if row.Precision.Valid {
		precision = row.Precision.Int32
	}

	return &AttrAttribute{
		Id:          row.ID,
		Name:        row.Name,
		ParentId:    parentID,
		Description: description,
		TypeId:      convertNullTypeID(row.TypeID),
		UnitId:      unitID,
		IsMultiple:  row.Multiple,
		Precision:   precision,
	}
}

func (s *AttrsGRPCServer) GetAttribute(ctx context.Context, in *AttrAttributeID) (*AttrAttribute, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "specifications", "edit"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	success, row, err := s.repository.Attribute(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !success {
		return nil, status.Error(codes.NotFound, "NotFound")
	}

	return convertAttribute(row), nil
}

func (s *AttrsGRPCServer) GetAttributes(
	ctx context.Context, in *AttrAttributesRequest,
) (*AttrAttributesResponse, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "specifications", "edit"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	rows, err := s.repository.Attributes(ctx, in.GetZoneId(), in.GetParentId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*AttrAttribute, len(rows))

	for idx, row := range rows {
		items[idx] = convertAttribute(row)
	}

	return &AttrAttributesResponse{
		Items: items,
	}, nil
}

func (s *AttrsGRPCServer) GetAttributeTypes(
	ctx context.Context, _ *emptypb.Empty,
) (*AttrAttributeTypesResponse, error) {
	rows, err := s.repository.AttributeTypes(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*AttrAttributeType, len(rows))
	for idx, row := range rows {
		items[idx] = &AttrAttributeType{
			Id:   convertTypeID(row.ID),
			Name: row.Name,
		}
	}

	return &AttrAttributeTypesResponse{
		Items: items,
	}, nil
}

func (s *AttrsGRPCServer) GetListOptions(
	ctx context.Context, in *AttrListOptionsRequest,
) (*AttrListOptionsResponse, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "specifications", "edit"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	rows, err := s.repository.ListOptions(ctx, in.GetAttributeId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	options := make([]*AttrListOption, len(rows))

	for idx, row := range rows {
		var parentID int64
		if row.ParentID.Valid {
			parentID = row.ParentID.Int64
		}

		options[idx] = &AttrListOption{
			Id:          row.ID,
			Name:        row.Name,
			AttributeId: row.AttributeID,
			ParentId:    parentID,
		}
	}

	return &AttrListOptionsResponse{
		Items: options,
	}, nil
}

func (s *AttrsGRPCServer) GetUnits(ctx context.Context, _ *emptypb.Empty) (*AttrUnitsResponse, error) {
	rows, err := s.repository.Units(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	units := make([]*AttrUnit, len(rows))
	for idx, row := range rows {
		units[idx] = &AttrUnit{
			Id:   row.ID,
			Name: row.Name,
			Abbr: row.Abbr,
		}
	}

	return &AttrUnitsResponse{
		Items: units,
	}, nil
}

func (s *AttrsGRPCServer) GetZoneAttributes(
	ctx context.Context, in *AttrZoneAttributesRequest,
) (*AttrZoneAttributesResponse, error) {
	rows, err := s.repository.ZoneAttributes(ctx, in.GetZoneId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*AttrZoneAttribute, len(rows))
	for idx, row := range rows {
		items[idx] = &AttrZoneAttribute{
			ZoneId:      row.ZoneID,
			AttributeId: row.AttributeID,
		}
	}

	return &AttrZoneAttributesResponse{
		Items: items,
	}, nil
}

func (s *AttrsGRPCServer) GetZones(ctx context.Context, _ *emptypb.Empty) (*AttrZonesResponse, error) {
	rows, err := s.repository.Zones(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*AttrZone, len(rows))
	for idx, row := range rows {
		items[idx] = &AttrZone{
			Id:   row.ID,
			Name: row.Name,
		}
	}

	return &AttrZonesResponse{
		Items: items,
	}, nil
}
