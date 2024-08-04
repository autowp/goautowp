package goautowp

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PicturesGRPCServer struct {
	UnimplementedPicturesServer
	repository          *pictures.Repository
	auth                *Auth
	enforcer            *casbin.Enforcer
	events              *Events
	hostManager         *hosts.Manager
	messagingRepository *messaging.Repository
	userRepository      *users.Repository
	i18n                *i18nbundle.I18n
}

func NewPicturesGRPCServer(
	repository *pictures.Repository, auth *Auth, enforcer *casbin.Enforcer, events *Events, hostManager *hosts.Manager,
	messagingRepository *messaging.Repository, userRepository *users.Repository, i18n *i18nbundle.I18n,
) *PicturesGRPCServer {
	return &PicturesGRPCServer{
		repository:          repository,
		auth:                auth,
		enforcer:            enforcer,
		events:              events,
		hostManager:         hostManager,
		messagingRepository: messagingRepository,
		userRepository:      userRepository,
		i18n:                i18n,
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

	if vote && currentStatus == pictures.StatusRemoving {
		err = s.repository.SetStatus(ctx, pictureID, pictures.StatusInbox, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if (!vote) && currentStatus == pictures.StatusAccepted {
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

	err = s.repository.SetStatus(ctx, pictureID, pictures.StatusInbox, userID)
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

		uri.Path = "/picture/" + url.QueryEscape(picture.Identity)
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
		Status: pictures.StatusAccepted,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	inboxCount, err := s.repository.Count(ctx, pictures.ListOptions{
		Status: pictures.StatusInbox,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &PicturesUserSummary{
		AcceptedCount: int32(acceptedCount),
		InboxCount:    int32(inboxCount),
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

	canNormalize := pic.Status == pictures.StatusInbox && s.enforcer.Enforce(role, "picture", action)
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
