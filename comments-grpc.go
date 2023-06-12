package goautowp

import (
	"context"
	"fmt"
	"net"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const MaxReplies = 500

type CommentsGRPCServer struct {
	UnimplementedCommentsServer
	auth               *Auth
	repository         *comments.Repository
	usersRepository    *users.Repository
	picturesRepository *pictures.Repository
	userExtractor      *users.UserExtractor
	enforcer           *casbin.Enforcer
}

func reverseConvertPicturesStatus(status pictures.Status) PictureStatus {
	switch status {
	case pictures.StatusAccepted:
		return PictureStatus_PICTURE_STATUS_ACCEPTED
	case pictures.StatusRemoving:
		return PictureStatus_PICTURE_STATUS_REMOVING
	case pictures.StatusInbox:
		return PictureStatus_PICTURE_STATUS_INBOX
	case pictures.StatusRemoved:
		return PictureStatus_PICTURE_STATUS_REMOVED
	}

	return PictureStatus_PICTURE_STATUS_UNKNOWN
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
	case CommentsType_UNKNOWN:
		return 0, nil
	}

	return 0, fmt.Errorf("`%v` is unknown comments type identifier", commentsType)
}

func reverseConvertType(commentsType comments.CommentType) (CommentsType, error) {
	switch commentsType {
	case comments.TypeIDPictures:
		return CommentsType_PICTURES_TYPE_ID, nil
	case comments.TypeIDItems:
		return CommentsType_ITEM_TYPE_ID, nil
	case comments.TypeIDVotings:
		return CommentsType_VOTINGS_TYPE_ID, nil
	case comments.TypeIDArticles:
		return CommentsType_ARTICLES_TYPE_ID, nil
	case comments.TypeIDForums:
		return CommentsType_FORUMS_TYPE_ID, nil
	}

	return 0, fmt.Errorf("`%v` is unknown comments type identifier", commentsType)
}

func convertModeratorAttention(value comments.ModeratorAttention) (ModeratorAttention, error) {
	switch value {
	case comments.ModeratorAttentionNone:
		return ModeratorAttention_NONE, nil
	case comments.ModeratorAttentionRequired:
		return ModeratorAttention_REQUIRED, nil
	case comments.ModeratorAttentionCompleted:
		return ModeratorAttention_COMPLETE, nil
	}

	return 0, fmt.Errorf("`%v` is unknown ModeratorAttention value", value)
}

func reverseConvertModeratorAttention(value ModeratorAttention) (comments.ModeratorAttention, error) {
	switch value {
	case ModeratorAttention_NONE:
		return comments.ModeratorAttentionNone, nil
	case ModeratorAttention_REQUIRED:
		return comments.ModeratorAttentionRequired, nil
	case ModeratorAttention_COMPLETE:
		return comments.ModeratorAttentionCompleted, nil
	}

	return 0, fmt.Errorf("`%v` is unknown ModeratorAttention value", value)
}

func extractMessage(
	ctx context.Context, row *comments.CommentMessage, repository *comments.Repository,
	picturesRepository *pictures.Repository, enforcer *casbin.Enforcer, userID int64, role string, canViewIP bool,
	fields *CommentMessageFields,
) (*APICommentsMessage, error) {
	canRemove := enforcer.Enforce(role, "comment", "remove")
	isModer := enforcer.Enforce(role, "global", "moderate")

	typeID, err := reverseConvertType(row.TypeID)
	if err != nil {
		return nil, err
	}

	parentID := row.ParentID.Int64
	if !row.ParentID.Valid {
		parentID = 0
	}

	ma, err := convertModeratorAttention(row.ModeratorAttention)
	if err != nil {
		return nil, err
	}

	isNew := false
	if fields.IsNew && userID > 0 {
		isNew, err = repository.IsNewMessage(ctx, row.TypeID, row.ItemID, row.CreatedAt, userID)
		if err != nil {
			return nil, err
		}
	}

	authorID := row.AuthorID.Int64
	if !row.AuthorID.Valid {
		authorID = 0
	}

	var (
		preview       string
		text          string
		vote          int32
		userVote      int32
		route         []string
		replies       []*APICommentsMessage
		pictureStatus PictureStatus
	)

	if canRemove || !row.Deleted {
		if fields.Preview {
			preview = util.GetTextPreview(row.Message, util.TextPreviewOptions{
				Maxlines:  1,
				Maxlength: comments.CommentMessagePreviewLength,
			})
		}

		if fields.Route {
			route, err = repository.MessageRowRoute(ctx, row.TypeID, row.ItemID, row.ID)
			if err != nil {
				return nil, err
			}
		}

		if fields.Text {
			text = row.Message
		}

		if fields.Vote {
			vote = row.Vote
		}

		if fields.UserVote {
			if userID != 0 {
				userVote, err = repository.UserVote(ctx, userID, row.ID)
				if err != nil {
					return nil, err
				}
			}
		}

		if fields.Replies {
			paginator := repository.Paginator(comments.Request{
				ItemID:       row.ItemID,
				TypeID:       row.TypeID,
				ParentID:     row.ID,
				PerPage:      MaxReplies,
				Order:        []exp.OrderedExpression{goqu.I("comment_message.datetime").Desc()},
				FetchMessage: fields.Preview || fields.Text,
				FetchVote:    fields.Vote,
				FetchIP:      canViewIP,
			})

			sqSelect, err := paginator.GetCurrentItems(ctx)
			if err != nil {
				return nil, err
			}

			rows := make([]*comments.CommentMessage, 0)

			err = sqSelect.ScanStructsContext(ctx, &rows)
			if err != nil {
				return nil, err
			}

			replies = make([]*APICommentsMessage, 0)

			for _, row := range rows {
				msg, err := extractMessage(ctx, row, repository, picturesRepository, enforcer, userID, role,
					canViewIP, fields)
				if err != nil {
					return nil, err
				}

				replies = append(replies, msg)
			}
		}

		if fields.Status && isModer {
			if row.TypeID == comments.TypeIDPictures {
				ps, err := picturesRepository.Status(ctx, row.ItemID)
				if err != nil {
					return nil, err
				}

				pictureStatus = reverseConvertPicturesStatus(ps)
			}
		}
	}

	ip := ""
	if canViewIP && row.IP != nil {
		ip = row.IP.String()
	}

	return &APICommentsMessage{
		Id:                 row.ID,
		TypeId:             typeID,
		ItemId:             row.ItemID,
		ParentId:           parentID,
		CreatedAt:          timestamppb.New(row.CreatedAt),
		Deleted:            row.Deleted,
		ModeratorAttention: ma,
		IsNew:              isNew,
		AuthorId:           authorID,
		Ip:                 ip,
		Preview:            preview,
		Text:               text,
		Vote:               vote,
		Route:              route,
		UserVote:           userVote,
		Replies:            replies,
		PictureStatus:      pictureStatus,
	}, nil
}

func NewCommentsGRPCServer(
	auth *Auth,
	commentsRepository *comments.Repository,
	usersRepository *users.Repository,
	picturesRepository *pictures.Repository,
	userExtractor *users.UserExtractor,
	enforcer *casbin.Enforcer,
) *CommentsGRPCServer {
	return &CommentsGRPCServer{
		auth:               auth,
		repository:         commentsRepository,
		usersRepository:    usersRepository,
		picturesRepository: picturesRepository,
		userExtractor:      userExtractor,
		enforcer:           enforcer,
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
		err = s.usersRepository.IncForumMessages(ctx, userID)
	} else {
		err = s.usersRepository.TouchLastMessage(ctx, userID)
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

func (s *CommentsGRPCServer) GetMessagePage(
	ctx context.Context, in *GetMessagePageRequest,
) (*APICommentsMessagePage, error) {
	itemID, typeID, page, err := s.repository.MessagePage(ctx, in.MessageId, in.PerPage)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	convertedTypeID, err := reverseConvertType(typeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APICommentsMessagePage{
		TypeId: convertedTypeID,
		ItemId: itemID,
		Page:   page,
	}, nil
}

func (s *CommentsGRPCServer) GetMessage(ctx context.Context, in *GetMessageRequest) (*APICommentsMessage, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	canViewIP := s.enforcer.Enforce(role, "user", "ip")

	fields := in.Fields
	if fields == nil {
		fields = &CommentMessageFields{}
	}

	row, err := s.repository.Message(ctx, in.Id, fields.Preview || fields.Text, fields.Vote, canViewIP)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if row == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	return extractMessage(ctx, row, s.repository, s.picturesRepository, s.enforcer, userID, role, canViewIP, fields)
}

func (s *CommentsGRPCServer) GetMessages(ctx context.Context, in *GetMessagesRequest) (*APICommentsMessages, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	isModer := s.enforcer.Enforce(role, "global", "moderate")
	canViewIP := s.enforcer.Enforce(role, "user", "ip")

	typeID, err := convertType(in.TypeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	fields := in.Fields
	if fields == nil {
		fields = &CommentMessageFields{}
	}

	options := comments.Request{
		ItemID:       in.ItemId,
		TypeID:       typeID,
		ParentID:     in.ParentId,
		NoParents:    in.NoParents,
		UserID:       in.UserId,
		Order:        []exp.OrderedExpression{goqu.I("comment_message.datetime").Desc()},
		FetchMessage: fields.Preview || fields.Text,
		FetchVote:    fields.Vote,
		FetchIP:      canViewIP,
		Page:         in.Page,
	}

	switch in.Order {
	case GetMessagesRequest_VOTE_DESC:
		options.Order = []exp.OrderedExpression{
			goqu.I("comment_message.vote").Desc(),
			goqu.I("comment_message.datetime DESC").Desc(),
		}
	case GetMessagesRequest_VOTE_ASC:
		options.Order = []exp.OrderedExpression{
			goqu.I("comment_message.vote").Asc(),
			goqu.I("comment_message.datetime DESC").Desc(),
		}
	case GetMessagesRequest_DATE_DESC:
		options.Order = []exp.OrderedExpression{goqu.I("comment_message.datetime").Desc()}
	case GetMessagesRequest_DATE_ASC, GetMessagesRequest_DEFAULT:
		options.Order = []exp.OrderedExpression{goqu.I("comment_message.datetime").Asc()}
	}

	if isModer {
		if len(in.UserIdentity) > 0 {
			options.UserID, err = s.usersRepository.UserIDByIdentity(ctx, in.UserIdentity)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		if in.ModeratorAttention != ModeratorAttention_NONE {
			ma, err := reverseConvertModeratorAttention(in.ModeratorAttention)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			options.ModeratorAttention = ma
		}

		if in.PicturesOfItemId > 0 {
			options.PicturesOfItemID = in.PicturesOfItemId
		}
	} else if in.ItemId == 0 && in.UserId == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if in.Limit <= 0 {
		in.Limit = 50000
	}

	options.PerPage = in.Limit

	paginator := s.repository.Paginator(options)

	msgs := make([]*APICommentsMessage, 0)

	if in.Limit > 0 {
		sqSelect, err := paginator.GetCurrentItems(ctx)
		if err != nil {
			return nil, err
		}

		rows := make([]*comments.CommentMessage, 0)

		err = sqSelect.ScanStructsContext(ctx, &rows)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			msg, err := extractMessage(ctx, row, s.repository, s.picturesRepository, s.enforcer, userID, role,
				canViewIP, fields)
			if err != nil {
				return nil, err
			}

			msgs = append(msgs, msg)
		}

		if userID > 0 && in.ItemId > 0 && in.TypeId > 0 {
			err = s.repository.SetSubscriptionSent(ctx, typeID, in.ItemId, userID, false)
			if err != nil {
				return nil, err
			}
		}
	}

	pages, err := paginator.GetPages(ctx)
	if err != nil {
		return nil, err
	}

	return &APICommentsMessages{
		Items: msgs,
		Paginator: &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
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
