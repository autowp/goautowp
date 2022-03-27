package goautowp

import (
	"context"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MessagingGRPCServer struct {
	UnimplementedMessagingServer
	repository *messaging.Repository
	auth       *Auth
}

func NewMessagingGRPCServer(repository *messaging.Repository, auth *Auth) *MessagingGRPCServer {
	return &MessagingGRPCServer{
		repository: repository,
		auth:       auth,
	}
}

func (s *MessagingGRPCServer) GetMessagesNewCount(ctx context.Context, _ *emptypb.Empty) (*APIMessageNewCount, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	count, err := s.repository.GetUserNewMessagesCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIMessageNewCount{
		Count: int32(count),
	}, nil
}

func (s *MessagingGRPCServer) GetMessagesSummary(ctx context.Context, _ *emptypb.Empty) (*APIMessageSummary, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	inbox, err := s.repository.GetInboxCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	inboxNew, err := s.repository.GetInboxNewCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	sent, err := s.repository.GetSentCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	system, err := s.repository.GetSystemCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	systemNew, err := s.repository.GetSystemNewCount(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIMessageSummary{
		InboxCount:     int32(inbox),
		InboxNewCount:  int32(inboxNew),
		SentCount:      int32(sent),
		SystemCount:    int32(system),
		SystemNewCount: int32(systemNew),
	}, nil

}

func (s *MessagingGRPCServer) DeleteMessage(ctx context.Context, in *MessagingDeleteMessage) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	err = s.repository.DeleteMessage(ctx, userID, in.MessageId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *MessagingGRPCServer) ClearFolder(ctx context.Context, in *MessagingClearFolder) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	switch in.GetFolder() {
	case "sent":
		err = s.repository.ClearSent(ctx, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

	case "system":
		err = s.repository.ClearSystem(ctx, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	default:
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *MessagingGRPCServer) CreateMessage(ctx context.Context, in *MessagingCreateMessage) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	fvs := make([]*errdetails.BadRequest_FieldViolation, 0)
	var problems []string

	message := in.GetMessage()

	messageInputFilter := validation.InputFilter{
		Filters:    []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{&validation.NotEmpty{}},
	}
	message, problems, err = messageInputFilter.IsValidString(message)
	if err != nil {
		return nil, err
	}
	for _, fv := range problems {
		fvs = append(fvs, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: fv,
		})
	}

	if len(fvs) > 0 {
		return nil, wrapFieldViolations(fvs)
	}

	err = s.repository.CreateMessage(ctx, userID, in.GetUserId(), message)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
