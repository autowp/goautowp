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
	GetBrandIcons(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BrandIcons, error)
	GetPerspectives(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivesItems, error)
	GetPerspectivePages(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PerspectivePagesItems, error)
	GetReCaptchaConfig(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ReCaptchaConfig, error)
	GetSpecs(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*SpecsItems, error)
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

func (c *autowpClient) GetBrandIcons(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BrandIcons, error) {
	out := new(BrandIcons)
	err := c.cc.Invoke(ctx, "/goautowp.Autowp/GetBrandIcons", in, out, opts...)
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
	GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error)
	GetPerspectives(context.Context, *emptypb.Empty) (*PerspectivesItems, error)
	GetPerspectivePages(context.Context, *emptypb.Empty) (*PerspectivePagesItems, error)
	GetReCaptchaConfig(context.Context, *emptypb.Empty) (*ReCaptchaConfig, error)
	GetSpecs(context.Context, *emptypb.Empty) (*SpecsItems, error)
	GetVehicleTypes(context.Context, *emptypb.Empty) (*VehicleTypeItems, error)
	mustEmbedUnimplementedAutowpServer()
}

// UnimplementedAutowpServer must be embedded to have forward compatible implementations.
type UnimplementedAutowpServer struct {
}

func (UnimplementedAutowpServer) AclEnforce(context.Context, *AclEnforceRequest) (*AclEnforceResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AclEnforce not implemented")
}
func (UnimplementedAutowpServer) GetBrandIcons(context.Context, *emptypb.Empty) (*BrandIcons, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBrandIcons not implemented")
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
			MethodName: "GetBrandIcons",
			Handler:    _Autowp_GetBrandIcons_Handler,
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
			MethodName: "GetVehicleTypes",
			Handler:    _Autowp_GetVehicleTypes_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "spec.proto",
}
