package goautowp

import (
	"context"
	"fmt"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UsersGRPCServer struct {
	UnimplementedUsersServer
	auth               *Auth
	enforcer           *casbin.Enforcer
	contactsRepository *ContactsRepository
	userRepository     *users.Repository
	events             *Events
	languages          map[string]config.LanguageConfig
	captcha            bool
	userExtractor      *UserExtractor
}

func NewUsersGRPCServer(
	auth *Auth,
	enforcer *casbin.Enforcer,
	contactsRepository *ContactsRepository,
	userRepository *users.Repository,
	events *Events,
	languages map[string]config.LanguageConfig,
	captcha bool,
	userExtractor *UserExtractor,
) *UsersGRPCServer {
	return &UsersGRPCServer{
		auth:               auth,
		enforcer:           enforcer,
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
		Fields: in.Fields,
	})
}

func (s *UsersGRPCServer) GetUser(_ context.Context, in *APIGetUserRequest) (*APIUser, error) {
	fields := in.Fields
	m := make(map[string]bool)

	for _, e := range fields {
		m[e] = true
	}

	dbUser, err := s.userRepository.User(users.GetUsersOptions{
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

func (s *UsersGRPCServer) DeleteUser(ctx context.Context, in *APIDeleteUserRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
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
