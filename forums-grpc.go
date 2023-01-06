package goautowp

import (
	"context"
	"net"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const MaxTopicNameLength = 100

type ForumsGRPCServer struct {
	UnimplementedForumsServer
	auth               *Auth
	forums             *Forums
	commentsRepository *comments.Repository
	usersRepository    *users.Repository
	enforcer           *casbin.Enforcer
}

func NewForumsGRPCServer(
	auth *Auth,
	forums *Forums,
	commentsRepository *comments.Repository,
	usersRepository *users.Repository,
	enforcer *casbin.Enforcer,
) *ForumsGRPCServer {
	return &ForumsGRPCServer{
		auth:               auth,
		forums:             forums,
		commentsRepository: commentsRepository,
		usersRepository:    usersRepository,
		enforcer:           enforcer,
	}
}

func (s *ForumsGRPCServer) GetUserSummary(ctx context.Context, _ *emptypb.Empty) (*APIForumsUserSummary, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	subscriptionsCount, err := s.forums.GetUserSummary(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIForumsUserSummary{
		SubscriptionsCount: int32(subscriptionsCount),
	}, nil
}

func (s *ForumsGRPCServer) CreateTopic(
	ctx context.Context,
	in *APICreateTopicRequest,
) (*APICreateTopicResponse, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	InvalidParams, err := in.Validate(ctx, s.commentsRepository, userID)
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	remoteAddr := "127.0.0.1"
	p, ok := peer.FromContext(ctx)

	if ok {
		nw := p.Addr.String()
		if nw != "bufconn" {
			ip, _, err := net.SplitHostPort(nw)
			if err != nil {
				logrus.Errorf("userip: %q is not IP:port", nw)
			} else {
				remoteAddr = ip
			}
		}
	}

	topicID, err := s.forums.AddTopic(ctx, in.ThemeId, in.Name, userID, remoteAddr)
	if err != nil {
		return nil, err
	}

	_, err = s.commentsRepository.Add(
		ctx,
		comments.TypeIDForums,
		topicID,
		0,
		userID,
		in.Message,
		remoteAddr,
		in.ModeratorAttention,
	)
	if err != nil {
		return nil, err
	}

	if in.Subscription {
		err = s.commentsRepository.Subscribe(ctx, userID, comments.TypeIDForums, topicID)
		if err != nil {
			return nil, err
		}
	}

	err = s.usersRepository.IncForumTopics(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &APICreateTopicResponse{
		Id: topicID,
	}, nil
}

func (s *APICreateTopicRequest) Validate(
	ctx context.Context,
	commentsRepository *comments.Repository,
	userID int64,
) ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Max: MaxTopicNameLength},
		},
	}
	s.Name, problems, err = nameInputFilter.IsValidString(s.Name)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	msgInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Max: comments.MaxMessageLength},
		},
	}
	s.Message, problems, err = msgInputFilter.IsValidString(s.Message)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: fv,
		})
	}

	needWait, err := commentsRepository.NeedWait(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if needWait {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: "Too often",
		})
	}

	return result, nil
}

func (s *ForumsGRPCServer) CloseTopic(ctx context.Context, in *APISetTopicStatusRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	forumAdmin := s.enforcer.Enforce(role, "forums", "moderate")
	if !forumAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.forums.Close(ctx, in.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ForumsGRPCServer) OpenTopic(ctx context.Context, in *APISetTopicStatusRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	forumAdmin := s.enforcer.Enforce(role, "forums", "moderate")
	if !forumAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.forums.Open(ctx, in.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ForumsGRPCServer) DeleteTopic(ctx context.Context, in *APISetTopicStatusRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	forumAdmin := s.enforcer.Enforce(role, "forums", "moderate")
	if !forumAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.forums.Delete(ctx, in.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
