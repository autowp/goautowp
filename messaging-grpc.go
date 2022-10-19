package goautowp

import (
	"context"

	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	count, err := s.repository.GetUserNewMessagesCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIMessageNewCount{
		Count: count,
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

	inbox, err := s.repository.GetInboxCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	inboxNew, err := s.repository.GetInboxNewCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	sent, err := s.repository.GetSentCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	system, err := s.repository.GetSystemCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	systemNew, err := s.repository.GetSystemNewCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIMessageSummary{
		InboxCount:     inbox,
		InboxNewCount:  inboxNew,
		SentCount:      sent,
		SystemCount:    system,
		SystemNewCount: systemNew,
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

	var (
		fvs      = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
	)

	message := in.GetText()

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

func (s *MessagingGRPCServer) GetMessages(
	ctx context.Context,
	in *MessagingGetMessagesRequest,
) (*MessagingGetMessagesResponse, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	var (
		messages []messaging.Message
		pages    *util.Pages
	)

	switch in.GetFolder() {
	case "inbox":
		messages, pages, err = s.repository.GetInbox(ctx, userID, in.GetPage())
	case "sent":
		messages, pages, err = s.repository.GetSentbox(ctx, userID, in.GetPage())
	case "system":
		messages, pages, err = s.repository.GetSystembox(ctx, userID, in.GetPage())
	case "dialog":
		messages, pages, err = s.repository.GetDialogbox(
			ctx,
			userID,
			in.GetUserId(),
			in.GetPage(),
		)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unexpected folder value")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*APIMessage, len(messages))

	for idx, msg := range messages {
		item := APIMessage{
			Id:              msg.ID,
			Text:            msg.Text,
			IsNew:           msg.IsNew,
			CanDelete:       msg.CanDelete,
			CanReply:        msg.CanReply,
			Date:            timestamppb.New(msg.Date),
			AllMessagesLink: msg.AllMessagesLink,
			DialogCount:     msg.DialogCount,
			ToUserId:        msg.ToUserID,
		}

		if msg.AuthorID != nil {
			item.AuthorId = *msg.AuthorID
		}

		if msg.DialogWithUserID != nil {
			item.DialogWithUserId = *msg.DialogWithUserID
		}

		items[idx] = &item
	}

	paginator := Pages{
		PageCount:        pages.PageCount,
		First:            pages.First,
		Current:          pages.Current,
		FirstPageInRange: pages.FirstPageInRange,
		LastPageInRange:  pages.LastPageInRange,
		PagesInRange:     pages.PagesInRange,
		TotalItemCount:   pages.TotalItemCount,
	}

	if pages.Next != nil {
		paginator.Next = *pages.Next
	}

	if pages.Previous != nil {
		paginator.Previous = *pages.Previous
	}

	return &MessagingGetMessagesResponse{
		Items:     items,
		Paginator: &paginator,
	}, nil
}
