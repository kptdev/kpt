// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.4
// source: evaluator.proto

package evaluator

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EvaluateFunctionRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Serialized ResourceList (https://kpt.dev/reference/schema/resource-list/)
	ResourceList []byte `protobuf:"bytes,1,opt,name=resource_list,json=resourceList,proto3" json:"resource_list,omitempty"`
	// kpt image identifying the function to evaluate
	Image string `protobuf:"bytes,2,opt,name=image,proto3" json:"image,omitempty"`
}

func (x *EvaluateFunctionRequest) Reset() {
	*x = EvaluateFunctionRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_evaluator_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluateFunctionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluateFunctionRequest) ProtoMessage() {}

func (x *EvaluateFunctionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_evaluator_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluateFunctionRequest.ProtoReflect.Descriptor instead.
func (*EvaluateFunctionRequest) Descriptor() ([]byte, []int) {
	return file_evaluator_proto_rawDescGZIP(), []int{0}
}

func (x *EvaluateFunctionRequest) GetResourceList() []byte {
	if x != nil {
		return x.ResourceList
	}
	return nil
}

func (x *EvaluateFunctionRequest) GetImage() string {
	if x != nil {
		return x.Image
	}
	return ""
}

// ConfigMap wraps a map<string, string> for use in oneof clause.
type ConfigMap struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data map[string]string `protobuf:"bytes,1,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ConfigMap) Reset() {
	*x = ConfigMap{}
	if protoimpl.UnsafeEnabled {
		mi := &file_evaluator_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigMap) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigMap) ProtoMessage() {}

func (x *ConfigMap) ProtoReflect() protoreflect.Message {
	mi := &file_evaluator_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigMap.ProtoReflect.Descriptor instead.
func (*ConfigMap) Descriptor() ([]byte, []int) {
	return file_evaluator_proto_rawDescGZIP(), []int{1}
}

func (x *ConfigMap) GetData() map[string]string {
	if x != nil {
		return x.Data
	}
	return nil
}

type EvaluateFunctionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Serialized ResourceList (https://kpt.dev/reference/schema/resource-list/),
	// including structured function results.
	ResourceList []byte `protobuf:"bytes,1,opt,name=resource_list,json=resourceList,proto3" json:"resource_list,omitempty"`
	// Additional log produced by the function (if any).
	Log []byte `protobuf:"bytes,2,opt,name=log,proto3" json:"log,omitempty"`
}

func (x *EvaluateFunctionResponse) Reset() {
	*x = EvaluateFunctionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_evaluator_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluateFunctionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluateFunctionResponse) ProtoMessage() {}

func (x *EvaluateFunctionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_evaluator_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluateFunctionResponse.ProtoReflect.Descriptor instead.
func (*EvaluateFunctionResponse) Descriptor() ([]byte, []int) {
	return file_evaluator_proto_rawDescGZIP(), []int{2}
}

func (x *EvaluateFunctionResponse) GetResourceList() []byte {
	if x != nil {
		return x.ResourceList
	}
	return nil
}

func (x *EvaluateFunctionResponse) GetLog() []byte {
	if x != nil {
		return x.Log
	}
	return nil
}

var File_evaluator_proto protoreflect.FileDescriptor

var file_evaluator_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x6f, 0x72, 0x1a, 0x0c, 0x73, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x54, 0x0a, 0x17, 0x45, 0x76,
	0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x46, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6d,
	0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x22, 0x78, 0x0a, 0x09, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4d, 0x61, 0x70, 0x12, 0x32, 0x0a,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x65, 0x76,
	0x61, 0x6c, 0x75, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4d, 0x61,
	0x70, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x64, 0x61, 0x74,
	0x61, 0x1a, 0x37, 0x0a, 0x09, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x51, 0x0a, 0x18, 0x45, 0x76,
	0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x46, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6c,
	0x6f, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x6c, 0x6f, 0x67, 0x32, 0x72, 0x0a,
	0x11, 0x46, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74,
	0x6f, 0x72, 0x12, 0x5d, 0x0a, 0x10, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x46, 0x75,
	0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x22, 0x2e, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74,
	0x6f, 0x72, 0x2e, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x46, 0x75, 0x6e, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x23, 0x2e, 0x65, 0x76, 0x61,
	0x6c, 0x75, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x46,
	0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x42, 0x3a, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x54,
	0x6f, 0x6f, 0x6c, 0x73, 0x2f, 0x6b, 0x70, 0x74, 0x2f, 0x70, 0x6f, 0x72, 0x63, 0x68, 0x2f, 0x66,
	0x75, 0x6e, 0x63, 0x2f, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x6f, 0x72, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_evaluator_proto_rawDescOnce sync.Once
	file_evaluator_proto_rawDescData = file_evaluator_proto_rawDesc
)

func file_evaluator_proto_rawDescGZIP() []byte {
	file_evaluator_proto_rawDescOnce.Do(func() {
		file_evaluator_proto_rawDescData = protoimpl.X.CompressGZIP(file_evaluator_proto_rawDescData)
	})
	return file_evaluator_proto_rawDescData
}

var file_evaluator_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_evaluator_proto_goTypes = []interface{}{
	(*EvaluateFunctionRequest)(nil),  // 0: evaluator.EvaluateFunctionRequest
	(*ConfigMap)(nil),                // 1: evaluator.ConfigMap
	(*EvaluateFunctionResponse)(nil), // 2: evaluator.EvaluateFunctionResponse
	nil,                              // 3: evaluator.ConfigMap.DataEntry
}
var file_evaluator_proto_depIdxs = []int32{
	3, // 0: evaluator.ConfigMap.data:type_name -> evaluator.ConfigMap.DataEntry
	0, // 1: evaluator.FunctionEvaluator.EvaluateFunction:input_type -> evaluator.EvaluateFunctionRequest
	2, // 2: evaluator.FunctionEvaluator.EvaluateFunction:output_type -> evaluator.EvaluateFunctionResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_evaluator_proto_init() }
func file_evaluator_proto_init() {
	if File_evaluator_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_evaluator_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluateFunctionRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_evaluator_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigMap); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_evaluator_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluateFunctionResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_evaluator_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_evaluator_proto_goTypes,
		DependencyIndexes: file_evaluator_proto_depIdxs,
		MessageInfos:      file_evaluator_proto_msgTypes,
	}.Build()
	File_evaluator_proto = out.File
	file_evaluator_proto_rawDesc = nil
	file_evaluator_proto_goTypes = nil
	file_evaluator_proto_depIdxs = nil
}
