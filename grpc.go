package goautowp

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func APIImageToGRPC(image *storage.Image) *APIImage {
	if image == nil {
		return nil
	}

	return &APIImage{
		Id:       int32(image.ID()),
		Src:      image.Src(),
		Width:    int32(image.Width()),
		Height:   int32(image.Height()),
		Filesize: int32(image.FileSize()),
	}
}

//func APIUserToGRPC(user *users.APIUser) *APIUser {
//	if user == nil {
//		return nil
//	}
//
//	var ts *timestamppb.Timestamp
//
//	if user.LastOnline != nil {
//		ts = timestamppb.New(*user.LastOnline)
//	}
//
//	return &APIUser{
//		Id:          user.ID,
//		Name:        user.Name,
//		Deleted:     user.Deleted,
//		LongAway:    user.LongAway,
//		Green:       user.Green,
//		Route:       user.Route,
//		Identity:    user.Identity,
//		Avatar:      APIImageToGRPC(user.Avatar),
//		Gravatar:    user.Gravatar,
//		LastOnline:  ts,
//		SpecsWeight: user.SpecsWeight,
//	}
//}

type GRPCServer struct {
	UnimplementedAutowpServer
	auth              *Auth
	catalogue         *Catalogue
	reCaptchaConfig   config.RecaptchaConfig
	fileStorageConfig config.FileStorageConfig
	enforcer          *casbin.Enforcer
	comments          *comments.Repository
	ipExtractor       *IPExtractor
	feedback          *Feedback
}

func NewGRPCServer(
	auth *Auth,
	catalogue *Catalogue,
	reCaptchaConfig config.RecaptchaConfig,
	fileStorageConfig config.FileStorageConfig,
	enforcer *casbin.Enforcer,
	comments *comments.Repository,
	ipExtractor *IPExtractor,
	feedback *Feedback,
) *GRPCServer {
	return &GRPCServer{
		auth:              auth,
		catalogue:         catalogue,
		reCaptchaConfig:   reCaptchaConfig,
		fileStorageConfig: fileStorageConfig,
		enforcer:          enforcer,
		comments:          comments,
		ipExtractor:       ipExtractor,
		feedback:          feedback,
	}
}

func (s *GRPCServer) GetSpecs(ctx context.Context, _ *emptypb.Empty) (*SpecsItems, error) {
	items, err := s.catalogue.getSpecs(ctx, 0)
	if err != nil {
		return nil, err
	}

	return &SpecsItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectives(ctx context.Context, _ *emptypb.Empty) (*PerspectivesItems, error) {
	items, err := s.catalogue.getPerspectives(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &PerspectivesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectivePages(ctx context.Context, _ *emptypb.Empty) (*PerspectivePagesItems, error) {
	items, err := s.catalogue.getPerspectivePages(ctx)
	if err != nil {
		return nil, err
	}

	return &PerspectivePagesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error) {
	return &ReCaptchaConfig{
		PublicKey: s.reCaptchaConfig.PublicKey,
	}, nil
}

func (s *GRPCServer) GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error) {
	if len(s.fileStorageConfig.S3.Endpoints) == 0 {
		return nil, errors.New("no endpoints provided")
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	endpoint := s.fileStorageConfig.S3.Endpoints[random.Intn(len(s.fileStorageConfig.S3.Endpoints))]

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.png"
	imageURL := parsedURL.String()

	parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.css"
	cssURL := parsedURL.String()

	return &BrandIcons{
		Image: imageURL,
		Css:   cssURL,
	}, nil
}

func (s *GRPCServer) AclEnforce( //nolint
	ctx context.Context,
	in *AclEnforceRequest,
) (*AclEnforceResult, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &AclEnforceResult{
		Result: s.enforcer.Enforce(role, in.Resource, in.Privilege),
	}, nil
}

func (s *GRPCServer) GetVehicleTypes(ctx context.Context, _ *emptypb.Empty) (*VehicleTypeItems, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	items, err := s.catalogue.getVehicleTypesTree(ctx, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &VehicleTypeItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetBrandVehicleTypes(
	ctx context.Context,
	in *GetBrandVehicleTypesRequest,
) (*BrandVehicleTypeItems, error) {
	items, err := s.catalogue.getBrandVehicleTypes(ctx, in.BrandId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &BrandVehicleTypeItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetIP(ctx context.Context, in *APIGetIPRequest) (*APIIP, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	m := make(map[string]bool)
	for _, e := range in.Fields {
		m[e] = true
	}

	result, err := s.ipExtractor.Extract(ctx, ip, m, role)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return result, nil
}

func (s *GRPCServer) CreateFeedback(ctx context.Context, in *APICreateFeedbackRequest) (*emptypb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed extract peer from context")
	}

	remoteAddr := p.Addr.String()

	fv, err := s.feedback.Create(CreateFeedbackRequest{
		Name:    in.Name,
		Email:   in.Email,
		Message: in.Message,
		Captcha: in.Captcha,
		IP:      remoteAddr,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func wrapFieldViolations(fv []*errdetails.BadRequest_FieldViolation) error {
	st := status.New(codes.InvalidArgument, "invalid request")
	br := &errdetails.BadRequest{
		FieldViolations: fv,
	}

	st, err := st.WithDetails(br)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return st.Err()
}
