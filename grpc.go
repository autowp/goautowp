package goautowp

import (
	"context"
	"errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	"net/url"
)

type GRPCServer struct {
	UnimplementedAutowpServer
	Catalogue         *Catalogue
	ReCaptchaConfig   RecaptchaConfig
	FileStorageConfig FileStorageConfig
}

func (s *GRPCServer) GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error) {
	items, err := s.Catalogue.getSpecs(0)
	if err != nil {
		return nil, err
	}

	return &SpecsItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error) {
	items, err := s.Catalogue.getPerspectives(nil)
	if err != nil {
		return nil, err
	}

	return &PerspectivesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error) {
	items, err := s.Catalogue.getPerspectivePages()
	if err != nil {
		return nil, err
	}

	return &PerspectivePagesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error) {
	return &ReCaptchaConfig{
		PublicKey: s.ReCaptchaConfig.PublicKey,
	}, nil
}

func (s *GRPCServer) GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error) {
	if len(s.FileStorageConfig.S3.Endpoints) <= 0 {
		return nil, errors.New("no endpoints provided")
	}

	endpoint := s.FileStorageConfig.S3.Endpoints[rand.Intn(len(s.FileStorageConfig.S3.Endpoints))]

	parsedURL, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	parsedURL.Path = "/" + url.PathEscape(s.FileStorageConfig.Bucket) + "/brands.png"
	imageURL := parsedURL.String()

	parsedURL.Path = "/" + url.PathEscape(s.FileStorageConfig.Bucket) + "/brands.css"
	cssURL := parsedURL.String()

	return &BrandIcons{
		Image: imageURL,
		Css:   cssURL,
	}, nil
}
