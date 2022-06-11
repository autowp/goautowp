package goautowp

import (
	"context"
	"fmt"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CommentsGRPCServer struct {
	UnimplementedCommentsServer
	auth            *Auth
	repository      *comments.Repository
	usersRepository *users.Repository
	userExtractor   *UserExtractor
	enforcer        *casbin.Enforcer
}

func convertType(commentsType CommentsType) (comments.CommentType, error) {
	switch commentsType {
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
	userExtractor *UserExtractor,
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

func (s *CommentsGRPCServer) GetCommentVotes(_ context.Context, in *GetCommentVotesRequest) (*CommentVoteItems, error) {
	votes, err := s.repository.GetVotes(in.CommentId)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if votes == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	result := make([]*CommentVote, 0)

	for _, user := range votes.PositiveVotes {
		extracted, err := s.userExtractor.Extract(&user, map[string]bool{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result = append(result, &CommentVote{
			Value: CommentVote_POSITIVE,
			User:  extracted,
		})
	}

	for _, user := range votes.NegativeVotes {
		extracted, err := s.userExtractor.Extract(&user, map[string]bool{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result = append(result, &CommentVote{
			Value: CommentVote_NEGATIVE,
			User:  extracted,
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

func (s *CommentsGRPCServer) VoteComment(ctx context.Context, in *CommentsVoteCommentRequest) (*CommentsVoteCommentResponse, error) {
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
