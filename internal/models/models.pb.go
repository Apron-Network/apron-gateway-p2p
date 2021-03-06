// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.6.1
// source: models.proto

package models

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ApronApiKey struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key       string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	ServiceId string `protobuf:"bytes,2,opt,name=service_id,json=serviceId,proto3" json:"service_id,omitempty"`
	IssuedAt  int64  `protobuf:"varint,3,opt,name=issued_at,json=issuedAt,proto3" json:"issued_at,omitempty"`
	ExpiredAt int64  `protobuf:"varint,4,opt,name=expired_at,json=expiredAt,proto3" json:"expired_at,omitempty"`
	AccountId string `protobuf:"bytes,5,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
}

func (x *ApronApiKey) Reset() {
	*x = ApronApiKey{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronApiKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronApiKey) ProtoMessage() {}

func (x *ApronApiKey) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronApiKey.ProtoReflect.Descriptor instead.
func (*ApronApiKey) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{0}
}

func (x *ApronApiKey) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *ApronApiKey) GetServiceId() string {
	if x != nil {
		return x.ServiceId
	}
	return ""
}

func (x *ApronApiKey) GetIssuedAt() int64 {
	if x != nil {
		return x.IssuedAt
	}
	return 0
}

func (x *ApronApiKey) GetExpiredAt() int64 {
	if x != nil {
		return x.ExpiredAt
	}
	return 0
}

func (x *ApronApiKey) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

type ApronServiceProvider struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// TODO: Those fields are not using now, will be activated later
	Id          string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Desc        string `protobuf:"bytes,3,opt,name=desc,proto3" json:"desc,omitempty"`
	CreatedAt   int64  `protobuf:"varint,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt   int64  `protobuf:"varint,5,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	ExtraDetail string `protobuf:"bytes,6,opt,name=extra_detail,json=extraDetail,proto3" json:"extra_detail,omitempty"`
	BaseUrl     string `protobuf:"bytes,20,opt,name=base_url,json=baseUrl,proto3" json:"base_url,omitempty"`
	Schema      string `protobuf:"bytes,21,opt,name=schema,proto3" json:"schema,omitempty"`
}

func (x *ApronServiceProvider) Reset() {
	*x = ApronServiceProvider{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronServiceProvider) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronServiceProvider) ProtoMessage() {}

func (x *ApronServiceProvider) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronServiceProvider.ProtoReflect.Descriptor instead.
func (*ApronServiceProvider) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{1}
}

func (x *ApronServiceProvider) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ApronServiceProvider) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ApronServiceProvider) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *ApronServiceProvider) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

func (x *ApronServiceProvider) GetUpdatedAt() int64 {
	if x != nil {
		return x.UpdatedAt
	}
	return 0
}

func (x *ApronServiceProvider) GetExtraDetail() string {
	if x != nil {
		return x.ExtraDetail
	}
	return ""
}

func (x *ApronServiceProvider) GetBaseUrl() string {
	if x != nil {
		return x.BaseUrl
	}
	return ""
}

func (x *ApronServiceProvider) GetSchema() string {
	if x != nil {
		return x.Schema
	}
	return ""
}

type ApronService struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Uniq id for service, will be generated automatically
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// name of this service
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Description of the service
	Desc string `protobuf:"bytes,3,opt,name=desc,proto3" json:"desc,omitempty"`
	// Logo image path for the service
	Logo string `protobuf:"bytes,4,opt,name=logo,proto3" json:"logo,omitempty"`
	// Brief usage
	Usage string `protobuf:"bytes,5,opt,name=usage,proto3" json:"usage,omitempty"`
	// Creator's user id
	UserId string `protobuf:"bytes,6,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	// Providers for this service
	Providers []*ApronServiceProvider `protobuf:"bytes,10,rep,name=providers,proto3" json:"providers,omitempty"`
	IsDeleted bool                    `protobuf:"varint,11,opt,name=is_deleted,json=isDeleted,proto3" json:"is_deleted,omitempty"`
}

func (x *ApronService) Reset() {
	*x = ApronService{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronService) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronService) ProtoMessage() {}

func (x *ApronService) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronService.ProtoReflect.Descriptor instead.
func (*ApronService) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{2}
}

func (x *ApronService) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ApronService) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ApronService) GetDesc() string {
	if x != nil {
		return x.Desc
	}
	return ""
}

func (x *ApronService) GetLogo() string {
	if x != nil {
		return x.Logo
	}
	return ""
}

func (x *ApronService) GetUsage() string {
	if x != nil {
		return x.Usage
	}
	return ""
}

func (x *ApronService) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *ApronService) GetProviders() []*ApronServiceProvider {
	if x != nil {
		return x.Providers
	}
	return nil
}

func (x *ApronService) GetIsDeleted() bool {
	if x != nil {
		return x.IsDeleted
	}
	return false
}

type ApronUser struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Email string `protobuf:"bytes,1,opt,name=email,proto3" json:"email,omitempty"`
}

func (x *ApronUser) Reset() {
	*x = ApronUser{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronUser) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronUser) ProtoMessage() {}

func (x *ApronUser) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronUser.ProtoReflect.Descriptor instead.
func (*ApronUser) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{3}
}

func (x *ApronUser) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

type AccessLog struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ts          int64  `protobuf:"varint,1,opt,name=ts,proto3" json:"ts,omitempty"`
	ServiceName string `protobuf:"bytes,2,opt,name=service_name,json=serviceName,proto3" json:"service_name,omitempty"`
	UserKey     string `protobuf:"bytes,3,opt,name=user_key,json=userKey,proto3" json:"user_key,omitempty"`
	RequestIp   string `protobuf:"bytes,4,opt,name=request_ip,json=requestIp,proto3" json:"request_ip,omitempty"`
	RequestPath string `protobuf:"bytes,5,opt,name=request_path,json=requestPath,proto3" json:"request_path,omitempty"`
}

func (x *AccessLog) Reset() {
	*x = AccessLog{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AccessLog) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AccessLog) ProtoMessage() {}

func (x *AccessLog) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AccessLog.ProtoReflect.Descriptor instead.
func (*AccessLog) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{4}
}

func (x *AccessLog) GetTs() int64 {
	if x != nil {
		return x.Ts
	}
	return 0
}

func (x *AccessLog) GetServiceName() string {
	if x != nil {
		return x.ServiceName
	}
	return ""
}

func (x *AccessLog) GetUserKey() string {
	if x != nil {
		return x.UserKey
	}
	return ""
}

func (x *AccessLog) GetRequestIp() string {
	if x != nil {
		return x.RequestIp
	}
	return ""
}

func (x *AccessLog) GetRequestPath() string {
	if x != nil {
		return x.RequestPath
	}
	return ""
}

type ApronServiceRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Service id indicate the service, which will be used to fetch service detail in service side node
	ServiceId string `protobuf:"bytes,1,opt,name=service_id,json=serviceId,proto3" json:"service_id,omitempty"`
	// request_id is used to distinguish various requests in same stream. The request id will be sent to remote node,
	// and all response from remote node for this request will have this id attached, then the sender node can pass
	// the request to correct client.
	RequestId string `protobuf:"bytes,2,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"`
	// request sender peer id, used to create response stream in remote node
	PeerId string `protobuf:"bytes,3,opt,name=peer_id,json=peerId,proto3" json:"peer_id,omitempty"`
	// Whether client request is ws, includes ws and wss
	IsWsRequest bool `protobuf:"varint,4,opt,name=is_ws_request,json=isWsRequest,proto3" json:"is_ws_request,omitempty"`
	// raw request data, serialized from fasthttp request
	RawRequest []byte `protobuf:"bytes,10,opt,name=raw_request,json=rawRequest,proto3" json:"raw_request,omitempty"`
}

func (x *ApronServiceRequest) Reset() {
	*x = ApronServiceRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronServiceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronServiceRequest) ProtoMessage() {}

func (x *ApronServiceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronServiceRequest.ProtoReflect.Descriptor instead.
func (*ApronServiceRequest) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{5}
}

func (x *ApronServiceRequest) GetServiceId() string {
	if x != nil {
		return x.ServiceId
	}
	return ""
}

func (x *ApronServiceRequest) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *ApronServiceRequest) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

func (x *ApronServiceRequest) GetIsWsRequest() bool {
	if x != nil {
		return x.IsWsRequest
	}
	return false
}

func (x *ApronServiceRequest) GetRawRequest() []byte {
	if x != nil {
		return x.RawRequest
	}
	return nil
}

type ApronServiceData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RequestId string `protobuf:"bytes,2,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"`
	RawData   []byte `protobuf:"bytes,10,opt,name=raw_data,json=rawData,proto3" json:"raw_data,omitempty"`
}

func (x *ApronServiceData) Reset() {
	*x = ApronServiceData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_models_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApronServiceData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApronServiceData) ProtoMessage() {}

func (x *ApronServiceData) ProtoReflect() protoreflect.Message {
	mi := &file_models_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApronServiceData.ProtoReflect.Descriptor instead.
func (*ApronServiceData) Descriptor() ([]byte, []int) {
	return file_models_proto_rawDescGZIP(), []int{6}
}

func (x *ApronServiceData) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *ApronServiceData) GetRawData() []byte {
	if x != nil {
		return x.RawData
	}
	return nil
}

var File_models_proto protoreflect.FileDescriptor

var file_models_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x99,
	0x01, 0x0a, 0x0b, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x41, 0x70, 0x69, 0x4b, 0x65, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12,
	0x1b, 0x0a, 0x09, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x08, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x41, 0x74, 0x12, 0x1d, 0x0a, 0x0a,
	0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x64, 0x41, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x22, 0xe2, 0x01, 0x0a, 0x14, 0x41,
	0x70, 0x72, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x76, 0x69,
	0x64, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x1d, 0x0a, 0x0a, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x75, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x65, 0x78, 0x74,
	0x72, 0x61, 0x5f, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0b, 0x65, 0x78, 0x74, 0x72, 0x61, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x12, 0x19, 0x0a, 0x08,
	0x62, 0x61, 0x73, 0x65, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x14, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x62, 0x61, 0x73, 0x65, 0x55, 0x72, 0x6c, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d,
	0x61, 0x18, 0x15, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x22,
	0xdd, 0x01, 0x0a, 0x0c, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x6f, 0x67, 0x6f,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x6f, 0x67, 0x6f, 0x12, 0x14, 0x0a, 0x05,
	0x75, 0x73, 0x61, 0x67, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x75, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x33, 0x0a, 0x09, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x15,
	0x2e, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f,
	0x76, 0x69, 0x64, 0x65, 0x72, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73,
	0x12, 0x1d, 0x0a, 0x0a, 0x69, 0x73, 0x5f, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x18, 0x0b,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x69, 0x73, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x22,
	0x21, 0x0a, 0x09, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x55, 0x73, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05,
	0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61,
	0x69, 0x6c, 0x22, 0x9b, 0x01, 0x0a, 0x09, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x4c, 0x6f, 0x67,
	0x12, 0x0e, 0x0a, 0x02, 0x74, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x74, 0x73,
	0x12, 0x21, 0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4e,
	0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x75, 0x73, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x12, 0x1d,
	0x0a, 0x0a, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x69, 0x70, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x70, 0x12, 0x21, 0x0a,
	0x0c, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x61, 0x74, 0x68,
	0x22, 0xb1, 0x01, 0x0a, 0x13, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x69,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x22, 0x0a, 0x0d, 0x69, 0x73, 0x5f, 0x77, 0x73, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x69, 0x73, 0x57, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x61, 0x77, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x72, 0x61, 0x77, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x4c, 0x0a, 0x10, 0x41, 0x70, 0x72, 0x6f, 0x6e, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x72, 0x61, 0x77, 0x5f, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x72, 0x61, 0x77, 0x44, 0x61,
	0x74, 0x61, 0x42, 0x1e, 0x5a, 0x1c, 0x61, 0x70, 0x72, 0x6f, 0x6e, 0x2e, 0x6e, 0x65, 0x74, 0x77,
	0x6f, 0x72, 0x6b, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_models_proto_rawDescOnce sync.Once
	file_models_proto_rawDescData = file_models_proto_rawDesc
)

func file_models_proto_rawDescGZIP() []byte {
	file_models_proto_rawDescOnce.Do(func() {
		file_models_proto_rawDescData = protoimpl.X.CompressGZIP(file_models_proto_rawDescData)
	})
	return file_models_proto_rawDescData
}

var file_models_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_models_proto_goTypes = []interface{}{
	(*ApronApiKey)(nil),          // 0: ApronApiKey
	(*ApronServiceProvider)(nil), // 1: ApronServiceProvider
	(*ApronService)(nil),         // 2: ApronService
	(*ApronUser)(nil),            // 3: ApronUser
	(*AccessLog)(nil),            // 4: AccessLog
	(*ApronServiceRequest)(nil),  // 5: ApronServiceRequest
	(*ApronServiceData)(nil),     // 6: ApronServiceData
}
var file_models_proto_depIdxs = []int32{
	1, // 0: ApronService.providers:type_name -> ApronServiceProvider
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_models_proto_init() }
func file_models_proto_init() {
	if File_models_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_models_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronApiKey); i {
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
		file_models_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronServiceProvider); i {
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
		file_models_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronService); i {
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
		file_models_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronUser); i {
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
		file_models_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AccessLog); i {
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
		file_models_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronServiceRequest); i {
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
		file_models_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApronServiceData); i {
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
			RawDescriptor: file_models_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_models_proto_goTypes,
		DependencyIndexes: file_models_proto_depIdxs,
		MessageInfos:      file_models_proto_msgTypes,
	}.Build()
	File_models_proto = out.File
	file_models_proto_rawDesc = nil
	file_models_proto_goTypes = nil
	file_models_proto_depIdxs = nil
}
