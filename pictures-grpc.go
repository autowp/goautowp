package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
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
}

func NewPicturesGRPCServer(
	repository *pictures.Repository, auth *Auth, enforcer *casbin.Enforcer, events *Events, hostManager *hosts.Manager,
	messagingRepository *messaging.Repository, userRepository *users.Repository, i18n *i18nbundle.I18n,
	duplicateFinder *DuplicateFinder, textStorageRepository *textstorage.Repository,
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
		UserId:  tpl.UserID,
		Message: tpl.Message,
		Vote:    tpl.Vote,
	}, nil
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
			UserId:  item.UserID,
		}
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "moder_vote"); !res {
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
		err = s.repository.SetStatus(ctx, pictureID, schema.PictureStatusInbox, userID)
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
			tpl := pictures.ModerVoteTemplate{
				UserID:  userID,
				Message: reason,
				Vote:    in.GetVote(),
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

func (s *PicturesGRPCServer) unaccept(ctx context.Context, pictureID int64, userID int64) error {
	picture, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	var previousStatusUserID int64
	if picture.ChangeStatusUserID.Valid {
		previousStatusUserID = picture.ChangeStatusUserID.Int64
	}

	err = s.repository.SetStatus(ctx, pictureID, schema.PictureStatusInbox, userID)
	if err != nil {
		return err
	}

	if picture.OwnerID.Valid {
		err = s.userRepository.RefreshPicturesCount(ctx, picture.OwnerID.Int64)
		if err != nil {
			return err
		}
	}

	err = s.events.Add(ctx, Event{
		UserID:   userID,
		Message:  fmt.Sprintf(`С картинки %d снят статус "принято"`, pictureID),
		Pictures: []int64{pictureID},
	})
	if err != nil {
		return err
	}

	if previousStatusUserID != userID {
		language, err := s.userRepository.UserLanguage(ctx, previousStatusUserID)
		if err != nil {
			return err
		}

		uri, err := s.hostManager.URIByLanguage(language)
		if err != nil {
			return err
		}

		uri.Path = s.pictureURLPath(picture.Identity)
		message := fmt.Sprintf(
			`С картинки %s снят статус "принято"`,
			uri.String(),
		)

		err = s.messagingRepository.CreateMessage(ctx, 0, previousStatusUserID, message)
		if err != nil {
			return err
		}
	}

	return nil
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

	owner, err := s.userRepository.User(ctx, users.GetUsersOptions{ID: picture.OwnerID.Int64})
	if err != nil {
		return err
	}

	if !s.enforcer.Enforce(owner.Role, "global", "moderate") {
		return nil
	}

	language, err := s.userRepository.UserLanguage(ctx, picture.OwnerID.Int64)
	if err != nil {
		return err
	}

	uri, err := s.hostManager.URIByLanguage(language)
	if err != nil {
		return err
	}

	uri.Path = "/moder/pictures/" + strconv.FormatInt(pictureID, 10)

	tpl := "pm/new-picture-%s-vote-%s/delete"
	if vote {
		tpl = "pm/new-picture-%s-vote-%s/accept"
	}

	localizer := s.i18n.Localizer(language)

	message, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: tpl,
		},
		TemplateData: map[string]interface{}{
			"Picture": uri.String(),
			"Reason":  reason,
		},
	})
	if err != nil {
		return err
	}

	err = s.messagingRepository.CreateMessage(ctx, 0, owner.ID, message)
	if err != nil {
		return err
	}

	return nil
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "move"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "move"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "move"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "crop"); !res {
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

	if res := s.enforcer.Enforce(role, "picture", "move"); !res {
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

func (s *PicturesGRPCServer) SetPicturePoint(ctx context.Context, in *SetPicturePointRequest) (*emptypb.Empty, error) {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

	userRows, _, err := s.userRepository.Users(ctx, users.GetUsersOptions{IDs: revUserIDs})
	if err != nil {
		return err
	}

	picture, err := s.repository.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	pictureURLPath := s.pictureURLPath(picture.Identity)

	for _, userRow := range userRows {
		pictureURL, err := s.hostManager.URIByLanguage(userRow.Language)
		if err != nil {
			return err
		}

		pictureURL.Path = pictureURLPath

		userURL, err := s.hostManager.URIByLanguage(userRow.Language)
		if err != nil {
			return err
		}

		userURL.Path = s.userURLPath(userRow.ID, userRow.Identity)

		localizer := s.i18n.Localizer(userRow.Language)

		message, err := localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "pm/user-%s-edited-picture-copyrights-%s-%s",
			},
			TemplateData: map[string]interface{}{
				"User":       userURL.String(),
				"PictureURL": pictureURL.String(),
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

func (s *PicturesGRPCServer) userURLPath(userID int64, identity *string) string {
	var resIdentity string
	if identity == nil || len(*identity) == 0 {
		resIdentity = "user" + strconv.FormatInt(userID, 10)
	} else {
		resIdentity = *identity
	}

	return "/users/" + url.QueryEscape(resIdentity)
}

func (s *PicturesGRPCServer) pictureURLPath(identity string) string {
	return "/picture/" + url.QueryEscape(identity)
}
