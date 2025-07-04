// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.30.2
// source: network.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NetworkIdentificationRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            uint32                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NetworkIdentificationRequest) Reset() {
	*x = NetworkIdentificationRequest{}
	mi := &file_network_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NetworkIdentificationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NetworkIdentificationRequest) ProtoMessage() {}

func (x *NetworkIdentificationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_network_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NetworkIdentificationRequest.ProtoReflect.Descriptor instead.
func (*NetworkIdentificationRequest) Descriptor() ([]byte, []int) {
	return file_network_proto_rawDescGZIP(), []int{0}
}

func (x *NetworkIdentificationRequest) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

type NetworkCreationRequest struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	InternetAccess bool                   `protobuf:"varint,1,opt,name=internet_access,json=internetAccess,proto3" json:"internet_access,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *NetworkCreationRequest) Reset() {
	*x = NetworkCreationRequest{}
	mi := &file_network_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NetworkCreationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NetworkCreationRequest) ProtoMessage() {}

func (x *NetworkCreationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_network_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NetworkCreationRequest.ProtoReflect.Descriptor instead.
func (*NetworkCreationRequest) Descriptor() ([]byte, []int) {
	return file_network_proto_rawDescGZIP(), []int{1}
}

func (x *NetworkCreationRequest) GetInternetAccess() bool {
	if x != nil {
		return x.InternetAccess
	}
	return false
}

type NetworkUpdateRequest struct {
	state          protoimpl.MessageState        `protogen:"open.v1"`
	Identification *NetworkIdentificationRequest `protobuf:"bytes,1,opt,name=identification,proto3" json:"identification,omitempty"`
	Update         *NetworkCreationRequest       `protobuf:"bytes,2,opt,name=update,proto3" json:"update,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *NetworkUpdateRequest) Reset() {
	*x = NetworkUpdateRequest{}
	mi := &file_network_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NetworkUpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NetworkUpdateRequest) ProtoMessage() {}

func (x *NetworkUpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_network_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NetworkUpdateRequest.ProtoReflect.Descriptor instead.
func (*NetworkUpdateRequest) Descriptor() ([]byte, []int) {
	return file_network_proto_rawDescGZIP(), []int{2}
}

func (x *NetworkUpdateRequest) GetIdentification() *NetworkIdentificationRequest {
	if x != nil {
		return x.Identification
	}
	return nil
}

func (x *NetworkUpdateRequest) GetUpdate() *NetworkCreationRequest {
	if x != nil {
		return x.Update
	}
	return nil
}

type Network struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	Id             uint32                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	InternetAccess bool                   `protobuf:"varint,2,opt,name=internet_access,json=internetAccess,proto3" json:"internet_access,omitempty"`
	CreatedAt      *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *Network) Reset() {
	*x = Network{}
	mi := &file_network_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Network) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Network) ProtoMessage() {}

func (x *Network) ProtoReflect() protoreflect.Message {
	mi := &file_network_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Network.ProtoReflect.Descriptor instead.
func (*Network) Descriptor() ([]byte, []int) {
	return file_network_proto_rawDescGZIP(), []int{3}
}

func (x *Network) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Network) GetInternetAccess() bool {
	if x != nil {
		return x.InternetAccess
	}
	return false
}

func (x *Network) GetCreatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

var File_network_proto protoreflect.FileDescriptor

const file_network_proto_rawDesc = "" +
	"\n" +
	"\rnetwork.proto\x12\bbx2cloud\x1a\x1bgoogle/protobuf/empty.proto\x1a\x1fgoogle/protobuf/timestamp.proto\".\n" +
	"\x1cNetworkIdentificationRequest\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\rR\x02id\"A\n" +
	"\x16NetworkCreationRequest\x12'\n" +
	"\x0finternet_access\x18\x01 \x01(\bR\x0einternetAccess\"\xa0\x01\n" +
	"\x14NetworkUpdateRequest\x12N\n" +
	"\x0eidentification\x18\x01 \x01(\v2&.bx2cloud.NetworkIdentificationRequestR\x0eidentification\x128\n" +
	"\x06update\x18\x02 \x01(\v2 .bx2cloud.NetworkCreationRequestR\x06update\"|\n" +
	"\aNetwork\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\rR\x02id\x12'\n" +
	"\x0finternet_access\x18\x02 \x01(\bR\x0einternetAccess\x128\n" +
	"\tcreatedAt\x18\x04 \x01(\v2\x1a.google.protobuf.TimestampR\tcreatedAt2\xcd\x02\n" +
	"\x0eNetworkService\x12@\n" +
	"\x03Get\x12&.bx2cloud.NetworkIdentificationRequest\x1a\x11.bx2cloud.Network\x123\n" +
	"\x04List\x12\x16.google.protobuf.Empty\x1a\x11.bx2cloud.Network0\x01\x12=\n" +
	"\x06Create\x12 .bx2cloud.NetworkCreationRequest\x1a\x11.bx2cloud.Network\x12;\n" +
	"\x06Update\x12\x1e.bx2cloud.NetworkUpdateRequest\x1a\x11.bx2cloud.Network\x12H\n" +
	"\x06Delete\x12&.bx2cloud.NetworkIdentificationRequest\x1a\x16.google.protobuf.EmptyB,Z*github.com/BenasB/bx2cloud/internal/api/pbb\x06proto3"

var (
	file_network_proto_rawDescOnce sync.Once
	file_network_proto_rawDescData []byte
)

func file_network_proto_rawDescGZIP() []byte {
	file_network_proto_rawDescOnce.Do(func() {
		file_network_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_network_proto_rawDesc), len(file_network_proto_rawDesc)))
	})
	return file_network_proto_rawDescData
}

var file_network_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_network_proto_goTypes = []any{
	(*NetworkIdentificationRequest)(nil), // 0: bx2cloud.NetworkIdentificationRequest
	(*NetworkCreationRequest)(nil),       // 1: bx2cloud.NetworkCreationRequest
	(*NetworkUpdateRequest)(nil),         // 2: bx2cloud.NetworkUpdateRequest
	(*Network)(nil),                      // 3: bx2cloud.Network
	(*timestamppb.Timestamp)(nil),        // 4: google.protobuf.Timestamp
	(*emptypb.Empty)(nil),                // 5: google.protobuf.Empty
}
var file_network_proto_depIdxs = []int32{
	0, // 0: bx2cloud.NetworkUpdateRequest.identification:type_name -> bx2cloud.NetworkIdentificationRequest
	1, // 1: bx2cloud.NetworkUpdateRequest.update:type_name -> bx2cloud.NetworkCreationRequest
	4, // 2: bx2cloud.Network.createdAt:type_name -> google.protobuf.Timestamp
	0, // 3: bx2cloud.NetworkService.Get:input_type -> bx2cloud.NetworkIdentificationRequest
	5, // 4: bx2cloud.NetworkService.List:input_type -> google.protobuf.Empty
	1, // 5: bx2cloud.NetworkService.Create:input_type -> bx2cloud.NetworkCreationRequest
	2, // 6: bx2cloud.NetworkService.Update:input_type -> bx2cloud.NetworkUpdateRequest
	0, // 7: bx2cloud.NetworkService.Delete:input_type -> bx2cloud.NetworkIdentificationRequest
	3, // 8: bx2cloud.NetworkService.Get:output_type -> bx2cloud.Network
	3, // 9: bx2cloud.NetworkService.List:output_type -> bx2cloud.Network
	3, // 10: bx2cloud.NetworkService.Create:output_type -> bx2cloud.Network
	3, // 11: bx2cloud.NetworkService.Update:output_type -> bx2cloud.Network
	5, // 12: bx2cloud.NetworkService.Delete:output_type -> google.protobuf.Empty
	8, // [8:13] is the sub-list for method output_type
	3, // [3:8] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_network_proto_init() }
func file_network_proto_init() {
	if File_network_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_network_proto_rawDesc), len(file_network_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_network_proto_goTypes,
		DependencyIndexes: file_network_proto_depIdxs,
		MessageInfos:      file_network_proto_msgTypes,
	}.Build()
	File_network_proto = out.File
	file_network_proto_goTypes = nil
	file_network_proto_depIdxs = nil
}
