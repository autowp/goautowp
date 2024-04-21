package goautowp

import (
	"context"
	"fmt"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/pictures"
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
	commentsRepository *comments.Repository
	picturesRepository *pictures.Repository
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
	commentsRepository *comments.Repository,
	picturesRepository *pictures.Repository,
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
		commentsRepository: commentsRepository,
		picturesRepository: picturesRepository,
		events:             events,
		languages:          languages,
		captcha:            captcha,
		userExtractor:      userExtractor,
	}
}

func (s *UsersGRPCServer) Me(ctx context.Context, _ *APIMeRequest) (*APIUser, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.GetUser(ctx, &APIGetUserRequest{
		UserId: userID,
	})
}

func (s *UsersGRPCServer) GetUser(ctx context.Context, in *APIGetUserRequest) (*APIUser, error) {
	dbUser, err := s.userRepository.User(ctx, users.GetUsersOptions{
		ID: in.UserId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if dbUser == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	apiUser, err := s.userExtractor.Extract(ctx, dbUser)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return apiUser, err
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

		match, err := s.userRepository.PasswordMatch(ctx, in.UserId, in.Password)
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

	success, err := s.userRepository.DeleteUser(ctx, in.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		err = s.contactsRepository.deleteUserEverywhere(ctx, in.UserId)
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

	if in.UserId == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.userRepository.SetDisableUserCommentsNotifications(ctx, userID, in.UserId, true)
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

	if in.UserId == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.userRepository.SetDisableUserCommentsNotifications(ctx, userID, in.UserId, false)
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

	if in.UserId == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	prefs, err := s.userRepository.UserPreferences(ctx, userID, in.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIUserPreferencesResponse{
		DisableCommentsNotifications: prefs.DisableCommentsNotifications,
	}, nil
}

func (s *UsersGRPCServer) GetUsers(ctx context.Context, in *APIUsersRequest) (*APIUsersResponse, error) {
	rows, pages, err := s.userRepository.Users(ctx, users.GetUsersOptions{
		IsOnline: true,
		Limit:    in.Limit,
		Page:     in.Page,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*APIUser, 0)

	for idx := range rows {
		apiUser, err := s.userExtractor.Extract(ctx, &rows[idx])
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		items = append(items, apiUser)
	}

	var paginator *Pages
	if pages != nil {
		paginator = &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
		}
	}

	return &APIUsersResponse{
		Items:     items,
		Paginator: paginator,
	}, nil
}

func (s *UsersGRPCServer) GetUsersRating(
	ctx context.Context, in *APIUsersRatingRequest,
) (*APIUsersRatingResponse, error) {
	const (
		usersRatingLimit          = 30
		detailedRatingForFirstNum = 10
	)

	switch in.Rating {
	case "likes":
		ratingUsers, err := s.commentsRepository.TopAuthors(ctx, usersRatingLimit)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result := make([]*APIUsersRatingUser, 0)
		for _, ratingUser := range ratingUsers {
			result = append(result, &APIUsersRatingUser{
				UserId: ratingUser.AuthorID,
				Volume: ratingUser.Volume,
				Brands: nil,
			})
		}

		return &APIUsersRatingResponse{
			Users: result,
		}, nil
	case "picture-likes":
		ratingUsers, err := s.picturesRepository.TopLikes(ctx, usersRatingLimit)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result := make([]*APIUsersRatingUser, 0)

		for idx, ratingUser := range ratingUsers {
			fansResult := make([]*APIUsersRatingUserFan, 0)

			if idx < detailedRatingForFirstNum {
				fans, err := s.picturesRepository.TopOwnerFans(ctx, ratingUser.OwnerID, 2)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				for _, fan := range fans {
					fansResult = append(fansResult, &APIUsersRatingUserFan{
						UserId: fan.UserID,
						Volume: fan.Volume,
					})
				}
			}

			result = append(result, &APIUsersRatingUser{
				UserId: ratingUser.OwnerID,
				Volume: ratingUser.Volume,
				Brands: nil,
				Fans:   fansResult,
			})
		}

		return &APIUsersRatingResponse{
			Users: result,
		}, nil
	}

	return nil, status.Error(codes.NotFound, "")
}
