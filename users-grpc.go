package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UsersGRPCServer struct {
	UnimplementedUsersServer
	oauthSecret        string
	db                 *sql.DB
	enforcer           *casbin.Enforcer
	contactsRepository *ContactsRepository
	userRepository     *users.Repository
	events             *Events
	languages          map[string]config.LanguageConfig
	captcha            bool
	passwordRecovery   *PasswordRecovery
	userExtractor      *UserExtractor
}

func NewUsersGRPCServer(
	oauthSecret string,
	db *sql.DB,
	enforcer *casbin.Enforcer,
	contactsRepository *ContactsRepository,
	userRepository *users.Repository,
	events *Events,
	languages map[string]config.LanguageConfig,
	captcha bool,
	passwordRecovery *PasswordRecovery,
	userExtractor *UserExtractor,
) *UsersGRPCServer {
	return &UsersGRPCServer{
		oauthSecret:        oauthSecret,
		db:                 db,
		enforcer:           enforcer,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		events:             events,
		languages:          languages,
		captcha:            captcha,
		passwordRecovery:   passwordRecovery,
		userExtractor:      userExtractor,
	}
}

func (s *UsersGRPCServer) CreateUser(ctx context.Context, in *APICreateUserRequest) (*emptypb.Empty, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed extract peer from context")
	}
	remoteAddr := p.Addr.String()

	language, ok := s.languages[in.Language]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "language `%s` is not defined", in.Language)
	}

	user := users.CreateUserOptions{
		Name:            in.Name,
		Email:           in.Email,
		Timezone:        language.Timezone,
		Language:        in.Language,
		Password:        in.Password,
		PasswordConfirm: in.PasswordConfirm,
		Captcha:         in.Captcha,
	}

	fv, err := s.userRepository.ValidateCreateUser(user, s.captcha, remoteAddr)
	if err != nil {
		return nil, err
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	_, err = s.userRepository.CreateUser(user)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) GetUser(_ context.Context, in *APIGetUserRequest) (*APIUser, error) {
	fields := in.Fields
	m := make(map[string]bool)
	for _, e := range fields {
		m[e] = true
	}

	dbUser, err := s.userRepository.GetUser(users.GetUsersOptions{
		ID:     in.UserId,
		Fields: m,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dbUser == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	return s.userExtractor.Extract(dbUser, m)
}

func (s *UsersGRPCServer) UpdateUser(ctx context.Context, in *APIUpdateUserRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated")
	}

	if userID != in.UserId {
		return nil, status.Error(codes.Internal, "Forbidden")
	}

	fv, err := s.userRepository.UpdateUser(ctx, userID, in.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) DeleteUser(ctx context.Context, in *APIDeleteUserRequest) (*emptypb.Empty, error) {
	userID, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	if !s.enforcer.Enforce(role, "user", "delete") {
		if userID != in.UserId {
			return nil, status.Errorf(codes.Internal, "Forbidden")
		}

		match, err := s.userRepository.PasswordMatch(in.UserId, in.Password)
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

	success, err := s.userRepository.DeleteUser(in.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.contactsRepository.deleteUserEverywhere(in.UserId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.events.Add(Event{
			UserID:  userID,
			Message: fmt.Sprintf("Удаление пользователя №%d", in.UserId),
			Users:   []int64{in.UserId},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) EmailChange(ctx context.Context, in *APIEmailChangeRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	fv, err := s.userRepository.EmailChangeStart(userID, in.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) EmailChangeConfirm(ctx context.Context, in *APIEmailChangeConfirmRequest) (*emptypb.Empty, error) {
	err := s.userRepository.EmailChangeFinish(ctx, in.Code)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) SetPassword(ctx context.Context, in *APISetPasswordRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	fv, err := s.userRepository.ValidateChangePassword(userID, in.OldPassword, in.NewPassword, in.NewPasswordConfirm)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	err = s.userRepository.SetPassword(ctx, userID, in.NewPassword)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) PasswordRecovery(ctx context.Context, in *APIPasswordRecoveryRequest) (*emptypb.Empty, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed extract peer from context")
	}
	remoteAddr := p.Addr.String()

	fv, err := s.passwordRecovery.Start(in.Email, in.Captcha, remoteAddr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) PasswordRecoveryCheckCode(_ context.Context, in *APIPasswordRecoveryCheckCodeRequest) (*emptypb.Empty, error) {

	if len(in.Code) <= 0 {
		return nil, status.Errorf(codes.Internal, "Invalid code")
	}

	userId, err := s.passwordRecovery.GetUserID(in.Code)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userId == 0 {
		return nil, status.Errorf(codes.NotFound, "Token not found")
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) PasswordRecoveryConfirm(ctx context.Context, in *APIPasswordRecoveryConfirmRequest) (*APIPasswordRecoveryConfirmResponse, error) {

	fv, userId, err := s.passwordRecovery.Finish(in.Code, in.Password, in.PasswordConfirm)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	err = s.userRepository.SetPassword(ctx, userId, in.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	login, err := s.userRepository.GetLogin(userId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIPasswordRecoveryConfirmResponse{
		Login: login,
	}, nil
}
