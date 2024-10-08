package goautowp

import (
	"context"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/users"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	usersRatingLimit    = 30
	numOfItemsInDetails = 3
)

type RatingGRPCServer struct {
	UnimplementedRatingServer
	picturesRepository *pictures.Repository
	userRepository     *users.Repository
	itemsRepository    *items.Repository
	commentsRepository *comments.Repository
	attrsRepository    *attrs.Repository
}

func NewRatingGRPCServer(
	picturesRepository *pictures.Repository,
	userRepository *users.Repository,
	itemsRepository *items.Repository,
	commentsRepository *comments.Repository,
	attrsRepository *attrs.Repository,
) *RatingGRPCServer {
	return &RatingGRPCServer{
		picturesRepository: picturesRepository,
		userRepository:     userRepository,
		itemsRepository:    itemsRepository,
		commentsRepository: commentsRepository,
		attrsRepository:    attrsRepository,
	}
}

func (s *RatingGRPCServer) GetUserPicturesRating(
	ctx context.Context, _ *emptypb.Empty,
) (*APIUsersRatingResponse, error) {
	falseRef := false
	trueRef := true

	rows, _, err := s.userRepository.Users(ctx, users.GetUsersOptions{
		Deleted:     &falseRef,
		Limit:       usersRatingLimit,
		Order:       []exp.OrderedExpression{schema.UserTablePicturesTotalCol.Desc()},
		HasPictures: &trueRef,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUser, len(rows))
	for idx, row := range rows {
		result[idx] = &APIUsersRatingUser{
			UserId: row.ID,
			Volume: row.PicturesTotal,
		}
	}

	return &APIUsersRatingResponse{
		Users: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserPicturesRatingBrands(
	ctx context.Context, in *UserRatingDetailsRequest,
) (*UserRatingBrandsResponse, error) {
	brands, _, err := s.itemsRepository.List(ctx, items.ListOptions{
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		DescendantPictures: &items.ItemPicturesOptions{
			Pictures: &items.PicturesOptions{
				OwnerID: in.GetUserId(),
				Status:  pictures.StatusAccepted,
			},
		},
		OrderBy: []exp.OrderedExpression{goqu.COUNT(goqu.DISTINCT(goqu.T("i_ipcd_pi_p").Col("id"))).Desc()},
		Limit:   numOfItemsInDetails,
		Fields: items.ListFields{
			NameOnly:             true,
			CurrentPicturesCount: true,
		},
		Language: in.GetLanguage(),
	}, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUserBrand, len(brands))
	for idx, brand := range brands {
		result[idx] = &APIUsersRatingUserBrand{
			Name:   brand.NameOnly,
			Route:  []string{"/", brand.Catname},
			Volume: int64(brand.CurrentPicturesCount),
		}
	}

	return &UserRatingBrandsResponse{
		Brands: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserCommentsRating(
	ctx context.Context, _ *emptypb.Empty,
) (*APIUsersRatingResponse, error) {
	ratingUsers, err := s.commentsRepository.TopAuthors(ctx, usersRatingLimit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUser, len(ratingUsers))
	for idx, ratingUser := range ratingUsers {
		result[idx] = &APIUsersRatingUser{
			UserId: ratingUser.AuthorID,
			Volume: ratingUser.Volume,
		}
	}

	return &APIUsersRatingResponse{
		Users: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserCommentsRatingFans(
	ctx context.Context, in *UserRatingDetailsRequest,
) (*GetUserRatingFansResponse, error) {
	fans, err := s.commentsRepository.AuthorsFans(ctx, in.GetUserId(), numOfItemsInDetails)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUserFan, len(fans))
	for idx, fan := range fans {
		result[idx] = &APIUsersRatingUserFan{
			UserId: fan.UserID,
			Volume: fan.Volume,
		}
	}

	return &GetUserRatingFansResponse{
		Fans: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserPictureLikesRating(
	ctx context.Context, _ *emptypb.Empty,
) (*APIUsersRatingResponse, error) {
	ratingUsers, err := s.picturesRepository.TopLikes(ctx, usersRatingLimit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUser, len(ratingUsers))
	for idx, ratingUser := range ratingUsers {
		result[idx] = &APIUsersRatingUser{
			UserId: ratingUser.OwnerID,
			Volume: ratingUser.Volume,
		}
	}

	return &APIUsersRatingResponse{
		Users: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserPictureLikesRatingFans(
	ctx context.Context, in *UserRatingDetailsRequest,
) (*GetUserRatingFansResponse, error) {
	fans, err := s.picturesRepository.TopOwnerFans(ctx, in.GetUserId(), numOfItemsInDetails)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUserFan, len(fans))
	for idx, fan := range fans {
		result[idx] = &APIUsersRatingUserFan{
			UserId: fan.UserID,
			Volume: fan.Volume,
		}
	}

	return &GetUserRatingFansResponse{
		Fans: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserSpecsRating(
	ctx context.Context, _ *emptypb.Empty,
) (*APIUsersRatingResponse, error) {
	falseRef := false
	trueRef := true

	ratingUsers, _, err := s.userRepository.Users(ctx, users.GetUsersOptions{
		Deleted:  &falseRef,
		HasSpecs: &trueRef,
		Limit:    usersRatingLimit,
		Order:    []exp.OrderedExpression{schema.UserTableSpecsVolumeCol.Desc()},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUser, len(ratingUsers))
	for idx, u := range ratingUsers {
		result[idx] = &APIUsersRatingUser{
			UserId: u.ID,
			Volume: u.SpecsVolume,
			Weight: u.SpecsWeight,
		}
	}

	return &APIUsersRatingResponse{
		Users: result,
	}, nil
}

func (s *RatingGRPCServer) GetUserSpecsRatingBrands(
	ctx context.Context, in *UserRatingDetailsRequest,
) (*UserRatingBrandsResponse, error) {
	brands, err := s.attrsRepository.TopUserBrands(ctx, in.GetUserId(), numOfItemsInDetails)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIUsersRatingUserBrand, len(brands))
	for idx, brand := range brands {
		result[idx] = &APIUsersRatingUserBrand{
			Name:   brand.Name,
			Route:  []string{"/", brand.Catname},
			Volume: brand.Volume,
		}
	}

	return &UserRatingBrandsResponse{
		Brands: result,
	}, nil
}
