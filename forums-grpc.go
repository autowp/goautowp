package goautowp

import (
	"context"
	"net"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	topicID, err := s.forums.AddTopic(ctx, in.GetThemeId(), in.GetName(), userID, remoteAddr)
	if err != nil {
		return nil, err
	}

	_, err = s.commentsRepository.Add(
		ctx,
		schema.CommentMessageTypeIDForums,
		topicID,
		0,
		userID,
		in.GetMessage(),
		remoteAddr,
		in.GetModeratorAttention(),
	)
	if err != nil {
		return nil, err
	}

	if in.GetSubscription() {
		err = s.commentsRepository.Subscribe(ctx, userID, schema.CommentMessageTypeIDForums, topicID)
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

	s.Name, problems, err = nameInputFilter.IsValidString(s.GetName())
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

	s.Message, problems, err = msgInputFilter.IsValidString(s.GetMessage())
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

	err = s.forums.Close(ctx, in.GetId())
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

	err = s.forums.Open(ctx, in.GetId())
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

	err = s.forums.Delete(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ForumsGRPCServer) MoveTopic(ctx context.Context, in *APIMoveTopicRequest) (*emptypb.Empty, error) {
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

	err = s.forums.MoveTopic(ctx, in.GetId(), in.GetThemeId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func convertTheme(theme *ForumsTheme) *APIForumsTheme {
	return &APIForumsTheme{
		Id:            theme.ID,
		Name:          theme.Name,
		TopicsCount:   theme.TopicsCount,
		MessagesCount: theme.MessagesCount,
		DisableTopics: theme.DisableTopics,
		Description:   theme.Description,
	}
}

func convertTopic(topic *ForumsTopic) *APIForumsTopic {
	return &APIForumsTopic{
		Id:           topic.ID,
		Name:         topic.Name,
		Status:       topic.Status,
		OldMessages:  topic.Messages - topic.NewMessages,
		NewMessages:  topic.NewMessages,
		CreatedAt:    timestamppb.New(topic.CreatedAt),
		UserId:       topic.UserID,
		ThemeId:      topic.ThemeID,
		Subscription: topic.Subscription,
	}
}

func (s *ForumsGRPCServer) GetTheme(ctx context.Context, in *APIGetForumsThemeRequest) (*APIForumsTheme, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	theme, err := s.forums.Theme(ctx, in.GetId(), isModerator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if theme == nil {
		return nil, status.Error(codes.NotFound, "Theme not found")
	}

	return convertTheme(theme), nil
}

func (s *ForumsGRPCServer) GetThemes(ctx context.Context, in *APIGetForumsThemesRequest) (*APIForumsThemes, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	themes, err := s.forums.Themes(ctx, in.GetThemeId(), isModerator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIForumsTheme, len(themes))
	for idx, theme := range themes {
		result[idx] = convertTheme(theme)
	}

	return &APIForumsThemes{
		Items: result,
	}, nil
}

func (s *ForumsGRPCServer) GetLastTopic(ctx context.Context, in *APIGetForumsThemeRequest) (*APIForumsTopic, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	topic, err := s.forums.LastTopic(ctx, in.GetId(), userID, isModerator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if topic == nil {
		return nil, status.Error(codes.NotFound, "Topic not found")
	}

	return convertTopic(topic), nil
}

func (s *ForumsGRPCServer) GetLastMessage(
	ctx context.Context,
	in *APIGetForumsTopicRequest,
) (*APICommentMessage, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	msg, err := s.forums.LastMessage(ctx, in.GetId(), isModerator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if msg == nil {
		return nil, status.Error(codes.NotFound, "Message not found")
	}

	var userID int64
	if msg.UserID.Valid {
		userID = msg.UserID.Int64
	}

	return &APICommentMessage{
		Id:        msg.ID,
		CreatedAt: timestamppb.New(msg.Datetime),
		UserId:    userID,
	}, nil
}

func (s *ForumsGRPCServer) GetTopics(ctx context.Context, in *APIGetForumsTopicsRequest) (*APIForumsTopics, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	topics, pages, err := s.forums.Topics(ctx, in.GetThemeId(), userID, isModerator, in.GetSubscription(), in.GetPage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIForumsTopic, len(topics))
	for idx, topic := range topics {
		result[idx] = convertTopic(topic)
	}

	return &APIForumsTopics{
		Items: result,
		Paginator: &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Last:             pages.Last,
			Previous:         pages.Previous,
			Next:             pages.Next,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
		},
	}, nil
}

func (s *ForumsGRPCServer) GetTopic(ctx context.Context, in *APIGetForumsTopicRequest) (*APIForumsTopic, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModerator := s.enforcer.Enforce(role, "forums", "moderate")

	topic, err := s.forums.Topic(ctx, in.GetId(), userID, isModerator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if topic == nil {
		return nil, status.Error(codes.NotFound, "Topic not found")
	}

	return convertTopic(topic), nil
}
