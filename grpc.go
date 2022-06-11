package goautowp

import (
	"context"
	"errors"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	"net"
	"net/url"
)

type GRPCServer struct {
	UnimplementedAutowpServer
	auth              *Auth
	catalogue         *Catalogue
	reCaptchaConfig   config.RecaptchaConfig
	fileStorageConfig config.FileStorageConfig
	enforcer          *casbin.Enforcer
	userExtractor     *UserExtractor
	comments          *comments.Repository
	ipExtractor       *IPExtractor
	feedback          *Feedback
	forums            *Forums
}

func NewGRPCServer(
	auth *Auth,
	catalogue *Catalogue,
	reCaptchaConfig config.RecaptchaConfig,
	fileStorageConfig config.FileStorageConfig,
	enforcer *casbin.Enforcer,
	userExtractor *UserExtractor,
	comments *comments.Repository,
	ipExtractor *IPExtractor,
	feedback *Feedback,
	forums *Forums,
) *GRPCServer {
	return &GRPCServer{
		auth:              auth,
		catalogue:         catalogue,
		reCaptchaConfig:   reCaptchaConfig,
		fileStorageConfig: fileStorageConfig,
		enforcer:          enforcer,
		userExtractor:     userExtractor,
		comments:          comments,
		ipExtractor:       ipExtractor,
		feedback:          feedback,
		forums:            forums,
	}
}

func (s *GRPCServer) GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error) {
	items, err := s.catalogue.getSpecs(0)
	if err != nil {
		return nil, err
	}

	return &SpecsItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error) {
	items, err := s.catalogue.getPerspectives(nil)
	if err != nil {
		return nil, err
	}

	return &PerspectivesItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error) {
	items, err := s.catalogue.getPerspectivePages()
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

	endpoint := s.fileStorageConfig.S3.Endpoints[rand.Intn(len(s.fileStorageConfig.S3.Endpoints))] // nolint: gosec

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

	items, err := s.catalogue.getVehicleTypesTree(0)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &VehicleTypeItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetBrandVehicleTypes(
	_ context.Context,
	in *GetBrandVehicleTypesRequest,
) (*BrandVehicleTypeItems, error) {
	items, err := s.catalogue.getBrandVehicleTypes(in.BrandId)

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

	result, err := s.ipExtractor.Extract(ip, m, role)
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

func (s *GRPCServer) GetForumsUserSummary(ctx context.Context, _ *emptypb.Empty) (*APIForumsUserSummary, error) {
	userID, _, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	subscriptionsCount, err := s.forums.GetUserSummary(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIForumsUserSummary{
		SubscriptionsCount: int32(subscriptionsCount),
	}, nil
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
