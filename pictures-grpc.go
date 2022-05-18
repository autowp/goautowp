package goautowp

import (
	"context"
	"github.com/autowp/goautowp/pictures"
	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PicturesGRPCServer struct {
	UnimplementedPicturesServer
	repository *pictures.Repository
	auth       *Auth
	enforcer   *casbin.Enforcer
}

func NewPicturesGRPCServer(repository *pictures.Repository, auth *Auth, enforcer *casbin.Enforcer) *PicturesGRPCServer {
	return &PicturesGRPCServer{
		repository: repository,
		auth:       auth,
		enforcer:   enforcer,
	}
}

func (s *PicturesGRPCServer) View(ctx context.Context, in *PicturesViewRequest) (*emptypb.Empty, error) {
	err := s.repository.IncView(ctx, in.GetPictureId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) Vote(ctx context.Context, in *PicturesVoteRequest) (*PicturesVoteSummary, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	err = s.repository.Vote(ctx, in.GetPictureId(), in.GetValue(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err, vote := s.repository.GetVote(ctx, in.GetPictureId(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &PicturesVoteSummary{
		Value:    vote.Value,
		Positive: vote.Positive,
		Negative: vote.Negative,
	}, nil
}

func (s *PicturesGRPCServer) CreateModerVoteTemplate(ctx context.Context, in *ModerVoteTemplate) (*ModerVoteTemplate, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	tpl := pictures.ModerVoteTemplate{
		UserID:  userID,
		Message: in.GetMessage(),
		Vote:    in.GetVote(),
	}

	fvs, err := tpl.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fvs) > 0 {
		return nil, wrapFieldViolations(fvs)
	}

	tpl, err = s.repository.CreateModerVoteTemplate(ctx, tpl)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ModerVoteTemplate{
		Id:      tpl.ID,
		UserID:  tpl.UserID,
		Message: tpl.Message,
		Vote:    tpl.Vote,
	}, nil
}

func (s *PicturesGRPCServer) DeleteModerVoteTemplate(ctx context.Context, in *DeleteModerVoteTemplateRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.repository.DeleteModerVoteTemplate(ctx, in.GetId(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) GetModerVoteTemplates(ctx context.Context, _ *emptypb.Empty) (*ModerVoteTemplates, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	items, err := s.repository.GetModerVoteTemplates(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*ModerVoteTemplate, len(items))
	for idx, item := range items {
		result[idx] = &ModerVoteTemplate{
			Id:      item.ID,
			Message: item.Message,
			Vote:    item.Vote,
			UserID:  item.UserID,
		}
	}

	return &ModerVoteTemplates{
		Items: result,
	}, nil
}
