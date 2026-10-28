package types

import (
	"context"

	"google.golang.org/grpc"
)

func init() {
	_Msg_serviceDesc.Methods = append(_Msg_serviceDesc.Methods,
		grpc.MethodDesc{
			MethodName: "RegisterAssessmentSigner",
			Handler:    _Msg_RegisterAssessmentSigner_Handler,
		},
		grpc.MethodDesc{
			MethodName: "SetAssessmentSignerStatus",
			Handler:    _Msg_SetAssessmentSignerStatus_Handler,
		},
	)
}

func _Msg_RegisterAssessmentSigner_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterAssessmentSigner)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterAssessmentSigner(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/uptick.appmining.v1.Msg/RegisterAssessmentSigner",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterAssessmentSigner(ctx, req.(*MsgRegisterAssessmentSigner))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SetAssessmentSignerStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSetAssessmentSignerStatus)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SetAssessmentSignerStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/uptick.appmining.v1.Msg/SetAssessmentSignerStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SetAssessmentSignerStatus(ctx, req.(*MsgSetAssessmentSignerStatus))
	}
	return interceptor(ctx, in, info, handler)
}
