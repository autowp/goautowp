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
