package goautowp

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCServer struct {
	UnimplementedAutowpServer
	Catalogue *Catalogue
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
