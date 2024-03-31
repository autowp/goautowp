package goautowp

import (
	"context"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/users"
	"github.com/doug-martin/goqu/v9/exp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ContactsGRPCServer struct {
	UnimplementedContactsServer
	auth               *Auth
	contactsRepository *ContactsRepository
	userRepository     *users.Repository
	userExtractor      *UserExtractor
}

func NewContactsGRPCServer(
	auth *Auth,
	contactsRepository *ContactsRepository,
	userRepository *users.Repository,
	userExtractor *UserExtractor,
) *ContactsGRPCServer {
	return &ContactsGRPCServer{
		auth:               auth,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		userExtractor:      userExtractor,
	}
}

func (s *ContactsGRPCServer) CreateContact(ctx context.Context, in *CreateContactRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if in.UserId == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	deleted := false

	user, err := s.userRepository.User(ctx, users.GetUsersOptions{ID: in.UserId, Deleted: &deleted})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "NotFound")
	}

	err = s.contactsRepository.create(ctx, userID, in.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactsGRPCServer) DeleteContact(ctx context.Context, in *DeleteContactRequest) (*emptypb.Empty, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.contactsRepository.delete(ctx, userID, in.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ContactsGRPCServer) GetContact(ctx context.Context, in *GetContactRequest) (*Contact, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if in.UserId == userID {
		return nil, status.Error(codes.InvalidArgument, "InvalidArgument")
	}

	exists, err := s.contactsRepository.isExists(ctx, userID, in.UserId)
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

func (s *ContactsGRPCServer) GetContacts(ctx context.Context, _ *GetContactsRequest) (*ContactItems, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	userRows, _, err := s.userRepository.Users(ctx, users.GetUsersOptions{
		InContacts: userID,
		Order: []exp.OrderedExpression{
			schema.UserTable.Col("deleted").Asc(),
			schema.UserTable.Col("name").Asc(),
		},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*Contact, len(userRows))

	for idx := range userRows {
		user, err := s.userExtractor.Extract(ctx, &userRows[idx])
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
