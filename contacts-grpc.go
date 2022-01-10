package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ContactsGRPCServer struct {
	UnimplementedContactsServer
	db                 *sql.DB
	contactsRepository *ContactsRepository
	userRepository     *users.Repository
	userExtractor      *UserExtractor
	keycloak           gocloak.GoCloak
	keycloakCfg        config.KeycloakConfig
}

func NewContactsGRPCServer(
	db *sql.DB,
	contactsRepository *ContactsRepository,
	userRepository *users.Repository,
	userExtractor *UserExtractor,
	keycloak gocloak.GoCloak,
	keycloakCfg config.KeycloakConfig,
) *ContactsGRPCServer {
	return &ContactsGRPCServer{
		db:                 db,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		userExtractor:      userExtractor,
		keycloak:           keycloak,
		keycloakCfg:        keycloakCfg,
	}
}

func (s *ContactsGRPCServer) CreateContact(ctx context.Context, in *CreateContactRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.keycloak, s.keycloakCfg)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if int64(in.UserId) == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	deleted := false
	user, err := s.userRepository.User(users.GetUsersOptions{ID: in.UserId, Deleted: &deleted})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "NotFound")
	}

	err = s.contactsRepository.create(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactsGRPCServer) DeleteContact(ctx context.Context, in *DeleteContactRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.keycloak, s.keycloakCfg)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.contactsRepository.delete(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactsGRPCServer) GetContact(ctx context.Context, in *GetContactRequest) (*Contact, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.keycloak, s.keycloakCfg)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if int64(in.UserId) == userID {
		return nil, status.Error(codes.InvalidArgument, "InvalidArgument")
	}

	exists, err := s.contactsRepository.isExists(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !exists {
		return nil, status.Error(codes.NotFound, "NotFound")
	}

	return &Contact{
		ContactUserId: in.UserId,
	}, nil
}

func (s *ContactsGRPCServer) GetContacts(ctx context.Context, in *GetContactsRequest) (*ContactItems, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.keycloak, s.keycloakCfg)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	fields := in.Fields
	m := make(map[string]bool)
	for _, e := range fields {
		m[e] = true
	}

	userRows, err := s.userRepository.Users(users.GetUsersOptions{
		InContacts: userID,
		Order:      []string{"users.deleted", "users.name"},
		Fields:     m,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*Contact, len(userRows))
	for idx, userRow := range userRows {
		user, err := s.userExtractor.Extract(&userRow, m)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		items[idx] = &Contact{
			ContactUserId: user.Id,
			User:          user,
		}
	}

	return &ContactItems{
		Items: items,
	}, nil
}
