package goautowp

import (
	"context"
	"fmt"
	"strings"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func convertUserFields(fields *UserFields, currentUserRoles []string) users.UserFields {
	lastIP := false
	if fields.GetLastIp() && util.Contains(currentUserRoles, users.RoleModer) {
		lastIP = true
	}

	login := false
	if fields.GetLogin() && util.Contains(currentUserRoles, users.RoleModer) {
		login = true
	}

	return users.UserFields{
		Email:         fields.GetEmail(),
		Timezone:      fields.GetTimezone(),
		Language:      fields.GetLanguage(),
		VotesPerDay:   fields.GetVotesPerDay(),
		VotesLeft:     fields.GetVotesLeft(),
		RegDate:       fields.GetRegDate(),
		PicturesAdded: fields.GetPicturesAdded(),
		LastIP:        lastIP,
		LastOnline:    fields.GetLastOnline(),
		Login:         login,
	}
}

type UsersGRPCServer struct {
	UnimplementedUsersServer
	auth               *Auth
	contactsRepository *ContactsRepository
	userRepository     *users.Repository
	events             *Events
	languages          map[string]config.LanguageConfig
	captcha            bool
	userExtractor      *UserExtractor
}

func NewUsersGRPCServer(
	auth *Auth,
	contactsRepository *ContactsRepository,
	userRepository *users.Repository,
	events *Events,
	languages map[string]config.LanguageConfig,
	captcha bool,
	userExtractor *UserExtractor,
) *UsersGRPCServer {
	return &UsersGRPCServer{
		auth:               auth,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		events:             events,
		languages:          languages,
		captcha:            captcha,
		userExtractor:      userExtractor,
	}
}

func (s *UsersGRPCServer) Me(ctx context.Context, in *APIMeRequest) (*APIUser, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.GetUser(ctx, &APIGetUserRequest{
		UserId: userID,
		Fields: in.GetFields(),
	})
}

func (s *UsersGRPCServer) GetUser(ctx context.Context, in *APIGetUserRequest) (*APIUser, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	dbUser, err := s.userRepository.User(ctx, &query.UserListOptions{
		ID:       in.GetUserId(),
		Identity: in.GetIdentity(),
	}, convertUserFields(in.GetFields(), role), users.OrderByNone)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if dbUser == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	apiUser, err := s.userExtractor.Extract(ctx, dbUser, in.GetFields(), userID, role)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return apiUser, err
}

func (s *UsersGRPCServer) DeleteUser(ctx context.Context, in *APIDeleteUserRequest) (*emptypb.Empty, error) {
	userID, roles, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !util.Contains(roles, users.RoleAdmin) {
		if userID != in.GetUserId() {
			return nil, status.Errorf(codes.Internal, "Forbidden")
		}

		match, err := s.userRepository.PasswordMatch(ctx, in.GetUserId(), in.GetPassword())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if !match {
			return nil, wrapFieldViolations([]*errdetails.BadRequest_FieldViolation{{
				Field:       "oldPassword",
				Description: "Password is incorrect",
			}})
		}
	}

	ctx = context.WithoutCancel(ctx)

	success, err := s.userRepository.DeleteUser(ctx, in.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.contactsRepository.deleteUserEverywhere(ctx, in.GetUserId())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.events.Add(ctx, Event{
			UserID:  userID,
			Message: fmt.Sprintf("Удаление пользователя №%d", in.GetUserId()),
			Users:   []int64{in.GetUserId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) DisableUserCommentsNotifications(
	ctx context.Context,
	in *APIUserPreferencesRequest,
) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if in.GetUserId() == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.userRepository.SetDisableUserCommentsNotifications(ctx, userID, in.GetUserId(), true)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) EnableUserCommentsNotifications(
	ctx context.Context,
	in *APIUserPreferencesRequest,
) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if in.GetUserId() == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.userRepository.SetDisableUserCommentsNotifications(ctx, userID, in.GetUserId(), false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) GetUserPreferences(
	ctx context.Context,
	in *APIUserPreferencesRequest,
) (*APIUserPreferencesResponse, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if in.GetUserId() == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	prefs, err := s.userRepository.UserPreferences(ctx, userID, in.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIUserPreferencesResponse{
		DisableCommentsNotifications: prefs.DisableCommentsNotifications,
	}, nil
}

func (s *UsersGRPCServer) GetUsers(ctx context.Context, in *APIUsersRequest) (*APIUsersResponse, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rows, pages, err := s.userRepository.Users(ctx, &query.UserListOptions{
		IsOnline: in.GetIsOnline(),
		Limit:    in.GetLimit(),
		Page:     in.GetPage(),
		Search:   in.GetSearch(),
		IDs:      in.GetId(),
	}, convertUserFields(in.GetFields(), role), users.OrderByNone)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUser, 0)

	for idx := range rows {
		apiUser, err := s.userExtractor.Extract(ctx, &rows[idx], in.GetFields(), userID, role)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result = append(result, apiUser)
	}

	var paginator *Pages
	if pages != nil {
		paginator = &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Last:             pages.Last,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
			Next:             pages.Next,
			Previous:         pages.Previous,
		}
	}

	return &APIUsersResponse{
		Items:     result,
		Paginator: paginator,
	}, nil
}

func (s *UsersGRPCServer) GetAccounts(ctx context.Context, _ *emptypb.Empty) (*APIAccountsResponse, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	rows, err := s.userRepository.UserAccounts(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	accounts := make([]*APIAccountsAccount, 0, len(rows))

	for _, row := range rows {
		if row.ServiceID != "keycloak" {
			canRemove, err := s.canRemoveAccount(ctx, userID, row.ID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			accounts = append(accounts, &APIAccountsAccount{
				Icon:      "fa fa-" + strings.ReplaceAll(row.ServiceID, "googleplus", "google-plus"),
				Id:        row.ID,
				Link:      row.Link,
				Name:      row.Name,
				CanRemove: canRemove,
			})
		}
	}

	return &APIAccountsResponse{
		Items: accounts,
	}, nil
}

func (s *UsersGRPCServer) canRemoveAccount(ctx context.Context, userID int64, id int64) (bool, error) {
	user, err := s.userRepository.User(ctx, &query.UserListOptions{ID: userID}, users.UserFields{}, users.OrderByNone)
	if err != nil {
		return false, err
	}

	if user.EMail != nil && len(*user.EMail) > 0 {
		return true, nil
	}

	return s.userRepository.HaveAccountsForOtherServices(ctx, userID, id)
}

func (s *UsersGRPCServer) DeleteUserAccount(ctx context.Context, in *DeleteUserAccountRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	canRemove, err := s.canRemoveAccount(ctx, userID, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !canRemove {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.userRepository.RemoveUserAccount(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) DeleteUserPhoto(ctx context.Context, in *DeleteUserPhotoRequest) (*emptypb.Empty, error) {
	userID, roles, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(roles, users.RoleUsersModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if in.GetId() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "id is zero")
	}

	success, err := s.userRepository.DeletePhoto(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.events.Add(ctx, Event{
			UserID:  userID,
			Message: fmt.Sprintf("Удаление фотографии пользователя №%d", in.GetId()),
			Users:   []int64{in.GetId()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) UpdateUser(ctx context.Context, in *UpdateUserRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	values := in.GetUser()

	if values.GetId() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "id is zero")
	}

	if userID != values.GetId() {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	maskPaths := in.GetUpdateMask().GetPaths()

	if maskPaths == nil {
		maskPaths = []string{}
	}

	InvalidParams, err := values.Validate(s.languages, maskPaths)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	set := schema.UsersRow{
		ID: userID,
	}

	if util.Contains(maskPaths, "language") {
		set.Language = values.GetLanguage()
	}

	if util.Contains(maskPaths, "timezone") {
		set.Timezone = values.GetTimezone()
	}

	err = s.userRepository.UpdateUser(ctx, set, maskPaths)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
