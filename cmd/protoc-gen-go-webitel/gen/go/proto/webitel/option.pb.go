// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        (unknown)
// source: proto/webitel/option.proto

package webitel

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Action int32

const (
	Action_ACTION_CREATE Action = 0
	Action_ACTION_READ   Action = 1
	Action_ACTION_UPDATE Action = 2
	Action_ACTION_DELETE Action = 3
)

// Enum value maps for Action.
var (
	Action_name = map[int32]string{
		0: "ACTION_CREATE",
		1: "ACTION_READ",
		2: "ACTION_UPDATE",
		3: "ACTION_DELETE",
	}
	Action_value = map[string]int32{
		"ACTION_CREATE": 0,
		"ACTION_READ":   1,
		"ACTION_UPDATE": 2,
		"ACTION_DELETE": 3,
	}
)

func (x Action) Enum() *Action {
	p := new(Action)
	*p = x
	return p
}

func (x Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Action) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_webitel_option_proto_enumTypes[0].Descriptor()
}

func (Action) Type() protoreflect.EnumType {
	return &file_proto_webitel_option_proto_enumTypes[0]
}

func (x Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Action.Descriptor instead.
func (Action) EnumDescriptor() ([]byte, []int) {
	return file_proto_webitel_option_proto_rawDescGZIP(), []int{0}
}

var file_proto_webitel_option_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.ServiceOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         50001,
		Name:          "option.objclass",
		Tag:           "bytes,50001,opt,name=objclass",
		Filename:      "proto/webitel/option.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ServiceOptions)(nil),
		ExtensionType: ([]string)(nil),
		Field:         50002,
		Name:          "option.additional_license",
		Tag:           "bytes,50002,rep,name=additional_license",
		Filename:      "proto/webitel/option.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MethodOptions)(nil),
		ExtensionType: (*Action)(nil),
		Field:         50002,
		Name:          "option.access",
		Tag:           "varint,50002,opt,name=access,enum=option.Action",
		Filename:      "proto/webitel/option.proto",
	},
}

// Extension fields to descriptorpb.ServiceOptions.
var (
	// optional string objclass = 50001;
	E_Objclass = &file_proto_webitel_option_proto_extTypes[0]
	// repeated string additional_license = 50002;
	E_AdditionalLicense = &file_proto_webitel_option_proto_extTypes[1]
)

// Extension fields to descriptorpb.MethodOptions.
var (
	// optional option.Action access = 50002;
	E_Access = &file_proto_webitel_option_proto_extTypes[2]
)

var File_proto_webitel_option_proto protoreflect.FileDescriptor

var file_proto_webitel_option_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x77, 0x65, 0x62, 0x69, 0x74, 0x65, 0x6c, 0x2f,
	0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a, 0x52, 0x0a, 0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x11, 0x0a, 0x0d, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x43, 0x52, 0x45, 0x41, 0x54,
	0x45, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x45,
	0x41, 0x44, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55,
	0x50, 0x44, 0x41, 0x54, 0x45, 0x10, 0x02, 0x12, 0x11, 0x0a, 0x0d, 0x41, 0x43, 0x54, 0x49, 0x4f,
	0x4e, 0x5f, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x03, 0x3a, 0x3d, 0x0a, 0x08, 0x6f, 0x62,
	0x6a, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd1, 0x86, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x6f, 0x62, 0x6a, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x3a, 0x50, 0x0a, 0x12, 0x61, 0x64, 0x64,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x6c, 0x69, 0x63, 0x65, 0x6e, 0x73, 0x65, 0x12,
	0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0xd2, 0x86, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x11, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x4c, 0x69, 0x63, 0x65, 0x6e, 0x73, 0x65, 0x3a, 0x48, 0x0a, 0x06, 0x61,
	0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x1e, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd2, 0x86, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e,
	0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x61,
	0x63, 0x63, 0x65, 0x73, 0x73, 0x42, 0xab, 0x01, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x2e, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x42, 0x0b, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x50, 0x01, 0x5a, 0x58, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x77, 0x65, 0x62, 0x69, 0x74, 0x65, 0x6c, 0x2f, 0x77, 0x65, 0x62, 0x69, 0x74, 0x65, 0x6c, 0x2d,
	0x67, 0x6f, 0x2d, 0x6b, 0x69, 0x74, 0x2f, 0x63, 0x6d, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x67, 0x6f, 0x2d, 0x77, 0x65, 0x62, 0x69, 0x74, 0x65, 0x6c,
	0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x67, 0x6f, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x77, 0x65,
	0x62, 0x69, 0x74, 0x65, 0x6c, 0x3b, 0x77, 0x65, 0x62, 0x69, 0x74, 0x65, 0x6c, 0xa2, 0x02, 0x03,
	0x4f, 0x58, 0x58, 0xaa, 0x02, 0x06, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0xca, 0x02, 0x06, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0xe2, 0x02, 0x12, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x06, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_webitel_option_proto_rawDescOnce sync.Once
	file_proto_webitel_option_proto_rawDescData = file_proto_webitel_option_proto_rawDesc
)

func file_proto_webitel_option_proto_rawDescGZIP() []byte {
	file_proto_webitel_option_proto_rawDescOnce.Do(func() {
		file_proto_webitel_option_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_webitel_option_proto_rawDescData)
	})
	return file_proto_webitel_option_proto_rawDescData
}

var file_proto_webitel_option_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_webitel_option_proto_goTypes = []interface{}{
	(Action)(0),                         // 0: option.Action
	(*descriptorpb.ServiceOptions)(nil), // 1: google.protobuf.ServiceOptions
	(*descriptorpb.MethodOptions)(nil),  // 2: google.protobuf.MethodOptions
}
var file_proto_webitel_option_proto_depIdxs = []int32{
	1, // 0: option.objclass:extendee -> google.protobuf.ServiceOptions
	1, // 1: option.additional_license:extendee -> google.protobuf.ServiceOptions
	2, // 2: option.access:extendee -> google.protobuf.MethodOptions
	0, // 3: option.access:type_name -> option.Action
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	3, // [3:4] is the sub-list for extension type_name
	0, // [0:3] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_webitel_option_proto_init() }
func file_proto_webitel_option_proto_init() {
	if File_proto_webitel_option_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_webitel_option_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 3,
			NumServices:   0,
		},
		GoTypes:           file_proto_webitel_option_proto_goTypes,
		DependencyIndexes: file_proto_webitel_option_proto_depIdxs,
		EnumInfos:         file_proto_webitel_option_proto_enumTypes,
		ExtensionInfos:    file_proto_webitel_option_proto_extTypes,
	}.Build()
	File_proto_webitel_option_proto = out.File
	file_proto_webitel_option_proto_rawDesc = nil
	file_proto_webitel_option_proto_goTypes = nil
	file_proto_webitel_option_proto_depIdxs = nil
}
