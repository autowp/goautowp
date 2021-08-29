package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UsersGRPCServer struct {
	UnimplementedUsersServer
	oauthConfig        OAuthConfig
	db                 *sql.DB
	enforcer           *casbin.Enforcer
	contactsRepository *ContactsRepository
	userRepository     *UserRepository
	events             *Events
	languages          map[string]LanguageConfig
	captcha            bool
}

func NewUsersGRPCServer(
	oauthConfig OAuthConfig,
	db *sql.DB,
	enforcer *casbin.Enforcer,
	contactsRepository *ContactsRepository,
	userRepository *UserRepository,
	events *Events,
	languages map[string]LanguageConfig,
	captcha bool,
) *UsersGRPCServer {
	return &UsersGRPCServer{
		oauthConfig:        oauthConfig,
		db:                 db,
		enforcer:           enforcer,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		events:             events,
		languages:          languages,
		captcha:            captcha,
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

	user := CreateUserOptions{
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
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *UsersGRPCServer) DeleteUser(ctx context.Context, in *APIDeleteUserRequest) (*emptypb.Empty, error) {
	userID, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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
			return nil, status.Errorf(codes.Internal, err.Error())
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
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if success {
		err = s.contactsRepository.deleteUserEverywhere(in.UserId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		err = s.events.Add(Event{
			UserID:  userID,
			Message: fmt.Sprintf("Удаление пользователя №%d", in.UserId),
			Users:   []int64{in.UserId},
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}
