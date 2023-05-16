// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: vmbridge/wasm/v1/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgStoreCode submit Wasm code to the system
type MsgSendToEvm struct {
	// Sender is the that actor that signed the messages
	Sender    string  `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender"`
	Contract  string  `protobuf:"bytes,2,opt,name=contract,proto3" json:"contract"`
	Recipient string  `protobuf:"bytes,3,opt,name=recipient,proto3" json:"recipient"`
	Amount    sdk.Int `protobuf:"bytes,4,opt,name=amount,proto3,customtype=Int" json:"amount"`
}

func (m *MsgSendToEvm) Reset()         { *m = MsgSendToEvm{} }
func (m *MsgSendToEvm) String() string { return proto.CompactTextString(m) }
func (*MsgSendToEvm) ProtoMessage()    {}
func (*MsgSendToEvm) Descriptor() ([]byte, []int) {
	return fileDescriptor_8bf6605aff77555b, []int{0}
}
func (m *MsgSendToEvm) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgSendToEvm) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgSendToEvm.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgSendToEvm) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgSendToEvm.Merge(m, src)
}
func (m *MsgSendToEvm) XXX_Size() int {
	return m.Size()
}
func (m *MsgSendToEvm) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgSendToEvm.DiscardUnknown(m)
}

var xxx_messageInfo_MsgSendToEvm proto.InternalMessageInfo

// MsgStoreCodeResponse returns store result data.
type MsgSendToEvmResponse struct {
	// CodeID is the reference to the stored WASM code
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

func (m *MsgSendToEvmResponse) Reset()         { *m = MsgSendToEvmResponse{} }
func (m *MsgSendToEvmResponse) String() string { return proto.CompactTextString(m) }
func (*MsgSendToEvmResponse) ProtoMessage()    {}
func (*MsgSendToEvmResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8bf6605aff77555b, []int{1}
}
func (m *MsgSendToEvmResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgSendToEvmResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgSendToEvmResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgSendToEvmResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgSendToEvmResponse.Merge(m, src)
}
func (m *MsgSendToEvmResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgSendToEvmResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgSendToEvmResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgSendToEvmResponse proto.InternalMessageInfo

// MsgStoreCode submit Wasm code to the system
type MsgCallToEvm struct {
	// Sender is the that actor that signed the messages
	Sender   string  `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender"`
	Evmaddr  string  `protobuf:"bytes,2,opt,name=evmaddr,proto3" json:"evmaddr"`
	Calldata string  `protobuf:"bytes,3,opt,name=calldata,proto3" json:"calldata"`
	Value    sdk.Int `protobuf:"bytes,4,opt,name=value,proto3,customtype=Int" json:"value"`
}

func (m *MsgCallToEvm) Reset()         { *m = MsgCallToEvm{} }
func (m *MsgCallToEvm) String() string { return proto.CompactTextString(m) }
func (*MsgCallToEvm) ProtoMessage()    {}
func (*MsgCallToEvm) Descriptor() ([]byte, []int) {
	return fileDescriptor_8bf6605aff77555b, []int{2}
}
func (m *MsgCallToEvm) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgCallToEvm) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgCallToEvm.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgCallToEvm) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgCallToEvm.Merge(m, src)
}
func (m *MsgCallToEvm) XXX_Size() int {
	return m.Size()
}
func (m *MsgCallToEvm) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgCallToEvm.DiscardUnknown(m)
}

var xxx_messageInfo_MsgCallToEvm proto.InternalMessageInfo

// MsgStoreCodeResponse returns store result data.
type MsgCallToEvmResponse struct {
	// CodeID is the reference to the stored WASM code
	Response string `protobuf:"bytes,1,opt,name=response,proto3" json:"response"`
}

func (m *MsgCallToEvmResponse) Reset()         { *m = MsgCallToEvmResponse{} }
func (m *MsgCallToEvmResponse) String() string { return proto.CompactTextString(m) }
func (*MsgCallToEvmResponse) ProtoMessage()    {}
func (*MsgCallToEvmResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8bf6605aff77555b, []int{3}
}
func (m *MsgCallToEvmResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgCallToEvmResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgCallToEvmResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgCallToEvmResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgCallToEvmResponse.Merge(m, src)
}
func (m *MsgCallToEvmResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgCallToEvmResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgCallToEvmResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgCallToEvmResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgSendToEvm)(nil), "vmbridge.wasm.v1.MsgSendToEvm")
	proto.RegisterType((*MsgSendToEvmResponse)(nil), "vmbridge.wasm.v1.MsgSendToEvmResponse")
	proto.RegisterType((*MsgCallToEvm)(nil), "vmbridge.wasm.v1.MsgCallToEvm")
	proto.RegisterType((*MsgCallToEvmResponse)(nil), "vmbridge.wasm.v1.MsgCallToEvmResponse")
}

func init() { proto.RegisterFile("vmbridge/wasm/v1/tx.proto", fileDescriptor_8bf6605aff77555b) }

var fileDescriptor_8bf6605aff77555b = []byte{
	// 403 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x52, 0xcd, 0xae, 0xd2, 0x40,
	0x18, 0x6d, 0x45, 0xf9, 0x19, 0x90, 0x90, 0x09, 0x8b, 0xca, 0x62, 0xaa, 0x35, 0x1a, 0x12, 0x93,
	0x56, 0xf4, 0x05, 0x0c, 0x86, 0x85, 0x0b, 0x36, 0xa3, 0x0b, 0xe2, 0x6e, 0x68, 0x27, 0x0d, 0x49,
	0xdb, 0x69, 0x3a, 0x43, 0xc5, 0xb7, 0xf0, 0x29, 0x5c, 0xba, 0xf7, 0x0d, 0x58, 0xb2, 0x34, 0x2e,
	0x9a, 0x7b, 0x61, 0xd7, 0xa7, 0xb8, 0x61, 0x3a, 0x1d, 0xc8, 0xfd, 0x21, 0x77, 0xc5, 0xe1, 0x9c,
	0x93, 0xe9, 0x39, 0x27, 0x1f, 0x78, 0x91, 0xc7, 0xcb, 0x6c, 0x15, 0x84, 0xd4, 0xfb, 0x41, 0x78,
	0xec, 0xe5, 0x13, 0x4f, 0x6c, 0xdc, 0x34, 0x63, 0x82, 0xc1, 0x41, 0x2d, 0xb9, 0x47, 0xc9, 0xcd,
	0x27, 0xa3, 0x61, 0xc8, 0x42, 0x26, 0x45, 0xef, 0x88, 0x2a, 0x9f, 0xf3, 0xc7, 0x04, 0xbd, 0x39,
	0x0f, 0xbf, 0xd2, 0x24, 0xf8, 0xc6, 0x66, 0x79, 0x0c, 0x1d, 0xd0, 0xe4, 0x34, 0x09, 0x68, 0x66,
	0x99, 0x2f, 0xcd, 0x71, 0x67, 0x0a, 0xca, 0xc2, 0x56, 0x0c, 0x56, 0xbf, 0x70, 0x0c, 0xda, 0x3e,
	0x4b, 0x44, 0x46, 0x7c, 0x61, 0x3d, 0x91, 0xae, 0x5e, 0x59, 0xd8, 0x9a, 0xc3, 0x1a, 0xc1, 0x77,
	0xa0, 0x93, 0x51, 0x7f, 0x95, 0xae, 0x68, 0x22, 0xac, 0x86, 0xb4, 0x3e, 0x2f, 0x0b, 0xfb, 0x44,
	0xe2, 0x13, 0x84, 0xaf, 0x41, 0x93, 0xc4, 0x6c, 0x9d, 0x08, 0xeb, 0xa9, 0x74, 0x76, 0xb7, 0x85,
	0x6d, 0xfc, 0x2f, 0xec, 0xc6, 0x97, 0x44, 0x60, 0x25, 0x39, 0xef, 0xc1, 0xf0, 0x3c, 0x2f, 0xa6,
	0x3c, 0x65, 0x09, 0xa7, 0xd0, 0x02, 0x2d, 0xbe, 0xf6, 0x7d, 0xca, 0xb9, 0x0c, 0xde, 0xc6, 0xf5,
	0x5f, 0xe7, 0x77, 0x55, 0xf1, 0x33, 0x89, 0xa2, 0xc7, 0x57, 0x7c, 0x03, 0x5a, 0x34, 0x8f, 0x49,
	0x10, 0x64, 0xaa, 0x61, 0xb7, 0x2c, 0xec, 0x9a, 0xc2, 0x35, 0x90, 0x4b, 0x90, 0x28, 0x0a, 0x88,
	0x20, 0xaa, 0x5e, 0xb5, 0x84, 0xe2, 0xb0, 0x46, 0xf0, 0x15, 0x78, 0x96, 0x93, 0x68, 0x4d, 0xef,
	0xeb, 0x56, 0x29, 0xce, 0x27, 0x59, 0x4d, 0xe7, 0xd4, 0xd5, 0xc6, 0xa0, 0x9d, 0x29, 0xac, 0x12,
	0xcb, 0x8f, 0xd4, 0x1c, 0xd6, 0xe8, 0xc3, 0x5f, 0x13, 0x34, 0xe6, 0x3c, 0x84, 0x0b, 0xd0, 0xd7,
	0x0b, 0xcd, 0xf2, 0xe3, 0xb6, 0xc8, 0xbd, 0x7d, 0x10, 0xee, 0xf9, 0x8c, 0xa3, 0xb7, 0x97, 0x75,
	0x9d, 0x65, 0x01, 0xfa, 0x3a, 0xe0, 0xa5, 0x97, 0xb5, 0xe9, 0x81, 0x97, 0xef, 0xb4, 0x9c, 0xba,
	0xdb, 0x6b, 0x64, 0x6c, 0xf7, 0xc8, 0xdc, 0xed, 0x91, 0x79, 0xb5, 0x47, 0xe6, 0xaf, 0x03, 0x32,
	0x76, 0x07, 0x64, 0xfc, 0x3b, 0x20, 0xe3, 0xfb, 0x60, 0xe3, 0xe9, 0x63, 0x17, 0x3f, 0x53, 0xca,
	0x97, 0x4d, 0x79, 0xc0, 0x1f, 0x6f, 0x02, 0x00, 0x00, 0xff, 0xff, 0x38, 0xf8, 0xdc, 0x02, 0x05,
	0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// SendToEvmEvent to exchange cw20 to erc20
	SendToEvmEvent(ctx context.Context, in *MsgSendToEvm, opts ...grpc.CallOption) (*MsgSendToEvmResponse, error)
	// CallToEvmEvent to call to evm contract
	CallToEvmEvent(ctx context.Context, in *MsgCallToEvm, opts ...grpc.CallOption) (*MsgCallToEvmResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) SendToEvmEvent(ctx context.Context, in *MsgSendToEvm, opts ...grpc.CallOption) (*MsgSendToEvmResponse, error) {
	out := new(MsgSendToEvmResponse)
	err := c.cc.Invoke(ctx, "/vmbridge.wasm.v1.Msg/SendToEvmEvent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) CallToEvmEvent(ctx context.Context, in *MsgCallToEvm, opts ...grpc.CallOption) (*MsgCallToEvmResponse, error) {
	out := new(MsgCallToEvmResponse)
	err := c.cc.Invoke(ctx, "/vmbridge.wasm.v1.Msg/CallToEvmEvent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// SendToEvmEvent to exchange cw20 to erc20
	SendToEvmEvent(context.Context, *MsgSendToEvm) (*MsgSendToEvmResponse, error)
	// CallToEvmEvent to call to evm contract
	CallToEvmEvent(context.Context, *MsgCallToEvm) (*MsgCallToEvmResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) SendToEvmEvent(ctx context.Context, req *MsgSendToEvm) (*MsgSendToEvmResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendToEvmEvent not implemented")
}
func (*UnimplementedMsgServer) CallToEvmEvent(ctx context.Context, req *MsgCallToEvm) (*MsgCallToEvmResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CallToEvmEvent not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_SendToEvmEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSendToEvm)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SendToEvmEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/vmbridge.wasm.v1.Msg/SendToEvmEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SendToEvmEvent(ctx, req.(*MsgSendToEvm))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_CallToEvmEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCallToEvm)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CallToEvmEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/vmbridge.wasm.v1.Msg/CallToEvmEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CallToEvmEvent(ctx, req.(*MsgCallToEvm))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "vmbridge.wasm.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendToEvmEvent",
			Handler:    _Msg_SendToEvmEvent_Handler,
		},
		{
			MethodName: "CallToEvmEvent",
			Handler:    _Msg_CallToEvmEvent_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "vmbridge/wasm/v1/tx.proto",
}

func (m *MsgSendToEvm) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgSendToEvm) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgSendToEvm) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Amount.Size()
		i -= size
		if _, err := m.Amount.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if len(m.Recipient) > 0 {
		i -= len(m.Recipient)
		copy(dAtA[i:], m.Recipient)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Recipient)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Contract) > 0 {
		i -= len(m.Contract)
		copy(dAtA[i:], m.Contract)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Contract)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgSendToEvmResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgSendToEvmResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgSendToEvmResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Success {
		i--
		if m.Success {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *MsgCallToEvm) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgCallToEvm) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgCallToEvm) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Value.Size()
		i -= size
		if _, err := m.Value.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if len(m.Calldata) > 0 {
		i -= len(m.Calldata)
		copy(dAtA[i:], m.Calldata)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Calldata)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Evmaddr) > 0 {
		i -= len(m.Evmaddr)
		copy(dAtA[i:], m.Evmaddr)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Evmaddr)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgCallToEvmResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgCallToEvmResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgCallToEvmResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Response) > 0 {
		i -= len(m.Response)
		copy(dAtA[i:], m.Response)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Response)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgSendToEvm) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Contract)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Recipient)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Amount.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgSendToEvmResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Success {
		n += 2
	}
	return n
}

func (m *MsgCallToEvm) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Evmaddr)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Calldata)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Value.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgCallToEvmResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Response)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgSendToEvm) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgSendToEvm: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgSendToEvm: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Contract", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Contract = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Recipient", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Recipient = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Amount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgSendToEvmResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgSendToEvmResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgSendToEvmResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Success", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Success = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgCallToEvm) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgCallToEvm: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgCallToEvm: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Evmaddr", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Evmaddr = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Calldata", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Calldata = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Value.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgCallToEvmResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgCallToEvmResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgCallToEvmResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Response", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Response = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
