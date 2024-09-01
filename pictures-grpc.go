package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/frontend"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/paulmach/orb"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func extractPictureModerVoteTemplate(tpl *schema.PictureModerVoteTemplateRow) *ModerVoteTemplate {
	return &ModerVoteTemplate{
		Id:      tpl.ID,
		UserId:  tpl.UserID,
		Message: tpl.Message,
		Vote:    int32(tpl.Vote),
	}
}

func convertPictureItemType(pictureItemType PictureItemType) schema.PictureItemType {
	switch pictureItemType {
	case PictureItemType_PICTURE_ITEM_UNKNOWN:
		return 0
	case PictureItemType_PICTURE_ITEM_CONTENT:
		return schema.PictureItemContent
	case PictureItemType_PICTURE_ITEM_AUTHOR:
		return schema.PictureItemAuthor
	case PictureItemType_PICTURE_ITEM_COPYRIGHTS:
		return schema.PictureItemCopyrights
	}

	return 0
}

type PicturesGRPCServer struct {
	UnimplementedPicturesServer
	repository            *pictures.Repository
	auth                  *Auth
	enforcer              *casbin.Enforcer
	events                *Events
	hostManager           *hosts.Manager
	messagingRepository   *messaging.Repository
	userRepository        *users.Repository
	i18n                  *i18nbundle.I18n
	duplicateFinder       *DuplicateFinder
	textStorageRepository *textstorage.Repository
	telegramService       *telegram.Service
	itemRepository        *items.Repository
	commentRepository     *comments.Repository
}

func NewPicturesGRPCServer(
	repository *pictures.Repository, auth *Auth, enforcer *casbin.Enforcer, events *Events, hostManager *hosts.Manager,
	messagingRepository *messaging.Repository, userRepository *users.Repository, i18n *i18nbundle.I18n,
	duplicateFinder *DuplicateFinder, textStorageRepository *textstorage.Repository, telegramService *telegram.Service,
	itemRepository *items.Repository, commentRepository *comments.Repository,
) *PicturesGRPCServer {
	return &PicturesGRPCServer{
		repository:            repository,
		auth:                  auth,
		enforcer:              enforcer,
		events:                events,
		hostManager:           hostManager,
		messagingRepository:   messagingRepository,
		userRepository:        userRepository,
		i18n:                  i18n,
		duplicateFinder:       duplicateFinder,
		textStorageRepository: textStorageRepository,
		telegramService:       telegramService,
		itemRepository:        itemRepository,
		commentRepository:     commentRepository,
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

	vote, err := s.repository.GetVote(ctx, in.GetPictureId(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &PicturesVoteSummary{
		Value:    vote.Value,
		Positive: vote.Positive,
		Negative: vote.Negative,
	}, nil
}

func (s *PicturesGRPCServer) ValidatePictureModerVoteTemplateRow(
	tpl *schema.PictureModerVoteTemplateRow,
) ([]*errdetails.BadRequest_FieldViolation, error) {
	result := make([]*errdetails.BadRequest_FieldViolation, 0)

	var (
		problems []string
		err      error
	)

	messageInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Max: schema.ModerVoteTemplateMessageMaxLength},
		},
	}

	tpl.Message, problems, err = messageInputFilter.IsValidString(tpl.Message)
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: fv,
		})
	}

	return result, nil
}

func (s *PicturesGRPCServer) CreateModerVoteTemplate(
	ctx context.Context,
	in *ModerVoteTemplate,
) (*ModerVoteTemplate, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	tpl := schema.PictureModerVoteTemplateRow{
		UserID:  userID,
		Message: in.GetMessage(),
		Vote:    int8(in.GetVote()), //nolint: gosec
	}

	fvs, err := s.ValidatePictureModerVoteTemplateRow(&tpl)
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

	return extractPictureModerVoteTemplate(&tpl), nil
}

func (s *PicturesGRPCServer) DeleteModerVoteTemplate(
	ctx context.Context,
	in *DeleteModerVoteTemplateRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
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

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	rows, err := s.repository.GetModerVoteTemplates(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*ModerVoteTemplate, len(rows))
	for idx, item := range rows {
		result[idx] = extractPictureModerVoteTemplate(&item)
	}

	return &ModerVoteTemplates{
		Items: result,
	}, nil
}

func (s *PicturesGRPCServer) DeleteModerVote(ctx context.Context, in *DeleteModerVoteRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureID := in.GetPictureId()

	success, err := s.repository.DeleteModerVote(ctx, pictureID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:   userID,
			Message:  fmt.Sprintf("Отменена заявка на принятие/удаление картинки %d", pictureID),
			Pictures: []int64{pictureID},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) UpdateModerVote(ctx context.Context, in *UpdateModerVoteRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "moder_vote") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	pictureID := in.GetPictureId()
	vote := in.GetVote() > 0
	reason := in.GetReason()

	success, err := s.repository.CreateModerVote(ctx, pictureID, userID, vote, reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !success {
		return &emptypb.Empty{}, nil
	}

	currentStatus, err := s.repository.Status(ctx, pictureID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if vote && currentStatus == schema.PictureStatusRemoving {
		err = s.restoreFromRemoving(ctx, pictureID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if (!vote) && currentStatus == schema.PictureStatusAccepted {
		err = s.unaccept(ctx, pictureID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if in.GetSave() {
		exists, err := s.repository.IsModerVoteTemplateExists(ctx, userID, reason)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if !exists {
			tpl := schema.PictureModerVoteTemplateRow{
				UserID:  userID,
				Message: reason,
				Vote:    int8(in.GetVote()), //nolint: gosec
			}

			_, err = s.repository.CreateModerVoteTemplate(ctx, tpl)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
	}

	msgTemplate := "Подана заявка на удаление картинки %d"
	if vote {
		msgTemplate = "Подана заявка на принятие картинки %d"
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf(msgTemplate, pictureID),
		Pictures: []int64{pictureID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.notifyVote(ctx, pictureID, vote, reason, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UpdateModerVoteRequest) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	reasonInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
		},
	}

	s.Reason, problems, err = reasonInputFilter.IsValidString(s.GetReason())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "reason",
			Description: fv,
		})
	}

	voteInputFilter := validation.InputFilter{
		Validators: []validation.ValidatorInterface{
			&validation.InArray{
				HaystackInt32: []int32{-1, 1},
			},
		},
	}

	s.Vote, problems, err = voteInputFilter.IsValidInt32(s.GetVote())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "vote",
			Description: fv,
		})
	}

	return result, nil
}

func (s *PicturesGRPCServer) restoreFromRemoving(ctx context.Context, pictureID int64, userID int64) error {
	pic, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	err = s.repository.SetStatus(ctx, pic.ID, schema.PictureStatusInbox, userID)
	if err != nil {
		return err
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf("Картинки `%d` восстановлена из очереди удаления", pic.ID),
		Pictures: []int64{pic.ID},
	})
	if err != nil {
		return err
	}

	if pic.OwnerID.Valid {
		err = s.userRepository.RefreshPicturesCount(ctx, pic.OwnerID.Int64)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PicturesGRPCServer) unaccept(ctx context.Context, pictureID int64, userID int64) error {
	picture, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	err = s.repository.SetStatus(ctx, pictureID, schema.PictureStatusInbox, userID)
	if err != nil {
		return err
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf(`С картинки %d снят статус "принято"`, pictureID),
		Pictures: []int64{pictureID},
	})
	if err != nil {
		return err
	}

	if picture.OwnerID.Valid {
		err = s.userRepository.RefreshPicturesCount(ctx, picture.OwnerID.Int64)
		if err != nil {
			return err
		}
	}

	return s.NotifyInboxed(ctx, picture, userID)
}

func (s *PicturesGRPCServer) notifyVote(
	ctx context.Context, pictureID int64, vote bool, reason string, userID int64,
) error {
	picture, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	if !picture.OwnerID.Valid || picture.OwnerID.Int64 == userID {
		return nil
	}

	owner, err := s.userRepository.User(ctx, query.ListUsersOptions{ID: picture.OwnerID.Int64}, users.UserFields{})
	if err != nil {
		return err
	}

	if !s.enforcer.Enforce(owner.Role, "global", "moderate") {
		return nil
	}

	tpl := "pm/new-picture-%s-vote-%s/delete"
	if vote {
		tpl = "pm/new-picture-%s-vote-%s/accept"
	}

	return s.sendLocalizedMessage(
		ctx, userID, picture.OwnerID, tpl,
		func(language string) (map[string]interface{}, error) {
			uri, err := s.hostManager.URIByLanguage(language)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"Picture": frontend.PictureModerURL(uri, pictureID),
				"Reason":  reason,
			}, nil
		})
}

func (s *PicturesGRPCServer) GetUserSummary(ctx context.Context, _ *emptypb.Empty) (*PicturesUserSummary, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	acceptedCount, err := s.repository.Count(ctx, pictures.ListOptions{
		Status: schema.PictureStatusAccepted,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	inboxCount, err := s.repository.Count(ctx, pictures.ListOptions{
		Status: schema.PictureStatusInbox,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &PicturesUserSummary{
		AcceptedCount: int32(acceptedCount), //nolint: gosec
		InboxCount:    int32(inboxCount),    //nolint: gosec
	}, nil
}

func (s *PicturesGRPCServer) enforcePictureImageOperation(
	ctx context.Context, pictureID int64, action string,
) (int64, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return 0, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return 0, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pic, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	if pic == nil {
		return 0, status.Errorf(codes.NotFound, "NotFound")
	}

	canNormalize := pic.Status == schema.PictureStatusInbox && s.enforcer.Enforce(role, "picture", action)
	if !canNormalize {
		return 0, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	return userID, nil
}

func (s *PicturesGRPCServer) Normalize(ctx context.Context, in *PictureIDRequest) (*emptypb.Empty, error) {
	pictureID := in.GetId()

	userID, err := s.enforcePictureImageOperation(ctx, pictureID, "normalize")
	if err != nil {
		return nil, err
	}

	err = s.repository.Normalize(ctx, pictureID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf("К картинке %d применён normalize", pictureID),
		Pictures: []int64{pictureID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) Flop(ctx context.Context, in *PictureIDRequest) (*emptypb.Empty, error) {
	pictureID := in.GetId()

	userID, err := s.enforcePictureImageOperation(ctx, pictureID, "flop")
	if err != nil {
		return nil, err
	}

	err = s.repository.Flop(ctx, pictureID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf("К картинке %d применён flop", pictureID),
		Pictures: []int64{pictureID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) DeleteSimilar(ctx context.Context, in *DeleteSimilarRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if err = s.duplicateFinder.HideSimilar(ctx, in.GetId(), in.GetSimilarPictureId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  "Отменёно предупреждение о повторе",
		Pictures: []int64{in.GetId(), in.GetSimilarPictureId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) Repair(ctx context.Context, in *PictureIDRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.repository.Repair(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPictureItemArea(
	ctx context.Context, in *SetPictureItemAreaRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureItemType := convertPictureItemType(in.GetType())

	err = s.repository.SetPictureItemArea(
		ctx, in.GetPictureId(), in.GetItemId(), pictureItemType, pictures.PictureItemArea{
			Left:   uint16(in.GetCropLeft()),   //nolint: gosec
			Top:    uint16(in.GetCropTop()),    //nolint: gosec
			Width:  uint16(in.GetCropWidth()),  //nolint: gosec
			Height: uint16(in.GetCropHeight()), //nolint: gosec
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  "Выделение области на картинке",
		Pictures: []int64{in.GetPictureId()},
		Items:    []int64{in.GetItemId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPictureItemPerspective(
	ctx context.Context, in *SetPictureItemPerspectiveRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		pic, err := s.repository.Picture(ctx, in.GetPictureId())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if !pic.OwnerID.Valid || pic.OwnerID.Int64 != userID || pic.Status != schema.PictureStatusInbox {
			return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
		}
	}

	pictureItemType := convertPictureItemType(in.GetType())

	err = s.repository.SetPictureItemPerspective(
		ctx, in.GetPictureId(), in.GetItemId(), pictureItemType, in.GetPerspectiveId(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  "Установка ракурса картинки",
		Pictures: []int64{in.GetPictureId()},
		Items:    []int64{in.GetItemId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPictureItemItemID(
	ctx context.Context, in *SetPictureItemItemIDRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "move") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureItemType := convertPictureItemType(in.GetType())

	err = s.repository.SetPictureItemItemID(
		ctx, in.GetPictureId(), in.GetItemId(), pictureItemType, in.GetNewItemId(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID: userID,
		Message: fmt.Sprintf(
			"Картинка %d перемещена из %d в %d",
			in.GetPictureId(), in.GetItemId(), in.GetNewItemId(),
		),
		Pictures: []int64{in.GetPictureId()},
		Items:    []int64{in.GetItemId(), in.GetNewItemId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) DeletePictureItem(
	ctx context.Context, in *DeletePictureItemRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "move") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureItemType := convertPictureItemType(in.GetType())

	success, err := s.repository.DeletePictureItem(ctx, in.GetPictureId(), in.GetItemId(), pictureItemType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !success {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf("Картинка %d отвязана от %d", in.GetPictureId(), in.GetItemId()),
		Pictures: []int64{in.GetPictureId()},
		Items:    []int64{in.GetItemId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) CreatePictureItem(
	ctx context.Context, in *CreatePictureItemRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "move") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureItemType := convertPictureItemType(in.GetType())

	success, err := s.repository.CreatePictureItem(
		ctx, in.GetPictureId(), in.GetItemId(), pictureItemType, in.GetPerspectiveId(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID: userID,
			Message: fmt.Sprintf(
				"Картинка %d связана с %d",
				in.GetPictureId(), in.GetItemId(),
			),
			Pictures: []int64{in.GetPictureId()},
			Items:    []int64{in.GetItemId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPictureCrop(ctx context.Context, in *SetPictureCropRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "crop") {
		pic, err := s.repository.Picture(ctx, in.GetPictureId())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, status.Errorf(codes.NotFound, "NotFound")
			}

			return nil, status.Error(codes.Internal, err.Error())
		}

		if !pic.OwnerID.Valid || pic.OwnerID.Int64 != userID || pic.Status != schema.PictureStatusInbox {
			return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
		}
	}

	err = s.repository.SetPictureCrop(
		ctx, in.GetPictureId(), sampler.Crop{
			Left:   int(in.GetCropLeft()),
			Top:    int(in.GetCropTop()),
			Width:  int(in.GetCropWidth()),
			Height: int(in.GetCropHeight()),
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  "Выделение области на картинке",
		Pictures: []int64{in.GetPictureId()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) ClearReplacePicture(ctx context.Context, in *PictureIDRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "picture", "move") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	success, err := s.repository.ClearReplacePicture(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:   userID,
			Message:  fmt.Sprintf("Замена для %d отклонена", in.GetId()),
			Pictures: []int64{in.GetId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) AcceptReplacePicture(ctx context.Context, in *PictureIDRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pic, err := s.repository.Picture(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !pic.ReplacePictureID.Valid {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	replacePicture, err := s.repository.Picture(ctx, pic.ReplacePictureID.Int64)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.canReplace(pic, replacePicture, role) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	// statuses
	if pic.Status != schema.PictureStatusAccepted {
		_, success, err := s.repository.Accept(ctx, pic.ID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "Accept error: "+err.Error())
		}

		if success && pic.OwnerID.Valid {
			err = s.userRepository.RefreshPicturesCount(ctx, pic.OwnerID.Int64)
			if err != nil {
				return nil, status.Error(codes.Internal, "RefreshPicturesCount error: "+err.Error())
			}
		}
	}

	if replacePicture.Status != schema.PictureStatusRemoving && replacePicture.Status != schema.PictureStatusRemoved {
		success, err := s.repository.QueueRemove(ctx, replacePicture.ID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "QueueRemove error: "+err.Error())
		}

		if success && replacePicture.OwnerID.Valid {
			err = s.userRepository.RefreshPicturesCount(ctx, replacePicture.OwnerID.Int64)
			if err != nil {
				return nil, status.Error(codes.Internal, "RefreshPicturesCount error: "+err.Error())
			}
		}
	}

	// comments
	err = s.commentRepository.MoveMessages(ctx,
		schema.CommentMessageTypeIDPictures, replacePicture.ID,
		schema.CommentMessageTypeIDPictures, pic.ID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// pms
	recipients := make(map[int64]sql.NullInt64)

	if pic.OwnerID.Valid {
		recipients[pic.OwnerID.Int64] = pic.OwnerID
	}

	if replacePicture.OwnerID.Valid {
		recipients[replacePicture.OwnerID.Int64] = replacePicture.OwnerID
	}

	user, err := s.userRepository.User(ctx, query.ListUsersOptions{ID: userID}, users.UserFields{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, recipient := range recipients {
		err = s.sendLocalizedMessage(
			ctx, userID, recipient, "pm/user-%s-accept-replace-%s-%s",
			func(language string) (map[string]interface{}, error) {
				uri, err := s.hostManager.URIByLanguage(language)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"ModeratorURL":          frontend.UserURL(uri, userID, user.Identity),
					"PictureURL":            frontend.PictureURL(uri, pic.Identity),
					"ReplacementPictureURL": frontend.PictureURL(uri, replacePicture.Identity),
				}, nil
			})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// log
	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf("Замена %d на %d", replacePicture.ID, pic.ID),
		Pictures: []int64{replacePicture.ID, pic.ID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPicturePoint(ctx context.Context, in *SetPicturePointRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	var (
		point    = in.GetPoint()
		orbPoint *orb.Point
	)

	if point.GetLatitude() != 0 || point.GetLongitude() != 0 {
		orbPoint = &orb.Point{point.GetLongitude(), point.GetLatitude()}
	}

	success, err := s.repository.SetPicturePoint(ctx, in.GetPictureId(), orbPoint)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:   userID,
			Message:  "Изменена точка для изображения",
			Pictures: []int64{in.GetPictureId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) UpdatePicture(ctx context.Context, in *UpdatePictureRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	inDate := in.GetTakenDate()

	success, err := s.repository.UpdatePicture(
		ctx, in.GetId(), in.GetName(),
		int16(inDate.GetYear()), int8(inDate.GetMonth()), int8(inDate.GetDay()), //nolint: gosec
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:   userID,
			Message:  "Редактирование изображения (дата, особое название)",
			Pictures: []int64{in.GetId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) SetPictureCopyrights(
	ctx context.Context, in *SetPictureCopyrightsRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pictureID := in.GetId()

	success, textID, err := s.repository.SetPictureCopyrights(ctx, pictureID, in.GetCopyrights(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "NotFound")
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:   userID,
			Message:  "Редактирование текста копирайтов изображения",
			Pictures: []int64{in.GetId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.notifyCopyrightsEdited(ctx, pictureID, textID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) notifyCopyrightsEdited(
	ctx context.Context, pictureID int64, textID int32, userID int64,
) error {
	revUserIDs, err := s.textStorageRepository.TextUserIDs(ctx, textID)
	if err != nil {
		return err
	}

	revUserIDs = util.RemoveValueFromArray(revUserIDs, userID)
	if len(revUserIDs) == 0 {
		return nil
	}

	userRows, _, err := s.userRepository.Users(ctx, query.ListUsersOptions{IDs: revUserIDs}, users.UserFields{})
	if err != nil {
		return err
	}

	picture, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	for _, userRow := range userRows {
		pictureURL, err := s.pictureURL(picture.Identity, userRow.Language)
		if err != nil {
			return err
		}

		userURL, err := s.userURL(userRow.ID, userRow.Identity, userRow.Language)
		if err != nil {
			return err
		}

		localizer := s.i18n.Localizer(userRow.Language)

		message, err := localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "pm/user-%s-edited-picture-copyrights-%s-%s",
			},
			TemplateData: map[string]interface{}{
				"User":       userURL,
				"PictureURL": pictureURL,
			},
		})
		if err != nil {
			return err
		}

		err = s.messagingRepository.CreateMessage(ctx, 0, userRow.ID, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PicturesGRPCServer) userURL(userID int64, identity *string, language string) (string, error) {
	userURL, err := s.hostManager.URIByLanguage(language)
	if err != nil {
		return "", err
	}

	userURL.Path = frontend.UserPath(userID, identity)

	return userURL.String(), nil
}

func (s *PicturesGRPCServer) pictureURL(identity string, language string) (string, error) {
	pictureURL, err := s.hostManager.URIByLanguage(language)
	if err != nil {
		return "", err
	}

	return frontend.PictureURL(pictureURL, identity), nil
}

func (s *PicturesGRPCServer) SetPictureStatus(
	ctx context.Context, in *SetPictureStatusRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	pic, err := s.repository.Picture(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	switch in.GetStatus() {
	case PictureStatus_PICTURE_STATUS_ACCEPTED:
		canAccept, err := s.canAccept(ctx, pic, role)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if !canAccept {
			return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
		}

		isFirstTimeAccepted, success, err := s.repository.Accept(ctx, pic.ID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "Accept error: "+err.Error())
		}

		if success {
			err = s.events.Add(ctx, Event{
				UserID:   userID,
				Message:  fmt.Sprintf("Картинка `%d` принята", pic.ID),
				Pictures: []int64{pic.ID},
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			if pic.OwnerID.Valid {
				err = s.userRepository.RefreshPicturesCount(ctx, pic.OwnerID.Int64)
				if err != nil {
					return nil, status.Error(codes.Internal, "RefreshPicturesCount error: "+err.Error())
				}
			}

			err = s.NotifyAccepted(ctx, pic, userID, isFirstTimeAccepted)
			if err != nil {
				return nil, status.Error(codes.Internal, "NotifyAccepted error: "+err.Error())
			}
		}
	case PictureStatus_PICTURE_STATUS_INBOX:
		if pic.Status == schema.PictureStatusRemoving {
			canRestore := s.enforcer.Enforce(role, "picture", "restore")
			if !canRestore {
				return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
			}

			err = s.restoreFromRemoving(ctx, pic.ID, userID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		} else if pic.Status == schema.PictureStatusAccepted {
			canUnaccept := s.enforcer.Enforce(role, "picture", "unaccept")
			if !canUnaccept {
				return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
			}

			err = s.unaccept(ctx, pic.ID, userID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
	case PictureStatus_PICTURE_STATUS_REMOVING:
		canDelete, err := s.pictureCanDelete(ctx, pic, role, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if !canDelete {
			return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
		}

		success, err := s.repository.QueueRemove(ctx, pic.ID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if success {
			err = s.notifyRemoving(ctx, pic, userID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			err = s.events.Add(ctx, Event{
				UserID:   userID,
				Message:  fmt.Sprintf("Картинка `%d` поставлена в очередь на удаление", pic.ID),
				Pictures: []int64{pic.ID},
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

	case PictureStatus_PICTURE_STATUS_REMOVED:
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")

	case PictureStatus_PICTURE_STATUS_UNKNOWN:
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	return &emptypb.Empty{}, nil
}

func (s *PicturesGRPCServer) sendMessage(
	ctx context.Context, userID int64, receiverID sql.NullInt64, messageFunc func(language string) (string, error),
) error {
	if !receiverID.Valid || (receiverID.Int64 == userID) {
		return nil
	}

	notDeleted := false

	receiver, err := s.userRepository.User(ctx, query.ListUsersOptions{ID: receiverID.Int64, Deleted: &notDeleted},
		users.UserFields{})
	if err != nil && !errors.Is(err, users.ErrUserNotFound) {
		return err
	}

	if receiver == nil {
		return nil
	}

	message, err := messageFunc(receiver.Language)
	if err != nil {
		return err
	}

	return s.messagingRepository.CreateMessage(ctx, 0, receiver.ID, message)
}

func (s *PicturesGRPCServer) sendLocalizedMessage(
	ctx context.Context, userID int64, receiverID sql.NullInt64, messageID string,
	templateDataFunc func(language string) (map[string]interface{}, error),
) error {
	if !receiverID.Valid || (receiverID.Int64 == userID) {
		return nil
	}

	notDeleted := false

	receiver, err := s.userRepository.User(ctx, query.ListUsersOptions{ID: receiverID.Int64, Deleted: &notDeleted},
		users.UserFields{})
	if err != nil && !errors.Is(err, users.ErrUserNotFound) {
		return err
	}

	if receiver == nil {
		return nil
	}

	localizer := s.i18n.Localizer(receiver.Language)

	templateData, err := templateDataFunc(receiver.Language)
	if err != nil {
		return err
	}

	message, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageID,
		},
		TemplateData: templateData,
	})
	if err != nil {
		return err
	}

	return s.messagingRepository.CreateMessage(ctx, 0, receiver.ID, message)
}

func (s *PicturesGRPCServer) NotifyAccepted(
	ctx context.Context, pic *schema.PictureRow, userID int64, isFirstTimeAccepted bool,
) error {
	if isFirstTimeAccepted {
		err := s.sendLocalizedMessage(
			ctx, userID, pic.OwnerID, "pm/your-picture-accepted-%s",
			func(language string) (map[string]interface{}, error) {
				pictureURL, err := s.pictureURL(pic.Identity, language)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"PictureURL": pictureURL,
				}, nil
			})
		if err != nil {
			return fmt.Errorf("sendLocalizedMessage: %w", err)
		}

		err = s.telegramService.NotifyPicture(ctx, pic, s.itemRepository)
		if err != nil {
			return fmt.Errorf("NotifyPicture: %w", err)
		}
	}

	err := s.sendMessage(
		ctx, userID, pic.ChangeStatusUserID, func(language string) (string, error) {
			pictureURL, err := s.pictureURL(pic.Identity, language)
			if err != nil {
				return "", err
			}

			return "Принята картинка " + pictureURL, nil
		})
	if err != nil {
		return fmt.Errorf("sendMessage: %w", err)
	}

	return nil
}

func (s *PicturesGRPCServer) NotifyInboxed(ctx context.Context, pic *schema.PictureRow, userID int64) error {
	if !pic.ChangeStatusUserID.Valid || pic.ChangeStatusUserID.Int64 == userID {
		return nil
	}

	return s.sendMessage(ctx, userID, pic.ChangeStatusUserID, func(language string) (string, error) {
		pictureURL, err := s.pictureURL(pic.Identity, language)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf(
			"С картинки `%s` снят статус \"принято\"",
			pictureURL,
		), nil
	})
}

func (s *PicturesGRPCServer) notifyRemoving(ctx context.Context, pic *schema.PictureRow, userID int64) error {
	return s.sendLocalizedMessage(
		ctx, userID, pic.OwnerID, "pm/your-picture-%s-enqueued-to-remove-%s",
		func(language string) (map[string]interface{}, error) {
			deleteRequests, err := s.repository.NegativeVotes(ctx, pic.ID)
			if err != nil {
				return nil, err
			}

			reasons := make([]string, 0, len(deleteRequests))

			for _, request := range deleteRequests {
				user, err := s.userRepository.User(ctx, query.ListUsersOptions{ID: request.UserID}, users.UserFields{})
				if err != nil {
					return nil, err
				}

				userURL, err := s.userURL(user.ID, user.Identity, user.Language)
				if err != nil {
					return nil, err
				}

				reasons = append(reasons, userURL+" : "+request.Reason)
			}

			pictureURL, err := s.pictureURL(pic.Identity, language)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"PictureURL": pictureURL,
				"Reasons":    strings.Join(reasons, "\n"),
			}, nil
		})
}

func (s *PicturesGRPCServer) canAccept(ctx context.Context, picture *schema.PictureRow, role string) (bool, error) {
	if !s.enforcer.Enforce(role, "picture", "accept") {
		return false, nil
	}

	return s.repository.CanAccept(ctx, picture)
}

func (s *PicturesGRPCServer) pictureCanDelete(
	ctx context.Context, picture *schema.PictureRow, role string, userID int64,
) (bool, error) {
	canDelete, err := s.repository.CanDelete(ctx, picture)
	if err != nil {
		return false, err
	}

	if !canDelete {
		return false, nil
	}

	if s.enforcer.Enforce(role, "picture", "remove") {
		return s.repository.HasVote(ctx, picture.ID, userID)
	}

	if s.enforcer.Enforce(role, "picture", "remove_by_vote") {
		hasVote, err := s.repository.HasVote(ctx, picture.ID, userID)
		if err != nil {
			return false, err
		}

		if hasVote {
			acceptVotes, err := s.repository.PositiveVotesCount(ctx, picture.ID)
			if err != nil {
				return false, err
			}

			deleteVotes, err := s.repository.NegativeVotesCount(ctx, picture.ID)
			if err != nil {
				return false, err
			}

			return deleteVotes > acceptVotes, nil
		}
	}

	return false, nil
}

func (s *PicturesGRPCServer) canReplace(picture, replacedPicture *schema.PictureRow, role string) bool {
	return (picture.Status == schema.PictureStatusAccepted ||
		picture.Status == schema.PictureStatusInbox && s.enforcer.Enforce(role, "picture", "accept")) &&
		(replacedPicture.Status == schema.PictureStatusRemoving ||
			replacedPicture.Status == schema.PictureStatusRemoved ||
			replacedPicture.Status == schema.PictureStatusInbox &&
				s.enforcer.Enforce(role, "picture", "remove_by_vote") ||
			replacedPicture.Status == schema.PictureStatusAccepted &&
				s.enforcer.Enforce(role, "picture", "unaccept") &&
				s.enforcer.Enforce(role, "picture", "remove_by_vote")) &&
		s.enforcer.Enforce(role, "picture", "move")
}
