package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	"net/url"
)

type GRPCServer struct {
	UnimplementedAutowpServer
	catalogue         *Catalogue
	reCaptchaConfig   RecaptchaConfig
	fileStorageConfig FileStorageConfig
	db                *sql.DB
	enforcer          *casbin.Enforcer
	oauthConfig       OAuthConfig
}

func NewGRPCServer(catalogue *Catalogue, reCaptchaConfig RecaptchaConfig, fileStorageConfig FileStorageConfig, db *sql.DB, enforcer *casbin.Enforcer, oauthConfig OAuthConfig) (*GRPCServer, error) {
	return &GRPCServer{
		catalogue:         catalogue,
		reCaptchaConfig:   reCaptchaConfig,
		fileStorageConfig: fileStorageConfig,
		db:                db,
		enforcer:          enforcer,
		oauthConfig:       oauthConfig,
	}, nil
}

func (s *GRPCServer) GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error) {
	items, err := s.catalogue.getSpecs(0)
	if err != nil {
		return nil, err
	}

	return &SpecsItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error) {
	items, err := s.catalogue.getPerspectives(nil)
	if err != nil {
		return nil, err
	}

	return &PerspectivesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error) {
	items, err := s.catalogue.getPerspectivePages()
	if err != nil {
		return nil, err
	}

	return &PerspectivePagesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error) {
	return &ReCaptchaConfig{
		PublicKey: s.reCaptchaConfig.PublicKey,
	}, nil
}

func (s *GRPCServer) GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error) {
	if len(s.fileStorageConfig.S3.Endpoints) <= 0 {
		return nil, errors.New("no endpoints provided")
	}

	endpoint := s.fileStorageConfig.S3.Endpoints[rand.Intn(len(s.fileStorageConfig.S3.Endpoints))]

	parsedURL, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.png"
	imageURL := parsedURL.String()

	parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.css"
	cssURL := parsedURL.String()

	return &BrandIcons{
		Image: imageURL,
		Css:   cssURL,
	}, nil
}

func (s *GRPCServer) AclEnforce(ctx context.Context, in *AclEnforceRequest) (*AclEnforceResult, error) {

	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &AclEnforceResult{
		Result: s.enforcer.Enforce(role, in.Resource, in.Privilege),
	}, nil
}

func (s *GRPCServer) GetVehicleTypes(ctx context.Context, in *emptypb.Empty) (*VehicleTypeItems, error) {
	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	items, err := s.catalogue.getVehicleTypesTree(0)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &VehicleTypeItems{
		Items: items,
	}, nil
}
