// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package goautowp

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AutowpClient is the client API for Autowp service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AutowpClient interface {
	AclEnforce(ctx context.Context, in *AclEnforceRequest, opts ...grpc.CallOption) (*AclEnforceResult, error)
	AddToTrafficBlacklist(ctx context.Context, in *AddToTrafficBlacklistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	AddToTrafficWhitelist(ctx context.Context, in *AddToTrafficWhitelistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	CreateContact(ctx context.Context, in *CreateContactRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	CreateFeedback(ctx context.Context, in *APICreateFeedbackRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteContact(ctx context.Context, in *DeleteContactRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteFromTrafficBlacklist(ctx context.Context, in *DeleteFromTrafficBlacklistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteFromTrafficWhitelist(ctx context.Context, in *DeleteFromTrafficWhitelistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetBrandIcons(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BrandIcons, error)
	GetBrandVehicleTypes(ctx context.Context, in *GetBrandVehicleTypesRequest, opts ...grpc.CallOption) (*BrandVehicleTypeItems, error)
	GetCommentVotes(ctx context.Context, in *GetCommentVotesRequest, opts ...grpc.CallOption) (*CommentVoteItems, error)
	GetContact(ctx context.Context, in *GetContactRequest, opts ...grpc.CallOption) (*Contact, error)
	GetContacts(ctx context.Context, in *GetContactsRequest, opts ...grpc.CallOption) (*ContactItems, error)
	GetIP(ctx context.Context, in *APIGetIPRequest, opts ...grpc.CallOption) (*APIIP, error)
	GetPerspectives(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivesItems, error)
	GetPerspectivePages(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivePagesItems, error)
	GetReCaptchaConfig(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ReCaptchaConfig, error)
	GetSpecs(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SpecsItems, error)
	GetTrafficTop(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*APITrafficTopResponse, error)
	GetTrafficWhitelist(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*APITrafficWhitelistItems, error)
	GetVehicleTypes(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VehicleTypeItems, error)
}

type autowpClient struct {
	cc grpc.ClientConnInterface
}

func NewAutowpClient(cc grpc.ClientConnInterface) AutowpClient {
	return &autowpClient{cc}
}

func (c *autowpClient) AclEnforce(ctx context.Context, in *AclEnforceRequest, opts ...grpc.CallOption) (*AclEnforceResult, error) {
	out := new(AclEnforceResult)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/AclEnforce", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) AddToTrafficBlacklist(ctx context.Context, in *AddToTrafficBlacklistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/AddToTrafficBlacklist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) AddToTrafficWhitelist(ctx context.Context, in *AddToTrafficWhitelistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/AddToTrafficWhitelist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) CreateContact(ctx context.Context, in *CreateContactRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/CreateContact", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) CreateFeedback(ctx context.Context, in *APICreateFeedbackRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/CreateFeedback", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) DeleteContact(ctx context.Context, in *DeleteContactRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/DeleteContact", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) DeleteFromTrafficBlacklist(ctx context.Context, in *DeleteFromTrafficBlacklistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/DeleteFromTrafficBlacklist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) DeleteFromTrafficWhitelist(ctx context.Context, in *DeleteFromTrafficWhitelistRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/DeleteFromTrafficWhitelist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetBrandIcons(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BrandIcons, error) {
	out := new(BrandIcons)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetBrandIcons", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetBrandVehicleTypes(ctx context.Context, in *GetBrandVehicleTypesRequest, opts ...grpc.CallOption) (*BrandVehicleTypeItems, error) {
	out := new(BrandVehicleTypeItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetBrandVehicleTypes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetCommentVotes(ctx context.Context, in *GetCommentVotesRequest, opts ...grpc.CallOption) (*CommentVoteItems, error) {
	out := new(CommentVoteItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetCommentVotes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetContact(ctx context.Context, in *GetContactRequest, opts ...grpc.CallOption) (*Contact, error) {
	out := new(Contact)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetContact", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetContacts(ctx context.Context, in *GetContactsRequest, opts ...grpc.CallOption) (*ContactItems, error) {
	out := new(ContactItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetContacts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetIP(ctx context.Context, in *APIGetIPRequest, opts ...grpc.CallOption) (*APIIP, error) {
	out := new(APIIP)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetIP", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetPerspectives(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivesItems, error) {
	out := new(PerspectivesItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetPerspectives", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetPerspectivePages(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivePagesItems, error) {
	out := new(PerspectivePagesItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetPerspectivePages", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetReCaptchaConfig(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ReCaptchaConfig, error) {
	out := new(ReCaptchaConfig)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetReCaptchaConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetSpecs(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SpecsItems, error) {
	out := new(SpecsItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetSpecs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetTrafficTop(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*APITrafficTopResponse, error) {
	out := new(APITrafficTopResponse)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetTrafficTop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetTrafficWhitelist(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*APITrafficWhitelistItems, error) {
	out := new(APITrafficWhitelistItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetTrafficWhitelist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *autowpClient) GetVehicleTypes(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VehicleTypeItems, error) {
	out := new(VehicleTypeItems)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetVehicleTypes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AutowpServer is the server API for Autowp service.
// All implementations must embed UnimplementedAutowpServer
// for forward compatibility
type AutowpServer interface {
	AclEnforce(context.Context, *AclEnforceRequest) (*AclEnforceResult, error)
	AddToTrafficBlacklist(context.Context, *AddToTrafficBlacklistRequest) (*emptypb.Empty, error)
	AddToTrafficWhitelist(context.Context, *AddToTrafficWhitelistRequest) (*emptypb.Empty, error)
	CreateContact(context.Context, *CreateContactRequest) (*emptypb.Empty, error)
	CreateFeedback(context.Context, *APICreateFeedbackRequest) (*emptypb.Empty, error)
	DeleteContact(context.Context, *DeleteContactRequest) (*emptypb.Empty, error)
	DeleteFromTrafficBlacklist(context.Context, *DeleteFromTrafficBlacklistRequest) (*emptypb.Empty, error)
	DeleteFromTrafficWhitelist(context.Context, *DeleteFromTrafficWhitelistRequest) (*emptypb.Empty, error)
	GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error)
	GetBrandVehicleTypes(context.Context, *GetBrandVehicleTypesRequest) (*BrandVehicleTypeItems, error)
	GetCommentVotes(context.Context, *GetCommentVotesRequest) (*CommentVoteItems, error)
	GetContact(context.Context, *GetContactRequest) (*Contact, error)
	GetContacts(context.Context, *GetContactsRequest) (*ContactItems, error)
	GetIP(context.Context, *APIGetIPRequest) (*APIIP, error)
	GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error)
	GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error)
	GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error)
	GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error)
	GetTrafficTop(context.Context, *emptypb.Empty) (*APITrafficTopResponse, error)
	GetTrafficWhitelist(context.Context, *emptypb.Empty) (*APITrafficWhitelistItems, error)
	GetVehicleTypes(context.Context, *emptypb.Empty) (*VehicleTypeItems, error)
	mustEmbedUnimplementedAutowpServer()
}

// UnimplementedAutowpServer must be embedded to have forward compatible implementations.
type UnimplementedAutowpServer struct {
}

func (UnimplementedAutowpServer) AclEnforce(context.Context, *AclEnforceRequest) (*AclEnforceResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AclEnforce not implemented")
}
func (UnimplementedAutowpServer) AddToTrafficBlacklist(context.Context, *AddToTrafficBlacklistRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddToTrafficBlacklist not implemented")
}
func (UnimplementedAutowpServer) AddToTrafficWhitelist(context.Context, *AddToTrafficWhitelistRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddToTrafficWhitelist not implemented")
}
func (UnimplementedAutowpServer) CreateContact(context.Context, *CreateContactRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateContact not implemented")
}
func (UnimplementedAutowpServer) CreateFeedback(context.Context, *APICreateFeedbackRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateFeedback not implemented")
}
func (UnimplementedAutowpServer) DeleteContact(context.Context, *DeleteContactRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteContact not implemented")
}
func (UnimplementedAutowpServer) DeleteFromTrafficBlacklist(context.Context, *DeleteFromTrafficBlacklistRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteFromTrafficBlacklist not implemented")
}
func (UnimplementedAutowpServer) DeleteFromTrafficWhitelist(context.Context, *DeleteFromTrafficWhitelistRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteFromTrafficWhitelist not implemented")
}
func (UnimplementedAutowpServer) GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBrandIcons not implemented")
}
func (UnimplementedAutowpServer) GetBrandVehicleTypes(context.Context, *GetBrandVehicleTypesRequest) (*BrandVehicleTypeItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBrandVehicleTypes not implemented")
}
func (UnimplementedAutowpServer) GetCommentVotes(context.Context, *GetCommentVotesRequest) (*CommentVoteItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCommentVotes not implemented")
}
func (UnimplementedAutowpServer) GetContact(context.Context, *GetContactRequest) (*Contact, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContact not implemented")
}
func (UnimplementedAutowpServer) GetContacts(context.Context, *GetContactsRequest) (*ContactItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContacts not implemented")
}
func (UnimplementedAutowpServer) GetIP(context.Context, *APIGetIPRequest) (*APIIP, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIP not implemented")
}
func (UnimplementedAutowpServer) GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPerspectives not implemented")
}
func (UnimplementedAutowpServer) GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPerspectivePages not implemented")
}
func (UnimplementedAutowpServer) GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReCaptchaConfig not implemented")
}
func (UnimplementedAutowpServer) GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSpecs not implemented")
}
func (UnimplementedAutowpServer) GetTrafficTop(context.Context, *emptypb.Empty) (*APITrafficTopResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTrafficTop not implemented")
}
func (UnimplementedAutowpServer) GetTrafficWhitelist(context.Context, *emptypb.Empty) (*APITrafficWhitelistItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTrafficWhitelist not implemented")
}
func (UnimplementedAutowpServer) GetVehicleTypes(context.Context, *emptypb.Empty) (*VehicleTypeItems, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVehicleTypes not implemented")
}
func (UnimplementedAutowpServer) mustEmbedUnimplementedAutowpServer() {}

// UnsafeAutowpServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AutowpServer will
// result in compilation errors.
type UnsafeAutowpServer interface {
	mustEmbedUnimplementedAutowpServer()
}

func RegisterAutowpServer(s grpc.ServiceRegistrar, srv AutowpServer) {
	s.RegisterService(&Autowp_ServiceDesc, srv)
}

func _Autowp_AclEnforce_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AclEnforceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).AclEnforce(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/AclEnforce",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).AclEnforce(ctx, req.(*AclEnforceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_AddToTrafficBlacklist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddToTrafficBlacklistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).AddToTrafficBlacklist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/AddToTrafficBlacklist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).AddToTrafficBlacklist(ctx, req.(*AddToTrafficBlacklistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_AddToTrafficWhitelist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddToTrafficWhitelistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).AddToTrafficWhitelist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/AddToTrafficWhitelist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).AddToTrafficWhitelist(ctx, req.(*AddToTrafficWhitelistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_CreateContact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateContactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).CreateContact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/CreateContact",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).CreateContact(ctx, req.(*CreateContactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_CreateFeedback_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(APICreateFeedbackRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).CreateFeedback(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/CreateFeedback",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).CreateFeedback(ctx, req.(*APICreateFeedbackRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_DeleteContact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteContactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).DeleteContact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/DeleteContact",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).DeleteContact(ctx, req.(*DeleteContactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_DeleteFromTrafficBlacklist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteFromTrafficBlacklistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).DeleteFromTrafficBlacklist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/DeleteFromTrafficBlacklist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).DeleteFromTrafficBlacklist(ctx, req.(*DeleteFromTrafficBlacklistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_DeleteFromTrafficWhitelist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteFromTrafficWhitelistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).DeleteFromTrafficWhitelist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/DeleteFromTrafficWhitelist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).DeleteFromTrafficWhitelist(ctx, req.(*DeleteFromTrafficWhitelistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetBrandIcons_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetBrandIcons(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetBrandIcons",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetBrandIcons(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetBrandVehicleTypes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBrandVehicleTypesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetBrandVehicleTypes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetBrandVehicleTypes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetBrandVehicleTypes(ctx, req.(*GetBrandVehicleTypesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetCommentVotes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCommentVotesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetCommentVotes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetCommentVotes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetCommentVotes(ctx, req.(*GetCommentVotesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetContact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetContact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetContact",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetContact(ctx, req.(*GetContactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetContacts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContactsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetContacts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetContacts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetContacts(ctx, req.(*GetContactsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetIP_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(APIGetIPRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetIP(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetIP",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetIP(ctx, req.(*APIGetIPRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetPerspectives_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetPerspectives(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetPerspectives",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetPerspectives(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetPerspectivePages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetPerspectivePages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetPerspectivePages",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetPerspectivePages(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetReCaptchaConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetReCaptchaConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetReCaptchaConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetReCaptchaConfig(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetSpecs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetSpecs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetSpecs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetSpecs(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetTrafficTop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetTrafficTop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetTrafficTop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetTrafficTop(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetTrafficWhitelist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetTrafficWhitelist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetTrafficWhitelist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetTrafficWhitelist(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Autowp_GetVehicleTypes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AutowpServer).GetVehicleTypes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/goautowp.Autowp/GetVehicleTypes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AutowpServer).GetVehicleTypes(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// Autowp_ServiceDesc is the grpc.ServiceDesc for Autowp service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Autowp_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "goautowp.Autowp",
	HandlerType: (*AutowpServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AclEnforce",
			Handler:    _Autowp_AclEnforce_Handler,
		},
		{
			MethodName: "AddToTrafficBlacklist",
			Handler:    _Autowp_AddToTrafficBlacklist_Handler,
		},
		{
			MethodName: "AddToTrafficWhitelist",
			Handler:    _Autowp_AddToTrafficWhitelist_Handler,
		},
		{
			MethodName: "CreateContact",
			Handler:    _Autowp_CreateContact_Handler,
		},
		{
			MethodName: "CreateFeedback",
			Handler:    _Autowp_CreateFeedback_Handler,
		},
		{
			MethodName: "DeleteContact",
			Handler:    _Autowp_DeleteContact_Handler,
		},
		{
			MethodName: "DeleteFromTrafficBlacklist",
			Handler:    _Autowp_DeleteFromTrafficBlacklist_Handler,
		},
		{
			MethodName: "DeleteFromTrafficWhitelist",
			Handler:    _Autowp_DeleteFromTrafficWhitelist_Handler,
		},
		{
			MethodName: "GetBrandIcons",
			Handler:    _Autowp_GetBrandIcons_Handler,
		},
		{
			MethodName: "GetBrandVehicleTypes",
			Handler:    _Autowp_GetBrandVehicleTypes_Handler,
		},
		{
			MethodName: "GetCommentVotes",
			Handler:    _Autowp_GetCommentVotes_Handler,
		},
		{
			MethodName: "GetContact",
			Handler:    _Autowp_GetContact_Handler,
		},
		{
			MethodName: "GetContacts",
			Handler:    _Autowp_GetContacts_Handler,
		},
		{
			MethodName: "GetIP",
			Handler:    _Autowp_GetIP_Handler,
		},
		{
			MethodName: "GetPerspectives",
			Handler:    _Autowp_GetPerspectives_Handler,
		},
		{
			MethodName: "GetPerspectivePages",
			Handler:    _Autowp_GetPerspectivePages_Handler,
		},
		{
			MethodName: "GetReCaptchaConfig",
			Handler:    _Autowp_GetReCaptchaConfig_Handler,
		},
		{
			MethodName: "GetSpecs",
			Handler:    _Autowp_GetSpecs_Handler,
		},
		{
			MethodName: "GetTrafficTop",
			Handler:    _Autowp_GetTrafficTop_Handler,
		},
		{
			MethodName: "GetTrafficWhitelist",
			Handler:    _Autowp_GetTrafficWhitelist_Handler,
		},
		{
			MethodName: "GetVehicleTypes",
			Handler:    _Autowp_GetVehicleTypes_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "spec.proto",
}
