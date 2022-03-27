package goautowp

import (
	"context"
	"github.com/autowp/goautowp/pictures"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PicturesGRPCServer struct {
	UnimplementedPicturesServer
	repository *pictures.Repository
}

func NewPicturesGRPCServer(repository *pictures.Repository) *PicturesGRPCServer {
	return &PicturesGRPCServer{
		repository: repository,
	}
}

func (s *PicturesGRPCServer) View(ctx context.Context, in *PicturesViewRequest) (*emptypb.Empty, error) {
	err := s.repository.IncView(ctx, in.GetPictureId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
