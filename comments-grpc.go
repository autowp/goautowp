package goautowp

import (
	"context"
	"fmt"
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

type CommentsGRPCServer struct {
	UnimplementedCommentsServer
	auth            *Auth
	repository      *comments.Repository
	usersRepository *users.Repository
	userExtractor   *users.UserExtractor
	enforcer        *casbin.Enforcer
}

func convertType(commentsType CommentsType) (comments.CommentType, error) {
	switch commentsType { //nolint:exhaustive
	case CommentsType_PICTURES_TYPE_ID:
		return comments.TypeIDPictures, nil
	case CommentsType_ITEM_TYPE_ID:
		return comments.TypeIDItems, nil
	case CommentsType_VOTINGS_TYPE_ID:
		return comments.TypeIDVotings, nil
	case CommentsType_ARTICLES_TYPE_ID:
		return comments.TypeIDArticles, nil
	case CommentsType_FORUMS_TYPE_ID:
		return comments.TypeIDForums, nil
	}

	return 0, fmt.Errorf("`%v` is unknown comments type identifier", commentsType)
}

func NewCommentsGRPCServer(
	auth *Auth,
	commentsRepository *comments.Repository,
	usersRepository *users.Repository,
	userExtractor *users.UserExtractor,
	enforcer *casbin.Enforcer,
) *CommentsGRPCServer {
	return &CommentsGRPCServer{
		auth:            auth,
		repository:      commentsRepository,
		usersRepository: usersRepository,
		userExtractor:   userExtractor,
		enforcer:        enforcer,
	}
}

func (s *CommentsGRPCServer) GetCommentVotes(
	ctx context.Context,
	in *GetCommentVotesRequest,
) (*CommentVoteItems, error) {
	votes, err := s.repository.GetVotes(ctx, in.CommentId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if votes == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	result := make([]*CommentVote, 0)

	for idx := range votes.PositiveVotes {
		extracted, err := s.userExtractor.Extract(ctx, &votes.PositiveVotes[idx], map[string]bool{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result = append(result, &CommentVote{
			Value: CommentVote_POSITIVE,
			User:  APIUserToGRPC(extracted),
		})
	}

	for idx := range votes.NegativeVotes {
		extracted, err := s.userExtractor.Extract(ctx, &votes.NegativeVotes[idx], map[string]bool{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result = append(result, &CommentVote{
			Value: CommentVote_NEGATIVE,
			User:  APIUserToGRPC(extracted),
		})
	}

	return &CommentVoteItems{
		Items: result,
	}, nil
}

func (s *CommentsGRPCServer) Subscribe(ctx context.Context, in *CommentsSubscribeRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	commentsType, err := convertType(in.GetTypeId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.repository.Subscribe(ctx, userID, commentsType, in.GetItemId())

	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, err
}

func (s *CommentsGRPCServer) UnSubscribe(ctx context.Context, in *CommentsUnSubscribeRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	commentsType, err := convertType(in.GetTypeId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.repository.UnSubscribe(ctx, userID, commentsType, in.GetItemId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *CommentsGRPCServer) View(ctx context.Context, in *CommentsViewRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	commentsType, err := convertType(in.GetTypeId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.repository.View(ctx, userID, commentsType, in.GetItemId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *CommentsGRPCServer) SetDeleted(ctx context.Context, in *CommentsSetDeletedRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if res := s.enforcer.Enforce(role, "comment", "remove"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if in.GetDeleted() {
		err = s.repository.QueueDeleteMessage(ctx, in.GetCommentId(), userID)
	} else {
		err = s.repository.RestoreMessage(ctx, in.GetCommentId())
	}

	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *CommentsGRPCServer) MoveComment(ctx context.Context, in *CommentsMoveCommentRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if res := s.enforcer.Enforce(role, "forums", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	commentType, err := s.repository.GetCommentType(ctx, in.GetCommentId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	if commentType != comments.TypeIDForums {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	commentsType, err := convertType(in.GetTypeId())
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.MoveMessage(ctx, in.GetCommentId(), commentsType, in.GetItemId())

	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *CommentsGRPCServer) VoteComment(
	ctx context.Context,
	in *CommentsVoteCommentRequest,
) (*CommentsVoteCommentResponse, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	votesLeft, err := s.usersRepository.GetVotesLeft(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if votesLeft <= 0 {
		return nil, status.Error(codes.PermissionDenied, "today vote limit reached")
	}

	votes, err := s.repository.VoteComment(ctx, userID, in.GetCommentId(), in.GetVote())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.usersRepository.DecVotes(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &CommentsVoteCommentResponse{
		Votes: votes,
	}, nil
}

func (s *AddCommentRequest) Validate(
	ctx context.Context,
	repository *comments.Repository,
	userID int64,
) ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

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

	needWait, err := repository.NeedWait(ctx, userID)
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

func (s *CommentsGRPCServer) Add(ctx context.Context, in *AddCommentRequest) (*AddCommentResponse, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate(ctx, s.repository, userID)
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	commentsType, err := convertType(in.GetTypeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.repository.AssertItem(ctx, commentsType, in.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	moderatorAttention := false
	if res := s.enforcer.Enforce(role, "comment", "moderator-attention"); res {
		moderatorAttention = in.ModeratorAttention
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

	messageID, err := s.repository.Add(
		ctx,
		commentsType, in.ItemId, in.ParentId, userID, in.Message, remoteAddr, moderatorAttention,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if messageID == 0 {
		return nil, status.Errorf(codes.Internal, "Message add failed")
	}

	if s.enforcer.Enforce(role, "global", "moderate") && in.ParentId > 0 && in.Resolve {
		err = s.repository.CompleteMessage(ctx, in.ParentId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if in.TypeId == CommentsType_FORUMS_TYPE_ID {
		err = s.usersRepository.IncForumMessages(ctx, in.ItemId)
	} else {
		err = s.usersRepository.TouchLastMessage(ctx, in.ItemId)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if in.ParentId > 0 {
		err = s.repository.NotifyAboutReply(ctx, messageID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = s.repository.NotifySubscribers(ctx, messageID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &AddCommentResponse{
		Id: messageID,
	}, nil
}
