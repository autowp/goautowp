package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math/rand"
	"net"
	"net/url"
	"time"
)

type GRPCServer struct {
	UnimplementedAutowpServer
	container          *Container
	catalogue          *Catalogue
	reCaptchaConfig    RecaptchaConfig
	fileStorageConfig  FileStorageConfig
	db                 *sql.DB
	enforcer           *casbin.Enforcer
	oauthConfig        OAuthConfig
	contactsRepository *ContactsRepository
	userRepository     *UserRepository
	userExtractor      *UserExtractor
	comments           *Comments
	traffic            *Traffic
	ipExtractor        *IPExtractor
	feedback           *Feedback
	forums             *Forums
	messages           *Messages
}

func NewGRPCServer(
	container *Container,
	catalogue *Catalogue,
	reCaptchaConfig RecaptchaConfig,
	fileStorageConfig FileStorageConfig,
	db *sql.DB,
	enforcer *casbin.Enforcer,
	oauthConfig OAuthConfig,
	contactsRepository *ContactsRepository,
	userRepository *UserRepository,
	userExtractor *UserExtractor,
	comments *Comments,
	traffic *Traffic,
	ipExtractor *IPExtractor,
	feedback *Feedback,
	forums *Forums,
	messages *Messages,
) (*GRPCServer, error) {
	return &GRPCServer{
		container:          container,
		catalogue:          catalogue,
		reCaptchaConfig:    reCaptchaConfig,
		fileStorageConfig:  fileStorageConfig,
		db:                 db,
		enforcer:           enforcer,
		oauthConfig:        oauthConfig,
		contactsRepository: contactsRepository,
		userRepository:     userRepository,
		userExtractor:      userExtractor,
		comments:           comments,
		traffic:            traffic,
		ipExtractor:        ipExtractor,
		feedback:           feedback,
		forums:             forums,
		messages:           messages,
	}, nil
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
	if len(s.fileStorageConfig.S3.Endpoints) <= 0 {
		return nil, errors.New("no endpoints provided")
	}

	endpoint := s.fileStorageConfig.S3.Endpoints[rand.Intn(len(s.fileStorageConfig.S3.Endpoints))]

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

func (s *GRPCServer) AclEnforce(ctx context.Context, in *AclEnforceRequest) (*AclEnforceResult, error) {

	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &AclEnforceResult{
		Result: s.enforcer.Enforce(role, in.Resource, in.Privilege),
	}, nil
}

func (s *GRPCServer) GetVehicleTypes(ctx context.Context, _ *emptypb.Empty) (*VehicleTypeItems, error) {
	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	items, err := s.catalogue.getVehicleTypesTree(0)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &VehicleTypeItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) GetBrandVehicleTypes(_ context.Context, in *GetBrandVehicleTypesRequest) (*BrandVehicleTypeItems, error) {
	items, err := s.catalogue.getBrandVehicleTypes(in.BrandId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &BrandVehicleTypeItems{
		Items: items,
	}, nil
}

func (s *GRPCServer) CreateContact(ctx context.Context, in *CreateContactRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if int64(in.UserId) == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	deleted := false
	user, err := s.userRepository.GetUser(GetUsersOptions{ID: int(in.UserId), Deleted: &deleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if user == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	err = s.contactsRepository.create(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) DeleteContact(ctx context.Context, in *DeleteContactRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.contactsRepository.delete(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) GetContact(ctx context.Context, in *GetContactRequest) (*Contact, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	if int64(in.UserId) == userID {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	exists, err := s.contactsRepository.isExists(userID, int64(in.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if !exists {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	return &Contact{
		ContactUserId: in.UserId,
	}, nil
}

func (s *GRPCServer) GetContacts(ctx context.Context, in *GetContactsRequest) (*ContactItems, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	fields := in.Fields
	m := make(map[string]bool)
	for _, e := range fields {
		m[e] = true
	}

	userRows, err := s.userRepository.GetUsers(GetUsersOptions{
		InContacts: userID,
		Order:      []string{"users.deleted", "users.name"},
		Fields:     m,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	items := make([]*Contact, len(userRows))
	for idx, userRow := range userRows {
		user, err := s.userExtractor.Extract(&userRow, m)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
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

func (s *GRPCServer) GetCommentVotes(_ context.Context, in *GetCommentVotesRequest) (*CommentVoteItems, error) {
	votes, err := s.comments.getVotes(int(in.CommentId))

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if votes == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound")
	}

	result := make([]*CommentVote, 0)

	for _, user := range votes.PositiveVotes {
		extracted, err := s.userExtractor.Extract(&user, map[string]bool{})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		result = append(result, &CommentVote{
			Value: CommentVote_POSITIVE,
			User:  extracted,
		})
	}

	for _, user := range votes.NegativeVotes {
		extracted, err := s.userExtractor.Extract(&user, map[string]bool{})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		result = append(result, &CommentVote{
			Value: CommentVote_NEGATIVE,
			User:  extracted,
		})
	}

	return &CommentVoteItems{
		Items: result,
	}, nil
}

func (s *GRPCServer) GetTrafficTop(_ context.Context, _ *emptypb.Empty) (*APITrafficTopResponse, error) {

	items, err := s.traffic.Monitoring.ListOfTop(50)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	result := make([]*APITrafficTopItem, len(items))
	for idx, item := range items {

		ban, err := s.traffic.Ban.Get(item.IP)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		inWhitelist, err := s.traffic.Whitelist.Exists(item.IP)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		var user *DBUser
		var topItemBan *APIBanItem

		if ban != nil {
			user, err = s.getUser(ban.ByUserID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}

			extractedUser, err := s.userExtractor.Extract(user, map[string]bool{})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}

			topItemBan = &APIBanItem{
				Until:    timestamppb.New(ban.Until),
				ByUserId: int32(ban.ByUserID),
				ByUser:   extractedUser,
				Reason:   ban.Reason,
			}
		}

		result[idx] = &APITrafficTopItem{
			Ip:          item.IP.String(),
			Count:       int32(item.Count),
			Ban:         topItemBan,
			InWhitelist: inWhitelist,
			WhoisUrl:    fmt.Sprintf("https://nic.ru/whois/?query=%s", url.QueryEscape(item.IP.String())),
		}
	}

	return &APITrafficTopResponse{
		Items: result,
	}, nil
}

func (s *GRPCServer) getUser(id int) (*DBUser, error) {
	rows, err := s.db.Query(`
		SELECT id, name, deleted, identity, last_online, role
		FROM users
		WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	if !rows.Next() {
		return nil, nil
	}

	var r DBUser
	err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (s *GRPCServer) GetIP(ctx context.Context, in *APIGetIPRequest) (*APIIP, error) {

	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
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
		return nil, status.Errorf(codes.Internal, err.Error())
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
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) DeleteFromTrafficBlacklist(ctx context.Context, in *DeleteFromTrafficBlacklistRequest) (*emptypb.Empty, error) {

	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "user", "ban"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.traffic.Ban.Remove(ip)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) DeleteFromTrafficWhitelist(ctx context.Context, in *DeleteFromTrafficWhitelistRequest) (*emptypb.Empty, error) {
	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.traffic.Whitelist.Remove(ip)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) AddToTrafficBlacklist(ctx context.Context, in *AddToTrafficBlacklistRequest) (*emptypb.Empty, error) {
	userID, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "user", "ban"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	duration := time.Hour * time.Duration(in.Period)

	err = s.traffic.Ban.Add(ip, duration, userID, in.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) AddToTrafficWhitelist(ctx context.Context, in *AddToTrafficWhitelistRequest) (*emptypb.Empty, error) {
	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.traffic.Whitelist.Add(ip, "manual click")
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	err = s.traffic.Ban.Remove(ip)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) GetTrafficWhitelist(ctx context.Context, _ *emptypb.Empty) (*APITrafficWhitelistItems, error) {
	_, role, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	list, err := s.traffic.Whitelist.List()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &APITrafficWhitelistItems{
		Items: list,
	}, nil
}

func (s *GRPCServer) GetForumsUserSummary(ctx context.Context, _ *emptypb.Empty) (*APIForumsUserSummary, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	subscriptionsCount, err := s.forums.GetUserSummary(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &APIForumsUserSummary{
		SubscriptionsCount: int32(subscriptionsCount),
	}, nil
}

func (s *GRPCServer) GetMessagesNewCount(ctx context.Context, _ *emptypb.Empty) (*APIMessageNewCount, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	count, err := s.messages.GetUserNewMessagesCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &APIMessageNewCount{
		Count: int32(count),
	}, nil
}

func (s *GRPCServer) GetMessagesSummary(ctx context.Context, _ *emptypb.Empty) (*APIMessageSummary, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	inbox, err := s.messages.GetInboxCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	inboxNew, err := s.messages.GetInboxNewCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	sent, err := s.messages.GetSentCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	system, err := s.messages.GetSystemCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	systemNew, err := s.messages.GetSystemNewCount(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &APIMessageSummary{
		InboxCount:     int32(inbox),
		InboxNewCount:  int32(inboxNew),
		SentCount:      int32(sent),
		SystemCount:    int32(system),
		SystemNewCount: int32(systemNew),
	}, nil

}

func (s *GRPCServer) CreateUser(ctx context.Context, in *APICreateUserRequest) (*emptypb.Empty, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed extract peer from context")
	}
	remoteAddr := p.Addr.String()

	config, err := s.container.GetConfig()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	language, ok := config.Languages[in.Language]
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

	fv, err := s.userRepository.ValidateCreateUser(user, config.Captcha, remoteAddr)
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

func (s *GRPCServer) PasswordRecovery(ctx context.Context, in *APIPasswordRecoveryRequest) (*emptypb.Empty, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed extract peer from context")
	}
	remoteAddr := p.Addr.String()

	pr, err := s.container.GetPasswordRecovery()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	fv, err := pr.Start(in.Email, in.Captcha, remoteAddr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) PasswordRecoveryCheckCode(_ context.Context, in *APIPasswordRecoveryCheckCodeRequest) (*emptypb.Empty, error) {

	if len(in.Code) <= 0 {
		return nil, status.Errorf(codes.Internal, "Invalid code")
	}

	pr, err := s.container.GetPasswordRecovery()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	userId, err := pr.GetUserID(in.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userId == 0 {
		return nil, status.Errorf(codes.NotFound, "Token not found")
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) PasswordRecoveryConfirm(_ context.Context, in *APIPasswordRecoveryConfirmRequest) (*APIPasswordRecoveryConfirmResponse, error) {

	pr, err := s.container.GetPasswordRecovery()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	fv, userId, err := pr.Finish(in.Code, in.Password, in.PasswordConfirm)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	users, err := s.container.GetUserRepository()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	err = users.SetPassword(userId, in.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	login, err := users.GetLogin(userId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &APIPasswordRecoveryConfirmResponse{
		Login: login,
	}, nil
}

func wrapFieldViolations(fv []*errdetails.BadRequest_FieldViolation) error {
	st := status.New(codes.InvalidArgument, "invalid request")
	br := &errdetails.BadRequest{
		FieldViolations: fv,
	}
	st, err := st.WithDetails(br)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return st.Err()
}

func (s *GRPCServer) EmailChange(ctx context.Context, in *APIEmailChangeRequest) (*emptypb.Empty, error) {
	userID, _, err := validateGRPCAuthorization(ctx, s.db, s.oauthConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}

	users, err := s.container.GetUserRepository()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	fv, err := users.EmailChangeStart(userID, in.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if len(fv) > 0 {
		return nil, wrapFieldViolations(fv)
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) EmailChangeConfirm(ctx context.Context, in *APIEmailChangeConfirmRequest) (*emptypb.Empty, error) {
	users, err := s.container.GetUserRepository()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	err = users.EmailChangeFinish(ctx, in.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
